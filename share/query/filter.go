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

var filterRegex = regexp.MustCompile(`^filter\[(\w+)](\[(\w+)])?`)

type FilterOperatorType int

const (
	FilterOperatorTypeEQ FilterOperatorType = iota
	FilterOperatorTypeGT
	FilterOperatorTypeLT
	FilterOperatorTypeSince
	FilterOperatorTypeUntil
)

func (fot FilterOperatorType) Code() string {
	return [...]string{"=", ">", "<", ">=", "<="}[fot]
}

func (fot FilterOperatorType) String() string {
	return [...]string{"eq", "gt", "lt", "since", "until"}[fot]
}
func ParseFilterOperatorType(filterOperator string) FilterOperatorType {
	switch strings.ToLower(filterOperator) {
	case FilterOperatorTypeGT.String():
		return FilterOperatorTypeGT
	case FilterOperatorTypeLT.String():
		return FilterOperatorTypeLT
	case FilterOperatorTypeSince.String():
		return FilterOperatorTypeSince
	case FilterOperatorTypeUntil.String():
		return FilterOperatorTypeUntil
	}

	return FilterOperatorTypeEQ
}

type FilterOption struct {
	Expression string
	Column     string
	Operator   FilterOperatorType
	Values     []string
}

func ValidateFilterOptions(fo []FilterOption, supportedFields map[string]bool) errors2.APIErrors {
	errs := errors2.APIErrors{}
	for i := range fo {
		ok := supportedFields[fo[i].Expression]
		if !ok {
			errs = append(errs, errors2.APIError{
				Message:    fmt.Sprintf("unsupported filter field '%s'", fo[i].Expression),
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

		expressionOperator := matches[2]
		expressionOperator = strings.TrimSpace(expressionOperator)

		filterOperator := matches[3]
		filterOperator = strings.TrimSpace(filterOperator)

		filterExpression := filterColumn + expressionOperator
		fo := FilterOption{
			Expression: filterExpression,
			Column:     filterColumn,
			Operator:   ParseFilterOperatorType(filterOperator),
			Values:     orValues,
		}

		res = append(res, fo)
	}

	return res
}

func IsLimitFilter(fo FilterOption) bool {
	return fo.Column == "limit"
}

func SortFiltersByOperator(a []FilterOption) {
	sort.Slice(a, func(i, j int) bool {
		return a[i].Operator < a[j].Operator
	})
}
