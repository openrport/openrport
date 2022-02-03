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

			regx := regexp.MustCompile(`[^\\]\*+`)
			for _, filterValue := range filter.Values {
				hasUnescapedWildCard := regx.MatchString(filterValue)
				if !hasUnescapedWildCard {
					if strings.EqualFold(filterValue, clientFieldValueToMatchStr) {
						return true, nil
					}

					continue
				}

				filterValueRegex, err := regexp.Compile("(?i)" + strings.ReplaceAll(filterValue, "*", ".*"))
				if err != nil {
					if strings.EqualFold(filterValue, clientFieldValueToMatchStr) {
						return true, nil
					}
					continue
				}

				if filterValueRegex.MatchString(clientFieldValueToMatchStr) {
					return true, nil
				}
			}
		}
	}

	return false, nil
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
