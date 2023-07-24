package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
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

		// check if value is a map[string]interface{} // assume values are reasonably printable into string
		clientFieldMapToMatch, ok := clientFieldValueToMatch.(map[string]interface{})
		if ok {
			clientFieldSliceToMatch = []interface{}{}
			for key, val := range clientFieldMapToMatch {
				clientFieldSliceToMatch = append(clientFieldSliceToMatch, fmt.Sprintf("%v: %v", key, val))
			}
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

				match, err := MatchIfDate(clientFieldValueToMatchStr, filterValue, filter)
				if err == nil {
					if match {
						matches[filterValue] = true
					}
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

func MatchIfDate(dateValueStr string, filterValueStr string, filter FilterOption) (match bool, err error) {
	filterDateValue, err := time.Parse(time.RFC3339, filterValueStr)
	if err != nil {
		filterDateValue, err = time.Parse("2006-01-02", filterValueStr)
	}
	if err == nil {
		dateValue, err := time.Parse(time.RFC3339, dateValueStr)
		if err == nil {
			if filter.Operator == "gt" {
				if dateValue.After(filterDateValue) {
					return true, nil
				}
				return false, nil
			}
			if filter.Operator == "lt" {
				if dateValue.Before(filterDateValue) {
					return true, nil
				}
			}
			// perform eq by check if outside the filter date
			if filter.Operator == "eq" {
				if dateValue.Before(filterDateValue) {
					return false, nil
				}
				if dateValue.After(filterDateValue.Add((1 * time.Hour * 24) - (1 * time.Nanosecond))) {
					return false, nil
				}
				return true, nil
			}
			return false, nil
		}
		return false, errors.New("value not valid RFC3339 date")
	}
	return false, errors.New("filter value not valid simple or RFC3339 date")
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
