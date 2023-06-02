package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/realvnc-labs/rport/share/models"

	chclient "github.com/realvnc-labs/rport/client"
	chshare "github.com/realvnc-labs/rport/share"
	"github.com/realvnc-labs/rport/share/clientconfig"
)

func readAttributesFile(cfgPath string) (models.Attributes, error) {

	attributes := models.Attributes{}

	data, err := os.ReadFile(cfgPath)

	if err != nil {
		return models.Attributes{}, err
	}

	err = json.Unmarshal(data, &attributes)

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
		file, err := readAttributesFile(config.Config.Client.AttributesFilePath)
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
