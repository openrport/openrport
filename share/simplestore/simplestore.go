package simplestore

import (
	"context"
	"encoding/json"
	"sort"
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

func (s *SimpleStore[T]) Get(ctx context.Context, key string) (T, bool, error) {
	s.RLock()
	defer s.RUnlock()
	t, found := s.memory[key]
	return t, found, nil
}

func (s *SimpleStore[T]) GetAll(ctx context.Context) ([]T, error) {
	s.RLock()
	keys := make([]string, len(s.memory))
	i := 0
	for k := range s.memory {
		keys[i] = k
		i++
	}
	s.RUnlock()

	sort.Strings(keys)

	s.RLock()
	defer s.RUnlock()

	ts := make([]T, len(s.memory))
	i = 0
	for _, k := range keys {
		ts[i] = s.memory[k]
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

func (s *SimpleStore[T]) Filter(ctx context.Context, sieve func(T) bool) ([]T, error) {

	all, err := s.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	tmp := []T{}

	for _, v := range all {
		if sieve(v) {
			tmp = append(tmp, v)
		}
	}

	return tmp, nil

}

//func EndpointFiltering[T any]() {
//	if options == nil {
//		return nil, fmt.Errorf("options for filtering and pagination is nil")
//	}
//
//	filter, err := filterer.CompileFromQueryListOptions[T](options.Filters)
//	if err != nil {
//		return nil, err
//	}
//
//	tunnels, err := t.kv.Filter(ctx, func(tunnel T) bool {
//		return tunnel.ClientID == clientID && filter.Run(tunnel)
//	})
//	if err != nil {
//		return nil, err
//	}
//
//	tunnels, err = dynops.FastSorter1(t.sortTranslationTable, tunnels, options.Sorts)
//	if err != nil {
//		return nil, err
//	}
//
//	tunnels = dynops.Paginator(tunnels, options.Pagination)
//
//	return simpleops.ToPointerSlice(tunnels), nil
//
//}
