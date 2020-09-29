package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type SaveToFileTask struct {
	log      *chshare.Logger
	clients  *ClientCache
	fileName string
}

// NewSaveToFileTask returns a task to save rport clients to a given file on disk.
func NewSaveToFileTask(log *chshare.Logger, clients *ClientCache, fileName string) *SaveToFileTask {
	return &SaveToFileTask{
		log:      log,
		clients:  clients,
		fileName: fileName,
	}
}

func (t *SaveToFileTask) Run() error {
	file, err := os.Create(t.fileName)
	if err != nil {
		return fmt.Errorf("failed to open rport clients file: %v", err)
	}
	defer file.Close()

	return t.getAndSave(file)
}

func (t *SaveToFileTask) getAndSave(w io.Writer) error {
	rClients := t.clients.GetAll()
	t.log.Debugf("Got %d rport clients from cache. Writing...", len(rClients))

	idPswdPairs := make(map[string]string, len(rClients))
	for _, v := range rClients {
		idPswdPairs[v.ID] = v.Password
	}

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(idPswdPairs); err != nil {
		return fmt.Errorf("failed to write rport clients: %v", err)
	}

	return nil
}
