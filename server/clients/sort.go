package clients

import "sort"

func SortByID(a []*Client, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		less := a[i].ID < a[j].ID
		if desc {
			return !less
		}
		return less
	})
}

func SortByName(a []*Client, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		less := a[i].Name < a[j].Name || a[i].Name == a[j].Name && a[i].ID < a[j].ID
		if desc {
			return !less
		}
		return less
	})
}

func SortByOS(a []*Client, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		less := a[i].OS < a[j].OS || a[i].OS == a[j].OS && a[i].ID < a[j].ID
		if desc {
			return !less
		}
		return less
	})
}

func SortByHostname(a []*Client, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		less := a[i].Hostname < a[j].Hostname || a[i].Hostname == a[j].Hostname && a[i].ID < a[j].ID
		if desc {
			return !less
		}
		return less
	})
}

func SortByVersion(a []*Client, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		less := a[i].Version < a[j].Version || a[i].Version == a[j].Version && a[i].ID < a[j].ID
		if desc {
			return !less
		}
		return less
	})
}
