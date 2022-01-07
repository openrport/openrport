package auditlog

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/cloudradar-monitoring/rport/server/api"
	"github.com/cloudradar-monitoring/rport/server/clients"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/query"
)

var (
	supportedFilters = map[string]bool{
		"timestamp[gt]":    true,
		"timestamp[lt]":    true,
		"timestamp[since]": true,
		"timestamp[until]": true,
		"username":         true,
		"remote_ip":        true,
		"application":      true,
		"action":           true,
		"affected_id":      true,
		"client_id":        true,
		"client_hostname":  true,
	}
	supportedSorts = map[string]bool{
		"timestamp":       true,
		"username":        true,
		"remote_ip":       true,
		"application":     true,
		"action":          true,
		"affected_id":     true,
		"client_id":       true,
		"client_hostname": true,
	}
)

type ClientGetter interface {
	GetByID(id string) (*clients.Client, error)
}

type Provider interface {
	io.Closer
	Save(e *Entry) error
	List(context.Context, *query.ListOptions) ([]*Entry, error)
	Count(context.Context, *query.ListOptions) (int, error)
}

type AuditLog struct {
	logger       *logger.Logger
	clientGetter ClientGetter
	provider     Provider
	config       Config
}

func New(l *logger.Logger, cg ClientGetter, dataDir string, cfg Config) (*AuditLog, error) {
	a := &AuditLog{
		logger:       l,
		clientGetter: cg,
		config:       cfg,
	}

	if cfg.Enable {
		rotation, err := newRotationProvider(
			l,
			cfg.rotationPeriod(),
			dataDir,
		)
		if err != nil {
			return nil, err
		}

		a.provider = rotation
	}

	return a, nil
}

func (a *AuditLog) Entry(application, action string) *Entry {
	// return nil if auditlog is not initialized, Entry handles nils so we don't panic unnecessarily
	if a == nil || !a.config.Enable {
		return nil
	}

	e := &Entry{
		Timestamp:   time.Now(),
		Application: application,
		Action:      action,

		al: a,
	}

	return e
}

func (a *AuditLog) Close() error {
	if a == nil || a.provider == nil {
		return nil
	}

	return a.provider.Close()
}

func (a *AuditLog) savePreparedEntry(e *Entry) error {
	if a.provider == nil {
		return nil
	}

	if a.config.UseIPObfuscation && e.RemoteIP != "" {
		ip := net.ParseIP(e.RemoteIP)
		if ip.To4() != nil {
			e.RemoteIP = strings.TrimSuffix(ip.Mask(net.CIDRMask(24, 32)).String(), "0") + "x"
		}
	}

	return a.provider.Save(e)
}

func (a *AuditLog) List(r *http.Request) (*api.SuccessPayload, error) {
	options := query.GetListOptions(r)

	err := query.ValidateListOptions(options, supportedSorts, supportedFilters, nil, &query.PaginationConfig{
		DefaultLimit: 10,
		MaxLimit:     100,
	})
	if err != nil {
		return nil, err
	}

	entries, err := a.provider.List(r.Context(), options)
	if err != nil {
		return nil, err
	}

	count, err := a.provider.Count(r.Context(), options)
	if err != nil {
		return nil, err
	}

	return &api.SuccessPayload{
		Data: entries,
		Meta: api.NewMeta(count),
	}, nil
}
