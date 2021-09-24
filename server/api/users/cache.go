package users

import "sync"

// UserCache is in memory user cache with thread-safe loading
type UserCache struct {
	byUsername map[string]*User
	mu         sync.RWMutex
}

func NewUserCache(initUsers []*User) *UserCache {
	r := &UserCache{}
	r.Load(initUsers)
	return r
}

// Load replaces users in cache with given users
func (r *UserCache) Load(users []*User) {
	m := make(map[string]*User, len(users))
	for _, u := range users {
		m[u.Username] = u
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byUsername = m
}

// GetByUsername returns user with the given username or nil
func (r *UserCache) GetByUsername(username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.byUsername[username], nil
}

func (r *UserCache) GetAll() ([]*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	users := make([]*User, 0, len(r.byUsername))
	for _, u := range r.byUsername {
		users = append(users, u)
	}
	return users, nil
}
