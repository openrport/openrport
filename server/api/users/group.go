package users

import (
	plusprm "github.com/realvnc-labs/rport/plus/capabilities/permission"
)

const (
	Administrators = "Administrators"
)

var AdministratorsGroup = Group{
	Name:        Administrators,
	Permissions: NewPermissions(AllPermissions...),
}

type (
	Group struct {
		Name               string                      `json:"name" db:"name"`
		Permissions        Permissions                 `json:"permissions" db:"permissions"`
		TunnelsRestricted  *plusprm.StringInterfaceMap `json:"tunnels_restricted" db:"tunnels_restricted"`
		CommandsRestricted *plusprm.StringInterfaceMap `json:"commands_restricted" db:"commands_restricted"`
	}
)

func NewGroup(name string, tr *plusprm.StringInterfaceMap, cr *plusprm.StringInterfaceMap, perms ...string) Group {
	if name == Administrators {
		return AdministratorsGroup
	}
	return Group{
		Name:               name,
		TunnelsRestricted:  tr,
		CommandsRestricted: cr,
		Permissions:        NewPermissions(perms...),
	}
}
