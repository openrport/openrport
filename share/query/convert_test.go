package query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudradar-monitoring/rport/share/query"
)

func TestConvertListOptionsToQuery(t *testing.T) {
	testCases := []struct {
		Name           string
		Options        *query.ListOptions
		ExpectedQuery  string
		ExpectedParams []interface{}
	}{
		{
			Name:           "no options",
			Options:        &query.ListOptions{},
			ExpectedQuery:  "SELECT * FROM res1",
			ExpectedParams: []interface{}{},
		}, {
			Name: "mixed options",
			Options: &query.ListOptions{
				Sorts: []query.SortOption{
					{
						Column: "field1",
						IsASC:  true,
					},
					{
						Column: "field2",
						IsASC:  false,
					},
				},
				Filters: []query.FilterOption{
					{
						Column: "field1",
						Values: []string{"val1", "val2", "val3"},
					},
					{
						Column: "field2",
						Values: []string{"value2"},
					},
				},
				Fields: []query.FieldsOption{
					{
						Resource: "res1",
						Fields:   []string{"field1", "field2"},
					},
				},
			},
			ExpectedQuery:  "SELECT res1.field1, res1.field2 FROM res1 WHERE (field1 = ? OR field1 = ? OR field1 = ?) AND field2 = ? ORDER BY field1 ASC, field2 DESC",
			ExpectedParams: []interface{}{"val1", "val2", "val3", "value2"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			query, params := query.ConvertListOptionsToQuery(tc.Options, "SELECT * FROM res1")

			assert.Equal(t, tc.ExpectedQuery, query)
			assert.Equal(t, tc.ExpectedParams, params)
		})
	}

}

func TestConvertOptionsToQuery(t *testing.T) {
	testCases := []struct {
		Name           string
		Query          string
		Options        *query.Options
		Params         []interface{}
		ExpectedQuery  string
		ExpectedParams []interface{}
	}{
		{
			Name:  "fields, no filters, no sorts",
			Query: "SELECT * FROM measurements as metrics",
			Options: &query.Options{
				Fields: []query.FieldsOption{
					{
						Resource: "metrics",
						Fields:   []string{"field1", "field2"},
					},
				},
			},
			Params:         []interface{}{},
			ExpectedQuery:  "SELECT metrics.field1, metrics.field2 FROM measurements as metrics",
			ExpectedParams: []interface{}{},
		}, {
			Name:  "fields, filters, sorts, params",
			Query: "SELECT * FROM measurements as metrics WHERE client_id = ?",
			Options: &query.Options{
				Sorts: []query.SortOption{
					{
						Column: "timestamp",
						IsASC:  false,
					},
				},
				Filters: []query.FilterOption{
					{
						Column:   "timestamp",
						Operator: query.FilterOperatorTypeGT,
						Values:   []string{"val1"},
					},
					{
						Column:   "timestamp",
						Operator: query.FilterOperatorTypeLT,
						Values:   []string{"value2"},
					},
				},
				Fields: []query.FieldsOption{
					{
						Resource: "metrics",
						Fields:   []string{"field1", "field2"},
					},
				},
			},
			ExpectedQuery:  "SELECT metrics.field1, metrics.field2 FROM measurements as metrics WHERE client_id = ? AND timestamp > ? AND timestamp < ? ORDER BY timestamp DESC",
			ExpectedParams: []interface{}{"val1", "value2"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()

			query, params := query.ConvertOptionsToQuery(tc.Options, tc.Query, tc.Params)

			assert.Equal(t, tc.ExpectedQuery, query)
			assert.Equal(t, tc.ExpectedParams, params)
		})
	}

}
