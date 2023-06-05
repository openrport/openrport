package simpleops

func Find[T any](list []T, matcher func(T) bool) (T, bool) {
	for _, e := range list {
		if matcher(e) {
			return e, true
		}
	}

	var m T
	return m, false
}
