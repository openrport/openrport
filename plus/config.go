package rportplus

import (
	"github.com/realvnc-labs/rport/plus/capabilities/oauth"
	"github.com/realvnc-labs/rport/plus/license"
)

// PluginConfig contains the config related to the plugin itself
type PluginConfig struct {
	PluginPath string `mapstructure:"plugin_path"`
}

// PlusConfig contains the overall config for the rport-plus plugin. note that
// each capability should have it's own config section in the config file.
type PlusConfig struct {
	PluginConfig  *PluginConfig   `mapstructure:"plus-plugin"`
	OAuthConfig   *oauth.Config   `mapstructure:"plus-oauth"`
	LicenseConfig *license.Config `mapstructure:"plus-license"`
}

// EDTODO: validateExtendedTunnelPermissions contains the logic for the validation and must be executed in rport-plus
