package users

const (
	Administrators = "Administrators"
)

// User represents API user.
type User struct {
	Username    string   `json:"username" db:"username"`
	Password    string   `json:"password" db:"password"`
	Groups      []string `json:"groups" db:"-"`
	TwoFASendTo string   `json:"two_fa_send_to" db:"two_fa_send_to"`
}
