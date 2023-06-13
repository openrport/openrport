package extendedpermission

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"plugin"
	"regexp"
	"strings"
	"time"

	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/share/logger"
)

type CapabilityEx interface {
	ValidateExtendedTunnelPermission(r *http.Request, tr []PermissionParams) error
	ValidateExtendedCommandPermission(r *http.Request, cr []PermissionParams) error
	ValidateExtendedCommandPermissionRaw(command string, isSudo bool, cr []PermissionParams) error
	ValidateExtendedDeleteNonOwnedTunnelPermissionRaw(tr []PermissionParams) error
}

type Config struct {
}

type Capability struct {
	Provider CapabilityEx

	Config *Config
	*logger.Logger
}

const (
	InitPlusExtendedPermissionCapabilityEx = "InitPlusExtendedPermissionCapabilityEx"
)

func (cap *Capability) GetInitFuncName() (name string) {
	return InitPlusExtendedPermissionCapabilityEx
}

func (cap *Capability) InitProvider(sym plugin.Symbol) {
	fn := sym.(func(cap *Capability) (capProvider CapabilityEx))
	cap.Provider = fn(cap)
}

func (cap *Capability) GetExtendedPermissionCapabilityEx() (capEx CapabilityEx) {
	return cap.Provider
}

func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return nil
}

type PermissionParams map[string]interface{}

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

func parseMinutes(m interface{}) (*time.Duration, error) {
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

func (m PermissionParams) Value() (driver.Value, error) {
	m = lowercaseKeys(m)
	for pName := range m {
		switch restriction := m[pName].(type) {
		case bool:
			break
		case string: // like with true or false but if the param content matches the regular expression
			_, err := regexp.Compile(restriction)
			if err != nil {
				return nil, fmt.Errorf("invalid restriction regular expression %q: %v", restriction, err)
			}
		case []interface{}: // [ "stuff", "like" "this" ]
			for _, restriction := range m[pName].([]interface{}) {
				switch restriction := restriction.(type) {
				case string: // need to check if are all strings
					if (pName == "allow") || (pName == "deny") {
						_, err := regexp.Compile(restriction)
						if err != nil {
							return nil, fmt.Errorf("invalid restriction regular expression %q: %v", restriction, err)
						}
					}
				default:
					return nil, fmt.Errorf("invalid restriction list %v of type %T", restriction, restriction)
				}
			}
		case map[string]interface{}: // stuff like this { "max": "60m", "min": "5m" }
			for rule := range restriction {
				if (rule != "max") && (rule != "min") {
					return nil, fmt.Errorf("invalid restriction rule '%v'", rule)
				}
				_, err := parseMinutes(restriction[rule])
				if err != nil {
					return nil, fmt.Errorf("restriction %v not parseable as time.duration: %v", restriction[rule], err)
				}
			}
		default:
			return nil, fmt.Errorf("restriction %v of type %T not recognized", m[pName], m[pName])
		}
	}
	if len(m) == 0 {
		return nil, nil
	}
	j, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return driver.Value(j), nil
}

func (m *PermissionParams) Scan(src interface{}) error {
	var source []byte
	_m := make(map[string]interface{})

	switch src := src.(type) {
	case string:
		source = []byte(src)
	case []uint8:
		source = src
	case nil:
		return nil
	default:
		return fmt.Errorf("incompatible type %T for PermissionParams", src)
	}
	err := json.Unmarshal(source, &_m)
	if err != nil {
		return err
	}
	*m = PermissionParams(_m)
	return nil
}
