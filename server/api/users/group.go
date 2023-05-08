package users

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	Administrators = "Administrators"
)

var AdministratorsGroup = Group{
	Name:        Administrators,
	Permissions: NewPermissions(AllPermissions...),
}

/*
	"commands_restricted": {
	    "allow": ["^sudo reboot$","^systemctl .* restart$"],		// I can reboot the machine. I can restart any service.
	    "deny": ["apache2","ssh"],									// I can restart any service except apache2 and ssh
	    "is_sudo": false											// I cannot use the global is_sudo switch. I can still prefix commands with sudo, if the keyword "sudo" is allowed.
	}
*/

/*
ED TODO: The list of deny and allow keywords are regular expressions.
Step 1: If the command matches against any of the deny expressions, the command is denied.
Step 2: The command must match against any of the allow expressions. Otherwise, the command is denied.

type CommandsRestricted struct {
	Allow  []string `json:"allow,omitempty"` // EDTODO: This is a regex, Using an empty list or omitting an object will remove any restrictions. For example, if allowed is not present, or if "allowed": [] then any command can be used.
	Deny   []string `json:"deny,omitempty"`  // EDTODO: If deny is missing or empty, the command is not validated against the deny patterns.
	IsSudo bool     `json:"is_sudo,omitempty"`
}

*/

type (
	StringInterfaceMap map[string]interface{}
	Group              struct {
		Name               string              `json:"name" db:"name"`
		Permissions        Permissions         `json:"permissions" db:"permissions"`
		TunnelsRestricted  *StringInterfaceMap `json:"tunnels_restricted" db:"tunnels_restricted"`
		CommandsRestricted *StringInterfaceMap `json:"commands_restricted" db:"commands_restricted"`
	}
)

// ED TODO: permissions saved in the db should be all lowercase

func (m StringInterfaceMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	j, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return driver.Value([]byte(j)), nil
}

func (m *StringInterfaceMap) Scan(src interface{}) error {
	var source []byte
	_m := make(map[string]interface{})

	switch src.(type) {
	case string:
		source = []byte(src.(string))
	case []uint8:
		source = []byte(src.([]uint8))
	case nil:
		return nil
	default:
		return errors.New(fmt.Sprintf("incompatible type %T for StringInterfaceMap", src))
	}
	err := json.Unmarshal(source, &_m)
	if err != nil {
		return err
	}
	*m = StringInterfaceMap(_m)
	return nil
}

func NewGroup(name string, tr *StringInterfaceMap, cr *StringInterfaceMap, perms ...string) Group {
	if name == Administrators {
		return AdministratorsGroup
	}
	return Group{
		Name:               name,
		Permissions:        NewPermissions(perms...),
		TunnelsRestricted:  tr,
		CommandsRestricted: cr,
	}
}
