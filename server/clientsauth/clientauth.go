package clientsauth

var SupportedFilters = map[string]bool{
	"id": true,
}

var SupportedSorts = map[string]bool{}

var SupportedFields = map[string]map[string]bool{
	"clients-auth": {
		"id": true,
	},
}

var ListDefaultFields = map[string][]string{
	"fields[clients-auth]": {
		"id",
	},
}

// ClientAuth represents rport client authentication credentials.
type ClientAuth struct {
	ID       string `json:"id" db:"id"`
	Password string `json:"password" db:"password"`
}
