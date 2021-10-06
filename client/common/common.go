package common

import "math"

func RoundToTwoDecimalPlaces(v float64) float64 {
	return math.Round(v*100) / 100
}

// StrInSlice returns true if search string found in slice
func StrInSlice(search string, slice []string) bool {
	for _, str := range slice {
		if str == search {
			return true
		}
	}
	return false
}
