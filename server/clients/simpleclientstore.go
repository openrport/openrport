package clients

import (
	"context"
	"time"

	"github.com/realvnc-labs/rport/share/logger"
)

type KVStore interface {
	GetAll(context.Context) ([]Client, error)
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

func (s SimpleClientStore) Save(ctx context.Context, client *Client) error {
	c := *client
	return s.kvstore.Save(ctx, c.ID, c)
}

func (s SimpleClientStore) DeleteObsolete(ctx context.Context, _ *logger.Logger) error {
	if s.keepDisconnectedClients == nil {
		return nil
	}
	all, err := s.kvstore.GetAll(ctx)
	if err != nil {
		return err
	}

	cutOff := time.Now().Add(-*s.keepDisconnectedClients)

	for _, c := range all { //nolint:govet
		if c.DisconnectedAt.After(cutOff) {
			err := s.kvstore.Delete(ctx, c.ID)
			if err != nil {
				return err
			}
		}
	}

	return nil
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
