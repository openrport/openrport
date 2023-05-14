package filterer

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/realvnc-labs/rport/share/dynops/dyncopy"
	"github.com/realvnc-labs/rport/share/query"
)

type fieldID[T any] int

func (fid fieldID[T]) GetFieldFromObject(o T) reflect.Value {
	v := reflect.ValueOf(o)
	return v.Field(int(fid))
}

type ComparatorSliceEQ[T any] struct {
	fieldID[T]
	comparator func(value string) bool
}

func (c ComparatorSliceEQ[T]) Run(o T) bool {
	strings := c.GetFieldFromObject(o).Interface().([]string)
	for _, ss := range strings {
		if c.comparator(ss) {
			return true
		}
	}
	return false
}

type ComparatorMapEQ[T any] struct {
	fieldID[T]
	comparator func(value string) bool
}

func (c ComparatorMapEQ[T]) Run(o T) bool {
	entries := c.GetFieldFromObject(o).Interface().(map[string]string)
	for k, v := range entries {
		if c.comparator(fmt.Sprintf("%v: %v", k, v)) {
			return true
		}
	}
	return false
}

type ComparatorStringEQ[T any] struct {
	fieldID[T]
	value string
}

type ComparatorStringWithComp[T any] struct {
	fieldID[T]
	comparator func(value string) bool
}

func (c ComparatorStringWithComp[T]) Run(o T) bool {
	return c.comparator(c.GetFieldFromObject(o).String())
}

func (c ComparatorStringEQ[T]) Run(o T) bool {
	return c.GetFieldFromObject(o).Interface() == c.value
}

type ComparatorSubStringEQ[T any] struct {
	fieldID[T]
	parts []string
}

func (c ComparatorSubStringEQ[T]) Run(o T) bool {
	txt := c.GetFieldFromObject(o).String()
	return StringMather(txt, c.parts)
}

type ComparatorTime[T any] struct {
	fieldID[T]
	value    time.Time
	operator query.FilterOperatorType
}

func (c ComparatorTime[T]) Run(o T) bool {
	value := c.GetFieldFromObject(o).Interface().(time.Time)
	switch c.operator {
	case query.FilterOperatorTypeEQ:
		return c.value == value
	case query.FilterOperatorTypeGT:
		return c.value.After(value)
	case query.FilterOperatorTypeLT:
		return c.value.Before(value)
	case query.FilterOperatorTypeSince:
		return c.value.After(value)
	case query.FilterOperatorTypeUntil:
		return c.value.Before(value)
	}

	return false
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

func CompileFromQueryListOptions[T any](filters []query.FilterOption) (Operation[T], error) {
	if len(filters) == 0 {
		return NewTrue[T](), nil
	}

	var proto T

	tt := dyncopy.BuildTranslationTable(proto)

	comparators := make([]Operation[T], len(filters))
	for i, filter := range filters {
		field, err := getColumnField(tt, filter)
		if err != nil {
			return nil, fmt.Errorf("%v on type: %v on filter: %v", err, reflect.TypeOf(proto), i)
		}

		switch len(filter.Values) {
		case 0:
			return nil, fmt.Errorf("no values for filter: %v", i)
		case 1:
			c, err := getComparator[T](filter.Operator, field, filter.Values[0])
			if err != nil {
				return nil, fmt.Errorf("%v on filter: %v", err, i)
			}

			comparators[i] = c
		default:
			ops := make([]Operation[T], len(filter.Values))
			for vid, v := range filter.Values {
				c, err := getComparator[T](filter.Operator, field, v)
				if err != nil {
					return nil, fmt.Errorf("%v on filter: %v", err, vid)
				}
				ops[vid] = c
			}

			switch filter.ValuesLogicalOperator {
			case query.FilterLogicalOperatorTypeOR:
				comparators[i] = NewOr(ops...)
			case query.FilterLogicalOperatorTypeAND:
				comparators[i] = NewAnd(ops...)
			default:
				return nil, fmt.Errorf("unknown logical operator: %v", filter.ValuesLogicalOperator)
			}
		}

	}
	return NewAnd[T](comparators...), nil
}

func getComparator[T any](operator query.FilterOperatorType, field dyncopy.Field, query string) (Operation[T], error) {

	switch field.Kind.Kind() {
	case reflect.String:
		return GetStringComparator[T](operator, field, query)
	case reflect.Slice:
		return GetSliceComparator[T](operator, field, query)
	case reflect.Map:
		return GetMapComparator[T](operator, field, query)
	case reflect.Int, reflect.Int64:
		return GetIntComparator[T](operator, field, query)
	default:
		switch field.Kind.Kind().String() {
		case "time.time":
			return GetTimeComparator[T](operator, field, query)
		}

		return nil, fmt.Errorf("can't compare field, unhandled type")
	}
}

func GetTimeComparator[T any](operator query.FilterOperatorType, field dyncopy.Field, v string) (Operation[T], error) {
	switch operator {
	case query.FilterOperatorTypeEQ:
	case query.FilterOperatorTypeGT:
	case query.FilterOperatorTypeLT:
	case query.FilterOperatorTypeSince:
	case query.FilterOperatorTypeUntil:
	default:
		return nil, fmt.Errorf("invalid time operator: %v", operator)
	}

	i, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, fmt.Errorf("can't parse value: %v into time: %v", v, err)
	}

	return ComparatorTime[T]{
		operator: operator,
		fieldID:  fieldID[T](field.ID),
		value:    i,
	}, nil
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

func StringMather(txt string, parts []string) bool {
	if len(txt) == 0 || len(parts) == 0 {
		return false
	}

	first := parts[0]

	subtext := txt
	i := strings.Index(subtext, first)
	if first != "" && i != 0 {
		return false
	}
	subtext = subtext[i+len(first):]

	if len(parts) > 2 {
		for _, part := range parts[1 : len(parts)-1] {
			i := strings.Index(subtext, part)
			if i < 0 {
				return false
			}
			subtext = subtext[i+len(part):]
		}
	}

	if len(parts) > 1 {
		last := parts[len(parts)-1]
		if strings.LastIndex(subtext, last)+len(last) != len(subtext) && last != "" {
			return false
		}
	}

	return true
}

func GenStringComparator(query string) func(value string) bool {
	if strings.Contains(query, "*") {
		parts := strings.Split(query, "*")
		return func(value string) bool {
			return StringMather(value, parts)
		}
	}

	return func(value string) bool {
		return value == query
	}
}

func GetStringComparator[T any](operator query.FilterOperatorType, field dyncopy.Field, v string) (Operation[T], error) {

	switch operator {
	case query.FilterOperatorTypeEQ:
		return ComparatorStringWithComp[T]{
			fieldID:    fieldID[T](field.ID),
			comparator: GenStringComparator(v),
		}, nil
	default:
		return nil, fmt.Errorf("invalid string operator: %v", operator)
	}
}

func GetSliceComparator[T any](operator query.FilterOperatorType, field dyncopy.Field, v string) (Operation[T], error) {
	switch operator {
	case query.FilterOperatorTypeEQ:
		return ComparatorSliceEQ[T]{
			fieldID:    fieldID[T](field.ID),
			comparator: GenStringComparator(v),
		}, nil
	default:
		return nil, fmt.Errorf("invalid string operator: %v", operator)
	}
}

func GetMapComparator[T any](operator query.FilterOperatorType, field dyncopy.Field, v string) (Operation[T], error) {

	switch operator {
	case query.FilterOperatorTypeEQ:
		return ComparatorMapEQ[T]{
			fieldID:    fieldID[T](field.ID),
			comparator: GenStringComparator(v),
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
