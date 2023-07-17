package types

import (
	"encoding/json"
	"time"
)

// TimeJSON is a wrapped time value that will marshal and unmarshal zero time to JSON
// as an empty string
type TimeJSON struct {
	time.Time
}

func NewTimeJSON(t time.Time) (nilTime TimeJSON) {
	nilTime = TimeJSON{}
	nilTime.Time = t
	return nilTime
}

func EmptyTimeJSON() (nilTime TimeJSON) {
	return NewTimeJSON(time.Time{})
}

func (et *TimeJSON) ToTime() (t time.Time) {
	return et.Time
}

func (et *TimeJSON) UnmarshalJSON(data []byte) error {
	if string(data) == `""` {
		et.Time = time.Time{}
		return nil
	}

	return json.Unmarshal(data, &et.Time)
}

func (et *TimeJSON) MarshalJSON() ([]byte, error) {
	if et.Time.IsZero() {
		return []byte(`""`), nil
	}

	return json.Marshal(et.Time)
}
