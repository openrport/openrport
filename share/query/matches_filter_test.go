package query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/share/query"
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
