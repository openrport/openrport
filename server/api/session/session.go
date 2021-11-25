package session

import (
	"context"
	"time"
)

type APISession struct {
	Token     string    `db:"token"`
	ExpiresAt time.Time `db:"expires_at"`
}

type Provider interface {
	Get(ctx context.Context, token string) (*APISession, error)
	GetAll(ctx context.Context) ([]*APISession, error)
	Save(ctx context.Context, session *APISession) error
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
	Close() error
}
