package dynops

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/realvnc-labs/rport/share/dynops/dyncopy"
	"github.com/realvnc-labs/rport/share/query"
)

func SlowBasicSorter[T any](list []T, sorts []query.SortOption) []T {

	if len(sorts) == 0 || len(list) == 0 {
		return list
	}

	sort.Slice(list, func(i, j int) bool {
		a := list[i]
		b := list[j]

		ra := reflect.ValueOf(a)
		rb := reflect.ValueOf(b)

		for _, s := range sorts {

			ka := fmt.Sprintf("%v", ra.FieldByName(s.Column).Interface())
			kb := fmt.Sprintf("%v", rb.FieldByName(s.Column).Interface())

			if ka == kb {
				continue
			}

			if s.IsASC {
				return ka > kb
			}

			return ka < kb

		}

		return true
	})

	return list
}

type ts struct {
	f     dyncopy.Field
	isAsc bool
}

// FastSorter1 in case of inefficiency there are plenty optimisations possible, ask Konrad
func FastSorter1[T any](tt map[string]dyncopy.Field, list []T, sorts []query.SortOption) ([]T, error) {

	if len(sorts) == 0 || len(list) == 0 {
		return list, nil
	}

	tsorts := make([]ts, len(sorts))

	for i, s := range sorts {
		translated, ok := tt[s.Column]
		if !ok {
			return list, fmt.Errorf("can't find column for sorting: %v", s.Column)
		}
		tsorts[i].isAsc = s.IsASC
		tsorts[i].f = translated
	}

	sort.Slice(list, func(i, j int) bool {
		a := list[i]
		b := list[j]

		ra := reflect.ValueOf(a)
		rb := reflect.ValueOf(b)

		for _, s := range tsorts {
			ka := fmt.Sprintf("%v", ra.Field(s.f.ID).Interface())
			kb := fmt.Sprintf("%v", rb.Field(s.f.ID).Interface())

			if ka == kb {
				continue
			}

			if s.isAsc {
				return ka > kb
			}

			return ka < kb

		}

		return true
	})

	return list, nil
}

func Paginator[T any](list []T, pagination *query.Pagination) []T {
	if pagination == nil {
		return list
	}
	start, end := pagination.GetStartEnd(len(list))
	return list[start:end]
}
