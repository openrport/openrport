package updates

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/cloudradar-monitoring/rport/share/comm"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/models"
)

type PackageManager interface {
	IsAvailable(context.Context) bool
	GetUpdatesStatus(context.Context, *logger.Logger) (*models.UpdatesStatus, error)
}

type Updates struct {
	// mtx protects both conn and status
	mtx    sync.RWMutex
	conn   ssh.Conn
	status *models.UpdatesStatus

	interval    time.Duration
	refreshChan chan struct{}

	pkgMgr PackageManager
	logger *logger.Logger
}

func New(logger *logger.Logger, interval time.Duration) *Updates {
	return &Updates{
		interval:    interval,
		refreshChan: make(chan struct{}),
		logger:      logger,
	}
}

func (u *Updates) Start(ctx context.Context) {
	if u.interval <= 0 {
		return
	}

	go u.refreshLoop(ctx)
}

func (u *Updates) getPackageManager(ctx context.Context) PackageManager {
	if u.pkgMgr != nil {
		return u.pkgMgr
	}
	for _, pm := range packageManagers {
		if pm.IsAvailable(ctx) {
			u.pkgMgr = pm
			return pm
		}
	}
	return nil
}

func (u *Updates) Refresh() {
	select {
	case u.refreshChan <- struct{}{}:
	default:
	}
}

func (u *Updates) refreshLoop(ctx context.Context) {
	for {
		u.refreshStatus(ctx)

		select {
		case <-ctx.Done():
			return
		case <-time.After(u.interval):
		case <-u.refreshChan:
		}
	}
}

func (u *Updates) refreshStatus(ctx context.Context) {
	var newStatus *models.UpdatesStatus

	pkgMgr := u.getPackageManager(ctx)
	if pkgMgr == nil {
		newStatus = &models.UpdatesStatus{
			Error: "no supported package manager found",
		}
	} else {
		u.logger.Infof("Using %v for updates", reflect.TypeOf(pkgMgr).Elem().Name())

		status, err := pkgMgr.GetUpdatesStatus(ctx, u.logger)
		if err != nil {
			newStatus = &models.UpdatesStatus{
				Error: err.Error(),
			}
		} else {
			newStatus = status
		}
	}
	newStatus.Refreshed = time.Now()

	if newStatus.Error != "" {
		u.logger.Infof("Refreshing OS patch level (pending updates) failed: %v", newStatus.Error)
	} else {
		u.logger.Infof("OS patch level (pending updates) refreshed, %v updates available (%v security)",
			newStatus.UpdatesAvailable, newStatus.SecurityUpdatesAvailable)
	}

	u.mtx.Lock()
	u.status = newStatus
	u.mtx.Unlock()

	go u.sendUpdates()
}

// sendUpdates sends updates in background, it's called both after status is refreshed or conn set
func (u *Updates) sendUpdates() {
	u.mtx.RLock()
	defer u.mtx.RUnlock()

	if u.conn != nil && u.status != nil {
		data, err := json.Marshal(u.status)
		if err != nil {
			u.logger.Errorf("Could not marshal json for updates status: %v", err)
			return
		}

		_, _, err = u.conn.SendRequest(comm.RequestTypeUpdatesStatus, false, data)
		if err != nil {
			u.logger.Errorf("Could not sent updates status: %v", err)
			return
		}
	}
}

func (u *Updates) SetConn(c ssh.Conn) {
	u.mtx.Lock()
	defer u.mtx.Unlock()

	u.conn = c
	go u.sendUpdates()
}
