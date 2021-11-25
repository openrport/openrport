package session

import (
	"context"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
)

const (
	saveInterval = time.Minute
)

type Cache struct {
	cache   *cache.Cache
	storage Provider
}

func NewCache(
	ctx context.Context,
	storage Provider,
	defaultExpiration,
	cleanupInterval time.Duration,
) (*Cache, error) {
	c := cache.New(defaultExpiration, cleanupInterval)

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
	// to avoid too many writes to Sqlite - save only new or after a given interval
	stored, err := p.storage.Get(ctx, session.Token)
	if err != nil {
		return err
	}

	if stored == nil || session.ExpiresAt.After(stored.ExpiresAt.Add(saveInterval)) {
		if err := p.storage.Save(ctx, session); err != nil {
			return err
		}
	}

	p.cache.SetDefault(session.Token, session)

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
