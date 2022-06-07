package clientsauth

var SupportedFilters = map[string]bool{
	"id": true,
}

var SupportedSorts = map[string]bool{}

// ClientAuth represents rport client authentication credentials.
type ClientAuth struct {
	ID       string `json:"id" db:"id"`
	Password string `json:"password" db:"password"`
}
