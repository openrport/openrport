package vault

import (
	"net/http"
	"regexp"
	"strings"
)

func ConvertGetParamsToFilterOptions(req *http.Request) *ListOptions {
	return &ListOptions{
		Sorts:   extractSortOptions(req),
		Filters: extractFilterOptions(req),
	}
}

func extractSortOptions(req *http.Request) []SortOption {
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

func extractFilterOptions(req *http.Request) []FilterOption {
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
