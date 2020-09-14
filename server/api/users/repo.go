package users

// UserRepository represents an in-memory repository of API users.
type UserRepository struct {
	users *UserCache
}

// NewUserRepository returns a new in-memory repository of API users.
func NewUserRepository(initUsers []*User) *UserRepository {
	return &UserRepository{
		users: NewUserCache(initUsers),
	}
}

func (r *UserRepository) GetByUsername(username string) (*User, error) {
	return r.users.GetByUsername(username), nil
}

func (r *UserRepository) Count() (int, error) {
	return r.users.Count(), nil
}
