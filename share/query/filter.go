package query

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

var filterRegex = regexp.MustCompile(`^filter\[(\w+)]`)

type FilterOption struct {
	Column string
	Values []string
}

func ValidateFilterOptions(fo []FilterOption, supportedFields map[string]bool) errors2.APIErrors {
	errs := errors2.APIErrors{}
	for i := range fo {
		ok := supportedFields[fo[i].Column]
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

		fo := FilterOption{
			Column: filterColumn,
			Values: orValues,
		}

		res = append(res, fo)
	}

	return res
}
