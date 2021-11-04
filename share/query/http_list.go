package query

import (
	"net/http"
	"strings"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

type ListOptions struct {
	Sorts   []SortOption
	Filters []FilterOption
	Fields  []FieldsOption
}

func GetListOptions(req *http.Request) *ListOptions {
	return &ListOptions{
		Sorts:   ExtractSortOptions(req),
		Filters: ExtractFilterOptions(req),
		Fields:  ExtractFieldsOptions(req),
	}
}

func NewOptions(req *http.Request, sortsDefault map[string][]string, filtersDefault map[string][]string, fieldsDefault map[string][]string) *ListOptions {
	qOptions := &ListOptions{}

	sorts := ExtractSortOptions(req)
	if len(sorts) > 0 {
		qOptions.Sorts = sorts
	} else {
		qOptions.Sorts = ParseSortOptions(sortsDefault)
	}
	filters := ExtractFilterOptions(req)
	if len(filters) > 0 {
		qOptions.Filters = filters
	} else {
		qOptions.Filters = ParseFilterOptions(filtersDefault)
	}

	fields := ExtractFieldsOptions(req)
	if len(fields) > 0 {
		qOptions.Fields = fields
	} else {
		qOptions.Fields = ParseFieldsOptions(fieldsDefault)
	}

	return qOptions
}

// when supportedFields is nil, the fields options are disabled and will not be validated or used
func ValidateListOptions(lo *ListOptions, supportedSortAndFilters map[string]bool, supportedFields map[string]map[string]bool) error {
	errs := errors2.APIErrors{}
	sortErrs := ValidateSortOptions(lo.Sorts, supportedSortAndFilters)
	if sortErrs != nil {
		errs = append(errs, sortErrs...)
	}

	filterErrs := ValidateFilterOptions(lo.Filters, supportedSortAndFilters)
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

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func ValidateOptions(options *ListOptions, supportedSortFields map[string]bool, supportedFilterFields map[string]bool, supportedFields map[string]map[string]bool) error {
	if err := ValidateSortOptions(options.Sorts, supportedSortFields); err != nil {
		return err
	}
	if err := ValidateFilterOptions(options.Filters, supportedFilterFields); err != nil {
		return err
	}
	if err := ValidateFieldsOptions(options.Fields, supportedFields); err != nil {
		return err
	}

	return nil
}

func (o *ListOptions) HasFilters() bool {
	return o.Filters != nil && len(o.Filters) > 0
}

func getOrValues(values []string) []string {
	orValues := make([]string, 0)
	for i := range values {
		orValue := strings.TrimSpace(values[i])
		if orValue == "" {
			continue
		}

		orValues = append(orValues, strings.Split(orValue, ",")...)
	}
	return orValues
}
