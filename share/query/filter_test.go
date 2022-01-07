package query

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

func TestValidateFilterOptions(t *testing.T) {
	testCases := []struct {
		Name                        string
		FilterOptions               []FilterOption
		SupportedFilterFields       map[string]bool
		ExpectedAPIErrors           errors2.APIErrors
		ExpectedFilterOptionColumns []string
	}{
		{
			Name: "filter fields without sub filter, ok",
			FilterOptions: []FilterOption{
				{
					Column: []string{"name"},
					Values: []string{"val1"},
				},
			},
			SupportedFilterFields: map[string]bool{"name": true},
			ExpectedAPIErrors:     nil,
		},
		{
			Name: "filter fields with sub filter, ok",
			FilterOptions: []FilterOption{
				{
					Column:   []string{"timestamp"},
					Operator: FilterOperatorTypeGT,
				},
				{
					Column:   []string{"timestamp"},
					Operator: FilterOperatorTypeLT,
				},
				{
					Column:   []string{"timestamp"},
					Operator: FilterOperatorTypeSince,
				},
				{
					Column:   []string{"timestamp"},
					Operator: FilterOperatorTypeUntil,
				},
			},
			SupportedFilterFields: map[string]bool{"timestamp[gt]": true, "timestamp[lt]": true, "timestamp[since]": true, "timestamp[until]": true},
			ExpectedAPIErrors:     nil,
		},
		{
			Name: "filter fields without sub filter, not ok",
			FilterOptions: []FilterOption{
				{
					Column: []string{"name"},
					Values: []string{"val1"},
				},
			},
			SupportedFilterFields: map[string]bool{"field1": true},
			ExpectedAPIErrors: errors2.APIErrors{
				errors2.APIError{
					Message:    fmt.Sprintf("unsupported filter field '%s'", "filter[name]"),
					HTTPStatus: http.StatusBadRequest,
				},
			},
		},
		{
			Name: "filter fields with sub filter, not ok",
			FilterOptions: []FilterOption{
				{
					Column:   []string{"timestamp"},
					Operator: "gt",
					Values:   []string{"val1"},
				},
				{
					Column:   []string{"timestamp"},
					Operator: "eq",
					Values:   []string{"value2"},
				},
			},
			SupportedFilterFields: map[string]bool{"timestamp[gt]": true, "timestamp[lt]": true},
			ExpectedAPIErrors: errors2.APIErrors{
				errors2.APIError{
					Message:    fmt.Sprintf("unsupported filter field '%s'", "filter[timestamp][eq]"),
					HTTPStatus: http.StatusBadRequest,
				},
			},
		},
		{
			Name: "wildcard filter",
			FilterOptions: []FilterOption{
				{
					Column: []string{"*"},
					Values: []string{"val1"},
				},
			},
			SupportedFilterFields:       map[string]bool{"field1": true, "field2": true},
			ExpectedAPIErrors:           nil,
			ExpectedFilterOptionColumns: []string{"field1", "field2"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			errs := ValidateFilterOptions(tc.FilterOptions, tc.SupportedFilterFields)

			assert.Equal(t, tc.ExpectedAPIErrors, errs)
			if tc.ExpectedFilterOptionColumns != nil {
				assert.ElementsMatch(t, tc.ExpectedFilterOptionColumns, tc.FilterOptions[0].Column)
			}
		})
	}

}

func TestParseFilterOptions(t *testing.T) {
	testCases := []struct {
		Name                  string
		Query                 map[string][]string
		ExpectedFilterOptions []FilterOption
	}{
		{
			Name: "filter fields with sub filter, ok",
			Query: map[string][]string{
				"filter[timestamp][gt]": {"1634303188"},
				"filter[timestamp][lt]": {"1634303609"},
			},
			ExpectedFilterOptions: []FilterOption{
				{
					Column:   []string{"timestamp"},
					Operator: FilterOperatorTypeGT,
					Values:   []string{"1634303188"},
				},
				{
					Column:   []string{"timestamp"},
					Operator: FilterOperatorTypeLT,
					Values:   []string{"1634303609"},
				},
			},
		},
		{
			Name: "filter fields with sub filter, not known sub filter",
			Query: map[string][]string{
				"filter[timestamp][xx]": {"1634303188"},
				"filter[timestamp][yy]": {"1634303609"},
			},
			ExpectedFilterOptions: []FilterOption{
				{
					Column:   []string{"timestamp"},
					Operator: "xx",
					Values:   []string{"1634303188"},
				},
				{
					Column:   []string{"timestamp"},
					Operator: "yy",
					Values:   []string{"1634303609"},
				},
			},
		},
		{
			Name: "filter fields without sub filter, ok",
			Query: map[string][]string{
				"filter[name]": {"val1"},
			},
			ExpectedFilterOptions: []FilterOption{
				{
					Column: []string{"name"},
					Values: []string{"val1"},
				},
			},
		},
		{
			Name: "multiple columns, ok",
			Query: map[string][]string{
				"filter[name|other]": {"val1"},
			},
			ExpectedFilterOptions: []FilterOption{
				{
					Column: []string{"name", "other"},
					Values: []string{"val1"},
				},
			},
		},
		{
			Name: "column with underscored",
			Query: map[string][]string{
				"filter[some_column_123]": {"val1"},
			},
			ExpectedFilterOptions: []FilterOption{
				{
					Column: []string{"some_column_123"},
					Values: []string{"val1"},
				},
			},
		},
		{
			Name: "filter without fields, not ok",
			Query: map[string][]string{
				"filter": {"val1"},
			},
			ExpectedFilterOptions: []FilterOption{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			filterOptions := ParseFilterOptions(tc.Query)

			assert.ElementsMatch(t, tc.ExpectedFilterOptions, filterOptions)
		})
	}
}

func TestSortFiltersByOperator(t *testing.T) {
	testCases := []struct {
		Name                  string
		FilterOptions         []FilterOption
		ExpectedFilterOptions []FilterOption
	}{
		{
			Name: "filter fields with sub filter",
			FilterOptions: []FilterOption{
				{
					Column:   []string{"timestamp"},
					Operator: "until",
					Values:   []string{"2021-09-29:11:00:00"},
				},
				{
					Column:   []string{"timestamp"},
					Operator: "lt",
					Values:   []string{"1634303609"},
				},
				{
					Column:   []string{"timestamp"},
					Operator: "since",
					Values:   []string{"2021-09-29:10:00:00"},
				},
				{
					Column:   []string{"timestamp"},
					Operator: "gt",
					Values:   []string{"1634303188"},
				},
			},
			ExpectedFilterOptions: []FilterOption{
				{
					Column:   []string{"timestamp"},
					Operator: "gt",
					Values:   []string{"1634303188"},
				},
				{
					Column:   []string{"timestamp"},
					Operator: "lt",
					Values:   []string{"1634303609"},
				},
				{
					Column:   []string{"timestamp"},
					Operator: "since",
					Values:   []string{"2021-09-29:10:00:00"},
				},
				{
					Column:   []string{"timestamp"},
					Operator: "until",
					Values:   []string{"2021-09-29:11:00:00"},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			filterOptions := tc.FilterOptions
			SortFiltersByOperator(filterOptions)

			assert.Equal(t, tc.ExpectedFilterOptions, filterOptions)
		})
	}
}
