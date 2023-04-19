package formatter

import (
	"fmt"
	"reflect"
	"strings"
)

func BuildTranslationTable(value any) map[string]int {
	tmp := map[string]int{}

	reflectedValue := reflect.ValueOf(value)
	if reflectedValue.Kind() == reflect.Pointer { // if pointer given resolve pointer
		reflectedValue = reflectedValue.Elem()
	}

	typeOf := reflectedValue.Type()

	for i := 0; i < reflectedValue.NumField(); i++ {
		reflectedField := typeOf.Field(i)
		jsonTag, _ := reflectedField.Tag.Lookup("json")
		jsonParts := strings.Split(jsonTag, ",")

		jsonAlias := jsonTag
		if len(jsonParts) != 0 {
			jsonAlias = jsonParts[0]
		}

		if len(jsonAlias) > 0 && jsonAlias != "-" {
			tmp[jsonAlias] = i
		} else if len(jsonAlias) == 0 {
			tmp[reflectedField.Name] = i
		}

	}

	return tmp
}

func BuildTranslator(translationTable map[string]int, fields []string) ([]Field, error) {
	tmp := make([]Field, len(fields))
	for i, field := range fields {
		id, found := translationTable[field]
		if !found {
			return nil, fmt.Errorf("requested field does not exist: %v", field)
		}

		tmp[i].Id = id
		tmp[i].Name = field
	}
	return tmp, nil
}

type Field struct {
	Id   int
	Name string
}

func Format(fields []Field, value any) map[string]interface{} {
	reflectedValue := reflect.ValueOf(value)
	if reflectedValue.Kind() == reflect.Pointer { // if pointer given resolve pointer
		reflectedValue = reflectedValue.Elem()
	}

	tmp := make(map[string]interface{}, len(fields))

	for _, field := range fields {
		tmp[field.Name] = reflectedValue.Field(field.Id).Interface()
	}
	return tmp
}

type Formatter struct {
	translationTable map[string]int
}

type Translator struct {
	fields []Field
}

func (t Translator) Format(object any) map[string]interface{} {
	return Format(t.fields, object)
}

func (f Formatter) NewTranslator(fields []string) (Translator, error) {
	tr, err := BuildTranslator(f.translationTable, fields)
	return Translator{fields: tr}, err
}

func NewFormatter(object any) Formatter {
	return Formatter{translationTable: BuildTranslationTable(object)}
}
