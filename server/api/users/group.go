package users

import (
	extperm "github.com/openrport/openrport/plus/capabilities/extendedpermission"
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
		Name               string                    `json:"name" db:"name"`
		Permissions        Permissions               `json:"permissions" db:"permissions"`
		TunnelsRestricted  *extperm.PermissionParams `json:"tunnels_restricted" db:"tunnels_restricted"`
		CommandsRestricted *extperm.PermissionParams `json:"commands_restricted" db:"commands_restricted"`
	}
)

func NewGroup(name string, tr *extperm.PermissionParams, cr *extperm.PermissionParams, perms ...string) Group {
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
