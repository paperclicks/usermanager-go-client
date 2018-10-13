package usermanager

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/paperclicks/golog"
	//needed for mysql
	_ "github.com/go-sql-driver/mysql"
)

//UserManager is a concrete instance of usermanager package
type UserManager struct {
	Gologger *golog.Golog
	DB       *sql.DB
	URL      string
	Username string
	Password string
	Token    string
}

//User represents infos about a optimizer user
type User struct {
	ID            int      `json:"id"`
	Firstname     string   `json:"firstname"`
	Lastname      string   `json:"lastname"`
	Email         string   `json:"email"`
	Username      string   `json:"username"`
	Roles         []string `json:"roles"`
	NativeAccess  bool     `json:"nativeAccess"`
	AmemberUserID int      `json:"amemberUserId"`
	MobileAccess  bool     `json:"mobileAccess"`
	Enabled       bool     `json:"enabled"`
}

//Token represents a login result
type Token struct {
	Token string `json:"token"`
}

var dialer *net.Dialer
var tr *http.Transport

func init() {

	//create a custom timout dialer
	dialer = &net.Dialer{Timeout: 30 * time.Second}

	//create a custom transport layer to use during API calls
	tr = &http.Transport{
		DialContext:         dialer.DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}
}

//New instantiates a new UserManager instance
func New(dsn string, url string, username string, password string, output io.Writer) *UserManager {

	gologger := golog.New(output)
	gologger.ShowCallerInfo = true

	token := ""

	db, err := sql.Open("mysql", dsn)

	if err != nil {
		gologger.Error("Error connecting to UserManager DB: %v", err)

		log.Fatalln("ERROR CONNECTING TO MYSQL: ", err)
	}

	return &UserManager{DB: db, URL: url, Username: username, Password: password, Token: token, Gologger: gologger}
}

func (umg *UserManager) login() error {

	client := &http.Client{Transport: tr}

	loginURL := umg.URL + "/api/login_check"

	//create form body
	form := url.Values{}
	form.Add("_username", umg.Username)
	form.Add("_password", umg.Password)

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(form.Encode()))

	if err != nil {
		umg.Gologger.Error(" Login::NewRequest - %v", err)
		return err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)

	if err != nil {
		umg.Gologger.Error(" Login::Request Error - %v", err)
		return err
	}

	defer resp.Body.Close()

	token := Token{}

	err = json.NewDecoder(resp.Body).Decode(&token)

	if err != nil {
		umg.Gologger.Error(" doPutRequest:: Decode Error - %v", err)
		return err
	}

	umg.Token = token.Token

	return nil

}

//GetUsersFromAPI return a map of all traffic source types in DB, having the unique_name as key
func (umg *UserManager) GetUsersFromAPI() (map[string]User, error) {

	users := make(map[string]User)

	//first try to get a token
	err := umg.login()
	if err != nil {
		umg.Gologger.Error(" Getusers::Authentication Error - %v", err)
		return users, err
	}

	client := &http.Client{Transport: tr}

	URL := umg.URL + "/users"

	req, err := http.NewRequest("GET", URL, nil)

	if err != nil {
		umg.Gologger.Error(" Getusers::NewRequest - %v", err)
		return users, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", umg.Token))

	//umg.Gologger.Debug("REQUEST: %v", req)

	resp, err := client.Do(req)

	if err != nil {
		umg.Gologger.Error(" GetUsers::Request Error - %v", err)
		return users, err
	}

	defer resp.Body.Close()

	var response []User

	err = json.NewDecoder(resp.Body).Decode(&response)

	if err != nil {
		umg.Gologger.Error(" GetUsers:: Decode Error - %v", err)
		return users, err
	}

	//create a map from the users array, and return the map. This is done because it is simpler to find a user using maps
	for _, v := range response {

		users[v.Username] = v
	}

	return users, nil
}

//GetUsersFromDB retrieves all users from DB
func (umg *UserManager) GetUsersFromDB() (map[string]User, error) {

	users := make(map[string]User)
	tableName := os.Getenv("USER_MANAGER_USERS_TABLE")

	q := fmt.Sprintf("SELECT id, IFNULL(amember_user_id,0), firstname, lastname, username, email, enabled, IFNULL(native_access,0), IFNULL(mobile_access,0) FROM %s", tableName)

	rows, err := umg.DB.Query(q)

	if err != nil {
		umg.Gologger.Error("An error occured while trying to get users from DB! %v", err)
		return users, err
	}

	defer rows.Close()

	for rows.Next() {

		u := User{}

		if err := rows.Scan(&u.ID, &u.AmemberUserID, &u.Firstname, &u.Lastname, &u.Username, &u.Email, &u.Enabled, &u.NativeAccess, &u.MobileAccess); err != nil {
			umg.Gologger.Error("%v", err)
			return users, err
		}

		users[u.Username] = u

	}

	return users, nil
}

//GetUsers returns a list of users; It can use the API or the DB based on the value of the env variable USER_MANAGER_USE_DB
// if USER_MANAGER_USE_DB = true it will use the DB directly else will use the API
func (umg *UserManager) GetUsers() (map[string]User, error) {

	useDB := os.Getenv("USER_MANAGER_USE_DB")

	if useDB == "true" {
		return umg.GetUsersFromDB()
	}

	return umg.GetUsersFromAPI()
}

//Status returns "OK" or "ERROR" based on the fact that the login process was successfull or not
func (umg *UserManager) Status() (string, error) {

	err := umg.login()

	if err != nil {
		return "ERROR", err
	}

	return "OK", nil
}
