package jobs

import (
	"sort"

	"github.com/cloudradar-monitoring/rport/share/models"
)

// SortByFinishedAt if desc sets at first jobs without finished time (always sorted by JID in ASC order),
// then jobs sorted by finished time. If asc - sets jobs without finished time at the end.
func SortByFinishedAt(a []*models.JobSummary, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		var less bool
		switch {
		case a[i].FinishedAt == nil && a[j].FinishedAt == nil:
			less = a[i].JID > a[j].JID
		case a[i].FinishedAt == nil:
			less = false
		case a[j].FinishedAt == nil:
			less = true
		default:
			less = a[i].FinishedAt.Before(*a[j].FinishedAt)
		}

		if desc {
			return !less
		}
		return less
	})
}
