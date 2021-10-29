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
	GetClientLatest(ctx context.Context, clientID string) (*models.Measurement, error)
	GetClientMetricsOne(ctx context.Context, clientID string, o *query.ListOptions) (*monitoring.ClientMetricsPayload, error)
	GetClientMetricsList(ctx context.Context, clientID string, o *query.ListOptions) ([]monitoring.ClientMetricsPayload, error)
	GetClientProcessesLatest(ctx context.Context, clientID string) (*monitoring.ClientProcessesPayload, error)
	GetClientProcessesFiltered(ctx context.Context, clientID string, filters []query.FilterOption) (*monitoring.ClientProcessesPayload, error)
	GetClientMountpointsLatest(ctx context.Context, clientID string) (*monitoring.ClientMountpointsPayload, error)
	GetClientMountpointsFiltered(ctx context.Context, clientID string, filters []query.FilterOption) (*monitoring.ClientMountpointsPayload, error)
}

var layoutSinceUntil = "2006-01-02:15:04:05"
var maxDataFetchHours = 48
var maxDataFetchDuration = time.Duration(maxDataFetchHours) * time.Hour

//var limitDownSamplingHours = 2

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

func (s *monitoringService) GetClientLatest(ctx context.Context, clientID string) (*models.Measurement, error) {
	return s.DBProvider.GetClientLatest(ctx, clientID)
}

func (s *monitoringService) GetClientProcessesLatest(ctx context.Context, clientID string) (*monitoring.ClientProcessesPayload, error) {
	return s.DBProvider.GetProcessesLatestByClientID(ctx, clientID)
}

func (s *monitoringService) GetClientProcessesFiltered(ctx context.Context, clientID string, filters []query.FilterOption) (*monitoring.ClientProcessesPayload, error) {
	t, err := time.Parse(layoutSinceUntil, filters[0].Values[0])
	if err != nil {
		return nil, fmt.Errorf("illegal time format:%v", filters[0].Values[0])
	}
	return s.DBProvider.GetProcessesNearestByClientID(ctx, clientID, t)
}

func (s *monitoringService) GetClientMountpointsLatest(ctx context.Context, clientID string) (*monitoring.ClientMountpointsPayload, error) {
	return s.DBProvider.GetMountpointsLatestByClientID(ctx, clientID)
}

func (s *monitoringService) GetClientMountpointsFiltered(ctx context.Context, clientID string, filters []query.FilterOption) (*monitoring.ClientMountpointsPayload, error) {
	t, err := time.Parse(layoutSinceUntil, filters[0].Values[0])
	if err == nil {
		return nil, fmt.Errorf("illegal time format:%v", filters[0].Values[0])
	}
	return s.DBProvider.GetMountpointsNearestByClientID(ctx, clientID, t)
}

func (s *monitoringService) GetClientMetricsOne(ctx context.Context, clientID string, o *query.ListOptions) (*monitoring.ClientMetricsPayload, error) {
	return s.DBProvider.GetMetricsLatestByClientID(ctx, clientID, o.Fields)
}

func (s monitoringService) GetClientMetricsList(ctx context.Context, clientID string, o *query.ListOptions) ([]monitoring.ClientMetricsPayload, error) {
	query.SortFiltersByOperator(o.Filters)
	if err := checkFilterOptions(o.Filters); err != nil {
		return nil, err
	}
	if err := parseAndConvertFilterValues(o.Filters); err != nil {
		return nil, err
	}
	if err := validateFilterValues(o.Filters); err != nil {
		return nil, err
	}
	return s.DBProvider.GetMetricsListByClientID(ctx, clientID, o)
}

func parseAndConvertFilterValues(filters []query.FilterOption) error {
	for _, fo := range filters {
		if (fo.Operator == query.FilterOperatorTypeGT) || (fo.Operator == query.FilterOperatorTypeLT) {
			if _, err := strconv.ParseInt(fo.Values[0], 10, 64); err != nil {
				return errors.APIError{Message: fmt.Sprintf("Illegal timestamp value %s", fo.Values[0]), HTTPStatus: http.StatusBadRequest}
			}
			continue
		}

		if (fo.Operator == query.FilterOperatorTypeSince) || (fo.Operator == query.FilterOperatorTypeUntil) {
			t, err := time.Parse(layoutSinceUntil, fo.Values[0])
			if err != nil {
				return errors.APIError{Message: "Illegal time value", HTTPStatus: http.StatusBadRequest}
			}
			fo.Values[0] = strconv.FormatInt(t.Unix(), 10)
		}
	}
	return nil
}

func validateFilterValues(filters []query.FilterOption) error {
	lower, _ := strconv.Atoi(filters[0].Values[0])
	upper, _ := strconv.Atoi(filters[1].Values[0])

	if lower > upper {
		return errors.APIError{Message: "Illegal time value (lower bound > upper bound)", HTTPStatus: http.StatusBadRequest}
	}
	span := time.Duration(upper)*time.Second - time.Duration(lower)*time.Second
	if span > maxDataFetchDuration {
		return errors.APIError{Message: fmt.Sprintf("Illegal period (max allowed: %d hours)", maxDataFetchHours), HTTPStatus: http.StatusBadRequest}
	}

	return nil
}

func checkFilterOptions(filters []query.FilterOption) error {
	if len(filters) != 2 {
		return errors.APIError{
			Message:    "Illegal number of filter options",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	if (filters[0].Operator == query.FilterOperatorTypeGT && filters[1].Operator != query.FilterOperatorTypeLT) ||
		(filters[0].Operator == query.FilterOperatorTypeSince && filters[1].Operator != query.FilterOperatorTypeUntil) {
		return errors.APIError{Message: fmt.Sprintf("Illegal filter pair %s %s", filters[0].Operator.String(), filters[1].Operator.String()), HTTPStatus: http.StatusBadRequest}
	}

	if len(filters[0].Values) != 1 || len(filters[1].Values) != 1 {
		return errors.APIError{Message: "Too much filter option values", HTTPStatus: http.StatusBadRequest}
	}

	return nil
}
