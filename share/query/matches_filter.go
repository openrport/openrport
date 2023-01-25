package query

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

func MatchesFilters(v interface{}, filterOptions []FilterOption) (bool, error) {
	valueMap, err := toMap(v)
	if err != nil {
		return false, err
	}
	for _, f := range filterOptions {
		matches, err := matchesFilter(valueMap, f)
		if err != nil {
			return false, err
		}
		if !matches {
			return false, nil
		}
	}

	return true, nil
}

func matchesFilter(valueMap map[string]interface{}, filter FilterOption) (bool, error) {
	matches := make(map[string]bool, len(filter.Values))

	for _, col := range filter.Column {
		clientFieldValueToMatch, ok := valueMap[col]
		if !ok {
			return false, fmt.Errorf("unsupported filter column: %s", col)
		}
		// Cast into slice if it's array field, otherwise set single value slice
		clientFieldSliceToMatch, ok := clientFieldValueToMatch.([]interface{})
		if !ok {
			clientFieldSliceToMatch = []interface{}{clientFieldValueToMatch}
		}

		for _, clientFieldValueToMatch := range clientFieldSliceToMatch {
			clientFieldValueToMatchStr := fmt.Sprint(clientFieldValueToMatch)

			// for each filter I cycle all the map matchFilter
			// OR == at least one filterOptions matches
			// AND == all filterOptions's need to match (count)
			for _, filterValue := range filter.Values {
				if matches[filterValue] { // this filter was already "assigned" to a match
					continue
				}
				hasUnescapedWildCard := strings.Contains(filterValue, "*")
				if !hasUnescapedWildCard {
					if strings.EqualFold(filterValue, clientFieldValueToMatchStr) {
						matches[filterValue] = true
					}
					continue
				}
				re := "(?i)^" + strings.ReplaceAll(filterValue, "*", ".*?") + "$"
				filterValueRegex, err := regexp.Compile(re)
				if err != nil {
					if strings.EqualFold(filterValue, clientFieldValueToMatchStr) {
						matches[filterValue] = true
					}
					continue
				}

				if filterValueRegex.MatchString(clientFieldValueToMatchStr) {
					matches[filterValue] = true
				}
			}
		}
	}

	switch filter.ValuesLogicalOperator {
	case FilterLogicalOperatorTypeAND:
		return len(matches) == len(filter.Values), nil
	}
	return len(matches) > 0, nil
}

func toMap(v interface{}) (map[string]interface{}, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	res := make(map[string]interface{})

	err = json.Unmarshal(bytes, &res)

	return res, err
}
