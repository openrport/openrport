package query

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertGetParamsToFilterOptions(t *testing.T) {
	testCases := []struct {
		name                string
		inputQuery          string
		expectedListOptions *ListOptions
	}{
		{
			name: "empty_query",
			expectedListOptions: &ListOptions{
				Sorts:   []SortOption{},
				Filters: []FilterOption{},
			},
		},
		{
			name:       "not_matching_query",
			inputQuery: "sord=date&filter=123&filter[]=345&sort=&sort=  &filter[k]=&filter[l]=    ",
			expectedListOptions: &ListOptions{
				Sorts:   []SortOption{},
				Filters: []FilterOption{},
			},
		},
		{
			name:       "all_possible_sorts_and_filters",
			inputQuery: "sort=date&sort=-user&filter[field1]=val1&filter[field1]=val2,val3&filter[field2]=value2,value3",
			expectedListOptions: &ListOptions{
				Sorts: []SortOption{
					{
						Column: "date",
						IsASC:  true,
					},
					{
						Column: "user",
						IsASC:  false,
					},
				},
				Filters: []FilterOption{
					{
						Column: "field1",
						Values: []string{"val1", "val2", "val3"},
					},
					{
						Column: "field2",
						Values: []string{"value2", "value3"},
					},
				},
			},
		},
	}

	for i := range testCases {
		t.Run(testCases[i].name, func(t *testing.T) {
			inputURL, err := url.Parse("/someu?" + testCases[i].inputQuery)
			require.NoError(t, err)

			req := &http.Request{
				URL: inputURL,
			}

			actualListOptions := GetSortAndFilterOptions(req)

			assert.ElementsMatch(t, testCases[i].expectedListOptions.Sorts, actualListOptions.Sorts)
			assert.ElementsMatch(t, testCases[i].expectedListOptions.Filters, actualListOptions.Filters)
		})
	}
}
