package users

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
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

func lowercaseKeys(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		lowercaseKey := strings.ToLower(k)
		switch value := v.(type) {
		case map[string]interface{}:
			result[lowercaseKey] = lowercaseKeys(value)
		default:
			result[lowercaseKey] = value
		}
	}
	return result
}

func (m StringInterfaceMap) Value() (driver.Value, error) {
	m = lowercaseKeys(m)
	for pName := range m {
		switch m[pName].(type) {
		case bool:
			break
		case string: // like with true or false but if the param content matches the regular expression
			restriction := m[pName].(string)
			_, err := regexp.Compile(restriction)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("invalid restriction regular expression %q: %v", restriction, err))
			}
			break //
		case []interface{}: // [ "stuff", "like" "this" ]
			fmt.Printf("[]interface{} %v\n", m[pName])
			for _, restriction := range m[pName].([]interface{}) {
				switch restriction.(type) {
				case string: // need to check if are all strings
					if (pName == "allow") || (pName == "deny") {
						_, err := regexp.Compile(restriction.(string))
						if err != nil {
							return nil, errors.New(fmt.Sprintf("invalid restriction regular expression %q: %v", restriction, err))
						}
					}
					break
				default:
					return nil, errors.New(fmt.Sprintf("invalid restriction list %v of type %T", restriction, restriction))
				}
			}
			break
		case map[string]interface{}: // stuff like this { "max": "60m", "min": "5m" }
			restriction := m[pName].(map[string]interface{})
			for rule := range restriction {
				if (rule != "max") && (rule != "min") {
					return nil, errors.New(fmt.Sprintf("invalid restriction rule '%v'", rule))
				}
				_, err := ParseMinutes(restriction[rule])
				if err != nil {
					return nil, errors.New(fmt.Sprintf("restriction %v not parseable as time.duration: %v", restriction[rule], err))
				}
			}
			break
		default:
			return nil, errors.New(fmt.Sprintf("restriction %v of type %T not recognized", m[pName], m[pName]))
		}
	}
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

// ED TODO: this ALSO INSIDE PLUS!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
func ParseMinutes(m interface{}) (*time.Duration, error) {
	parseable := fmt.Sprintf("%v", m)
	dur, err := time.ParseDuration(parseable)
	if err != nil {
		parseable = fmt.Sprintf("%vm", m)
		dur, err = time.ParseDuration(parseable)
		if err != nil {
			return nil, errors.New("invalid type")
		}
		return &dur, nil
	}
	return &dur, nil
}
