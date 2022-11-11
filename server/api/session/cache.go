package session

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/patrickmn/go-cache"
)

type Cache struct {
	cache   CacheProvider
	storage StorageProvider
}

func NewCache(
	ctx context.Context,
	defaultExpiration,
	cleanupInterval time.Duration,
	storage StorageProvider,
	c CacheProvider,
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
		c.Set(cur.Token, cur, cur.ExpiresAt.Sub(now))
	}

	return &Cache{
		cache:   c,
		storage: storage,
	}, nil
}

func (p *Cache) Get(ctx context.Context, token string) (*APISession, error) {
	return p.getFromCache(token)
}

func (p *Cache) Save(ctx context.Context, session *APISession) error {
	// always save to storage as we want the token last access time saved
	sessionID, err := p.storage.Save(ctx, session)
	if err != nil {
		return err
	}

	// make sure the session id is included in the cache version. this also updates
	// the session ID in the supplied session.
	session.SessionID = sessionID

	p.cache.Set(session.Token, session, time.Until(session.ExpiresAt))

	return nil
}

func (p *Cache) Delete(ctx context.Context, token string) error {
	if err := p.storage.Delete(ctx, token); err != nil {
		return err
	}

	p.cache.Delete(token)

	return nil
}

func (p *Cache) DeleteExpired(ctx context.Context) error {
	return p.storage.DeleteExpired(ctx)
}

func (p *Cache) Close() error {
	return p.storage.Close()
}

func (p *Cache) getFromCache(token string) (*APISession, error) {
	existingObj, _ := p.cache.Get(token)
	if existingObj == nil {
		return nil, nil
	}

	existing, ok := existingObj.(*APISession)
	if !ok {
		return nil, fmt.Errorf("invalid cache entry: expected *APISession, got %T", existingObj)
	}

	return existing, nil
}

func (p *Cache) GetAllByUser(ctx context.Context, username string) (sessions []*APISession, err error) {
	count := p.cache.ItemCount()
	sessions = make([]*APISession, 0, count)
	for _, item := range p.cache.Items() {
		session, ok := item.Object.(*APISession)
		if !ok {
			return nil, fmt.Errorf("invalid cache entry: expected *APISession, got %T", item.Object)
		}
		if session.Username == username {
			sessions = append(sessions, session)
		}
	}

	// TODO: check with TH whether oldest should be first
	// sort into oldest accessed first
	sort.SliceStable(sessions, func(a, b int) bool {
		return sessions[a].LastAccessAt.Before(sessions[b].LastAccessAt)
	})
	return sessions, nil
}

func (p *Cache) DeleteByUser(ctx context.Context, username string, sessionID int64) (err error) {
	err = p.deleteUserSessionFromCache(username, sessionID)
	if err != nil {
		return err
	}
	err = p.deleteUserSessionFromStorage(ctx, username, sessionID)
	if err != nil {
		return err
	}
	return nil
}

func (p *Cache) deleteUserSessionFromCache(username string, sessionID int64) (err error) {
	// Items() returns a copy of the underlying unexpired cache items and Delete
	// won't error if item not found. this should be thread safe.
	for _, item := range p.cache.Items() {
		session, ok := item.Object.(*APISession)
		if !ok {
			return fmt.Errorf("invalid cache entry: expected *APISession, got %T", item.Object)
		}
		if session.Username == username && session.SessionID == sessionID {
			p.cache.Delete(session.Token)
			break
		}
	}

	return nil
}

func (p *Cache) deleteUserSessionFromStorage(ctx context.Context, username string, sessionID int64) (err error) {
	err = p.storage.DeleteByUser(ctx, username, sessionID)
	if err != nil {
		return fmt.Errorf("unable to delete session from cache: %w", err)
	}
	return nil
}

func (p *Cache) DeleteAllByUser(ctx context.Context, username string) (err error) {
	err = p.deleteUserSessionsFromCache(username)
	if err != nil {
		return err
	}
	err = p.deleteUserSessionsFromStorage(ctx, username)
	if err != nil {
		return err
	}
	return nil
}

func (p *Cache) deleteUserSessionsFromCache(username string) (err error) {
	for _, item := range p.cache.Items() {
		session, ok := item.Object.(*APISession)
		if !ok {
			return fmt.Errorf("invalid cache entry: expected *APISession, got %T", item.Object)
		}
		if session.Username == username {
			p.cache.Delete(session.Token)
		}
	}
	return nil
}

func (p *Cache) deleteUserSessionsFromStorage(ctx context.Context, username string) (err error) {
	err = p.storage.DeleteAllByUser(ctx, username)
	if err != nil {
		return fmt.Errorf("unable to delete sessions from cache: %w", err)
	}
	return nil
}
