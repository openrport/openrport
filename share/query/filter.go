package query

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

var filterRegex = regexp.MustCompile(`^filter\[([\w|]+)](\[(\w+)])?`)

type FilterOperatorType string

const (
	FilterOperatorTypeEQ    FilterOperatorType = "eq"
	FilterOperatorTypeGT    FilterOperatorType = "gt"
	FilterOperatorTypeLT    FilterOperatorType = "lt"
	FilterOperatorTypeSince FilterOperatorType = "since"
	FilterOperatorTypeUntil FilterOperatorType = "until"
)

func (fot FilterOperatorType) Code() string {
	code, ok := map[FilterOperatorType]string{
		"eq":    "=",
		"gt":    ">",
		"lt":    "<",
		"since": ">=",
		"until": "<=",
	}[fot]
	if !ok {
		return "="
	}
	return code
}

type FilterOption struct {
	Column   []string // Columns filters are ORed together
	Operator FilterOperatorType
	Values   []string
}

func (fo FilterOption) String() string {
	s := fmt.Sprintf("filter[%s]", strings.Join(fo.Column, "|"))
	if fo.Operator != "" {
		s += fmt.Sprintf("[%s]", fo.Operator)
	}
	return s
}

func (fo FilterOption) isSupported(supportedFields map[string]bool) bool {
	for _, col := range fo.Column {
		if fo.Operator == "" && supportedFields[col] {
			continue
		}
		if supportedFields[fmt.Sprintf("%s[%s]", col, fo.Operator)] {
			continue
		}
		return false
	}
	return true
}

func ValidateFilterOptions(fo []FilterOption, supportedFields map[string]bool) errors2.APIErrors {
	errs := errors2.APIErrors{}
	for i := range fo {
		ok := fo[i].isSupported(supportedFields)
		if !ok {
			errs = append(errs, errors2.APIError{
				Message:    fmt.Sprintf("unsupported filter field '%s'", fo[i]),
				HTTPStatus: http.StatusBadRequest,
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func ParseFilterOptions(values url.Values) []FilterOption {
	res := make([]FilterOption, 0)
	for filterKey, filterValues := range values {
		if !strings.HasPrefix(filterKey, "filter") || len(filterValues) == 0 {
			continue
		}

		orValues := getOrValues(filterValues)

		if len(orValues) == 0 {
			continue
		}

		matches := filterRegex.FindStringSubmatch(filterKey)
		if matches == nil || len(matches) < 4 {
			continue
		}

		filterColumn := matches[1]
		filterColumn = strings.TrimSpace(filterColumn)
		if filterColumn == "" {
			continue
		}
		filterColumns := strings.Split(filterColumn, "|")

		filterOperator := matches[3]
		filterOperator = strings.TrimSpace(filterOperator)

		fo := FilterOption{
			Column:   filterColumns,
			Operator: FilterOperatorType(filterOperator),
			Values:   orValues,
		}

		res = append(res, fo)
	}

	return res
}

func SortFiltersByOperator(a []FilterOption) {
	sort.Slice(a, func(i, j int) bool {
		return a[i].Operator < a[j].Operator
	})
}
