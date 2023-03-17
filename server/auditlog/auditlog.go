package auditlog

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/cloudradar-monitoring/rport/server/api/users"
	"github.com/cloudradar-monitoring/rport/server/auditlog/config"

	"github.com/cloudradar-monitoring/rport/db/sqlite"

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
	config       config.Config
}

type NotAllowedError struct {
	Msg string
}

func (e *NotAllowedError) Error() string {
	return e.Msg
}

func New(l *logger.Logger, cg ClientGetter, dataDir string, cfg config.Config, dataSourceOptions sqlite.DataSourceOptions) (*AuditLog, error) {
	a := &AuditLog{
		logger:       l,
		clientGetter: cg,
		config:       cfg,
	}

	if cfg.Enable {
		rotation, err := newRotationProvider(
			l,
			cfg.RotationPeriod(),
			dataDir,
			dataSourceOptions,
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

func (a *AuditLog) List(r *http.Request, user *users.User) (*api.SuccessPayload, error) {
	options := query.GetListOptions(r)
	if !user.IsAdmin() {
		// Deny none-admins looking for foreign audit logs
		for _, v := range options.Filters {
			for _, col := range v.Column {
				if col == "username" {
					return nil, &NotAllowedError{"only members of group Administrators can filter by usernames"}
				}
			}
		}
		// Add a forced filter so none-admins cannot inspect what others have done
		options.Filters = append(options.Filters, query.FilterOption{
			Column: []string{"username"},
			Values: []string{user.Username},
		})
	}
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
