package oauth_test

// const (
// 	defaultPluginPath = "../../../../rport-plus/plugin.so"
// )

// func TestShouldFailToRegisterOAuthCapabilityDueToMissingProvider(t *testing.T) {
// 	cap := oauth.Capability{
// 		Config: &oauth.Config{
// 			Enabled: true,
// 		},
// 	}

// 	err := cap.LoadSymbols(defaultPluginPath)
// 	require.NoError(t, err)

// 	err = cap.ValidateConfig()
// 	assert.EqualError(t, err, "no provider specified")
// }

// func TestShouldFailToRegisterOAuthCapabilityDueToUnknownProvider(t *testing.T) {
// 	cap := oauth.Capability{
// 		Config: &oauth.Config{
// 			Enabled:  true,
// 			Provider: "OAuth",
// 		},
// 	}

// 	err := cap.LoadSymbols(defaultPluginPath)
// 	require.NoError(t, err)

// 	err = cap.ValidateConfig()
// 	assert.EqualError(t, err, "provider (OAuth) not currently supported")
// }

// func TestShouldoRegisterOAuthCapability(t *testing.T) {
// 	cap := oauth.Capability{
// 		Config: &oauth.Config{
// 			Enabled:  true,
// 			Provider: oauth.GitHubOAuthProvider,
// 		},
// 	}

// 	err := cap.LoadSymbols(defaultPluginPath)
// 	require.NoError(t, err)

// 	err = cap.ValidateConfig()
// 	assert.NoError(t, err)

// 	count := cap.GetSymbolTable().GetSymbolCount()
// 	assert.Equal(t, 4, count)
// }
