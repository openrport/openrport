package query

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRetrieveOptions(t *testing.T) {
	testCases := []struct {
		name                    string
		inputQuery              string
		expectedRetrieveOptions *RetrieveOptions
	}{
		{
			name: "empty_query",
			expectedRetrieveOptions: &RetrieveOptions{
				Fields: []FieldsOption{},
			},
		},
		{
			name:       "not_matching_query",
			inputQuery: "fields=abc&fields[]=def",
			expectedRetrieveOptions: &RetrieveOptions{
				Fields: []FieldsOption{},
			},
		},
		{
			name:       "all_possible_sorts_and_filters",
			inputQuery: "fields[res1]=f1,f2&fields[res2]=f1,f3",
			expectedRetrieveOptions: &RetrieveOptions{
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

			actualRetrieveOptions := GetRetrieveOptions(req)

			assert.ElementsMatch(t, testCases[i].expectedRetrieveOptions.Fields, actualRetrieveOptions.Fields)
		})
	}
}

func TestValidateRetrieveOptionsWithErrors(t *testing.T) {
	supportedFields := map[string]map[string]bool{
		"res1": map[string]bool{
			"f1": true,
			"f2": true,
		},
	}
	options := &RetrieveOptions{
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
	}

	err := ValidateRetrieveOptions(options, supportedFields)
	assert.Equal(t, err.Error(), `unsupported field "f3" for resource "res1", unsupported resource in fields: "res2"`)
}

func TestValidateRetrieveOptionsOk(t *testing.T) {
	supportedFields := map[string]map[string]bool{
		"res1": map[string]bool{
			"f1": true,
			"f2": true,
		},
	}
	options := &RetrieveOptions{
		Fields: []FieldsOption{
			{
				Resource: "res1",
				Fields:   []string{"f1", "f2"},
			},
		},
	}

	err := ValidateRetrieveOptions(options, supportedFields)
	assert.NoError(t, err)
}
