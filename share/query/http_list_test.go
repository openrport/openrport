package query

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetListOptions(t *testing.T) {
	testCases := []struct {
		name                string
		inputQuery          string
		expectedListOptions *ListOptions
	}{
		{
			name: "empty_query",
			expectedListOptions: &ListOptions{
				Sorts:      []SortOption{},
				Filters:    []FilterOption{},
				Fields:     []FieldsOption{},
				Pagination: &Pagination{Offset: "0"},
			},
		},
		{
			name:       "not_matching_query",
			inputQuery: "sord=date&filter=123&filter[]=345&fields=abc&fields[]=def&page[other]=3",
			expectedListOptions: &ListOptions{
				Sorts:      []SortOption{},
				Filters:    []FilterOption{},
				Fields:     []FieldsOption{},
				Pagination: &Pagination{Offset: "0"},
			},
		},
		{
			name:       "all_possible_sorts_and_filters",
			inputQuery: "sort=date&sort=-user&filter[field1]=val1&filter[field1]=val2,val3&filter[field2]=value2,value3&filter[field3]=&fields[res1]=f1,f2&fields[res2]=f1,f3&page[offset]=3&page[limit]=10",
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
						Column: []string{"field1"},
						Values: []string{"val1", "val2", "val3"},
					},
					{
						Column: []string{"field2"},
						Values: []string{"value2", "value3"},
					},
					{
						Column: []string{"field3"},
						Values: []string{""},
					},
				},
				Fields: []FieldsOption{
					{
						Resource: "res1",
						Fields:   []string{"f1", "f2"},
					},
					{
						Resource: "res2",
						Fields:   []string{"f1", "f3"},
					},
				},
				Pagination: &Pagination{
					Offset: "3",
					Limit:  "10",
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

			actualListOptions := GetListOptions(req)

			assert.ElementsMatch(t, testCases[i].expectedListOptions.Sorts, actualListOptions.Sorts)
			assert.ElementsMatch(t, testCases[i].expectedListOptions.Filters, actualListOptions.Filters)
			assert.ElementsMatch(t, testCases[i].expectedListOptions.Fields, actualListOptions.Fields)
			assert.Equal(t, testCases[i].expectedListOptions.Pagination, actualListOptions.Pagination)
		})
	}
}

func TestValidateListOptionsWithErrors(t *testing.T) {
	supportedFields := map[string]map[string]bool{
		"res1": map[string]bool{
			"f1": true,
			"f2": true,
		},
	}
	supportedSorts := map[string]bool{
		"f3": true,
		"f4": true,
	}
	supportedFilters := map[string]bool{
		"f4": true,
		"f5": true,
	}
	paginationConfig := &PaginationConfig{
		MaxLimit: 10,
	}
	options := &ListOptions{
		Sorts: []SortOption{
			{
				Column: "f4",
				IsASC:  true,
			},
			{
				Column: "f5",
				IsASC:  false,
			},
		},
		Filters: []FilterOption{
			{
				Column: []string{"f3"},
				Values: []string{"v1", "v2"},
			},
			{
				Column: []string{"f4"},
				Values: []string{"v1", "v2"},
			},
		},
		Fields: []FieldsOption{
			{
				Resource: "res1",
				Fields:   []string{"f1", "f3"},
			},
			{
				Resource: "res2",
				Fields:   []string{"f1", "f3"},
			},
		},
		Pagination: &Pagination{
			Offset: "-1",
			Limit:  "20",
		},
	}

	err := ValidateListOptions(options, supportedSorts, supportedFilters, supportedFields, paginationConfig)
	assert.Equal(t, err.Error(), `unsupported sort field 'f5', unsupported filter field 'filter[f3]', unsupported field "f3" for resource "res1", unsupported resource in fields: "res2", pagination limit too big (20) maximum is 10, pagination offset must not be negative`)
}

func TestValidateListOptionsOk(t *testing.T) {
	supportedFields := map[string]map[string]bool{
		"res1": map[string]bool{
			"f1": true,
			"f2": true,
		},
	}
	supportedSorts := map[string]bool{
		"f3": true,
		"f4": true,
	}
	supportedFilters := map[string]bool{
		"f4": true,
		"f5": true,
	}
	paginationConfig := &PaginationConfig{
		MaxLimit: 10,
	}
	options := &ListOptions{
		Sorts: []SortOption{
			{
				Column: "f3",
				IsASC:  true,
			},
			{
				Column: "f4",
				IsASC:  false,
			},
		},
		Filters: []FilterOption{
			{
				Column: []string{"f4"},
				Values: []string{"v1", "v2"},
			},
			{
				Column: []string{"f5"},
				Values: []string{"v1", "v2"},
			},
		},
		Fields: []FieldsOption{
			{
				Resource: "res1",
				Fields:   []string{"f1", "f2"},
			},
		},
		Pagination: &Pagination{
			Offset: "3",
			Limit:  "10",
		},
	}

	err := ValidateListOptions(options, supportedSorts, supportedFilters, supportedFields, paginationConfig)
	assert.NoError(t, err)
}

func TestValidateListOptionsNoFieldsNoPagination(t *testing.T) {
	supportedSortAndFilters := map[string]bool{
		"f3": true,
		"f4": true,
	}
	options := &ListOptions{
		Sorts: []SortOption{
			{
				Column: "f3",
				IsASC:  true,
			},
			{
				Column: "f4",
				IsASC:  false,
			},
		},
		Filters: []FilterOption{
			{
				Column: []string{"f3"},
				Values: []string{"v1", "v2"},
			},
			{
				Column: []string{"f4"},
				Values: []string{"v1", "v2"},
			},
		},
		Fields: []FieldsOption{
			{
				Resource: "res1",
				Fields:   []string{"f1", "f2"},
			},
		},
		Pagination: &Pagination{
			Limit:  "10",
			Offset: "3",
		},
	}

	err := ValidateListOptions(options, supportedSortAndFilters, supportedSortAndFilters, nil, nil)
	assert.NoError(t, err)

	assert.Nil(t, options.Fields)
	assert.Nil(t, options.Pagination)
}
