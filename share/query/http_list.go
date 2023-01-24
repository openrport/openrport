package query

import (
	"net/http"
	"strings"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

type ListOptions struct {
	Sorts      []SortOption
	Filters    []FilterOption
	Fields     []FieldsOption
	Pagination *Pagination
}

func GetListOptions(req *http.Request) *ListOptions {
	return NewOptions(req, nil, nil, nil)
}

func NewOptions(req *http.Request, sortsDefault map[string][]string, filtersDefault map[string][]string, fieldsDefault map[string][]string) *ListOptions {
	qOptions := &ListOptions{}

	sorts := ParseSortOptions(req.URL.Query())
	if len(sorts) > 0 {
		qOptions.Sorts = sorts
	} else {
		qOptions.Sorts = ParseSortOptions(sortsDefault)
	}
	filters := ParseFilterOptions(req.URL.Query())
	if len(filters) > 0 {
		qOptions.Filters = filters
	} else {
		qOptions.Filters = ParseFilterOptions(filtersDefault)
	}

	fields := ParseFieldsOptions(req.URL.Query())
	if len(fields) > 0 {
		qOptions.Fields = fields
	} else {
		qOptions.Fields = ParseFieldsOptions(fieldsDefault)
	}

	qOptions.Pagination = ParsePagination(req.URL.Query())

	return qOptions
}

// ValidateListOptions when supportedFields is nil, the fields options are disabled and will not be validated or used, same for pagination
func ValidateListOptions(lo *ListOptions, supportedSorts map[string]bool, supportedFilters map[string]bool, supportedFields map[string]map[string]bool, paginationConfig *PaginationConfig) error {
	errs := errors2.APIErrors{}
	sortErrs := ValidateSortOptions(lo.Sorts, supportedSorts)
	if sortErrs != nil {
		errs = append(errs, sortErrs...)
	}

	filterErrs := ValidateFilterOptions(lo.Filters, supportedFilters)
	if filterErrs != nil {
		errs = append(errs, filterErrs...)
	}

	if supportedFields != nil {
		fieldErrs := ValidateFieldsOptions(lo.Fields, supportedFields)
		if fieldErrs != nil {
			errs = append(errs, fieldErrs...)
		}
	} else {
		lo.Fields = nil
	}

	if paginationConfig != nil {
		paginationErrs := ValidatePagination(lo.Pagination, paginationConfig)
		if paginationErrs != nil {
			errs = append(errs, paginationErrs...)
		}
	} else {
		lo.Pagination = nil
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func getValues(values []string) ([]string, FilterLogicalOperator) {
	var op FilterLogicalOperator
	outValues := make([]string, 0)
	for i := range values {
		value := strings.TrimSpace(values[i])

		matchesAndOr := valuesLogicalOpsblock.FindStringSubmatch(value)
		if matchesAndOr != nil {
			op = FilterLogicalOperator(matchesAndOr[1])
			value = matchesAndOr[2]
		}

		operands := strings.Split(value, ",")
		for i := range operands {
			operands[i] = strings.TrimSpace(operands[i])
		}
		outValues = append(outValues, operands...)
	}
	return outValues, op
}
