package clients

import "sort"

// ClientAuth represents rport client authentication credentials.
type ClientAuth struct {
	ID       string `json:"id" db:"id"`
	Password string `json:"password" db:"password"`
}

func SortByID(a []*ClientAuth, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		less := a[i].ID < a[j].ID
		if desc {
			return !less
		}
		return less
	})
}
