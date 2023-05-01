package permission

import (
	"plugin"

	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/share/logger"
)

type CapabilityEx interface {
	GetPermissionInfo() (info *PlusPermissionInfo)
}

type Config struct {
}

type Capability struct {
	Provider CapabilityEx

	Config *Config
	*logger.Logger
}

const (
	InitPlusPermissionCapabilityEx = "InitPlusPermissionCapabilityEx"
)

type PlusPermissionInfo struct {
	FooVar1 string `json:"foo_var1"`
}

func (cap *Capability) GetInitFuncName() (name string) {
	return InitPlusPermissionCapabilityEx
}

func (cap *Capability) InitProvider(sym plugin.Symbol) {
	fn := sym.(func(cap *Capability) (capProvider CapabilityEx))
	cap.Provider = fn(cap)
}

func (cap *Capability) GetPermissionCapabilityEx() (capEx CapabilityEx) {
	return cap.Provider
}

func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return nil
}
