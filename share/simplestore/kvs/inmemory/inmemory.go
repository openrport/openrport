package inmemory

import (
	"context"
	"sync"
)

type InMemory struct {
	sync.RWMutex
	datas map[string][]byte
}

func NewInMemory() *InMemory {
	return &InMemory{datas: make(map[string][]byte)}
}

func (m *InMemory) Read(ctx context.Context, key string) ([]byte, bool, error) {
	m.RLock()
	defer m.RUnlock()
	bytes, found := m.datas[key]
	return bytes, found, nil
}

func (m *InMemory) ReadAll(ctx context.Context, reader func(key string, data []byte) error) error {
	m.RLock()
	defer m.RUnlock()

	for k, o := range m.datas {
		err := reader(k, o)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *InMemory) Put(_ context.Context, key string, data []byte) error {
	m.Lock()
	m.datas[key] = data
	m.Unlock()
	return nil
}

func (m *InMemory) Delete(ctx context.Context, key string) error {
	m.Lock()
	delete(m.datas, key)
	m.Unlock()
	return nil
}
