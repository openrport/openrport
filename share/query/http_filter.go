package query

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

type SortOption struct {
	Column string
	IsASC  bool
}

type FilterOption struct {
	Column string
	Values []string
}

type ListOptions struct {
	Sorts   []SortOption
	Filters []FilterOption
}

func GetSortAndFilterOptions(req *http.Request) *ListOptions {
	return &ListOptions{
		Sorts:   ExtractSortOptions(req),
		Filters: ExtractFilterOptions(req),
	}
}

func ExtractSortOptions(req *http.Request) []SortOption {
	res := make([]SortOption, 0)
	query := req.URL.Query()

	sorts, ok := query["sort"]
	if !ok || len(sorts) == 0 {
		return res
	}

	for _, sort := range sorts {
		sort = strings.TrimSpace(sort)
		if sort == "" {
			continue
		}
		sortOption := SortOption{
			IsASC: true,
		}
		if strings.HasPrefix(sort, "-") {
			sortOption.IsASC = false
			sortOption.Column = strings.TrimLeft(sort, "-")
		} else {
			sortOption.Column = sort
		}

		res = append(res, sortOption)
	}

	return res
}

func ValidateFilterOptions(fo []FilterOption, supportedFields map[string]bool) errors2.APIErrors {
	errs := errors2.APIErrors{}
	for i := range fo {
		ok := supportedFields[fo[i].Column]
		if !ok {
			errs = append(errs, errors2.APIError{
				Message: fmt.Sprintf("unsupported filter field '%s'", fo[i].Column),
				Code:    http.StatusBadRequest,
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func ValidateSortOptions(so []SortOption, supportedFields map[string]bool) errors2.APIErrors {
	errs := errors2.APIErrors{}
	for i := range so {
		ok := supportedFields[so[i].Column]
		if !ok {
			errs = append(errs, errors2.APIError{
				Message: fmt.Sprintf("unsupported sort field '%s'", so[i].Column),
				Code:    http.StatusBadRequest,
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func ValidateListOptions(lo *ListOptions, supportedFields map[string]bool) error {
	errs := errors2.APIErrors{}
	sortErrs := ValidateSortOptions(lo.Sorts, supportedFields)
	if sortErrs != nil {
		errs = append(errs, sortErrs...)
	}

	filterErrs := ValidateFilterOptions(lo.Filters, supportedFields)
	if filterErrs != nil {
		errs = append(errs, filterErrs...)
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func ExtractFilterOptions(req *http.Request) []FilterOption {
	res := make([]FilterOption, 0)
	for filterKey, filterValues := range req.URL.Query() {
		if !strings.HasPrefix(filterKey, "filter") || len(filterValues) == 0 {
			continue
		}

		orValues := make([]string, 0)
		for i := range filterValues {
			orValue := strings.TrimSpace(filterValues[i])
			if orValue == "" {
				continue
			}

			orValues = append(orValues, strings.Split(orValue, ",")...)
		}
		if len(orValues) == 0 {
			continue
		}

		reg := regexp.MustCompile(`^filter\[(\w+)]`)
		matches := reg.FindStringSubmatch(filterKey)
		if matches == nil || len(matches) < 2 {
			continue
		}

		filterColumn := matches[1]
		filterColumn = strings.TrimSpace(filterColumn)
		if filterColumn == "" {
			continue
		}

		fo := FilterOption{
			Column: filterColumn,
			Values: orValues,
		}

		res = append(res, fo)
	}

	return res
}
