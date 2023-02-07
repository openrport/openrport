package license

type Config struct {
	ID       string `mapstructure:"id"`
	Key      string `mapstructure:"key"`
	ProxyURL string `mapstructure:"proxy_url"`
	DataDir  string // the main rportd data_dir is used
	// TODO: (rs): remove this now?
	CheckingEnabled bool // allow license checking to be disabled/enabled. will be removed for 1.0.
}
