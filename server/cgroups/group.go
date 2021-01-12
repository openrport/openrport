package cgroups

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type ClientGroup struct {
	ID          string        `json:"id" db:"id"`
	Description string        `json:"description" db:"description"`
	Params      *ClientParams `json:"params" db:"params"`
}

type ClientParams struct {
	ClientID     []string `json:"client_id"`
	Name         []string `json:"name"`
	OS           []string `json:"os"`
	OSArch       []string `json:"os_arch"`
	OSFamily     []string `json:"os_family"`
	OSKernel     []string `json:"os_kernel"`
	Hostname     []string `json:"hostname"`
	IPv4         []string `json:"ipv4"`
	IPv6         []string `json:"ipv6"`
	Tag          []string `json:"tag"`
	Version      []string `json:"version"`
	Address      []string `json:"address"`
	ClientAuthID []string `json:"client_auth_id"`
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
