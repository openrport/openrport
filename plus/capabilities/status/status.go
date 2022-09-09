package status

import (
	"plugin"

	"github.com/cloudradar-monitoring/rport/plus/validator"
	"github.com/cloudradar-monitoring/rport/share/logger"
)

type CapabilityEx interface {
	GetStatusInfo() (info *PlusStatusInfo)
}

type Config struct {
}

type Capability struct {
	Provider CapabilityEx

	Config *Config
	*logger.Logger
}

const (
	InitPlusStatusCapabilityEx = "InitPlusStatusCapabilityEx"
)

type PlusStatusInfo struct {
	PlusVersion    string
	PlusBuildTime  string
	PlusLocalBuild string
	RportGitRef    string
	RportCommitID  string
}

func (cap *Capability) GetInitFuncName() (name string) {
	return InitPlusStatusCapabilityEx
}

func (cap *Capability) SetProvider(sym plugin.Symbol) {
	fn := sym.(func(cap *Capability) (capProvider CapabilityEx))
	cap.Provider = fn(cap)
}

func (cap *Capability) GetStatusCapabilityEx() (capEx CapabilityEx) {
	return cap.Provider
}

func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return nil
}
