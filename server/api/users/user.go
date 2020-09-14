package users

// User represents API user.
type User struct {
	Username string
	Password string
	Groups   []string
}
