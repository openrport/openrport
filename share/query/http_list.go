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
