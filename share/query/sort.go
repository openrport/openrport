package query

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

type SortOption struct {
	Column string
	IsASC  bool
}

func ExtractSortOptions(req *http.Request) []SortOption {
	return ParseSortOptions(req.URL.Query())
}

func ParseSortOptions(values url.Values) []SortOption {
	res := make([]SortOption, 0)

	sorts, ok := values["sort"]
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

func ValidateSortOptions(so []SortOption, supportedFields map[string]bool) errors2.APIErrors {
	errs := errors2.APIErrors{}
	for i := range so {
		ok := supportedFields[so[i].Column]
		if !ok {
			errs = append(errs, errors2.APIError{
				Message:    fmt.Sprintf("unsupported sort field '%s'", so[i].Column),
				HTTPStatus: http.StatusBadRequest,
			})
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}
