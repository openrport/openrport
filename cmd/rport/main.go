//go:generate goversioninfo -icon=../../opt/resource/app.ico
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/cloudradar-monitoring/rport/cmd/rport/cliboilerplate"
	"github.com/cloudradar-monitoring/rport/share/files"

	chclient "github.com/cloudradar-monitoring/rport/client"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/clientconfig"
)

var (
	RootCmd *cobra.Command
)

func init() {
	// Assign root cmd late to avoid initialization loop
	RootCmd = &cobra.Command{
		Version: chshare.BuildVersion,
		Run:     runMain,
	}

	// set help message
	RootCmd.SetUsageFunc(func(*cobra.Command) error {
		fmt.Print(cliboilerplate.ClientHelp)
		os.Exit(1)
		return nil
	})

	pFlags := RootCmd.PersistentFlags()

	cliboilerplate.SetPFlags(pFlags)
}

// main this binary can be run in 3 ways
// 1 - interactive - for development or other advanced use
// 2 - as an OS service
// 3 - as interface for managing OS service (start, stop, install, uninstall, etc) (install needs to check config and create dirs)
func main() {
	if err := RootCmd.Execute(); err != nil {
		log.Fatalf("failed executing RootCmd: %v", err)
	}
}

func runMain(*cobra.Command, []string) {
	serviceManager, err := isServiceManager()
	if err != nil {
		log.Fatal(err)
	}

	if serviceManager {
		if err := manageService(); err != nil { // app run to change state of OS service
			log.Fatal(err)
		}
	} else {
		if err := runClient(); err != nil { // app run as rport client
			log.Fatal(err)
		}
	}
}

func manageService() error {
	var svcUser string
	pFlags := RootCmd.PersistentFlags()
	cfgPath, err := pFlags.GetString("config")
	if err != nil {
		return err
	}

	svcCommand, err := pFlags.GetString("service")
	if err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		svcUser, err = pFlags.GetString("service-user")
		if err != nil {
			return err
		}
	}

	if svcCommand == "install" {
		// validate config file without command line args before installing it for the service
		// other service commands do not change config file specified at install

		config, err := decodeConfig(cfgPath, false)
		if err != nil {
			return fmt.Errorf("invalid config: %v. Check your config file", err)
		}

		err = config.ParseAndValidate(true)
		if err != nil {
			return fmt.Errorf("config validation failed: %v", err)
		}

	}

	return cliboilerplate.HandleSvcCommand(svcCommand, cfgPath, svcUser)
}

func runClient() error {
	pFlags := RootCmd.PersistentFlags()

	cfgPath, err := pFlags.GetString("config")
	if err != nil {
		return err
	}

	config, err := decodeConfig(cfgPath, service.Interactive())
	if err != nil {
		return fmt.Errorf("invalid config: %v. Check your config file", err)
	}

	err = config.Logging.LogOutput.Start()
	if err != nil {
		return fmt.Errorf("failed starting log output: %v", err)
	}
	defer config.Logging.LogOutput.Shutdown()

	err = chclient.PrepareDirs(config)
	if err != nil {
		return fmt.Errorf("failed preparing directories: %v", err)
	}

	err = config.ParseAndValidate(false)
	if err != nil {
		return fmt.Errorf("config validation failed: %v", err)
	}

	err = checkRootOK(config)
	if err != nil {
		return fmt.Errorf("root check failed: %v", err)
	}

	fileAPI := files.NewFileSystem()
	c, err := chclient.NewClient(config, fileAPI)
	if err != nil {
		return fmt.Errorf("failed creating client: %v", err)
	}

	if service.Interactive() { // if run from command line

		go chshare.GoStats()

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		defer cancel()
		return c.Run(ctx)

	}
	// if run as OS service
	return cliboilerplate.RunAsService(c, cfgPath)
}

func isServiceManager() (bool, error) {
	pFlags := RootCmd.PersistentFlags()
	svcCommand, err := pFlags.GetString("service")
	return svcCommand != "", err
}

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

func decodeConfig(cfgPath string, overrideConfigWithCLIArgs bool) (*chclient.ClientConfigHolder, error) {

	viperCfg := viper.New()
	viperCfg.SetConfigType("toml")

	cliboilerplate.SetViperConfigDefaults(viperCfg)

	if cfgPath != "" {
		viperCfg.SetConfigFile(cfgPath)
	} else {
		viperCfg.AddConfigPath(".")
		viperCfg.SetConfigName("rport.conf")
	}

	config := &chclient.ClientConfigHolder{Config: &clientconfig.Config{}}

	pFlags := RootCmd.PersistentFlags()

	if overrideConfigWithCLIArgs {
		cliboilerplate.BindPFlagsToViperConfig(pFlags, viperCfg)
	}

	if err := chshare.DecodeViperConfig(viperCfg, config.Config, nil); err != nil {
		return nil, err
	}

	if config.InterpreterAliases == nil {
		config.InterpreterAliases = map[string]string{}
	}

	if overrideConfigWithCLIArgs {
		args := pFlags.Args()

		if len(args) > 0 {
			config.Client.Server = args[0]
			config.Client.Remotes = args[1:]
		}

		scheme, err := pFlags.GetString("scheme")
		if err != nil {
			return nil, err
		}
		config.Tunnels.Scheme = scheme

		proxy, err := pFlags.GetBool("enable-reverse-proxy")
		if err != nil {
			return nil, err
		}
		config.Tunnels.ReverseProxy = proxy

		HostHeader, err := pFlags.GetString("host-header")
		if err != nil {
			return nil, err
		}
		config.Tunnels.HostHeader = HostHeader
	}

	if len(config.Config.Client.AttributesFilePath) > 0 {
		file, err := readConfigFile(config.Config.Client.AttributesFilePath)
		if err != nil {
			log.Println("error reading attributes_file", err)
			log.Println("ignoring attributes_file")
		} else {
			fmt.Printf("extending config by extra client attributes file %v\n", file)
			config.Client.Tags = file.Tags
			config.Client.Labels = file.Labels
		}
	}
	return config, nil
}

func checkRootOK(config *chclient.ClientConfigHolder) error {
	if !config.Client.AllowRoot && chshare.IsRunningAsRoot() {
		return errors.New("by default running as root is not allowed")
	}
	return nil
}
