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
		TunnelsRestricted:  tr,
		CommandsRestricted: cr,
		Permissions:        NewPermissions(perms...),
	}
}
