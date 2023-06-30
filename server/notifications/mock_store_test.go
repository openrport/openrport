package notifications_test

import (
	"context"
	"sync"

	"github.com/realvnc-labs/rport/server/notifications"
)

type MockStore struct {
	notifications map[string]notifications.NotificationDetails
	ch            map[notifications.Target]chan notifications.NotificationDetails
	sync.RWMutex
}

func (m *MockStore) LogRunning(ctx context.Context, nid string) error {
	m.Lock()
	defer m.Unlock()

	n := m.notifications[nid]
	n.State = notifications.ProcessingStateRunning
	m.notifications[nid] = n
	return nil
}

func (m *MockStore) LogDone(ctx context.Context, nid string) error {
	m.Lock()
	defer m.Unlock()

	n := m.notifications[nid]
	n.State = notifications.ProcessingStateDone
	m.notifications[nid] = n
	return nil
}

func (m *MockStore) LogError(ctx context.Context, nid string, error string) error {
	m.Lock()
	defer m.Unlock()

	n := m.notifications[nid]
	n.State = notifications.ProcessingStateError
	n.Out = error
	m.notifications[nid] = n
	return nil
}

func NewMockStore() *MockStore {
	return &MockStore{
		notifications: make(map[string]notifications.NotificationDetails),
		ch:            map[notifications.Target]chan notifications.NotificationDetails{},
	}
}

func (m *MockStore) NotificationStream(target notifications.Target) chan notifications.NotificationDetails {
	m.Lock()
	defer m.Unlock()

	ch, found := m.ch[target]
	if !found {
		ch = make(chan notifications.NotificationDetails, len(m.notifications))
		for _, n := range m.notifications {
			if n.Target == target {
				ch <- n
			}
		}
		m.ch[target] = ch
	}

	return ch
}

func (m *MockStore) Create(ctx context.Context, notification notifications.NotificationDetails) error {
	m.Lock()
	defer m.Unlock()

	m.notifications[notification.ID.ID()] = notification
	ch, found := m.ch[notification.Target]
	if found {
		ch <- notification
	}

	return nil
}

func (m *MockStore) List(ctx context.Context) ([]notifications.NotificationSummary, error) {
	m.Lock()
	defer m.Unlock()

	tmp := make([]notifications.NotificationSummary, len(m.notifications))
	i := 0
	for _, n := range m.notifications {
		tmp[i] = notifications.NotificationSummary{
			State: n.State,
		}
		i++
	}
	return tmp, nil
}

func (m *MockStore) Details(ctx context.Context, notificationID notifications.NotificationID) (notifications.NotificationDetails, bool, error) {
	m.Lock()
	defer m.Unlock()

	details, found := m.notifications[notificationID.ID()]
	return details, found, nil
}

func (m *MockStore) Close() error {
	m.Lock()
	defer m.Unlock()

	for _, ch := range m.ch {
		close(ch)
	}
	return nil
}
