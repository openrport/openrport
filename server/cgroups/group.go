package cgroups

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type ClientGroup struct {
	ID          string        `json:"id" db:"id"`
	Description string        `json:"description" db:"description"`
	Params      *ClientParams `json:"params" db:"params"`
}

type ClientParams struct {
	ClientID     ParamValues `json:"client_id"`
	Name         ParamValues `json:"name"`
	OS           ParamValues `json:"os"`
	OSArch       ParamValues `json:"os_arch"`
	OSFamily     ParamValues `json:"os_family"`
	OSKernel     ParamValues `json:"os_kernel"`
	Hostname     ParamValues `json:"hostname"`
	IPv4         ParamValues `json:"ipv4"`
	IPv6         ParamValues `json:"ipv6"`
	Tag          ParamValues `json:"tag"`
	Version      ParamValues `json:"version"`
	Address      ParamValues `json:"address"`
	ClientAuthID ParamValues `json:"client_auth_id"`
}

type Param string
type ParamValues []Param

func (p ParamValues) MatchesOneOf(values ...string) bool {
	if len(values) == 0 || len(p) == 0 {
		return true
	}

	for _, curParam := range p {
		for _, curValue := range values {
			if curParam.matches(curValue) {
				return true
			}
		}
	}
	return false
}

func (p Param) matches(value string) bool {
	if value == "" {
		return true
	}

	str := string(p)
	if len(str) == 0 {
		return false
	}

	if strings.HasSuffix(str, "*") {
		return strings.Contains(value, str[:len(str)-1])
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
