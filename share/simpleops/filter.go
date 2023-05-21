package simpleops

func FilterSlice[T any](slice []T, cond func(T) bool) []T {
	var tmp []T
	for _, entry := range slice {
		if cond(entry) {
			tmp = append(tmp, entry)
		}
	}
	return tmp
}
