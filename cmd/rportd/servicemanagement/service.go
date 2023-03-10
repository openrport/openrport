package servicemanagement

import (
	"context"
	"log"
	"path/filepath"

	"github.com/kardianos/service"

	chserver "github.com/cloudradar-monitoring/rport/server"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

var svcConfig = &service.Config{
	Name:        "rportd",
	DisplayName: "Rport Server",
	Description: "Create reverse tunnels with ease.",
	Option: service.KeyValue{
		"LimitNOFILE":         1048576,
		"AmbientCapabilities": "CAP_NET_BIND_SERVICE",
	},
}

func HandleSvcCommand(svcCommand string, configPath string, user *string) error {
	svc, err := getService(nil, configPath, user)
	if err != nil {
		return err
	}

	return chshare.HandleServiceCommand(svc, svcCommand)
}

func RunAsService(s *chserver.Server, configPath string) error {
	svc, err := getService(s, configPath, nil)
	if err != nil {
		return err
	}

	return svc.Run()
}

func getService(s *chserver.Server, configPath string, user *string) (service.Service, error) {
	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, err
	}
	svcConfig.Arguments = []string{"-c", absConfigPath}
	if user != nil {
		svcConfig.UserName = *user
	}
	return service.New(&serviceWrapper{s}, svcConfig)
}

type serviceWrapper struct {
	*chserver.Server
}

func (w *serviceWrapper) Start(service.Service) error {
	if w.Server == nil {
		return nil
	}
	go func() {
		ctx := context.Background()
		if err := w.Server.Run(ctx); err != nil {
			log.Println(err)
		}
	}()
	return nil
}

func (w *serviceWrapper) Stop(service.Service) error {
	return w.Server.Close()
}
