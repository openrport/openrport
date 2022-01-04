package query

import (
	"fmt"
	"strings"
)

func ConvertListOptionsToQuery(lo *ListOptions, q string) (qOut string, params []interface{}) {
	return AppendOptionsToQuery(lo, q, nil)
}

func ConvertRetrieveOptionsToQuery(ro *RetrieveOptions, q string) string {
	qOut := ReplaceStarSelect(ro.Fields, q)

	return qOut
}

func AppendOptionsToQuery(o *ListOptions, q string, params []interface{}) (string, []interface{}) {
	q, params = AddWhere(o.Filters, q, params)
	q = AddOrderBy(o.Sorts, q)
	q = ReplaceStarSelect(o.Fields, q)
	q, params = addLimitOffset(o.Pagination, q, params)

	return q, params
}

func AddWhere(filterOptions []FilterOption, q string, params []interface{}) (string, []interface{}) {
	if len(filterOptions) == 0 {
		return q, params
	}

	whereParts := make([]string, 0, len(filterOptions))
	for i := range filterOptions {
		orParts := make([]string, 0, len(filterOptions[i].Values))
		for _, col := range filterOptions[i].Column {
			for y := range filterOptions[i].Values {
				orParts = append(orParts, fmt.Sprintf("%s %s ?", col, filterOptions[i].Operator.Code()))
				params = append(params, filterOptions[i].Values[y])
			}
		}

		if len(orParts) > 1 {
			whereParts = append(whereParts, fmt.Sprintf("(%s)", strings.Join(orParts, " OR ")))
		} else {
			whereParts = append(whereParts, orParts[0])
		}
	}

	concat := " WHERE "
	qUpper := strings.ToUpper(q)
	if strings.Contains(qUpper, " WHERE ") {
		concat = " AND "
	}
	q += concat + strings.Join(whereParts, " AND ")

	return q, params
}

func AddOrderBy(sortOptions []SortOption, q string) string {
	if len(sortOptions) == 0 {
		return q
	}
	orderByValues := make([]string, 0, len(sortOptions))
	for i := range sortOptions {
		direction := "ASC"
		if !sortOptions[i].IsASC {
			direction = "DESC"
		}
		orderByValues = append(orderByValues, fmt.Sprintf("%s %s", sortOptions[i].Column, direction))
	}
	if len(orderByValues) > 0 {
		q += " ORDER BY " + strings.Join(orderByValues, ", ")
	}

	return q
}

func ReplaceStarSelect(fieldOptions []FieldsOption, q string) string {
	if !strings.HasPrefix(strings.ToUpper(q), "SELECT * ") {
		return q
	}
	if len(fieldOptions) == 0 {
		return q
	}

	fields := []string{}
	for _, fo := range fieldOptions {
		for _, field := range fo.Fields {
			fields = append(fields, fmt.Sprintf("%s.%s", fo.Resource, field))
		}
	}

	return strings.Replace(q, "*", strings.Join(fields, ", "), 1)
}

func addLimitOffset(pagination *Pagination, q string, params []interface{}) (string, []interface{}) {
	if pagination == nil {
		return q, params
	}

	q += " LIMIT ? OFFSET ?"
	params = append(params, pagination.Limit, pagination.Offset)

	return q, params
}
