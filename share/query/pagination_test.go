package query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/share/query"
)

func TestValidatePagination(t *testing.T) {
	testCases := []struct {
		Name          string
		Pagination    *query.Pagination
		ExpectedLimit string
		ExpectedErrs  []string
	}{
		{
			Name: "set default limit",
			Pagination: &query.Pagination{
				Offset: "0",
			},
			ExpectedLimit: "10",
		}, {
			Name: "ok",
			Pagination: &query.Pagination{
				Offset: "0",
				Limit:  "15",
			},
			ExpectedLimit: "15",
		}, {

			Name: "not numbers",
			Pagination: &query.Pagination{
				Offset: "ab",
				Limit:  "cd",
			},
			ExpectedErrs: []string{
				"pagination limit must be a number",
				"pagination offset must be a number",
			},
		}, {
			Name: "negative",
			Pagination: &query.Pagination{
				Offset: "-1",
				Limit:  "0",
			},
			ExpectedErrs: []string{
				"pagination limit must be positive",
				"pagination offset must not be negative",
			},
		}, {
			Name: "too big",
			Pagination: &query.Pagination{
				Offset: "0",
				Limit:  "1000",
			},
			ExpectedErrs: []string{
				"pagination limit too big (1000) maximum is 100",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			errs := query.ValidatePagination(tc.Pagination, &query.PaginationConfig{
				DefaultLimit: 10,
				MaxLimit:     100,
			})

			assert.Equal(t, len(tc.ExpectedErrs), len(errs))
			for i := range tc.ExpectedErrs {
				assert.Equal(t, tc.ExpectedErrs[i], errs[i].Message)
			}
			if tc.ExpectedLimit != "" {
				assert.Equal(t, tc.ExpectedLimit, tc.Pagination.Limit)
			}
		})
	}
}

func TestGetStartEnd(t *testing.T) {
	testCases := []struct {
		Name          string
		TotalCount    int
		ExpectedStart int
		ExpectedEnd   int
	}{
		{
			Name:          "total greater",
			TotalCount:    30,
			ExpectedStart: 10,
			ExpectedEnd:   20,
		},
		{
			Name:          "total less than end",
			TotalCount:    15,
			ExpectedStart: 10,
			ExpectedEnd:   15,
		},
		{
			Name:          "total less than start",
			TotalCount:    5,
			ExpectedStart: 5,
			ExpectedEnd:   5,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			pagination := &query.Pagination{
				ValidatedOffset: 10,
				ValidatedLimit:  10,
			}
			start, end := pagination.GetStartEnd(tc.TotalCount)

			assert.Equal(t, tc.ExpectedStart, start)
			assert.Equal(t, tc.ExpectedEnd, end)
		})
	}
}
