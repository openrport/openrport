package types

// JSONString is a string containing JSON encoded data, it prevents further json encoding when used inside a struct that gets encoded
// cant' use json.RawMessage because of:sql: Scan error on column index 1, name "processes": unsupported Scan, storing driver.Value type string into type *json.RawMessage
type JSONString string

func (js JSONString) MarshalJSON() ([]byte, error) {
	if js == "" {
		return []byte("null"), nil
	}
	return []byte(js), nil
}

func (js *JSONString) UnmarshalJSON(data []byte) error {
	*js = JSONString(data)
	return nil
}
