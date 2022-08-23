package version

import (
	"plugin"

	"github.com/cloudradar-monitoring/rport/rport-plus/validator"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

type CapabilityEx interface {
	GetVersionInfo() (info *Info)
}

type Config struct {
}

type Capability struct {
	Provider CapabilityEx

	Config *Config
	*logger.Logger
}

const (
	InitVersionCapabilityEx = "InitVersionCapabilityEx"
)

type Info struct {
	PlusVersion    string
	PlusBuildTime  string
	PlusLocalBuild string
	RportBranch    string
	RportCommitID  string
}

func (cap *Capability) GetInitFuncName() (name string) {
	return InitVersionCapabilityEx
}

func (cap *Capability) SetProvider(sym plugin.Symbol) {
	fn := sym.(func(cap *Capability) (capProvider CapabilityEx))
	cap.Provider = fn(cap)
}

func (cap *Capability) GetVersionCapabilityEx() (capEx CapabilityEx) {
	return cap.Provider
}

func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return nil
}
