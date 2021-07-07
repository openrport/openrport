package collections

type StringBoolMap map[string]bool

// Has returns true if a map contains a given key with 'true' value.
func (m StringBoolMap) Has(key string) bool {
	return m[key]
}

func ConvertToStringBoolMap(list []string) StringBoolMap {
	m := make(map[string]bool)
	for _, cur := range list {
		m[cur] = true
	}
	return m
}
