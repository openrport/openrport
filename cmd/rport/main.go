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

	"github.com/spf13/pflag"

	"github.com/openrport/openrport/cmd/rport/servicemanagement"

	"github.com/kardianos/service"
	"github.com/spf13/cobra"

	"github.com/openrport/openrport/cmd/rport/cli"
	"github.com/openrport/openrport/share/files"

	chclient "github.com/openrport/openrport/client"
	chshare "github.com/openrport/openrport/share"
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
		fmt.Print(cli.ClientHelp)
		os.Exit(1)
		return nil
	})

	pFlags := RootCmd.PersistentFlags()

	cli.SetPFlags(pFlags)
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
	pFlags := RootCmd.PersistentFlags()
	serviceManager, err := isServiceManager(pFlags)
	if err != nil {
		log.Fatal(err)
	}

	if serviceManager {
		if err := ManageService(pFlags); err != nil { // app run to change state of OS service
			log.Fatal(err)
		}
	} else {
		if err := runClient(pFlags); err != nil { // app run as rport client
			log.Fatal(err)
		}
	}
}

func ManageService(pFlags *pflag.FlagSet) error {
	var svcUser string

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

		config, err := cli.DecodeConfig(cfgPath, pFlags, false)
		if err != nil {
			return fmt.Errorf("invalid config: %v. Check your config file", err)
		}

		err = config.ParseAndValidate(true)
		if err != nil {
			return fmt.Errorf("config validation failed: %v", err)
		}

	}

	return servicemanagement.HandleSvcCommand(svcCommand, cfgPath, svcUser)
}

func runClient(pFlags *pflag.FlagSet) error {

	cfgPath, err := pFlags.GetString("config")
	if err != nil {
		return err
	}

	config, err := cli.DecodeConfig(cfgPath, pFlags, service.Interactive())
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
	return servicemanagement.RunAsService(c, cfgPath)
}

func isServiceManager(pFlags *pflag.FlagSet) (bool, error) {
	svcCommand, err := pFlags.GetString("service")
	return svcCommand != "", err
}

func checkRootOK(config *chclient.ClientConfigHolder) error {
	if !config.Client.AllowRoot && chshare.IsRunningAsRoot() {
		return errors.New("by default running as root is not allowed")
	}
	return nil
}
