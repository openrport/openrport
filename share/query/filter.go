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

var filterRegex = regexp.MustCompile(`^filter\[([\w|*]+)](\[(\w+)])?`)

type FilterOperatorType string
type FilterColumnOperatorType string

const (
	FilterOperatorTypeEQ    FilterOperatorType = "eq"
	FilterOperatorTypeGT    FilterOperatorType = "gt"
	FilterOperatorTypeLT    FilterOperatorType = "lt"
	FilterOperatorTypeSince FilterOperatorType = "since"
	FilterOperatorTypeUntil FilterOperatorType = "until"
)
const (
	FilterColumnOperatorTypeOR  FilterColumnOperatorType = "or"
	FilterColumnOperatorTypeAND FilterColumnOperatorType = "and"
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
	Column         []string // Columns filters are [ColumnOperator]ed together (only AND, OR, default OR)
	ColumnOperator FilterColumnOperatorType

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

func (fo *FilterOption) setWildcardColumns(supportedFields map[string]bool) {
	fo.Column = make([]string, 0, len(supportedFields))
	for field := range supportedFields {
		fo.Column = append(fo.Column, field)
	}
}

func ValidateFilterOptions(fo []FilterOption, supportedFields map[string]bool) errors2.APIErrors {
	errs := errors2.APIErrors{}
	for i := range fo {
		if len(fo[i].Column) == 1 && fo[i].Column[0] == "*" {
			fo[i].setWildcardColumns(supportedFields)
		}
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

func SplitFilters(options []FilterOption, keys map[string]bool) ([]FilterOption, []FilterOption) {
	var these, other []FilterOption
	for _, o := range options {
		isThese := false
		for _, c := range o.Column {
			if _, ok := keys[c]; ok {
				isThese = true
			}
		}
		if isThese {
			these = append(these, o)
		} else {
			other = append(other, o)
		}
	}
	return these, other
}
