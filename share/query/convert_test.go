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
			ExpectedParams: nil,
		}, {
			Name:           "nil options",
			Options:        nil,
			ExpectedQuery:  "SELECT * FROM res1",
			ExpectedParams: nil,
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
						Column: []string{"field1"},
						Values: []string{"val1", "val2", "val3"},
					},
					{
						Column: []string{"field2"},
						Values: []string{"value2"},
					},
					{
						Column: []string{"field3", "field4"},
						Values: []string{"value1", "value3"},
					},
					{
						Column: []string{"field5"},
						Values: []string{""},
					},
					{
						Column: []string{"field6"},
						Values: []string{"abc*", "*def*"},
					},
				},
				Fields: []query.FieldsOption{
					{
						Resource: "res1",
						Fields:   []string{"field1", "field2"},
					},
				},
				Pagination: &query.Pagination{
					Offset: "10",
					Limit:  "5",
				},
			},
			ExpectedQuery:  "SELECT res1.field1, res1.field2 FROM res1 WHERE (field1 = ? OR field1 = ? OR field1 = ?) AND field2 = ? AND (field3 = ? OR field3 = ? OR field4 = ? OR field4 = ?) AND (field5 = ? OR field5 IS NULL) AND (LOWER(field6) LIKE ? OR LOWER(field6) LIKE ?) ORDER BY field1 ASC, field2 DESC LIMIT ? OFFSET ?",
			ExpectedParams: []interface{}{"val1", "val2", "val3", "value2", "value1", "value3", "value1", "value3", "", "abc%", "%def%", "5", "10"},
		},
		{
			Name: "wildcard option",
			Options: &query.ListOptions{
				Sorts: []query.SortOption{
					{
						Column: "field1",
						IsASC:  true,
					},
				},
				Filters: []query.FilterOption{
					{
						Column: []string{"field1"},
						Values: []string{"val*"},
					},
					{
						Column: []string{"field2"},
						Values: []string{"val*"},
					},
				},
			},
			ExpectedQuery:  `SELECT * FROM res1 WHERE field1 LIKE ? ESCAPE '\' AND field2 LIKE ? ESCAPE '\' ORDER BY field1 ASC`,
			ExpectedParams: []interface{}{"val%", "val%"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Run("convert", func(t *testing.T) {
				t.Parallel()

				query, params := query.ConvertListOptionsToQuery(tc.Options, "SELECT * FROM res1")

				assert.Equal(t, tc.ExpectedQuery, query)
				assert.Equal(t, tc.ExpectedParams, params)
			})
			t.Run("append", func(t *testing.T) {
				t.Parallel()

				query, params := query.AppendOptionsToQuery(tc.Options, "SELECT * FROM res1", []interface{}{123, "abc"})

				assert.Equal(t, tc.ExpectedQuery, query)
				assert.Equal(t, append([]interface{}{123, "abc"}, tc.ExpectedParams...), params)
			})

		})
	}

}
