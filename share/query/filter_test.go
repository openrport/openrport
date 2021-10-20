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
		Name                  string
		FilterOptions         []FilterOption
		SupportedFilterFields map[string]bool
		ExpectedAPIErrors     errors2.APIErrors
	}{
		{
			Name: "filter fields without sub filter, ok",
			FilterOptions: []FilterOption{
				{
					Expression: "name",
					Column:     "name",
					Values:     []string{"val1"},
				},
			},
			SupportedFilterFields: map[string]bool{"name": true},
			ExpectedAPIErrors:     nil,
		},
		{
			Name: "filter fields with sub filter, ok",
			FilterOptions: []FilterOption{
				{
					Expression: "timestamp[gt]",
					Operator:   FilterOperatorTypeGT,
				},
				{
					Expression: "timestamp[lt]",
					Operator:   FilterOperatorTypeLT,
				},
				{
					Expression: "timestamp[since]",
					Operator:   FilterOperatorTypeSince,
				},
				{
					Expression: "timestamp[until]",
					Operator:   FilterOperatorTypeUntil,
				},
			},
			SupportedFilterFields: map[string]bool{"timestamp[gt]": true, "timestamp[lt]": true, "timestamp[since]": true, "timestamp[until]": true},
			ExpectedAPIErrors:     nil,
		},
		{
			Name: "filter fields without sub filter, not ok",
			FilterOptions: []FilterOption{
				{
					Expression: "name",
					Values:     []string{"val1"},
				},
			},
			SupportedFilterFields: map[string]bool{"field1": true},
			ExpectedAPIErrors: errors2.APIErrors{
				errors2.APIError{
					Message:    fmt.Sprintf("unsupported filter field '%s'", "name"),
					HTTPStatus: http.StatusBadRequest,
				},
			},
		},
		{
			Name: "filter fields with sub filter, not ok",
			FilterOptions: []FilterOption{
				{
					Expression: "timestamp[gt]",
					Operator:   FilterOperatorTypeGT,
					Values:     []string{"val1"},
				},
				{
					Expression: "timestamp[to]",
					Operator:   FilterOperatorTypeEQ,
					Values:     []string{"value2"},
				},
			},
			SupportedFilterFields: map[string]bool{"timestamp[gt]": true, "timestamp[lt]": true},
			ExpectedAPIErrors: errors2.APIErrors{
				errors2.APIError{
					Message:    fmt.Sprintf("unsupported filter field '%s'", "timestamp[to]"),
					HTTPStatus: http.StatusBadRequest,
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			errs := ValidateFilterOptions(tc.FilterOptions, tc.SupportedFilterFields)

			assert.Equal(t, tc.ExpectedAPIErrors, errs)
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
					Expression: "timestamp[gt]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeGT,
					Values:     []string{"1634303188"},
				},
				{
					Expression: "timestamp[lt]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeLT,
					Values:     []string{"1634303609"},
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
					Expression: "timestamp[xx]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeEQ,
					Values:     []string{"1634303188"},
				},
				{
					Expression: "timestamp[yy]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeEQ,
					Values:     []string{"1634303609"},
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
					Expression: "name",
					Column:     "name",
					Operator:   FilterOperatorTypeEQ,
					Values:     []string{"val1"},
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

			assert.Equal(t, tc.ExpectedFilterOptions, filterOptions)
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
					Expression: "timestamp[until]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeUntil,
					Values:     []string{"2021-09-29:11:00:00"},
				},
				{
					Expression: "timestamp[lt]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeLT,
					Values:     []string{"1634303609"},
				},
				{
					Expression: "timestamp[since]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeSince,
					Values:     []string{"2021-09-29:10:00:00"},
				},
				{
					Expression: "timestamp[gt]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeGT,
					Values:     []string{"1634303188"},
				},
			},
			ExpectedFilterOptions: []FilterOption{
				{
					Expression: "timestamp[gt]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeGT,
					Values:     []string{"1634303188"},
				},
				{
					Expression: "timestamp[lt]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeLT,
					Values:     []string{"1634303609"},
				},
				{
					Expression: "timestamp[since]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeSince,
					Values:     []string{"2021-09-29:10:00:00"},
				},
				{
					Expression: "timestamp[until]",
					Column:     "timestamp",
					Operator:   FilterOperatorTypeUntil,
					Values:     []string{"2021-09-29:11:00:00"},
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
