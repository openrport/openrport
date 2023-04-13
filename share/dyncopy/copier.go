package dyncopy

import (
	"fmt"
	"reflect"
)

type FromToPair struct {
	from string
	to   string
}

type Field struct {
	id   int
	kind reflect.Type
}

func BuildTranslationTable(value any) map[string]Field {
	tmp := map[string]Field{}

	reflectedValue := reflect.ValueOf(value)
	if reflectedValue.Kind() == reflect.Pointer { // if pointer given resolve pointer
		reflectedValue = reflectedValue.Elem()
	}

	typeOf := reflectedValue.Type()

	for i := 0; i < reflectedValue.NumField(); i++ {
		reflectedField := typeOf.Field(i)
		tmp[reflectedField.Name] = Field{id: i, kind: reflectedField.Type}
	}

	return tmp
}

func NewPair(from string, to string) FromToPair {
	return FromToPair{from: from, to: to}
}

type converterPair struct {
	from int
	to   int
}

func NewCopier[FromType any, ToType any](example1 FromType, example2 ToType, pairs []FromToPair) (func(FromType, *ToType), error) {
	if len(pairs) == 0 {
		return func(from FromType, to *ToType) {}, nil
	}

	fromTT := BuildTranslationTable(example1)
	toTT := BuildTranslationTable(example2)

	converterPairs := make([]converterPair, len(pairs))

	for i, pair := range pairs {
		fromField, ok := fromTT[pair.from]
		if !ok {
			return nil, fmt.Errorf("no such field in %v: \"%v\"", reflect.TypeOf(example1), pair.from)
		}
		toField, ok := toTT[pair.to]
		if !ok {
			return nil, fmt.Errorf("no such field in %v: \"%v\"", reflect.TypeOf(example2), pair.to)
		}
		if fromField.kind != toField.kind {
			return nil, fmt.Errorf("types of fields don't align - copy impossible for fields: %v, %v", pair.from, pair.to)
		}

		converterPairs[i] = converterPair{from: fromField.id, to: toField.id}
	}

	return func(from FromType, to *ToType) {
		reflectedFrom := reflect.ValueOf(from)
		if reflectedFrom.Kind() == reflect.Pointer { // if pointer given resolve pointer
			reflectedFrom = reflectedFrom.Elem()
		}

		reflectedTo := reflect.ValueOf(to)

		for _, pair := range converterPairs {
			field := reflectedTo.Elem().Field(pair.to)
			field.Set(reflectedFrom.Field(pair.from))
		}

	}, nil

}
