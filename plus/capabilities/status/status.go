package status

import (
	"plugin"

	licensecap "github.com/realvnc-labs/rport/plus/capabilities/license"
	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/share/logger"
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
	IsEnabled    bool                       `json:"is_enabled"`
	IsTrial      bool                       `json:"is_trial"`
	ValidLicense bool                       `json:"valid_license"`
	LicenseInfo  licensecap.PlusLicenseInfo `json:"lic_info"`

	PlusVersion    string `json:"plus_version"`
	PlusBuildTime  string `json:"build_time"`
	PlusLocalBuild string `json:"local_build"`
	RportGitRef    string `json:"rport_git_ref"`
	RportCommitID  string `json:"rport_commit_id"`
}

func (cap *Capability) GetInitFuncName() (name string) {
	return InitPlusStatusCapabilityEx
}

func (cap *Capability) InitProvider(sym plugin.Symbol) {
	fn := sym.(func(cap *Capability) (capProvider CapabilityEx))
	cap.Provider = fn(cap)
}

func (cap *Capability) GetStatusCapabilityEx() (capEx CapabilityEx) {
	return cap.Provider
}

func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return nil
}
