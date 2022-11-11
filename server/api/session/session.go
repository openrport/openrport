package session

import (
	"context"
	"time"

	"github.com/patrickmn/go-cache"
)

type APISession struct {
	SessionID    int64     `db:"session_id"`
	Token        string    `db:"token"`
	ExpiresAt    time.Time `db:"expires_at"`
	LastAccessAt time.Time `db:"last_access_at"`
	Username     string    `db:"username"`
	UserAgent    string    `db:"user_agent"`
	IPAddress    string    `db:"ip_address"`
}

// current implementation provided by go-cache
type InternalCacheProvider interface {
	Set(k string, x interface{}, d time.Duration)
	Get(k string) (interface{}, bool)
	Delete(k string)
	ItemCount() int
	// using `cache.Item` creates a interface dependency on go-cache but currently
	// not worth de-coupling. if alternative cache implementations are required then
	// deal with this then.
	Items() map[string]cache.Item
}

type StorageProvider interface {
	Get(ctx context.Context, token string) (*APISession, error)
	GetAll(ctx context.Context) ([]*APISession, error)
	Save(ctx context.Context, session *APISession) (sessionID int64, err error)
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
	Close() error

	DeleteAllByUser(ctx context.Context, username string) (err error)
	DeleteByUser(ctx context.Context, username string, sessionID int64) (err error)
}
