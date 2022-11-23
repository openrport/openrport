package chconfig

import (
	"reflect"

	"github.com/gobeam/stringy"
	"github.com/oleiade/reflections"
)

// ConfigReplaceDeprecated maps deprecated config settings to their new replacements.
func ConfigReplaceDeprecated(s any) (map[string]string, error) {
	replaced := make(map[string]string)
	var structTags map[string]string
	structTags, err := reflections.Tags(s, "replaced_by")
	if err != nil {
		return replaced, err
	}
	for field, tag := range structTags {
		if tag == "" {
			continue
		}
		// Get the value of the struct field
		value, err := reflections.GetField(s, field)
		if err != nil {
			return replaced, err
		}
		// If value is the default value, the deprecated setting isn't used. So skip it.
		if reflect.ValueOf(value).IsZero() {
			continue
		}
		// map the value of the deprecated setting to the new one
		err = reflections.SetField(s, tag, value)
		if err != nil {
			return replaced, err
		}
		// Create a list of replaced settings to be included in the log
		scField := stringy.New(field).SnakeCase().ToLower()
		scReplacement := stringy.New(tag).SnakeCase().ToLower()
		replaced[scField] = scReplacement
	}
	return replaced, nil
}
