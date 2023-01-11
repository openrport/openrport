package enums

type APITokenScope string

const (
	APITokenRead        APITokenScope = "read"
	APITokenReadWrite   APITokenScope = "read+write"
	APITokenClientsAuth APITokenScope = "clients-auth"
)
