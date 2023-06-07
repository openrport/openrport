package licensecap

import (
	"plugin"

	"github.com/realvnc-labs/rport/plus/validator"
	"github.com/realvnc-labs/rport/share/logger"
)

const (
	InitPlusLicenseCapabilityEx = "InitPlusLicenseCapabilityEx"
)

type LicenseInfoAvailableNotifier func()

type CapabilityEx interface {
	SetLicenseInfoAvailableNotifier(notifyFn LicenseInfoAvailableNotifier)
	LicenseInfoAvailable() (avail bool)

	IsTrialMode() (isTrial bool)
	GetLicenseInfo() (licenseInfo *PlusLicenseInfo)

	GetMaxClients() (maxClients int)
	GetMaxUsers() (maxUsers int)
}

type Config struct {
}

type Capability struct {
	Provider CapabilityEx

	Config *Config
	*logger.Logger
}

type PlusLicenseInfo struct {
	MaxClients int `json:"max_clients"`
	MaxUsers   int `json:"max_users"`
}

func (cap *Capability) GetInitFuncName() (name string) {
	return InitPlusLicenseCapabilityEx
}

func (cap *Capability) InitProvider(sym plugin.Symbol) {
	fn := sym.(func(cap *Capability) (capProvider CapabilityEx))
	cap.Provider = fn(cap)
}

func (cap *Capability) GetLicenseCapabilityEx() (capEx CapabilityEx) {
	return cap.Provider
}

func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return nil
}
