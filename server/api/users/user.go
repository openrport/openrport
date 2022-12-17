package users

type APIToken struct {
	Prefix string
	Scope  string
	// EDTODO: should User know expires_at field also? (YES)
	Token string
}

// User represents API user.
type User struct {
	Username        string   `json:"username" db:"username"`
	Password        string   `json:"password" db:"password"`
	PasswordExpired *bool    `json:"password_expired" db:"password_expired"`
	Groups          []string `json:"groups" db:"-"`
	TwoFASendTo     string   `json:"two_fa_send_to" db:"two_fa_send_to"`
	Token           *[]APIToken
	TotP            string `json:"totp_secret,omitempty" db:"totp_secret"`
}

func (u User) GetGroups() []string {
	return u.Groups
}

func (u User) GetUsername() string {
	return u.Username
}

func (u User) IsAdmin() bool {
	for _, group := range u.Groups {
		if group == Administrators {
			return true
		}
	}
	return false
}

func PasswordExpired(f bool) *bool {
	return &f
}

func Token(s APIToken) *APIToken {
	return &s
}
