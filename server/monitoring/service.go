package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/server/api/monitoring"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type Service interface {
	SaveMeasurement(ctx context.Context, measurement *models.Measurement) error
	DeleteMeasurementsOlderThan(ctx context.Context, period time.Duration) (int64, error)
	GetClientMetricsLatest(ctx context.Context, clientID string, o *query.ListOptions) (*monitoring.ClientMetricsPayload, error)
	GetClientMetricsFiltered(ctx context.Context, clientID string, o *query.ListOptions) ([]monitoring.ClientMetricsPayload, error)
	GetClientProcessesLatest(ctx context.Context, clientID string) (*monitoring.ClientProcessesPayload, error)
	GetClientProcessesFiltered(ctx context.Context, clientID string, filters []query.FilterOption) (*monitoring.ClientProcessesPayload, error)
	GetClientMountpointsLatest(ctx context.Context, clientID string) (*monitoring.ClientMountpointsPayload, error)
	GetClientMountpointsFiltered(ctx context.Context, clientID string, filters []query.FilterOption) (*monitoring.ClientMountpointsPayload, error)
}

const layoutAPI = time.RFC3339
const layoutDb = "2006-01-02 15:04:05"
const maxLimit = 120
const maxDataFetchHours = 48
const maxDataFetchDuration = time.Duration(maxDataFetchHours) * time.Hour
const thresholdDownsamplingDuration = time.Duration(2) * time.Hour

type monitoringService struct {
	DBProvider DBProvider
}

func NewService(dbProvider DBProvider) Service {
	return &monitoringService{DBProvider: dbProvider}
}
func (s *monitoringService) SaveMeasurement(ctx context.Context, measurement *models.Measurement) error {
	return s.DBProvider.CreateMeasurement(ctx, measurement)
}

func (s *monitoringService) DeleteMeasurementsOlderThan(ctx context.Context, period time.Duration) (int64, error) {
	compare := time.Now().Add(-period)
	return s.DBProvider.DeleteMeasurementsBefore(ctx, compare)
}

func (s *monitoringService) GetClientProcessesLatest(ctx context.Context, clientID string) (*monitoring.ClientProcessesPayload, error) {
	return s.DBProvider.GetProcessesLatestByClientID(ctx, clientID)
}

func (s *monitoringService) GetClientProcessesFiltered(ctx context.Context, clientID string, filters []query.FilterOption) (*monitoring.ClientProcessesPayload, error) {
	if err := checkAllowedFilterOptions(filters, 1); err != nil {
		return nil, err
	}
	if err := parseAndConvertFilterValues(filters); err != nil {
		return nil, err
	}
	return s.DBProvider.GetProcessesNearestByClientID(ctx, clientID, filters)
}

func (s *monitoringService) GetClientMountpointsLatest(ctx context.Context, clientID string) (*monitoring.ClientMountpointsPayload, error) {
	return s.DBProvider.GetMountpointsLatestByClientID(ctx, clientID)
}

func (s *monitoringService) GetClientMountpointsFiltered(ctx context.Context, clientID string, filters []query.FilterOption) (*monitoring.ClientMountpointsPayload, error) {
	if err := checkAllowedFilterOptions(filters, 1); err != nil {
		return nil, err
	}
	if err := parseAndConvertFilterValues(filters); err != nil {
		return nil, err
	}
	return s.DBProvider.GetMountpointsNearestByClientID(ctx, clientID, filters)
}

func (s *monitoringService) GetClientMetricsLatest(ctx context.Context, clientID string, o *query.ListOptions) (*monitoring.ClientMetricsPayload, error) {
	return s.DBProvider.GetMetricsLatestByClientID(ctx, clientID, o.Fields)
}

func (s monitoringService) GetClientMetricsFiltered(ctx context.Context, clientID string, o *query.ListOptions) ([]monitoring.ClientMetricsPayload, error) {
	//query.SortFiltersByOperator(o.Filters)
	if err := checkAllowedFilterOptions(o.Filters, 2); err != nil {
		return nil, err
	}
	if err := checkAllowedFilterCombinations(o.Filters); err != nil {
		return nil, err
	}
	if err := parseAndConvertFilterValues(o.Filters); err != nil {
		return nil, err
	}
	if query.IsLimitFilter(o.Filters[1]) {
		return s.getClientMetricsFilteredLimited(ctx, clientID, o)
	}
	return s.getClientMetricsFilteredRange(ctx, clientID, o)
}

