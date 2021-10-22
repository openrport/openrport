package auditlog

import (
	"time"

	"github.com/cloudradar-monitoring/rport/server/clients"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type ClientGetter interface {
	GetByID(id string) (*clients.Client, error)
}

type Provider interface {
	Save(e *Entry) error
}

type AuditLog struct {
	logger       *chshare.Logger
	clientGetter ClientGetter
	provider     Provider
}

func New(l *chshare.Logger, cg ClientGetter) *AuditLog {
	return &AuditLog{
		logger:       l,
		clientGetter: cg,
	}
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

func (a *AuditLog) savePreparedEntry(e *Entry) error {
	if a.provider == nil {
		return nil
	}

	return a.provider.Save(e)
}
