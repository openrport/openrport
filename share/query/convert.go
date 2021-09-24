package query

import (
	"fmt"
	"strings"
)

func ConvertListOptionsToQuery(lo *ListOptions, q string) (qOut string, params []interface{}) {
	qOut, params = addWhere(lo, q)
	qOut = addOrderBy(lo, qOut)
	qOut = replaceStarSelect(lo.Fields, qOut)

	return qOut, params
}

func ConvertRetrieveOptionsToQuery(ro *RetrieveOptions, q string) string {
	qOut := replaceStarSelect(ro.Fields, q)

	return qOut
}

func addWhere(lo *ListOptions, q string) (qOut string, params []interface{}) {
	params = []interface{}{}
	if len(lo.Filters) == 0 {
		return q, params
	}

	whereParts := make([]string, 0, len(lo.Filters))
	for i := range lo.Filters {
		if len(lo.Filters[i].Values) == 1 {
			whereParts = append(whereParts, fmt.Sprintf("%s = ?", lo.Filters[i].Column))
			params = append(params, lo.Filters[i].Values[0])
		} else {
			orParts := make([]string, 0, len(lo.Filters[i].Values))
			for y := range lo.Filters[i].Values {
				orParts = append(orParts, fmt.Sprintf("%s = ?", lo.Filters[i].Column))
				params = append(params, lo.Filters[i].Values[y])
			}

			whereParts = append(whereParts, fmt.Sprintf("(%s)", strings.Join(orParts, " OR ")))
		}
	}

	q += " WHERE " + strings.Join(whereParts, " AND ") + " "

	return q, params
}

func addOrderBy(lo *ListOptions, q string) string {
	if len(lo.Sorts) == 0 {
		return q
	}
	orderByValues := make([]string, 0, len(lo.Sorts))
	for i := range lo.Sorts {
		direction := "ASC"
		if !lo.Sorts[i].IsASC {
			direction = "DESC"
		}
		orderByValues = append(orderByValues, fmt.Sprintf("%s %s", lo.Sorts[i].Column, direction))
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
