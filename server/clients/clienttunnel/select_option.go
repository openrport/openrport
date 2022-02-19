package clienttunnel

type SelectOption struct {
	Value       string
	Description string
	Selected    bool
}

type SelectOptions []SelectOption

func CreateOptions(keys []string, values []string, selKey string) []SelectOption {
	var options []SelectOption
	if len(keys) < 1 || len(keys) != len(values) {
		return options
	}

	for i := 0; i < len(keys); i++ {
		o := SelectOption{
			Value:       keys[i],
			Description: values[i],
			Selected:    keys[i] == selKey,
		}
		options = append(options, o)
	}

	return options
}
