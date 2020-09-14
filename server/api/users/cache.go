package users

type UserCache struct {
	byUsername map[string]*User
}

func NewUserCache(initUsers []*User) *UserCache {
	m := make(map[string]*User, len(initUsers))
	for _, u := range initUsers {
		m[u.Username] = u
	}
	return &UserCache{byUsername: m}
}

func (r *UserCache) GetByUsername(username string) *User {
	return r.byUsername[username]
}

func (r *UserCache) Count() int {
	return len(r.byUsername)
}
