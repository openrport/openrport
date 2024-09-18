package inventory

import (
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"github.com/openrport/openrport/share/comm"
	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
	"golang.org/x/crypto/ssh"
)

type SoftwareInventoryManager interface {
	IsAvaliable(context.Context) bool
	GetInventory(context.Context, *logger.Logger) ([]models.SoftwareInventory, error)
}

type ContainerInventoryManager interface {
	IsAvaliable(context.Context) bool
	GetInventory(context.Context, *logger.Logger) ([]models.ContainerInventory, error)
}

type Inventory struct {
	mtx  sync.RWMutex
	conn ssh.Conn
	inv  *models.Inventory

	interval    time.Duration
	refreshChan chan struct{}

	sofInvMgr SoftwareInventoryManager
	conInvMgr ContainerInventoryManager
	logger    *logger.Logger
}

func New(logger *logger.Logger, interval time.Duration) *Inventory {
	return &Inventory{
		interval:    interval,
		refreshChan: make(chan struct{}),
		logger:      logger,
	}
}

func (i *Inventory) Start(ctx context.Context) {
	if i.interval <= 0 {
		return
	}

	go i.refreshLoop(ctx)
}

func (i *Inventory) getSoftwareInventoryManager(ctx context.Context) SoftwareInventoryManager {
	if i.sofInvMgr != nil {
		return i.sofInvMgr
	}

	for _, sim := range softwareInventoryManagers {
		if sim.IsAvaliable(ctx) {
			i.sofInvMgr = sim
			return sim
		}
	}

	return nil
}

func (i *Inventory) getContainerInventoryManager(ctx context.Context) ContainerInventoryManager {
	if i.conInvMgr != nil {
		return i.conInvMgr
	}

	for _, cim := range containerInventoryManagers {
		if cim.IsAvaliable(ctx) {
			i.conInvMgr = cim
			return cim
		}
	}

	return nil
}

func (i *Inventory) refreshLoop(ctx context.Context) {
	for {
		i.refreshInventory(ctx)

		select {
		case <-ctx.Done():
			i.logger.Debugf("Inventory refreshLoop finished")
			return
		case <-time.After(i.interval):
		case <-i.refreshChan:
		}
	}
}

func (i *Inventory) refreshInventory(ctx context.Context) {
	var newInventory *models.Inventory

	sofInvMgr := i.getSoftwareInventoryManager(ctx)
	if sofInvMgr == nil {
		newInventory = &models.Inventory{
			SoftwareInventory: make([]models.SoftwareInventory, 0),
		}
	} else {
		i.logger.Infof("Using %v for software inventory", reflect.TypeOf(sofInvMgr).Elem().Name())

		software_inventory, err := sofInvMgr.GetInventory(ctx, i.logger)
		if err != nil {
			i.logger.Infof("Could not get software inventory from system!")
			newInventory = &models.Inventory{
				SoftwareInventory: make([]models.SoftwareInventory, 0),
			}
		} else {
			newInventory = &models.Inventory{
				SoftwareInventory: software_inventory,
			}
		}
	}

	conInvMgr := i.getContainerInventoryManager(ctx)
	if conInvMgr == nil {
		newInventory = &models.Inventory{
			ContainerInventory: []models.ContainerInventory{},
		}
	} else {
		i.logger.Infof("Using %v for container inventory", reflect.TypeOf(conInvMgr).Elem().Name())

		container_inventory, err := conInvMgr.GetInventory(ctx, i.logger)
		if err != nil {
			i.logger.Infof("Could not get container inventory from system!")
			newInventory = &models.Inventory{
				ContainerInventory: []models.ContainerInventory{},
			}
		} else {
			newInventory = &models.Inventory{
				ContainerInventory: container_inventory,
			}
		}

	}

	newInventory.Refreshed = time.Now()

	i.logger.Infof("Current software and container inventory refreshed!")

	i.mtx.Lock()
	i.inv = newInventory
	i.mtx.Unlock()

	go i.sendInventory()
}

func (i *Inventory) sendInventory() {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	if i.conn != nil && i.inv != nil {
		data, err := json.Marshal(i.inv)
		if err != nil {
			i.logger.Errorf("Could not marshal json for inventory: %v", err)
			return
		}

		_, _, err = i.conn.SendRequest(comm.RequestTypeInventory, false, data)
		if err != nil {
			i.logger.Errorf("Could not sent inventory: %v", err)
			return
		}
	}
}

func (i *Inventory) SetConn(c ssh.Conn) {
	i.mtx.RLock()
	defer i.mtx.RUnlock()

	i.conn = c
}

func (i *Inventory) Stop() {
	i.conn = nil
}
