package extendedpermissionlocal

import (
	"net/http"
	"plugin"

	"github.com/openrport/openrport/plus/capabilities/extendedpermission"
	"github.com/openrport/openrport/plus/validator"
	"github.com/openrport/openrport/share/logger"
)

type Provider struct{}

type Capability struct {
	Provider *Provider
	Config   *extendedpermission.Config
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

func (cap *Capability) GetExtendedPermissionCapabilityEx() (capEx extendedpermission.CapabilityEx) {
	return cap.Provider
}

func (cap *Capability) GetConfigValidator() (v validator.Validator) {
	return nil
}

func (p *Provider) ValidateExtendedTunnelPermission(r *http.Request, tr []extendedpermission.PermissionParams) error {
	return nil
}

func (p *Provider) ValidateExtendedCommandPermission(r *http.Request, cr []extendedpermission.PermissionParams) error {
	return nil
}

func (p *Provider) ValidateExtendedCommandPermissionRaw(command string, isSudo bool, cr []extendedpermission.PermissionParams) error {
	return nil
}

func (p *Provider) ValidateExtendedDeleteNonOwnedTunnelPermissionRaw(tr []extendedpermission.PermissionParams) error {
	return nil
}
