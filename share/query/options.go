package query

import "net/http"

type Options struct {
	Sorts   []SortOption
	Filters []FilterOption
	Fields  []FieldsOption
}

func NewOptions(req *http.Request, sortsDefault map[string][]string, filtersDefault map[string][]string, fieldsDefault map[string][]string) *Options {
	qOptions := &Options{}

	sorts := ExtractSortOptions(req)
	if len(sorts) > 0 {
		qOptions.Sorts = sorts
	} else {
		qOptions.Sorts = ParseSortOptions(sortsDefault)
	}
	filters := ExtractFilterOptions(req)
	if len(filters) > 0 {
		qOptions.Filters = filters
	} else {
		qOptions.Filters = ParseFilterOptions(filtersDefault)
	}

	fields := ExtractFieldsOptions(req)
	if len(fields) > 0 {
		qOptions.Fields = fields
	} else {
		qOptions.Fields = ParseFieldsOptions(fieldsDefault)
	}

	return qOptions
}

func (o *Options) HasSorts() bool {
	return o.Sorts != nil && len(o.Sorts) > 0
}

func (o *Options) HasFilters() bool {
	return o.Filters != nil && len(o.Filters) > 0
}

func (o *Options) HasFields() bool {
	return o.Fields != nil && len(o.Fields) > 0
}

func ValidateOptions(options *Options, supportedSortFields map[string]bool, supportedFilterFields map[string]bool, supportedFields map[string]map[string]bool) error {
	if err := ValidateSortOptions(options.Sorts, supportedSortFields); err != nil {
		return err
	}
	if err := ValidateFilterOptions(options.Filters, supportedFilterFields); err != nil {
		return err
	}
	if err := ValidateFieldsOptions(options.Fields, supportedFields); err != nil {
		return err
	}

	return nil
}
