package query

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

//var filterRegex = regexp.MustCompile(`^filter\[(\w+)]`)

var filterRegex = regexp.MustCompile(`^filter\[(\w+)](\[(\w+)])?`)

type FilterOperatorType int

//
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
	Column   string
	Operator FilterOperatorType
	Values   []string
}

func ValidateFilterOptions(fo []FilterOption, supportedFields map[string]bool) errors2.APIErrors {
	errs := errors2.APIErrors{}
	for i := range fo {
		filterExpr := fo[i].Column
		if fo[i].Operator != FilterOperatorTypeEQ {
			filterExpr = filterExpr + "[" + fo[i].Operator.String() + "]"
		}
		ok := supportedFields[filterExpr]
		if !ok {
			errs = append(errs, errors2.APIError{
				Message:    fmt.Sprintf("unsupported filter field '%s'", fo[i].Column),
				HTTPStatus: http.StatusBadRequest,
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func ExtractFilterOptions(req *http.Request) []FilterOption {
	return ParseFilterOptions(req.URL.Query())
}

func ParseFilterOptions(query map[string][]string) []FilterOption {

	res := make([]FilterOption, 0)
	for filterKey, filterValues := range query {
		if !strings.HasPrefix(filterKey, "filter") || len(filterValues) == 0 {
			continue
		}

		orValues := getOrValues(filterValues)

		if len(orValues) == 0 {
			continue
		}

		matches := filterRegex.FindStringSubmatch(filterKey)
		if matches == nil || len(matches) < 2 {
			continue
		}

		filterColumn := matches[1]
		filterColumn = strings.TrimSpace(filterColumn)
		if filterColumn == "" {
			continue
		}

		filterOperator := FilterOperatorTypeEQ.Code()
		if len(matches) == 4 {
			filterOperator = matches[3]
		}
		fo := FilterOption{
			Column:   filterColumn,
			Operator: ParseFilterOperatorType(filterOperator),
			Values:   orValues,
		}

		res = append(res, fo)
	}

	return res
}
