package license

type Config struct {
	ID       string `mapstructure:"id"`
	Key      string `mapstructure:"key"`
	ProxyURL string `mapstructure:"proxy_url"`
	DataDir  string // the main rportd data_dir is used
}
