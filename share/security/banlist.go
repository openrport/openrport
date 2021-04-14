package security

import (
	"sync"
	"time"
)

type BanList struct {
	banDuration time.Duration
	mu          sync.RWMutex
	visitors    map[string]time.Time
}

func NewBanList(banDuration time.Duration) *BanList {
	return &BanList{
		banDuration: banDuration,
		visitors:    make(map[string]time.Time),
	}
}

func (l *BanList) Add(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.visitors[key] = time.Now().Add(l.banDuration)
}

func (l *BanList) IsBanned(key string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	banExpiry, found := l.visitors[key]
	return found && banExpiry.After(time.Now())
}
