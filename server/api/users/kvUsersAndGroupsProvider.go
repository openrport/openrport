package users

import (
	"context"
	"github.com/realvnc-labs/rport/share/enums"
	"github.com/realvnc-labs/rport/share/simpleops"
	"github.com/realvnc-labs/rport/share/simplestore"
)

type UserView interface {
	GetAll(ctx context.Context) ([]User, error)
	Get(ctx context.Context, username string) (User, bool, error)
}

type GroupView interface {
	GetAll(ctx context.Context) ([]Group, error)
	Get(ctx context.Context, groupID string) (Group, bool, error)
}

type kvUsersAndGroupsProvider struct {
	store     simplestore.TransactionalStore
	userView  UserView
	groupView GroupView
}

func NewKvUsersAndGroupsProvider(store simplestore.TransactionalStore) *kvUsersAndGroupsProvider {
	return &kvUsersAndGroupsProvider{
		store:     store,
		userView:  simplestore.WithType[User](store),
		groupView: simplestore.WithType[Group](store),
	}
}

func (k kvUsersAndGroupsProvider) Type() enums.ProviderSource {
	return enums.ProviderSourceKV
}

func (k kvUsersAndGroupsProvider) SupportsGroupPermissions() bool {
	return true
}

func (k kvUsersAndGroupsProvider) GetAll() ([]*User, error) {

	all, err := k.userView.GetAll(context.Background())

	return simpleops.ToPointerSlice(all), err
}

func (k kvUsersAndGroupsProvider) ListGroups() ([]Group, error) {
	return k.groupView.GetAll(context.Background())
}

func (k kvUsersAndGroupsProvider) GetGroup(groupID string) (Group, error) {
	group, found, err := k.groupView.Get(context.Background(), groupID)
	if err != nil {
		return Group{}, err
	}
	if found {
		return group, nil
	}

	return NewGroup(groupID), nil
}

func (k kvUsersAndGroupsProvider) UpdateGroup(groupID string, group Group) error {
	return k.store.Transaction(context.Background(), func(ctx context.Context, tx simplestore.Transaction) error {
		return simplestore.TransactionWithType[Group](tx).Save(groupID, group)
	})
}

func (k kvUsersAndGroupsProvider) DeleteGroup(groupID string) error {
	return k.store.Transaction(context.Background(), func(ctx context.Context, tx simplestore.Transaction) error {
		onUsers := simplestore.TransactionWithType[User](tx)
		users, err := onUsers.GetAll()
		if err != nil {
			return err
		}
		for _, user := range users {
			user.Groups = simpleops.FilterSlice(user.Groups, func(s string) bool {
				return s != groupID
			})

			err = onUsers.Save(user.Username, user)
			if err != nil {
				return err
			}
		}

		return simplestore.TransactionWithType[Group](tx).Delete(groupID)
	})

}

func (k kvUsersAndGroupsProvider) GetByUsername(username string) (*User, error) {
	user, ok, err := k.userView.Get(context.Background(), username)
	if err != nil {
		return nil, err
	}
	if ok {
		return &user, nil
	}
	return nil, nil
}

func (k kvUsersAndGroupsProvider) Add(usr *User) error {
	return k.store.Transaction(context.Background(), func(ctx context.Context, tx simplestore.Transaction) error {
		return simplestore.TransactionWithType[User](tx).Save(usr.Username, *usr)
	})
}

func (k kvUsersAndGroupsProvider) Update(usr *User, usernameToUpdate string) error {
	return k.store.Transaction(context.Background(), func(ctx context.Context, tx simplestore.Transaction) error {
		return simplestore.TransactionWithType[User](tx).Save(usernameToUpdate, *usr)
	})
}

func (k kvUsersAndGroupsProvider) Delete(usernameToDelete string) error {
	return k.store.Transaction(context.Background(), func(ctx context.Context, tx simplestore.Transaction) error {
		return simplestore.TransactionWithType[User](tx).Delete(usernameToDelete)
	})
}
