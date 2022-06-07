package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringSlice is used for storing string slice in sqlite
type StringSlice []string

func (s *StringSlice) Scan(value interface{}) error {
	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected to have string, got %T", value)
	}
	err := json.Unmarshal([]byte(valueStr), s)
	if err != nil {
		return fmt.Errorf("failed to decode string slice: %v", err)
	}
	return nil
}

func (s StringSlice) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to encode string slice: %v", err)
	}
	return string(b), nil
}
