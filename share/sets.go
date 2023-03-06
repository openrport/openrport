package chshare

import (
	mapset "github.com/deckarep/golang-set"
)

func SetFromRange(start, end int) mapset.Set {
	s := mapset.NewSet()
	for i := 0; i <= end-start; i++ {
		s.Add(start + i)
	}
	return s
}
