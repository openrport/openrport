package clients

import (
	"sort"
	"strings"
)

func SortByID(a []*CalculatedClient, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		less := strings.ToLower(a[i].GetID()) < strings.ToLower(a[j].GetID())
		if desc {
			return !less
		}
		return less
	})
}

func SortByName(a []*CalculatedClient, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		aiName := strings.ToLower(a[i].GetName())
		ajName := strings.ToLower(a[j].GetName())
		less := aiName < ajName || aiName == ajName && strings.ToLower(a[i].GetID()) < strings.ToLower(a[j].GetID())
		if desc {
			return !less
		}
		return less
	})
}

func SortByOS(a []*CalculatedClient, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		aiOS := strings.ToLower(a[i].GetOS())
		ajOS := strings.ToLower(a[j].GetOS())
		less := aiOS < ajOS || aiOS == ajOS && strings.ToLower(a[i].GetID()) < strings.ToLower(a[j].GetID())
		if desc {
			return !less
		}
		return less
	})
}

func SortByHostname(a []*CalculatedClient, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		aiHostname := strings.ToLower(a[i].GetHostname())
		ajHostname := strings.ToLower(a[j].GetHostname())
		less := aiHostname < ajHostname || aiHostname == ajHostname && strings.ToLower(a[i].GetID()) < strings.ToLower(a[j].GetID())
		if desc {
			return !less
		}
		return less
	})
}

func SortByVersion(a []*CalculatedClient, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		aiVersion := strings.ToLower(a[i].GetVersion())
		ajVersion := strings.ToLower(a[j].GetVersion())
		less := aiVersion < ajVersion || aiVersion == ajVersion && strings.ToLower(a[i].GetID()) < strings.ToLower(a[j].GetID())
		if desc {
			return !less
		}
		return less
	})
}
