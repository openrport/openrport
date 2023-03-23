package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/realvnc-labs/rport/server/api"
	"github.com/realvnc-labs/rport/server/api/errors"
	"github.com/realvnc-labs/rport/share/models"
	"github.com/realvnc-labs/rport/share/query"
)

type Service interface {
	SaveMeasurement(ctx context.Context, measurement *models.Measurement) error
	DeleteMeasurementsOlderThan(ctx context.Context, period time.Duration) (int64, error)
	ListClientMetrics(context.Context, string, *query.ListOptions) (*api.SuccessPayload, error)
	ListClientGraph(context.Context, string, *query.ListOptions, string, *models.NetworkCard, *models.NetworkCard) (*api.SuccessPayload, error)
	ListClientGraphMetrics(context.Context, string, *query.ListOptions, *query.RequestInfo, bool, bool) (*api.SuccessPayload, error)
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
const oneMBitBytes = 125000.0 // for converting MBits to Bytes

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

func (s *monitoringService) ListClientGraphMetrics(ctx context.Context, clientID string, lo *query.ListOptions, ri *query.RequestInfo, netLan bool, netWan bool) (*api.SuccessPayload, error) {
	span, err := s.validateAndParseGraphOptions(lo)
	if err != nil {
		return nil, err
	}

	entries, err := s.DBProvider.ListGraphMetricsByClientID(ctx, clientID, span.Hours(), lo)
	if err != nil {
		return nil, err
	}

	links := &GraphMetricsLinksPayload{
		CPUUsagePercent: NewGraphMetricsLink(ri, LinkCPUPercent),
		MemUsagePercent: NewGraphMetricsLink(ri, LinkMemPercent),
		IOUsagePercent:  NewGraphMetricsLink(ri, LinkIOPercent),
	}
	if netLan {
		links.NetLanUsagePercent = NewGraphMetricsLink(ri, LinkNetPercentLan)
		links.NetLanUsageBPS = NewGraphMetricsLink(ri, LinkNetBPSLan)
	}
	if netWan {
		links.NetWanUsagePercent = NewGraphMetricsLink(ri, LinkNetPercentWan)
		links.NetWanUsageBPS = NewGraphMetricsLink(ri, LinkNetBPSWan)
	}

	return &api.SuccessPayload{
		Data:  entries,
		Links: links,
	}, nil
}

func (s *monitoringService) ListClientGraph(ctx context.Context, clientID string, lo *query.ListOptions, graph string, lanCard *models.NetworkCard, wanCard *models.NetworkCard) (*api.SuccessPayload, error) {
	if strings.HasSuffix(graph, "_lan") && lanCard == nil ||
		strings.HasSuffix(graph, "_wan") && wanCard == nil {
		return nil, errors.APIError{
			Message:    fmt.Sprintf("graph data %s not available for client with id %s", graph, clientID),
			HTTPStatus: http.StatusNotFound,
		}
	}

	span, err := s.validateAndParseGraphOptions(lo)
	if err != nil {
		return nil, err
	}

	entries, err := s.DBProvider.ListGraphByClientID(ctx, clientID, span.Hours(), lo, graph)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(graph, "net_usage_percent_") {
		calculatePercentValues(&entries, lanCard, wanCard)
	}

	return &api.SuccessPayload{
		Data: entries,
	}, nil
}

func calculatePercentValues(entries *[]*ClientGraphMetricsGraphPayload, lanCard *models.NetworkCard, wanCard *models.NetworkCard) {
	if entries == nil {
		return
	}

	bytesMaxLan := oneMBitBytes
	if lanCard != nil {
		bytesMaxLan = bytesMaxLan * float64(lanCard.MaxSpeed)
	}
	bytesMaxWan := oneMBitBytes
	if wanCard != nil {
		bytesMaxWan = bytesMaxWan * float64(wanCard.MaxSpeed)
	}

	var bytes float64
	var percent float64
	for _, entry := range *entries {
		if entry.NetUsagePercentLan != nil {
			if entry.NetUsagePercentLan.InAvg != nil {
				bytes = *entry.NetUsagePercentLan.InAvg
				percent = calculateBytesPercent(bytes, bytesMaxLan)
				*entry.NetUsagePercentLan.InAvg = percent
			}
			if entry.NetUsagePercentLan.InMin != nil {
				bytes = *entry.NetUsagePercentLan.InMin
				percent = calculateBytesPercent(bytes, bytesMaxLan)
				*entry.NetUsagePercentLan.InMin = percent
			}
			if entry.NetUsagePercentLan.InMax != nil {
				bytes = *entry.NetUsagePercentLan.InMax
				percent = calculateBytesPercent(bytes, bytesMaxLan)
				*entry.NetUsagePercentLan.InMax = percent
			}
			if entry.NetUsagePercentLan.OutAvg != nil {
				bytes = *entry.NetUsagePercentLan.OutAvg
				percent = calculateBytesPercent(bytes, bytesMaxLan)
				*entry.NetUsagePercentLan.OutAvg = percent
			}
			if entry.NetUsagePercentLan.OutMin != nil {
				bytes = *entry.NetUsagePercentLan.OutMin
				percent = calculateBytesPercent(bytes, bytesMaxLan)
				*entry.NetUsagePercentLan.OutMin = percent
			}
			if entry.NetUsagePercentLan.OutMax != nil {
				bytes = *entry.NetUsagePercentLan.OutMax
				percent = calculateBytesPercent(bytes, bytesMaxLan)
				*entry.NetUsagePercentLan.OutMax = percent
			}
		}
		if entry.NetUsagePercentWan != nil {
			if entry.NetUsagePercentWan.InAvg != nil {
				bytes = *entry.NetUsagePercentWan.InAvg
				percent = calculateBytesPercent(bytes, bytesMaxWan)
				*entry.NetUsagePercentWan.InAvg = percent
			}
			if entry.NetUsagePercentWan.InMin != nil {
				bytes = *entry.NetUsagePercentWan.InMin
				percent = calculateBytesPercent(bytes, bytesMaxWan)
				*entry.NetUsagePercentWan.InMin = percent
			}
			if entry.NetUsagePercentWan.InMax != nil {
				bytes = *entry.NetUsagePercentWan.InMax
				percent = calculateBytesPercent(bytes, bytesMaxWan)
				*entry.NetUsagePercentWan.InMax = percent
			}
			if entry.NetUsagePercentWan.OutAvg != nil {
				bytes = *entry.NetUsagePercentWan.OutAvg
				percent = calculateBytesPercent(bytes, bytesMaxWan)
				*entry.NetUsagePercentWan.OutAvg = percent
			}
			if entry.NetUsagePercentWan.OutMin != nil {
				bytes = *entry.NetUsagePercentWan.OutMin
				percent = calculateBytesPercent(bytes, bytesMaxWan)
				*entry.NetUsagePercentWan.OutMin = percent
			}
			if entry.NetUsagePercentWan.OutMax != nil {
				bytes = *entry.NetUsagePercentWan.OutMax
				percent = calculateBytesPercent(bytes, bytesMaxWan)
				*entry.NetUsagePercentWan.OutMax = percent
			}
		}
	}
}

func calculateBytesPercent(bytes float64, bytesMax float64) float64 {
	return bytes / bytesMax * 100
}

func (s *monitoringService) validateAndParseGraphOptions(lo *query.ListOptions) (*time.Duration, error) {
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

	query.SortFiltersByOperator(lo.Filters) //important for next check
	if (lo.Filters[0].Operator == query.FilterOperatorTypeGT && lo.Filters[1].Operator == query.FilterOperatorTypeLT) ||
		(lo.Filters[0].Operator == query.FilterOperatorTypeSince && lo.Filters[1].Operator == query.FilterOperatorTypeUntil) {
		//these are the allowed filter combinations
	} else {
		return nil, errors.APIError{Message: fmt.Sprintf("Illegal filter pair %s %s", lo.Filters[0], lo.Filters[1]), HTTPStatus: http.StatusBadRequest}
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

	return &span, nil
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
