package alertinglocal

import (
	"plugin"

	alertingcap "github.com/openrport/openrport/plus/capabilities/alerting"
	"github.com/openrport/openrport/plus/capabilities/alerting/alertingmock"
	"github.com/openrport/openrport/plus/validator"
	"github.com/openrport/openrport/share/logger"
)

type Capability struct {
	Provider alertingcap.CapabilityEx
	Config   *alertingcap.Config
	Logger   *logger.Logger
}

func (cap *Capability) GetInitFuncName() (name string) {
	return ""
}

func (cap *Capability) InitProvider(_ plugin.Symbol) {
	if cap.Provider == nil {
		mockCap := &alertingmock.Capability{}
		mockCap.InitProvider(nil)
		cap.Provider = mockCap.Provider
	}
}

func (cap *Capability) GetAlertingCapabilityEx() (capEx alertingcap.CapabilityEx) {
	return cap.Provider
}

func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	if cap.Provider == nil {
		return nil
	}
	if vv, ok := cap.Provider.(validator.Validator); ok {
		return vv
	}
	return nil
}
