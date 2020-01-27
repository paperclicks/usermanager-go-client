package usermanagerclient

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bitbucket.org/paperclicks/ms-go-database/database/model"
	"bitbucket.org/paperclicks/ms-go-database/database/model/usermanager"

	"bitbucket.org/paperclicks/ms-go-database/database/postgres"

	"github.com/paperclicks/golog"
	//needed for mysql
	_ "github.com/go-sql-driver/mysql"
)

//UserManager is a concrete instance of usermanager package
type UserManager struct {
	Gologger *golog.Golog
	DB       *postgres.Database
	URL      string
	Username string
	Password string
	Token    string
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
func New(db *postgres.Database, APIUrl string, APIUser string, APIPass string, gl *golog.Golog) *UserManager {

	return &UserManager{DB: db, URL: APIUrl, Username: APIPass, Password: APIPass, Gologger: gl}
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
func (umg *UserManager) GetUsersFromAPI() (map[string]*usermanager.User, error) {

	users := make(map[string]*usermanager.User)

	//first try to get a token
	err := umg.login()
	if err != nil {
		return users, err
	}

	client := &http.Client{Transport: tr}

	URL := umg.URL + "/users"

	req, err := http.NewRequest("GET", URL, nil)

	if err != nil {
		return users, err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", umg.Token))

	//umg.Gologger.Debug("REQUEST: %v", req)

	resp, err := client.Do(req)

	if err != nil {
		return users, err
	}

	defer resp.Body.Close()

	var response []usermanager.User

	err = json.NewDecoder(resp.Body).Decode(&response)

	if err != nil {
		return users, err
	}

	//create a map from the users array, and return the map. This is done because it is simpler to find a user using maps
	for _, v := range response {

		users[v.Username] = &v
	}

	return users, nil
}

//GetUsersFromDB returns a map of users from DB, having username as key
func (umg *UserManager) GetUsersFromDB(conditions []model.Condition) (map[string]*usermanager.User, error) {

	users := make(map[string]*usermanager.User)

	uc := usermanager.UserCollection{}

	umg.DB.Select(&usermanager.User{}, conditions, &uc, 10000)

	for _, u := range uc.Collection {

		users[u.Username] = &u
	}

	return users, nil
}

//GetUserFromDB returns a single user from DB based on the username
func (umg *UserManager) GetUserFromDB(username string) (usermanager.User, error) {
	user := usermanager.User{}

	err := umg.DB.GetByField(&user, "username", username)
	if err != nil {
		return user, err
	}

	return user, nil

}

//GetUsers returns a list of users; It can use the API or the DB based on the value of the env variable USER_MANAGER_USE_DB
// if USER_MANAGER_USE_DB = true it will use the DB directly else will use the API
func (umg *UserManager) GetUsers(useDB bool, conditions []model.Condition) (map[string]*usermanager.User, error) {

	if useDB {
		return umg.GetUsersFromDB(conditions)
	}

	return umg.GetUsersFromAPI()
}

//UpsertUser updates an existing user or creates a new one
func (umg *UserManager) UpsertUser(user usermanager.User) error {
	err := umg.DB.Upsert(&user)
	if err != nil {
		return err
	}

	return nil
}

//UpsertUserRole updates an existing user or creates a new one
func (umg *UserManager) UpsertUserRole(user usermanager.User, roleID int32) error {
	userRole := usermanager.UserRolePivot{UserID: user.ID, RoleID: roleID}
	err := umg.DB.Upsert(&userRole)
	if err != nil {
		return err
	}

	return nil
}
