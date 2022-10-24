package rportplus

import "github.com/cloudradar-monitoring/rport/plus/capabilities/oauth"

// PluginConfig contains the config related to the plugin itself
type PluginConfig struct {
	PluginPath string `mapstructure:"plugin_path"`
}

// PlusConfig contains the overall config for the rport-plus plugin. note that
// each capability should have it's own config section in the config file.
type PlusConfig struct {
	PluginConfig *PluginConfig `mapstructure:"plus-plugin"`
	OAuthConfig  *oauth.Config `mapstructure:"plus-oauth"`
}
