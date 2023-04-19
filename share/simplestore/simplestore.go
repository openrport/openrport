package simplestore

import (
	"context"
	"encoding/json"
	"sync"
)

type KVStore[T any] interface {
	Put(ctx context.Context, key string, data []byte) error
	ReadAll(ctx context.Context, reader func(key string, data []byte) error) error
	Delete(ctx context.Context, key string) error
}

type SimpleStore[T any] struct {
	sync.RWMutex
	memory  map[string]T
	kvstore KVStore[T]
}

func NewSimpleStore[T any](ctx context.Context, store KVStore[T]) (*SimpleStore[T], error) {

	memory := map[string]T{}

	err := store.ReadAll(ctx, func(key string, data []byte) error {
		var obj T
		err := json.Unmarshal(data, &obj)
		if err != nil {
			return err
		}
		memory[key] = obj

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &SimpleStore[T]{
		memory:  memory,
		kvstore: store,
	}, nil
}

func (s *SimpleStore[T]) GetAll(ctx context.Context) ([]T, error) {
	s.RLock()
	defer s.RUnlock()
	ts := make([]T, len(s.memory))
	i := 0
	for _, o := range s.memory {
		ts[i] = o
		i++
	}

	return ts, nil
}

func (s *SimpleStore[T]) Save(ctx context.Context, key string, obj T) error {

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	s.Lock()
	defer s.Unlock()

	if err := s.kvstore.Put(ctx, key, data); err != nil {
		return err
	}
	s.memory[key] = obj

	return nil
}

func (s *SimpleStore[T]) Delete(ctx context.Context, key string) error {
	s.Lock()
	defer s.Unlock()
	err := s.kvstore.Delete(ctx, key)
	if err != nil {
		return err
	}
	delete(s.memory, key)

	return nil
}
