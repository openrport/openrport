package clients

import (
	"context"
	"github.com/realvnc-labs/rport/share/simpleops"
	"time"

	"github.com/realvnc-labs/rport/share/logger"
)

type KVStore interface {
	GetAll(context.Context) ([]Client, error)
	Get(context.Context, string) (Client, bool, error)
	Save(context.Context, string, Client) error
	Delete(context.Context, string) error
}

type SimpleClientStore struct {
	kvstore                 KVStore
	keepDisconnectedClients *time.Duration
}

func NewSimpleClientStore(kvstore KVStore, keepDisconnectedClients *time.Duration) *SimpleClientStore {
	return &SimpleClientStore{kvstore: kvstore, keepDisconnectedClients: keepDisconnectedClients}
}

func (s SimpleClientStore) Get(ctx context.Context, id string, l *logger.Logger) (*Client, error) {
	get, found, err := s.kvstore.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	get.Logger = l
	return &get, nil
}

func (s SimpleClientStore) GetAll(
	ctx context.Context,
	l *logger.Logger,
) ([]*Client, error) {
	all, err := s.kvstore.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	allPointers := make([]*Client, len(all))
	for i, c := range all { //nolint:govet
		allPointers[i] = &c //nolint:gosec
		allPointers[i].Logger = l
	}

	return allPointers, nil
}

func (s SimpleClientStore) GetNonObsoleteClients(
	ctx context.Context,
	l *logger.Logger,
) ([]*Client, error) {
	all, err := s.kvstore.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	all = simpleops.FilterSlice(all, func(client Client) bool {
		return client.Obsolete(s.keepDisconnectedClients)
	})

	allPointers := make([]*Client, len(all))
	for i, c := range all { //nolint:govet
		allPointers[i] = &c //nolint:gosec
		allPointers[i].Logger = l
	}

	return allPointers, nil
}

func (s SimpleClientStore) Save(ctx context.Context, client *Client) error {
	c := *client
	return s.kvstore.Save(ctx, c.ID, c)
}

func (s SimpleClientStore) Delete(
	ctx context.Context,
	id string,
	_ *logger.Logger,
) error {
	return s.kvstore.Delete(ctx, id)
}

func (s SimpleClientStore) Close() error {
	return nil
}
