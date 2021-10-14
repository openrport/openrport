package query

import (
	"fmt"
	"strings"
)

func ConvertListOptionsToQuery(lo *ListOptions, q string) (qOut string, params []interface{}) {
	qOut, params = addWhere(lo.Filters, q)
	qOut = addOrderBy(lo.Sorts, qOut)
	qOut = replaceStarSelect(lo.Fields, qOut)

	return qOut, params
}

func ConvertRetrieveOptionsToQuery(ro *RetrieveOptions, q string) string {
	qOut := replaceStarSelect(ro.Fields, q)

	return qOut
}

func ConvertOptionsToQuery(o *Options, q string) (qOut string, params []interface{}) {
	qOut, params = addWhere(o.Filters, q)
	qOut = addOrderBy(o.Sorts, qOut)
	qOut = replaceStarSelect(o.Fields, qOut)

	return qOut, params
}

func addWhere(filterOptions []FilterOption, q string) (qOut string, params []interface{}) {
	params = []interface{}{}
	if len(filterOptions) == 0 {
		return q, params
	}

	whereParts := make([]string, 0, len(filterOptions))
	for i := range filterOptions {
		if len(filterOptions[i].Values) == 1 {
			whereParts = append(whereParts, fmt.Sprintf("%s = ?", filterOptions[i].Column))
			params = append(params, filterOptions[i].Values[0])
		} else {
			orParts := make([]string, 0, len(filterOptions[i].Values))
			for y := range filterOptions[i].Values {
				orParts = append(orParts, fmt.Sprintf("%s = ?", filterOptions[i].Column))
				params = append(params, filterOptions[i].Values[y])
			}

			whereParts = append(whereParts, fmt.Sprintf("(%s)", strings.Join(orParts, " OR ")))
		}
	}

	q += " WHERE " + strings.Join(whereParts, " AND ") + " "

	return q, params
}

func addOrderBy(sortOptions []SortOption, q string) string {
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
		q += "ORDER BY " + strings.Join(orderByValues, ", ")
	}

	return q
}

func replaceStarSelect(fieldOptions []FieldsOption, q string) string {
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
