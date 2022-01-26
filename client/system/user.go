package system

import (
	"fmt"
	"os/user"
	"strconv"
)

type SysUserLookup interface {
	GetUIDByName(user string) (uid uint32, err error)
	GetGidByName(group string) (gid uint32, err error)
}

type SysUserProvider struct{}

func (sup SysUserProvider) GetUIDByName(user string) (uid uint32, err error) {
	return GetUIDByName(user)
}

func (sup SysUserProvider) GetGidByName(group string) (gid uint32, err error) {
	return GetGidByName(group)
}

func GetUIDByName(name string) (uid uint32, err error) {
	usr, err := user.Lookup(name)
	if err != nil {
		return 0, err
	}

	u64, err := strconv.ParseUint(usr.Uid, 10, 32)
	if err != nil {
		fmt.Println(err)
	}

	return uint32(u64), nil
}

func GetGidByName(group string) (gid uint32, err error) {
	gr, err := user.LookupGroup(group)
	if err != nil {
		return 0, err
	}

	u64, err := strconv.ParseUint(gr.Gid, 10, 32)
	if err != nil {
		fmt.Println(err)
	}

	return uint32(u64), nil
}
