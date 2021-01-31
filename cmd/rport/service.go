package main

import (
	"log"
	"path/filepath"

	"github.com/kardianos/service"

	chclient "github.com/cloudradar-monitoring/rport/client"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

var svcConfig = &service.Config{
	Name:        "rport",
	DisplayName: "Rport Client",
	Description: "Create reverse tunnels with ease.",
}

func handleSvcCommand(svcCommand string, configPath string, user *string) error {
	svc, err := getService(nil, configPath, user)
	if err != nil {
		return err
	}

	return chshare.HandleServiceCommand(svc, svcCommand)
}

func runAsService(c *chclient.Client, configPath string) error {
	svc, err := getService(c, configPath, nil)
	if err != nil {
		return err
	}

	return svc.Run()
}

func getService(c *chclient.Client, configPath string, user *string) (service.Service, error) {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}
	svcConfig.Arguments = []string{"-c", absConfigPath}
	if user != nil {
		svcConfig.UserName = *user
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
	go func() {
		if err := w.Client.Run(); err != nil {
			log.Println(err)
		}
	}()
	return nil
}

func (w *serviceWrapper) Stop(service.Service) error {
	return w.Client.Close()
}
