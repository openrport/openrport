package chshare

import (
	mapset "github.com/deckarep/golang-set"
)

func SetFromRange(start, end int) mapset.Set {
	s := mapset.NewThreadUnsafeSet()
	for i := 0; i <= end-start; i++ {
		s.Add(start + i)
	}
	return s
}
