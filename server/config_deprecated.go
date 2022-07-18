package chserver

import (
	"time"

	"github.com/gobeam/stringy"
	"github.com/oleiade/reflections"
)

// ServerConfigReplaceDeprecated maps deprecated config settings to their new replacements.
func ServerConfigReplaceDeprecated(s ServerConfig) (ServerConfig, map[string]string, error) {
	replaced := make(map[string]string)
	var structTags map[string]string
	structTags, err := reflections.Tags(s, "replaced_by")
	if err != nil {
		return s, replaced, err
	}
	for field, tag := range structTags {
		if tag == "" {
			continue
		}
		// Get the value of the struct field
		value, err := reflections.GetField(s, field)
		if err != nil {
			return s, replaced, err
		}
		// If value is the default value, the deprecated setting isn't used. So skip it.
		switch value.(type) {
		case string:
			if value == "" {
				continue
			}
		case time.Duration:
			if value == 0*time.Second {
				continue
			}
		case bool:
			if value == false {
				continue
			}
		default:
			if value == 0 {
				continue
			}
		}
		// map the value of the deprecated setting to the new one
		err = reflections.SetField(&s, tag, value)
		if err != nil {
			return s, replaced, err
		}
		// Create a list of replaced settings to be included in the log
		scField := stringy.New(field).SnakeCase().ToLower()
		scReplacement := stringy.New(tag).SnakeCase().ToLower()
		replaced[scField] = scReplacement
	}
	return s, replaced, nil
}
