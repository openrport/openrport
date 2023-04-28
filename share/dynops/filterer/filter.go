package filterer

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/realvnc-labs/rport/share/dynops/dyncopy"
	"github.com/realvnc-labs/rport/share/query"
)

type fieldID[T any] int

func (fid fieldID[T]) GetFieldFromObject(o T) reflect.Value {
	v := reflect.ValueOf(o)
	return v.Field(int(fid))
}

type ComparatorStringEQ[T any] struct {
	fieldID[T]
	value string
}

func (c ComparatorStringEQ[T]) Run(o T) bool {
	return c.GetFieldFromObject(o).Interface() == c.value
}

type ComparatorInt[T any] struct {
	fieldID[T]
	value    int64
	operator query.FilterOperatorType
}

func (c ComparatorInt[T]) Run(o T) bool {
	value := c.GetFieldFromObject(o)
	switch c.operator {
	case query.FilterOperatorTypeEQ:
		return c.value == value.Int()
	case query.FilterOperatorTypeGT:
		return c.value > value.Int()
	case query.FilterOperatorTypeLT:
		return c.value < value.Int()
	case query.FilterOperatorTypeSince:
		return c.value >= value.Int()
	case query.FilterOperatorTypeUntil:
		return c.value <= value.Int()
	}

	return false
}

func CompileFromQueryListOptions[T any](options *query.ListOptions) (Operation[T], error) {
	if options == nil || len(options.Filters) == 0 {
		return NewTrue[T](), nil
	}

	var proto T

	tt := dyncopy.BuildTranslationTable(proto)

	comparators := make([]Operation[T], len(options.Filters))
	for i, filter := range options.Filters {
		field, err := getColumnField(tt, filter)
		if err != nil {
			return nil, fmt.Errorf("%v on type: %v on filter: %v", err, reflect.TypeOf(proto), i)
		}

		v, err := getValue(filter)
		if err != nil {
			return nil, fmt.Errorf("%v on filter: %v", err, i)
		}

		c, err := getComparator[T](filter.Operator, field, v)
		if err != nil {
			return nil, fmt.Errorf("%v on filter: %v", err, i)
		}

		comparators[i] = c
	}
	return NewAnd[T](comparators...), nil
}

func getComparator[T any](operator query.FilterOperatorType, field dyncopy.Field, v string) (Operation[T], error) {

	switch field.Kind.Kind() {
	case reflect.String:
		return GetStringComparator[T](operator, field, v)

	case reflect.Int, reflect.Int64:
		return GetIntComparator[T](operator, field, v)
	default:
		return nil, fmt.Errorf("can't compare field, unhandled type")
	}
}

func GetIntComparator[T any](operator query.FilterOperatorType, field dyncopy.Field, v string) (Operation[T], error) {
	switch operator {
	case query.FilterOperatorTypeEQ:
	case query.FilterOperatorTypeGT:
	case query.FilterOperatorTypeLT:
	case query.FilterOperatorTypeSince:
	case query.FilterOperatorTypeUntil:
	default:
		return nil, fmt.Errorf("invalid int operator: %v", operator)
	}

	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("can't parse value: %v into int: %v", v, err)
	}

	return ComparatorInt[T]{
		operator: operator,
		fieldID:  fieldID[T](field.ID),
		value:    i,
	}, nil
}

func GetStringComparator[T any](operator query.FilterOperatorType, field dyncopy.Field, v string) (Operation[T], error) {
	switch operator {
	case query.FilterOperatorTypeEQ:
		return ComparatorStringEQ[T]{
			fieldID: fieldID[T](field.ID),
			value:   v,
		}, nil
	default:
		return nil, fmt.Errorf("invalid string operator: %v", operator)
	}
}

func getColumnField(tt map[string]dyncopy.Field, option query.FilterOption) (dyncopy.Field, error) {
	switch len(option.Column) {
	case 0:
		return dyncopy.Field{}, fmt.Errorf("column not set")
	case 1:
		col := option.Column[0]
		field, found := tt[col]
		if found {
			return field, nil
		}
		return dyncopy.Field{}, fmt.Errorf("field: %v does not exist", col)
	default:
		return dyncopy.Field{}, fmt.Errorf("multiple columns set")
	}
}

func getValue(option query.FilterOption) (string, error) {
	switch len(option.Values) {
	case 0:
		return "", fmt.Errorf("value not set")
	case 1:
		return option.Values[0], nil
	default:
		return "", fmt.Errorf("multiple values set")
	}
}
