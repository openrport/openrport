package rportplus

func IsPlusEnabled(config PlusConfig) bool {
	if config.OAuthConfig != nil {
		return true
	}
	return config.PluginConfig != nil
}

func HasLicenseConfig(config PlusConfig) bool {
	return IsPlusEnabled(config) && config.LicenseConfig != nil
}

func IsPlusOAuthEnabled(config PlusConfig) bool {
	return IsPlusEnabled(config) && config.OAuthConfig != nil
}

func IsOAuthPermittedUserList(config PlusConfig) bool {
	if !IsPlusEnabled(config) {
		return false
	}
	return config.OAuthConfig != nil && config.OAuthConfig.PermittedUserList
}
