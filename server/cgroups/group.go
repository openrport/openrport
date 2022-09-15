package cgroups

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/cloudradar-monitoring/rport/share/types"
)

type ClientGroup struct {
	ID                string            `json:"id" db:"id"`
	Description       string            `json:"description" db:"description"`
	Params            *ClientParams     `json:"params" db:"params"`
	AllowedUserGroups types.StringSlice `json:"allowed_user_groups" db:"allowed_user_groups"`
	// ClientIDs shows what clients belong to a given group. Note: it's populated separately.
	ClientIDs []string `json:"client_ids" db:"-"`
}

type ClientParams struct {
	ClientID     *ParamValues `json:"client_id"`
	Name         *ParamValues `json:"name"`
	OS           *ParamValues `json:"os"`
	OSArch       *ParamValues `json:"os_arch"`
	OSFamily     *ParamValues `json:"os_family"`
	OSKernel     *ParamValues `json:"os_kernel"`
	Hostname     *ParamValues `json:"hostname"`
	IPv4         *ParamValues `json:"ipv4"`
	IPv6         *ParamValues `json:"ipv6"`
	Tag          *ParamValues `json:"tag"`
	Version      *ParamValues `json:"version"`
	Address      *ParamValues `json:"address"`
	ClientAuthID *ParamValues `json:"client_auth_id"`
}

type Param string
type ParamValues []Param

func (p *ParamValues) MatchesOneOf(values ...string) bool {
	if p == nil || len(*p) == 0 && len(values) == 0 {
		return true
	}

	for _, curParam := range *p {
		for _, curValue := range values {
			if curParam.matches(curValue) {
				return true
			}
		}
	}
	return false
}

func (p Param) matches(value string) bool {
	str := strings.ToLower(string(p))
	value = strings.ToLower(value)
	if strings.Contains(str, "*") {
		parts := strings.Split(str, "*")
		if !strings.HasPrefix(value, parts[0]) || !strings.HasSuffix(value, parts[len(parts)-1]) {
			return false
		}

		for _, part := range parts {
			i := strings.Index(value, part)
			if i == -1 {
				return false
			}
			value = value[(i + len(part)):]
		}

		return true
	}

	return str == value
}

func (p *ClientParams) Scan(value interface{}) error {
	if p == nil {
		return errors.New("'params' cannot be nil")
	}
	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected to have string, got %T", value)
	}
	err := json.Unmarshal([]byte(valueStr), p)
	if err != nil {
		return fmt.Errorf("failed to decode 'params' field: %v", err)
	}
	return nil
}

func (p *ClientParams) Value() (driver.Value, error) {
	if p == nil {
		return nil, errors.New("'params' cannot be nil")
	}
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to encode 'params' field: %v", err)
	}
	return string(b), nil
}

var noParams ClientParams

func (p *ClientParams) HasNoParams() bool {
	if p == nil {
		return true
	}
	return reflect.DeepEqual(*p, noParams)
}

func (g *ClientGroup) UserGroupIsAllowed(requiredUserGroup string) bool {
	for _, AllowedUserGroup := range g.AllowedUserGroups {
		if AllowedUserGroup == requiredUserGroup {
			return true
		}
	}
	return false
}

func (g *ClientGroup) OneOfUserGroupsIsAllowed(userGroups []string) bool {
	for _, userGroup := range userGroups {
		if g.UserGroupIsAllowed(userGroup) {
			return true
		}
	}
	return false
}
