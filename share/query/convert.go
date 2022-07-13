package query

import (
	"fmt"
	"strings"
)

type SQLConverter struct {
	dbDriverName string
}

// NewSQLConverter converts query parameters into SQL language
func NewSQLConverter(dbDriverName string) *SQLConverter {
	return &SQLConverter{
		dbDriverName: dbDriverName,
	}
}

func (c *SQLConverter) ConvertListOptionsToQuery(lo *ListOptions, q string) (qOut string, params []interface{}) {
	return c.AppendOptionsToQuery(lo, q, nil)
}

func (c *SQLConverter) ConvertRetrieveOptionsToQuery(ro *RetrieveOptions, q string) string {
	qOut := c.ReplaceStarSelect(ro.Fields, q)

	return qOut
}

func (c *SQLConverter) AppendOptionsToQuery(o *ListOptions, q string, params []interface{}) (string, []interface{}) {
	if o == nil {
		return q, params
	}
	q, params = c.AddWhere(o.Filters, q, params)
	q = c.AddOrderBy(o.Sorts, q)
	q = c.ReplaceStarSelect(o.Fields, q)
	q, params = c.addLimitOffset(o.Pagination, q, params)

	return q, params
}

func (c *SQLConverter) AddWhere(filterOptions []FilterOption, q string, params []interface{}) (string, []interface{}) {
	if len(filterOptions) == 0 {
		return q, params
	}

	whereParts := make([]string, 0, len(filterOptions))
	for i := range filterOptions {
		orParts := make([]string, 0, len(filterOptions[i].Values))
		for _, col := range filterOptions[i].Column {
			for _, val := range filterOptions[i].Values {
				part := fmt.Sprintf("%s %s ?", col, filterOptions[i].Operator.Code())
				if val == "" {
					part = fmt.Sprintf("(%s OR %s IS NULL)", part, col)
				} else if strings.Contains(val, "*") && filterOptions[i].Operator.Code() == "=" {
					// Implement a SQL LIKE search triggered by a wildcard
					if c.dbDriverName == "mysql" {
						//MySQL needs the backslash escaped, that means double-backslash;  WHERE LOWER(id) LIKE 'op\%' escape "\\";
						part = fmt.Sprintf("LOWER(%s) LIKE ? ESCAPE '\\\\'", col)
					} else {
						//SQLite needs a single backslash
						part = fmt.Sprintf("LOWER(%s) LIKE ? ESCAPE '\\'", col)
					}
					// Escape the % sign to treat it literally, on the API side % must not become a wildcard
					val = strings.Replace(val, "%", "\\%", -1)
					// Make search case-insensitive
					val = strings.ToLower(val)
					// Replace wildcard * by sql wildcard %
					val = strings.ReplaceAll(val, "*", "%")
				}
				orParts = append(orParts, part)
				params = append(params, val)
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

func (c *SQLConverter) AddOrderBy(sortOptions []SortOption, q string) string {
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

func (c *SQLConverter) ReplaceStarSelect(fieldOptions []FieldsOption, q string) string {
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

func (c *SQLConverter) addLimitOffset(pagination *Pagination, q string, params []interface{}) (string, []interface{}) {
	if pagination == nil {
		return q, params
	}

	q += " LIMIT ? OFFSET ?"
	params = append(params, pagination.Limit, pagination.Offset)

	return q, params
}
