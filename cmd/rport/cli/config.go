package cli

import (
	"fmt"
	"log"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	chclient "github.com/cloudradar-monitoring/rport/client"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/clientconfig"
)

type ClientAttributesConfigHolder struct {
	Tags   []string
	Labels map[string]string
}

func readConfigFile(cfgPath string) (ClientAttributesConfigHolder, error) {

	viperCfg := viper.New()
	viperCfg.SetConfigFile(cfgPath)

	attributes := ClientAttributesConfigHolder{}

	if err := viperCfg.ReadInConfig(); err != nil {
		return ClientAttributesConfigHolder{}, fmt.Errorf("error reading config file: %s", err)

	}
	err := viperCfg.Unmarshal(&attributes)
	return attributes, err
}

func DecodeConfig(cfgPath string, pFlags *pflag.FlagSet, overrideConfigWithCLIArgs bool) (*chclient.ClientConfigHolder, error) {

	viperCfg := preconfigureViperReader(cfgPath)

	config := &chclient.ClientConfigHolder{Config: &clientconfig.Config{}}

	if overrideConfigWithCLIArgs {
		BindPFlagsToViperConfig(pFlags, viperCfg)
	}

	if err := chshare.DecodeViperConfig(viperCfg, config.Config, nil); err != nil {
		return nil, err
	}

	if config.InterpreterAliases == nil {
		config.InterpreterAliases = map[string]string{}
	}

	if overrideConfigWithCLIArgs {
		if err := readArgsFromCLI(pFlags, config); err != nil {
			return nil, err
		}
	}

	readAdditionalAttributes(config)

	return config, nil
}

func preconfigureViperReader(cfgPath string) *viper.Viper {
	viperCfg := viper.New()
	viperCfg.SetConfigType("toml")

	SetViperConfigDefaults(viperCfg)

	if cfgPath != "" {
		viperCfg.SetConfigFile(cfgPath)
	} else {
		viperCfg.AddConfigPath(".")
		viperCfg.SetConfigName("rport.conf")
	}
	return viperCfg
}

func readArgsFromCLI(pFlags *pflag.FlagSet, config *chclient.ClientConfigHolder) error {
	args := pFlags.Args()

	if len(args) > 0 {
		config.Client.Server = args[0]
		config.Client.Remotes = args[1:]
	}

	scheme, err := pFlags.GetString("scheme")
	if err != nil {
		return err
	}
	config.Tunnels.Scheme = scheme

	proxy, err := pFlags.GetBool("enable-reverse-proxy")
	if err != nil {
		return err
	}
	config.Tunnels.ReverseProxy = proxy

	HostHeader, err := pFlags.GetString("host-header")
	if err != nil {
		return err
	}
	config.Tunnels.HostHeader = HostHeader

	return nil
}

func readAdditionalAttributes(config *chclient.ClientConfigHolder) {
	if len(config.Config.Client.AttributesFilePath) > 0 {
		file, err := readConfigFile(config.Config.Client.AttributesFilePath)
		if err != nil {
			log.Println("error reading attributes_file", err) // logger is not initialized yet
			log.Println("ignoring attributes_file")
			// don't panic - file doesn't have to be read successfully
		} else {
			fmt.Printf("extending config by extra client attributes file %v\n", file)
			config.Client.Tags = file.Tags
			config.Client.Labels = file.Labels
		}
	}
}
