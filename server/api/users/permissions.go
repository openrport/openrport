package users

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	PermissionTunnels    = "tunnels"
	PermissionScripts    = "scripts"
	PermissionCommands   = "commands"
	PermissionVault      = "vault"
	PermissionScheduler  = "scheduler"
	PermissionMonitoring = "monitoring"
	PermissionUploads    = "uploads"
	PermissionsAuditLog  = "auditlog"
)

var AllPermissions = []string{
	PermissionTunnels,
	PermissionScripts,
	PermissionCommands,
	PermissionVault,
	PermissionScheduler,
	PermissionMonitoring,
	PermissionUploads,
	PermissionsAuditLog,
}

type Permissions struct {
	data map[string]bool
}

func NewPermissions(perms ...string) Permissions {
	permissions := Permissions{
		data: make(map[string]bool),
	}
	for _, p := range perms {
		permissions.data[p] = true
	}
	return permissions
}

func (permissions Permissions) All() map[string]bool {
	result := make(map[string]bool)
	for _, p := range AllPermissions {
		result[p] = permissions.Has(p)
	}
	return result
}

func (permissions Permissions) Has(p string) bool {
	if permissions.data == nil {
		return false
	}
	return permissions.data[p]
}

func (permissions *Permissions) Scan(value interface{}) error {
	if permissions == nil {
		return errors.New("'permissions' cannot be nil")
	}

	var err error
	switch d := value.(type) {
	case string:
		// Handle sqlite json decoding
		if d == "" {
			return nil
		}
		err = json.Unmarshal([]byte(d), &permissions.data)
	case []uint8:
		// Handle MySQL json decoding
		if len(d) == 0 {
			return nil
		}
		err = json.Unmarshal(d, &permissions.data)
	default:
		return fmt.Errorf("failed to decode json column: unknown comlumn type %T", value)
	}

	if err != nil {
		return fmt.Errorf("failed to decode 'permissions' field: %v", err)
	}
	return nil
}

func (permissions Permissions) Value() (driver.Value, error) {
	b, err := json.Marshal(permissions.data)
	if err != nil {
		return nil, fmt.Errorf("failed to encode 'permissions' field: %v", err)
	}
	return string(b), nil
}

func (permissions Permissions) MarshalJSON() ([]byte, error) {
	return json.Marshal(permissions.All())
}

func (permissions *Permissions) UnmarshalJSON(data []byte) error {
	result := make(map[string]bool)
	err := json.Unmarshal(data, &result)
	if err != nil {
		return err
	}

	for key := range result {
		isPermission := false
		for _, permission := range AllPermissions {
			if key == permission {
				isPermission = true
				break
			}
		}
		if !isPermission {
			return fmt.Errorf("invalid permission: %v", key)
		}
	}

	permissions.data = result
	return nil
}