func (s *monitoringService) getClientMetricsFilteredLimited(ctx context.Context, clientID string, o *query.ListOptions) ([]monitoring.ClientMetricsPayload, error) {
	limit, _ := strconv.Atoi(o.Filters[1].Values[0])
	if limit < 1 || limit > maxLimit {
		return nil, errors.APIError{Message: fmt.Sprintf("Illegal limit (allowed: 1 to %d)", maxLimit), HTTPStatus: http.StatusBadRequest}
	}
	return s.DBProvider.GetMetricsListByClientID(ctx, clientID, o)
}

func (s *monitoringService) getClientMetricsFilteredRange(ctx context.Context, clientID string, o *query.ListOptions) ([]monitoring.ClientMetricsPayload, error) {
	lower, _ := time.Parse(layoutDb, o.Filters[0].Values[0])
	upper, _ := time.Parse(layoutDb, o.Filters[1].Values[0])

	if upper.Before(lower) {
		return nil, errors.APIError{Message: "Illegal time value (upper before lower)", HTTPStatus: http.StatusBadRequest}
	}
	span := upper.Sub(lower)
	if span > maxDataFetchDuration {
		return nil, errors.APIError{Message: fmt.Sprintf("Illegal period (max allowed: %d hours)", maxDataFetchHours), HTTPStatus: http.StatusBadRequest}
	}
	if span > thresholdDownsamplingDuration {
		return s.DBProvider.GetMetricsListDownsampledByClientID(ctx, clientID, span.Hours(), o)
	}

	return s.DBProvider.GetMetricsListByClientID(ctx, clientID, o)
}

func parseAndConvertFilterValues(filters []query.FilterOption) error {
	for _, fo := range filters {
		if (fo.Operator == query.FilterOperatorTypeGT) || (fo.Operator == query.FilterOperatorTypeLT) {
			ti, err := strconv.ParseInt(fo.Values[0], 10, 64)
			if err != nil {
				return errors.APIError{Message: fmt.Sprintf("Illegal timestamp value %s", fo.Values[0]), HTTPStatus: http.StatusBadRequest}
			}
			t := time.Unix(ti, 0)
			fo.Values[0] = t.Format(layoutDb)
			continue
		}

		if (fo.Operator == query.FilterOperatorTypeSince) || (fo.Operator == query.FilterOperatorTypeUntil) {
			t, err := time.Parse(layoutAPI, fo.Values[0])
			if err != nil {
				return errors.APIError{Message: "Illegal time value", HTTPStatus: http.StatusBadRequest}
			}
			//fo.Values[0] = strconv.FormatInt(t.Unix(), 10)
			fo.Values[0] = t.Format(layoutDb)
			continue
		}

		if query.IsLimitFilter(fo) {
			if _, err := strconv.Atoi(fo.Values[0]); err != nil {
				return errors.APIError{Message: "Illegal limit value", HTTPStatus: http.StatusBadRequest}
			}
		}
	}
	return nil
}

func checkAllowedFilterOptions(filters []query.FilterOption, filterCount int) error {
	if len(filters) != filterCount {
		return errors.APIError{
			Message:    "Illegal number of filter options",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	for _, filter := range filters {
		if len(filter.Values) != 1 {
			return errors.APIError{Message: "Too much filter option values", HTTPStatus: http.StatusBadRequest}
		}
	}

	return nil
}

func checkAllowedFilterCombinations(filters []query.FilterOption) error {
	if (filters[0].Operator == query.FilterOperatorTypeGT && filters[1].Operator == query.FilterOperatorTypeLT) ||
		(filters[0].Operator == query.FilterOperatorTypeGT && filters[1].Operator == query.FilterOperatorTypeEQ) ||
		(filters[0].Operator == query.FilterOperatorTypeSince && filters[1].Operator != query.FilterOperatorTypeUntil) ||
		(filters[0].Operator == query.FilterOperatorTypeSince && filters[1].Operator != query.FilterOperatorTypeEQ) {
		//these are the allowed filter combinations
	} else {
		return errors.APIError{Message: fmt.Sprintf("Illegal filter pair %s %s", filters[0].Expression, filters[1].Expression), HTTPStatus: http.StatusBadRequest}
	}

	return nil
}
