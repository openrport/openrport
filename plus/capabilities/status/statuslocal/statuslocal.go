package statuslocal

import (
	"plugin"

	"github.com/openrport/openrport/plus/capabilities/status"
	"github.com/openrport/openrport/plus/validator"
	"github.com/openrport/openrport/share/logger"
)

type Provider struct{}

type Capability struct {
	Provider *Provider
	Config   *status.Config
	Logger   *logger.Logger
}

func (cap *Capability) GetInitFuncName() (name string) {
	return ""
}

func (cap *Capability) InitProvider(_ plugin.Symbol) {
	if cap.Provider == nil {
		cap.Provider = &Provider{}
	}
}

func (cap *Capability) GetStatusCapabilityEx() (capEx status.CapabilityEx) {
	return cap.Provider
}

func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return nil
}

func (p *Provider) GetStatusInfo() (info *status.PlusStatusInfo) {
	return &status.PlusStatusInfo{
		PlusVersion:   "native",
		PlusBuildTime: "native",
		RportGitRef:   "native",
		RportCommitID: "native",
	}
}
