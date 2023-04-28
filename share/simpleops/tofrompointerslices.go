package simpleops

func ToPointerSlice[T any](slice []T) []*T {
	tmp := make([]*T, len(slice))
	for k, v := range slice {
		tmp[k] = &v
	}
	return tmp
}

func ToValueSlice[T any](slice []*T) []T {
	tmp := make([]T, len(slice))
	for k, v := range slice {
		tmp[k] = *v
	}
	return tmp
}
