package ptr

import "time"

func Time(t time.Time) *time.Time {
	return &t
}

func Bool(b bool) *bool {
	return &b
}

func String(s string) *string {
	return &s
}

func Int(i int) *int {
	return &i
}
