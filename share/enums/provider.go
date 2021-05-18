package enums

type ProviderSource string

const (
	ProviderSourceStatic ProviderSource = "Static Credentials"
	ProviderSourceFile   ProviderSource = "File"
	ProviderSourceDB     ProviderSource = "DB"
	ProviderSourceMock   ProviderSource = "Mock"
)
