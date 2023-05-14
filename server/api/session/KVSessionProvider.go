package session

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

type KVStore interface {
	Get(ctx context.Context, key string) (APISession, bool, error)
	GetAll(ctx context.Context) ([]APISession, error)
	Delete(ctx context.Context, sessionID string) error
	Save(ctx context.Context, sessionID string, session APISession) error
	Filter(ctx context.Context, sieve func(session APISession) bool) ([]APISession, error)
}

type KVSessionProvider struct {
	kv KVStore
}

func (K KVSessionProvider) Get(ctx context.Context, sessionID int64) (bool, APISession, error) {
	session, found, err := K.kv.Get(ctx, GenKeyForAPISession(sessionID))
	return found, session, err
}

func GenKeyForAPISession(sessionID int64) string {
	return fmt.Sprintf("%v", sessionID)
}

func (K KVSessionProvider) GetAll(ctx context.Context) ([]APISession, error) {
	return K.kv.GetAll(ctx)
}

func (K KVSessionProvider) Save(ctx context.Context, session APISession) (int64, error) {
	if session.SessionID == 0 {
		session.SessionID = time.Now().UnixNano()*1000 + rand.Int63n(999)
	}
	return session.SessionID, K.kv.Save(ctx, GenKeyForAPISession(session.SessionID), session)
}

func (K KVSessionProvider) Delete(ctx context.Context, sessionID int64) error {
	return K.kv.Delete(ctx, GenKeyForAPISession(sessionID))
}

func (K KVSessionProvider) DeleteExpired(ctx context.Context) error {
	now := time.Now()

	return K.DeleteMatching(ctx, func(session APISession) bool {
		return session.ExpiresAt.Before(now)
	})
}

func (K KVSessionProvider) DeleteMatching(ctx context.Context, sieve func(session APISession) bool) error {

	expired, err := K.kv.Filter(ctx, sieve)

	if err != nil {
		return err
	}

	for _, exp := range expired {
		err = K.Delete(ctx, exp.SessionID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (K KVSessionProvider) Close() error {
	// nop
	return nil
}

func (K KVSessionProvider) DeleteAllByUser(ctx context.Context, username string) (err error) {
	return K.DeleteMatching(ctx, func(session APISession) bool {
		return session.Username == username
	})
}

func (K KVSessionProvider) DeleteByID(ctx context.Context, username string, sessionID int64) (err error) {
	return K.DeleteMatching(ctx, func(session APISession) bool {
		return session.Username == username && session.SessionID == sessionID
	})
}

func NewKVSessionProvider(kv KVStore) StorageProvider {
	return &KVSessionProvider{kv: kv}
}
