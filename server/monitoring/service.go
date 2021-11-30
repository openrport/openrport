package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/share/models"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type Service interface {
	SaveMeasurement(ctx context.Context, measurement *models.Measurement) error
	DeleteMeasurementsOlderThan(ctx context.Context, period time.Duration) (int64, error)
	ListClientMetrics(context.Context, string, *query.ListOptions) (*api.SuccessPayload, error)
	ListClientGraphMetrics(context.Context, string, *query.ListOptions) ([]*ClientGraphMetricsPayload, error)
	ListClientMountpoints(context.Context, string, *query.ListOptions) (*api.SuccessPayload, error)
	ListClientProcesses(context.Context, string, *query.ListOptions) (*api.SuccessPayload, error)
}

const layoutAPI = time.RFC3339
const layoutDb = "2006-01-02 15:04:05"
const defaultLimitMetrics = 1
const maxLimitMetrics = 120
const defaultLimitMountpoints = 1
const maxLimitMountpoints = 100
const defaultLimitProcesses = 1
const maxLimitProcesses = 10
const minDownsamplingHours = 2
const minDownsamplingDuration = time.Duration(minDownsamplingHours) * time.Hour
const maxDownsamplingHours = 48
const maxDownsamplingDuration = time.Duration(maxDownsamplingHours) * time.Hour

type monitoringService struct {
	DBProvider DBProvider
}

func NewService(dbProvider DBProvider) Service {
	return &monitoringService{DBProvider: dbProvider}
}
func (s *monitoringService) SaveMeasurement(ctx context.Context, measurement *models.Measurement) error {
	measurement.Timestamp = time.Now().UTC()
	return s.DBProvider.CreateMeasurement(ctx, measurement)
}

func (s *monitoringService) DeleteMeasurementsOlderThan(ctx context.Context, period time.Duration) (int64, error) {
	compare := time.Now().Add(-period)
	return s.DBProvider.DeleteMeasurementsBefore(ctx, compare)
}

func (s *monitoringService) ListClientGraphMetrics(ctx context.Context, clientID string, lo *query.ListOptions) ([]*ClientGraphMetricsPayload, error) {
	err := query.ValidateListOptions(lo, ClientGraphMetricsSortFields, ClientGraphMetricsFilterFields, ClientGraphMetricsFields, nil)
	if err != nil {
		return nil, err
	}
	if err := parseAndConvertFilterValues(lo.Filters); err != nil {
		return nil, err
	}

	if len(lo.Filters) != 2 {
		return nil, errors.APIError{
			Message:    "Illegal number of filter options",
			HTTPStatus: http.StatusBadRequest,
		}
	}

	if (lo.Filters[0].Operator == query.FilterOperatorTypeGT && lo.Filters[1].Operator == query.FilterOperatorTypeLT) ||
		(lo.Filters[0].Operator == query.FilterOperatorTypeSince && lo.Filters[1].Operator == query.FilterOperatorTypeUntil) {
		//these are the allowed filter combinations
	} else {
		return nil, errors.APIError{Message: fmt.Sprintf("Illegal filter pair %s %s", lo.Filters[0].Expression, lo.Filters[1].Expression), HTTPStatus: http.StatusBadRequest}
	}

	lower, _ := time.Parse(layoutDb, lo.Filters[0].Values[0])
	upper, _ := time.Parse(layoutDb, lo.Filters[1].Values[0])

	if upper.Before(lower) {
		return nil, errors.APIError{Message: "Illegal time value (upper before lower)", HTTPStatus: http.StatusBadRequest}
	}
	span := upper.Sub(lower)
	if span < minDownsamplingDuration || span > maxDownsamplingDuration {
		return nil, errors.APIError{Message: fmt.Sprintf("Illegal period (min,max allowed: %d,%d hours)", minDownsamplingHours, maxDownsamplingHours), HTTPStatus: http.StatusBadRequest}
	}

	return s.DBProvider.ListGraphMetricsByClientID(ctx, clientID, span.Hours(), lo)
}

func (s *monitoringService) ListClientMetrics(ctx context.Context, clientID string, options *query.ListOptions) (*api.SuccessPayload, error) {
	err := query.ValidateListOptions(options, ClientMetricsSortFields, ClientMetricsFilterFields, ClientMetricsFields, &query.PaginationConfig{
		DefaultLimit: defaultLimitMetrics,
		MaxLimit:     maxLimitMetrics,
	})
	if err != nil {
		return nil, err
	}
	if err := parseAndConvertFilterValues(options.Filters); err != nil {
		return nil, err
	}

	entries, err := s.DBProvider.ListMetricsByClientID(ctx, clientID, options)
	if err != nil {
		return nil, err
	}
	count, err := s.DBProvider.CountByClientID(ctx, clientID, options)
	if err != nil {
		return nil, err
	}

	return &api.SuccessPayload{
		Data: entries,
		Meta: api.NewMeta(count),
	}, nil
}

func (s *monitoringService) ListClientMountpoints(ctx context.Context, clientID string, options *query.ListOptions) (*api.SuccessPayload, error) {
	err := query.ValidateListOptions(options, ClientMountpointsSortFields, ClientMountpointsFilterFields, ClientMountpointsFields, &query.PaginationConfig{
		DefaultLimit: defaultLimitMountpoints,
		MaxLimit:     maxLimitMountpoints,
	})
	if err != nil {
		return nil, err
	}
	if err := parseAndConvertFilterValues(options.Filters); err != nil {
		return nil, err
	}

	entries, err := s.DBProvider.ListMountpointsByClientID(ctx, clientID, options)
	if err != nil {
		return nil, err
	}
	count, err := s.DBProvider.CountByClientID(ctx, clientID, options)
	if err != nil {
		return nil, err
	}

	return &api.SuccessPayload{
		Data: entries,
		Meta: api.NewMeta(count),
	}, nil
}

func (s *monitoringService) ListClientProcesses(ctx context.Context, clientID string, options *query.ListOptions) (*api.SuccessPayload, error) {
	err := query.ValidateListOptions(options, ClientProcessesSortFields, ClientProcessesFilterFields, ClientProcessesFields, &query.PaginationConfig{
		DefaultLimit: defaultLimitProcesses,
		MaxLimit:     maxLimitProcesses,
	})
	if err != nil {
		return nil, err
	}
	if err := parseAndConvertFilterValues(options.Filters); err != nil {
		return nil, err
	}

	entries, err := s.DBProvider.ListProcessesByClientID(ctx, clientID, options)
	if err != nil {
		return nil, err
	}
	count, err := s.DBProvider.CountByClientID(ctx, clientID, options)
	if err != nil {
		return nil, err
	}

	return &api.SuccessPayload{
		Data: entries,
		Meta: api.NewMeta(count),
	}, nil
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
	}
	return nil
}
