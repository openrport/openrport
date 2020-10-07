package main

import (
	"path/filepath"

	"github.com/kardianos/service"

	chclient "github.com/cloudradar-monitoring/rport/client"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

var svcConfig = &service.Config{
	Name:        "rport",
	DisplayName: "CloudRadar Rport Client",
	Description: "Create reverse tunnels with ease.",
}

func handleSvcCommand(svcCommand string, configPath string) error {
	svc, err := getService(nil, configPath)
	if err != nil {
		return err
	}

	return chshare.HandleServiceCommand(svc, svcCommand)
}

func runAsService(c *chclient.Client, configPath string) error {
	svc, err := getService(c, configPath)
	if err != nil {
		return err
	}

	return svc.Run()
}

func getService(c *chclient.Client, configPath string) (service.Service, error) {
	if configPath != "" {
		absConfigPath, err := filepath.Abs(configPath)
		if err != nil {
			return nil, err
		}
		svcConfig.Arguments = []string{"-c", absConfigPath}
	}
	return service.New(&serviceWrapper{c}, svcConfig)
}

type serviceWrapper struct {
	*chclient.Client
}

func (w *serviceWrapper) Start(service.Service) error {
	if w.Client == nil {
		return nil
	}
	go w.Client.Run()
	return nil
}

func (w *serviceWrapper) Stop(service.Service) error {
	return w.Client.Close()
}
