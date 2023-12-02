package usermanagerclient

import "time"

type ViewUser struct {
	Id                      int32     `json:"id"`
	FistName                string    `json:"firstname"`
	LastName                string    `json:"lastname"`
	Username                string    `json:"username"`
	Email                   string    `json:"email"`
	CreatedAt               time.Time `json:"created_at"`
	SubscriptionPlan        string    `json:"native_subscription_plan"`
	NativeAccess            bool      `json:"native_access"`
	MobileAccess            bool      `json:"mobile_access"`
	Notes                   string    `json:"notes"`
	Vertical                string    `json:"vertical"`
	Subusers                int       `json:"sub_users"`
	ConnectedTrafficSources string    `json:"connected_traffic_sources"`
	Currencies              string    `json:"currencies"`
	ConnectedTrackers       string    `json:"connected_trackers"`
}
