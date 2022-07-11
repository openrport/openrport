package users

const (
	Administrators = "Administrators"
)

var AdministratorsGroup = Group{
	Name:        Administrators,
	Permissions: NewPermissions(AllPermissions...),
}

type Group struct {
	Name        string      `json:"name" db:"name"`
	Permissions Permissions `json:"permissions" db:"permissions"`
}

func NewGroup(name string) Group {
	if name == Administrators {
		return AdministratorsGroup
	}
	return Group{
		Name:        name,
		Permissions: NewPermissions(),
	}
}
