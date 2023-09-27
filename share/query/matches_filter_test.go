package query_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openrport/openrport/share/query"
)

func TestMatchesFilters(t *testing.T) {
	value := struct {
		Name   string            `json:"name"`
		Tags   []int             `json:"tags"`
		Labels map[string]string `json:"labels"`
	}{
		Name:   "abcde",
		Tags:   []int{123, 456},
		Labels: map[string]string{"country": "Germany", "city": "Cologne", "datacenter": "NetCologne GmbH"},
	}
	testCases := []struct {
		Name           string
		Filters        []query.FilterOption
		ExpectedResult bool
	}{
		{
			Name: "single value",
			Filters: []query.FilterOption{
				{
					Column: []string{"name"},
					Values: []string{
						"abcde",
					},
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "case insensitive",
			Filters: []query.FilterOption{
				{
					Column: []string{"name"},
					Values: []string{
						"aBcDe",
					},
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "wildcard",
			Filters: []query.FilterOption{
				{
					Column: []string{"name"},
					Values: []string{
						"ab*",
					},
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "wildcard case insensitive",
			Filters: []query.FilterOption{
				{
					Column: []string{"name"},
					Values: []string{
						"aB*",
					},
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "array value",
			Filters: []query.FilterOption{
				{
					Column: []string{"tags"},
					Values: []string{
						"123",
					},
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "map value",
			Filters: []query.FilterOption{
				{
					Column: []string{"labels"},
					Values: []string{
						"city: Cologne",
					},
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "multiple values",
			Filters: []query.FilterOption{
				{
					Column: []string{"name"},
					Values: []string{
						"ab123",
						"abcde",
					},
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "multiple filters",
			Filters: []query.FilterOption{
				{
					Column: []string{"name"},
					Values: []string{
						"abcde",
					},
				},
				{
					Column: []string{"tags"},
					Values: []string{
						"123",
					},
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "or columns",
			Filters: []query.FilterOption{
				{
					Column: []string{"name", "tags"},
					Values: []string{
						"123",
					},
				},
			},
			ExpectedResult: true,
		},
		{
			Name: "no match",
			Filters: []query.FilterOption{
				{
					Column: []string{"name"},
					Values: []string{
						"12345",
						"defgh",
					},
				},
				{
					Column: []string{"tags"},
					Values: []string{
						"123",
					},
				},
			},
			ExpectedResult: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			result, err := query.MatchesFilters(value, tc.Filters)
			require.NoError(t, err)
			assert.Equal(t, tc.ExpectedResult, result)
		})
	}
}

func TestMatchesFiltersUnsupported(t *testing.T) {
	value := struct {
		Name string `json:"name"`
		Tags []int  `json:"tags"`
	}{
		Name: "abcde",
		Tags: []int{123, 456},
	}

	_, err := query.MatchesFilters(value, []query.FilterOption{
		{
			Column: []string{"other"},
			Values: []string{
				"12345",
			},
		},
	})
	assert.EqualError(t, err, "unsupported filter column: other")
}

func TestMatchIfDate(t *testing.T) {
	testCases := []struct {
		name           string
		dateValueStr   string
		filterValueStr string
		filter         query.FilterOption
		expectedMatch  bool
		expectedError  error
	}{
		{
			name:           "date is after filter date with gt operator",
			dateValueStr:   "2023-07-22T00:00:00Z",
			filterValueStr: "2023-07-21T00:00:00Z",
			filter:         query.FilterOption{Operator: "gt"},
			expectedMatch:  true,
			expectedError:  nil,
		},
		{
			name:           "date is after filter date with gt operator",
			dateValueStr:   "2023-07-22T00:00:00Z",
			filterValueStr: "2023-07-21",
			filter:         query.FilterOption{Operator: "gt"},
			expectedMatch:  true,
			expectedError:  nil,
		},
		{
			name:           "date is before filter date with lt operator",
			dateValueStr:   "2023-07-20T00:00:00Z",
			filterValueStr: "2023-07-21T00:00:00Z",
			filter:         query.FilterOption{Operator: "lt"},
			expectedMatch:  true,
			expectedError:  nil,
		},
		{
			name:           "date is before filter date with lt operator",
			dateValueStr:   "2023-07-20T00:00:00Z",
			filterValueStr: "2023-07-21",
			filter:         query.FilterOption{Operator: "lt"},
			expectedMatch:  true,
			expectedError:  nil,
		},
		{
			name:           "date eq exact start filter date with eq operator",
			dateValueStr:   "2023-07-20T00:00:00Z",
			filterValueStr: "2023-07-20",
			filter:         query.FilterOption{Operator: "eq"},
			expectedMatch:  true,
			expectedError:  nil,
		},
		{
			name:           "date eq filter date with eq operator",
			dateValueStr:   "2023-07-20T01:00:00Z",
			filterValueStr: "2023-07-20",
			filter:         query.FilterOption{Operator: "eq"},
			expectedMatch:  true,
			expectedError:  nil,
		},
		{
			name:           "date not eq exact end filter date with eq operator",
			dateValueStr:   "2023-07-21T00:00:00Z",
			filterValueStr: "2023-07-20",
			filter:         query.FilterOption{Operator: "eq"},
			expectedMatch:  false,
			expectedError:  nil,
		},
		{
			name:           "date eq filter date with eq operator",
			dateValueStr:   "2023-07-20T00:10:00Z",
			filterValueStr: "2023-07-20",
			filter:         query.FilterOption{Operator: "eq"},
			expectedMatch:  true,
			expectedError:  nil,
		},
		{
			name:           "date not eq filter date with eq operator",
			dateValueStr:   "2023-07-19T00:00:00Z",
			filterValueStr: "2023-07-20",
			filter:         query.FilterOption{Operator: "eq"},
			expectedMatch:  false,
			expectedError:  nil,
		},
		{
			name:           "invalid date value",
			dateValueStr:   "invalid-date",
			filterValueStr: "2023-07-21T00:00:00Z",
			filter:         query.FilterOption{Operator: "gt"},
			expectedMatch:  false,
			expectedError:  errors.New("value not valid RFC3339 date"),
		},
		{
			name:           "invalid filter date value",
			dateValueStr:   "2023-07-22T00:00:00Z",
			filterValueStr: "invalid-date",
			filter:         query.FilterOption{Operator: "gt"},
			expectedMatch:  false,
			expectedError:  errors.New("filter value not valid simple or RFC3339 date"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			match, err := query.MatchIfDate(tc.dateValueStr, tc.filterValueStr, tc.filter)
			assert.Equal(t, tc.expectedMatch, match)
			assert.Equal(t, tc.expectedError, err)
		})
	}
}
