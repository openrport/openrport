package session

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
)

type Cache struct {
	cache   InternalCacheProvider
	storage StorageProvider
}

func NewCache(
	ctx context.Context,
	defaultExpiration,
	cleanupInterval time.Duration,
	storage StorageProvider,
	c InternalCacheProvider,
) (*Cache, error) {
	if c == nil {
		c = cache.New(defaultExpiration, cleanupInterval)
	}
	now := time.Now()
	validSessions, err := storage.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get api sessions from storage: %w", err)
	}

	for _, cur := range validSessions {
		c.Set(formatID(cur.SessionID), cur, cur.ExpiresAt.Sub(now))
	}

	return &Cache{
		cache:   c,
		storage: storage,
	}, nil
}

func (p *Cache) Get(ctx context.Context, sessionID int64) (found bool, sessionInfo APISession, err error) {
	return p.getFromCache(sessionID)
}

func (p *Cache) Save(ctx context.Context, session APISession) (sessionID int64, err error) {
	// always save to storage as we want the token last access time saved
	sessionID, err = p.storage.Save(ctx, session)
	if err != nil {
		return -1, err
	}

	// make sure the session id is included in the cache version.
	session.SessionID = sessionID

	p.cache.Set(formatID(sessionID), session, time.Until(session.ExpiresAt))

	return sessionID, nil
}

func (p *Cache) Delete(ctx context.Context, sessionID int64) error {
	if err := p.storage.Delete(ctx, sessionID); err != nil {
		return err
	}

	p.cache.Delete(formatID(sessionID))

	return nil
}

func (p *Cache) DeleteExpired(ctx context.Context) error {
	// only delete expired token in storage as go-cache will have already expired
	// the internal cache tokens.
	return p.storage.DeleteExpired(ctx)
}

func (p *Cache) Close() error {
	return p.storage.Close()
}

func (p *Cache) getFromCache(sessionID int64) (found bool, sessionInfo APISession, err error) {
	existingObj, found := p.cache.Get(formatID(sessionID))
	if !found {
		return false, APISession{}, nil
	}

	existing, ok := existingObj.(APISession)
	if !ok {
		return false, APISession{}, fmt.Errorf("invalid cache entry: expected APISession, got %T", existingObj)
	}

	return true, existing, nil
}

func (p *Cache) GetAllByUser(ctx context.Context, username string) (sessions []APISession, err error) {
	sessions = make([]APISession, 0)
	// just query the go-cache tokens. they will be more up to date than the storage tokens.
	for _, item := range p.cache.Items() {
		session, ok := item.Object.(APISession)
		if !ok {
			return nil, fmt.Errorf("invalid cache entry: expected *APISession, got %T", item.Object)
		}
		if session.Username == username {
			sessions = append(sessions, session)
		}
	}

	// sort into oldest accessed first
	sort.SliceStable(sessions, func(a, b int) bool {
		return sessions[a].LastAccessAt.Before(sessions[b].LastAccessAt)
	})
	return sessions, nil
}

func (p *Cache) DeleteByID(ctx context.Context, username string, sessionID int64) (err error) {
	err = p.deleteUserSessionsFromCache(username, sessionID)
	if err != nil {
		return err
	}
	err = p.deleteUserSessionFromStorage(ctx, username, sessionID)
	if err != nil {
		return err
	}
	return nil
}

func (p *Cache) deleteUserSessionsFromCache(username string, sessionID int64) (err error) {
	if sessionID != -1 {
		p.cache.Delete(formatID(sessionID))
		return
	}

	// Items() returns a copy of the underlying unexpired cache items and Delete
	// won't error if item not found. this should be thread safe.
	for _, item := range p.cache.Items() {
		session, ok := item.Object.(APISession)
		if !ok {
			return fmt.Errorf("invalid cache entry: expected *APISession, got %T", item.Object)
		}
		if session.Username == username {
			p.cache.Delete(formatID(session.SessionID))
		}
	}

	return nil
}

func (p *Cache) deleteUserSessionFromStorage(ctx context.Context, username string, sessionID int64) (err error) {
	err = p.storage.DeleteByID(ctx, username, sessionID)
	if err != nil {
		return fmt.Errorf("unable to delete session from storage: %w", err)
	}
	return nil
}

func (p *Cache) DeleteAllByUser(ctx context.Context, username string) (err error) {
	err = p.deleteUserSessionsFromCache(username, -1 /* delete all sessions for user */)
	if err != nil {
		return err
	}
	err = p.deleteUserSessionsFromStorage(ctx, username)
	if err != nil {
		return err
	}
	return nil
}

func (p *Cache) deleteUserSessionsFromStorage(ctx context.Context, username string) (err error) {
	err = p.storage.DeleteAllByUser(ctx, username)
	if err != nil {
		return fmt.Errorf("unable to delete sessions from storage: %w", err)
	}
	return nil
}

func formatID(sessionID int64) (id string) {
	return strconv.FormatInt(sessionID, 10)
}
