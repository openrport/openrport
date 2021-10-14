package query

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

var fieldsRegex = regexp.MustCompile(`^fields\[(\w+)]`)

type FieldsOption struct {
	Resource string
	Fields   []string
}

func ValidateFieldsOptions(fieldOptions []FieldsOption, supportedFields map[string]map[string]bool) errors2.APIErrors {
	errs := errors2.APIErrors{}
	for _, fo := range fieldOptions {
		_, ok := supportedFields[fo.Resource]
		if !ok {
			errs = append(errs, errors2.APIError{
				Message:    fmt.Sprintf("unsupported resource in fields: %q", fo.Resource),
				HTTPStatus: http.StatusBadRequest,
			})
			continue
		}
		for _, field := range fo.Fields {
			ok := supportedFields[fo.Resource][field]
			if !ok {
				errs = append(errs, errors2.APIError{
					Message:    fmt.Sprintf("unsupported field %q for resource %q", field, fo.Resource),
					HTTPStatus: http.StatusBadRequest,
				})
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}

	return nil
}

func ExtractFieldsOptions(req *http.Request) []FieldsOption {
	return ParseFieldsOptions(req.URL.Query())
}

func ParseFieldsOptions(query map[string][]string) []FieldsOption {
	res := make([]FieldsOption, 0)
	for fieldsKey, fieldsValues := range query {
		if !strings.HasPrefix(fieldsKey, "fields") || len(fieldsValues) == 0 {
			continue
		}

		orValues := getOrValues(fieldsValues)
		if len(orValues) == 0 {
			continue
		}

		matches := fieldsRegex.FindStringSubmatch(fieldsKey)
		if matches == nil || len(matches) < 2 {
			continue
		}

		fieldsResource := matches[1]
		fieldsResource = strings.TrimSpace(fieldsResource)
		if fieldsResource == "" {
			continue
		}

		fo := FieldsOption{
			Resource: fieldsResource,
			Fields:   orValues,
		}

		res = append(res, fo)
	}

	return res
}
