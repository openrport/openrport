package notifications_test

import (
	"context"

	"github.com/realvnc-labs/rport/server/notifications"
	"github.com/realvnc-labs/rport/server/notifications/repository/inmemory"
)

type MockStore struct {
	notifications map[string]notifications.NotificationDetails
	ch            map[notifications.Target]chan notifications.NotificationDetails
}

func NewMockStore() *MockStore {
	return &MockStore{
		notifications: make(map[string]notifications.NotificationDetails),
		ch:            map[notifications.Target]chan notifications.NotificationDetails{},
	}
}

func (m *MockStore) UpdatesFor(target notifications.Target) chan notifications.NotificationDetails {
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

func (m *MockStore) Save(ctx context.Context, notification notifications.NotificationDetails) error {
	m.notifications[notification.ID.String()] = notification
	if notification.State == notifications.ProcessingStateQueued {
		ch, found := m.ch[notification.Target]
		if found {
			ch <- notification
		}
	}
	return nil
}

func (m *MockStore) List(ctx context.Context) ([]inmemory.NotificationSummary, error) {
	tmp := make([]inmemory.NotificationSummary, len(m.notifications))
	i := 0
	for _, n := range m.notifications {
		tmp[i] = inmemory.NotificationSummary{
			State: n.State,
		}
		i++
	}
	return tmp, nil
}

func (m *MockStore) Details(ctx context.Context, notificationID inmemory.NotificationID) (notifications.NotificationDetails, bool, error) {
	details, found := m.notifications[notificationID.String()]
	return details, found, nil
}

func (m *MockStore) Close() error {
	for _, ch := range m.ch {
		close(ch)
	}
	return nil
}
