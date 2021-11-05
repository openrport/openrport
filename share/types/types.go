package types

// JSONString is a string containing JSON encoded data, it prevents further json encoding when used inside a struct that gets encoded
type JSONString string

func (js JSONString) MarshalJSON() ([]byte, error) {
	return []byte(js), nil
}
