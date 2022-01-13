package clients

import (
	"sort"
	"strings"
)

func SortByID(a []*CalculatedClient, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		less := strings.ToLower(a[i].ID) < strings.ToLower(a[j].ID)
		if desc {
			return !less
		}
		return less
	})
}

func SortByName(a []*CalculatedClient, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		aiName := strings.ToLower(a[i].Name)
		ajName := strings.ToLower(a[j].Name)
		less := aiName < ajName || aiName == ajName && strings.ToLower(a[i].ID) < strings.ToLower(a[j].ID)
		if desc {
			return !less
		}
		return less
	})
}

func SortByOS(a []*CalculatedClient, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		aiOS := strings.ToLower(a[i].OS)
		ajOS := strings.ToLower(a[j].OS)
		less := aiOS < ajOS || aiOS == ajOS && strings.ToLower(a[i].ID) < strings.ToLower(a[j].ID)
		if desc {
			return !less
		}
		return less
	})
}

func SortByHostname(a []*CalculatedClient, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		aiHostname := strings.ToLower(a[i].Hostname)
		ajHostname := strings.ToLower(a[j].Hostname)
		less := aiHostname < ajHostname || aiHostname == ajHostname && strings.ToLower(a[i].ID) < strings.ToLower(a[j].ID)
		if desc {
			return !less
		}
		return less
	})
}

func SortByVersion(a []*CalculatedClient, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		aiVersion := strings.ToLower(a[i].Version)
		ajVersion := strings.ToLower(a[j].Version)
		less := aiVersion < ajVersion || aiVersion == ajVersion && strings.ToLower(a[i].ID) < strings.ToLower(a[j].ID)
		if desc {
			return !less
		}
		return less
	})
}
