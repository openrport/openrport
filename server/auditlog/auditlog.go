package auditlog

import (
	"io"
	"net"
	"time"

	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type ClientGetter interface {
	GetByID(id string) (*clients.Client, error)
}

type Provider interface {
	io.Closer
	Save(e *Entry) error
}

type AuditLog struct {
	logger       *chshare.Logger
	clientGetter ClientGetter
	provider     Provider
	config       Config
}

func New(l *chshare.Logger, cg ClientGetter, dataDir string, cfg Config) (*AuditLog, error) {
	if !cfg.Enable {
		return nil, nil
	}

	sqlite, err := newSQLiteProvider(dataDir)
	if err != nil {
		return nil, err
	}
	return &AuditLog{
		logger:       l,
		clientGetter: cg,
		provider:     sqlite,
		config:       cfg,
	}, nil
}

func (a *AuditLog) Entry(application, action string) *Entry {
	// return nil if auditlog is not initialized, Entry handles nils so we don't panic unnecessarily
	if a == nil {
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
