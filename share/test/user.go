package test

import "github.com/stretchr/testify/mock"

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
