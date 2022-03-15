package test

import (
	"os/user"

	"github.com/stretchr/testify/mock"
)

type SysUserProviderMock struct {
	mock.Mock
}

func (supm *SysUserProviderMock) GetUIDByName(user string) (uid uint32, err error) {
	args := supm.Called(user)

	return args.Get(0).(uint32), args.Error(1)
}

func (supm *SysUserProviderMock) GetGidByName(group string) (gid uint32, err error) {
	args := supm.Called(group)

	return args.Get(0).(uint32), args.Error(1)
}

func (supm *SysUserProviderMock) GetCurrentUserAndGroup() (*user.User, *user.Group, error) {
	args := supm.Called()

	var usr *user.User
	usrI := args.Get(0)
	if usrI != nil {
		usr = usrI.(*user.User)
	}

	var gr *user.Group
	grI := args.Get(1)
	if grI != nil {
		gr = grI.(*user.Group)
	}

	return usr, gr, args.Error(2)
}
