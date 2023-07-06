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

func (m *MockStore) SetDone(ctx context.Context, details notifications.NotificationDetails) error {
	return m.logDone(ctx, details.ID.ID())
}

func (m *MockStore) SetError(ctx context.Context, details notifications.NotificationDetails, out string) error {
	return m.logError(ctx, details.ID.ID(), out)
}

func (m *MockStore) logDone(_ context.Context, nid string) error {
	m.Lock()
	defer m.Unlock()

	n := m.notifications[nid]
	n.State = notifications.ProcessingStateDone
	m.notifications[nid] = n
	return nil
}

func (m *MockStore) logError(_ context.Context, nid string, error string) error {
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
	notifications2 := m.notifications

	if !found {
		ch = make(chan notifications.NotificationDetails, len(notifications2))
		for _, n := range notifications2 {
			if n.Target == target {
				ch <- n
			}
		}

		m.ch[target] = ch
	}

	return ch
}

func (m *MockStore) Create(_ context.Context, notification notifications.NotificationDetails) error {
	m.Lock()
	m.notifications[notification.ID.ID()] = notification
	ch, found := m.ch[notification.Target]
	m.Unlock()

	if found {
		ch <- notification
	}

	return nil
}

func (m *MockStore) List(_ context.Context) ([]notifications.NotificationSummary, error) {
	m.RLock()
	defer m.RUnlock()

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

func (m *MockStore) Details(_ context.Context, notificationID notifications.NotificationID) (notifications.NotificationDetails, bool, error) {
	m.RLock()
	defer m.RUnlock()

	details, found := m.notifications[notificationID.ID()]
	return details, found, nil
}

func (m *MockStore) Close() error {
	m.RLock()
	defer m.RUnlock()

	for _, ch := range m.ch {
		close(ch)
	}
	return nil
}
