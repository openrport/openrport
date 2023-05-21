package simplestore

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

type Transaction interface {
	get(key string) (any, bool, error)
	getAll() ([]any, error)
	save(key string, entry any) error
	delete(key string) error
}

type TransactionalStore interface {
	get(ctx context.Context, key string) (any, bool, error)
	getAll(ctx context.Context) ([]any, error)
	Transaction(ctx context.Context, updater func(ctx context.Context, tx Transaction) error) error
}

type view[T any] struct {
	store     TransactionalStore
	keyPrefix string
}

func WithType[T any](store TransactionalStore) view[T] {
	return view[T]{
		store:     store,
		keyPrefix: genKeyPrefix[T](),
	}
}

type transactionView[T any] struct {
	tx        Transaction
	keyPrefix string
}

func (t transactionView[T]) Get(key string) (T, bool, error) {
	var emptyEntry T

	get, b, err := t.tx.get(t.prefix(key))
	if err != nil {
		return emptyEntry, false, err
	}
	if !b {
		return emptyEntry, false, nil
	}

	return get.(T), true, nil
}

func (t transactionView[T]) GetAll() ([]T, error) {

	all, err := t.tx.getAll()
	if err != nil {
		return nil, err
	}

	ts := make([]T, len(all))
	if err != nil {
		return nil, err
	}

	return ts, nil
}

func (t transactionView[T]) Save(key string, entry T) error {
	return t.tx.save(t.prefix(key), entry)
}

func (t transactionView[T]) Delete(key string) error {
	return t.tx.delete(t.prefix(key))
}

func (t transactionView[T]) prefix(key string) string {
	return t.keyPrefix + key
}

type TransactionOperations[T any] interface {
	Get(key string) (T, bool, error)
	GetAll() ([]T, error)
	Save(key string, entry T) error
	Delete(key string) error
}

func TransactionWithType[T any](tx Transaction) TransactionOperations[T] {
	return transactionView[T]{
		tx:        tx,
		keyPrefix: genKeyPrefix[T](),
	}
}

func genKeyPrefix[T any]() string {
	var t T
	return fmt.Sprintf("%T---", t)
}

func (v view[T]) Get(ctx context.Context, key string) (T, bool, error) {
	var emptyEntry T

	get, b, err := v.store.get(ctx, v.keyPrefix+key)
	if err != nil {
		return emptyEntry, false, err
	}
	if !b {
		return emptyEntry, false, nil
	}

	return get.(T), true, nil
}

func (v view[T]) GetAll(ctx context.Context) ([]T, error) {

	// prefix := genKeyPrefix[T]()

	all, err := v.store.getAll(ctx)
	if err != nil {
		return nil, err
	}

	var ts []T
	if err != nil {
		return nil, err
	}

	for _, e := range all {
		ec, ok := e.(T)
		if ok {
			ts = append(ts, ec)
		}
	}

	return ts, nil

}

type store struct {
	sync.RWMutex
	entries map[string]any
	kvStore KVStore
	key     string
}

func NewTransactionalStore(ctx context.Context, name string, kvStore KVStore) (*store, error) {
	var entries map[string]interface{}

	data, found, err := kvStore.Read(ctx, name)
	if err != nil {
		return nil, err
	}

	if found {
		err = json.Unmarshal(data, &entries)

		if err != nil {
			return nil, err
		}
	}

	return &store{
		kvStore: kvStore,
		entries: entries,
		key:     name,
	}, nil
}

func (s *store) get(ctx context.Context, key string) (any, bool, error) {
	s.RLock()
	defer s.RUnlock()
	a, ok := s.entries[key]
	return a, ok, nil
}

func (s *store) getAll(ctx context.Context) ([]any, error) {
	s.RLock()
	defer s.RUnlock()
	var res []any
	for _, e := range s.entries {
		res = append(res, e)
	}
	return res, nil
}

func (s *store) Transaction(ctx context.Context, updater func(ctx context.Context, tx Transaction) error) error {
	s.Lock()
	defer s.Unlock()
	tx := toInternalTransaction(s.entries)
	err := updater(ctx, tx)
	if err != nil {
		return err
	}

	data, err := json.Marshal(tx.entries)
	if err != nil {
		return err
	}

	err = s.kvStore.Put(ctx, s.key, data)
	if err != nil {
		return err
	}
	s.entries = tx.entries
	return nil
}

type internalTransaction struct {
	entries map[string]any
}

func (i internalTransaction) get(key string) (any, bool, error) {
	a, ok := i.entries[key]
	return a, ok, nil
}

func (i internalTransaction) getAll() ([]any, error) {
	out := make([]any, len(i.entries))
	c := 0
	for _, v := range i.entries {
		out[c] = v
		c++
	}

	return out, nil
}

func (i internalTransaction) save(key string, entry any) error {
	i.entries[key] = entry
	return nil
}

func (i internalTransaction) delete(key string) error {
	delete(i.entries, key)
	return nil
}

func toInternalTransaction(org map[string]any) internalTransaction {
	newMap := make(map[string]any, len(org))
	for k, v := range org {
		newMap[k] = v
	}

	return internalTransaction{
		entries: newMap,
	}
}
