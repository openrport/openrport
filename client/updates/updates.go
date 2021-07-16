package updates

import (
	"context"
	"sync"
	"time"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

var packageManagers = []PackageManager{
	NewAptPackageManager(),
}

type PackageManager interface {
	IsAvailable(context.Context) bool
	GetUpdatesStatus(context.Context, *chshare.Logger) (*Status, error)
}

type Updates struct {
	status *Status
	mtx    sync.RWMutex

	interval time.Duration

	pkgMgr PackageManager
	logger *chshare.Logger
}

type Status struct {
	Refreshed                time.Time
	UpdatesAvailable         int
	SecurityUpdatesAvailable int
	UpdateSummaries          []Summary
	RebootPending            bool
	Error                    string
	Hint                     string
}

type Summary struct {
	Title            string
	Description      string
	RebootRequired   bool
	IsSecurityUpdate bool
}

func New(ctx context.Context, logger *chshare.Logger, interval time.Duration) *Updates {
	u := &Updates{
		interval: interval,
		logger:   logger,
	}
	if interval < 0 {
		return u
	}

	for _, pm := range packageManagers {
		if pm.IsAvailable(ctx) {
			u.pkgMgr = pm
		}
	}

	if u.pkgMgr == nil {
		u.status = &Status{
			Error: "no supported package manager found",
		}
		u.logger.Errorf("Update status not available: %v", u.status.Error)
	} else {
		go u.refreshLoop(ctx)
	}

	return u
}

func (u *Updates) refreshLoop(ctx context.Context) {
	for {
		u.refreshStatus(ctx)

		status := u.GetStatus()
		if status.Error != "" {
			u.logger.Errorf("Update status refresh failed: %v", status.Error)
		} else {
			u.logger.Infof("Update status refreshed, %v updates available (%v security)",
				status.UpdatesAvailable, status.SecurityUpdatesAvailable)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(u.interval):
		}
	}
}

func (u *Updates) refreshStatus(ctx context.Context) {
	var newStatus *Status

	status, err := u.pkgMgr.GetUpdatesStatus(ctx, u.logger)
	if err != nil {
		newStatus = &Status{
			Error: err.Error(),
		}
	} else {
		newStatus = status
	}
	newStatus.Refreshed = time.Now()

	u.mtx.Lock()
	defer u.mtx.Unlock()

	u.status = newStatus
}

func (u *Updates) GetStatus() *Status {
	u.mtx.RLock()
	defer u.mtx.RUnlock()

	return u.status
}
