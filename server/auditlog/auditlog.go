package auditlog

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
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
}

type AuditLog struct {
	logger       *chshare.Logger
	clientGetter ClientGetter
	provider     Provider
	config       Config
}

func New(l *chshare.Logger, cg ClientGetter, dataDir string, cfg Config) (*AuditLog, error) {
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
			e.RemoteIP = ip.Mask(net.CIDRMask(24, 32)).String()
		}
	}

	return a.provider.Save(e)
}

func (a *AuditLog) List(r *http.Request) ([]*Entry, error) {
	options := query.GetListOptions(r)

	err := query.ValidateOptions(options, supportedSorts, supportedFilters, nil)
	if err != nil {
		return nil, err
	}

	return a.provider.List(r.Context(), options)
}
