package csr

import (
	"encoding/json"
	"fmt"
	"os"

	chserver "github.com/cloudradar-monitoring/rport/server"
	chshare "github.com/cloudradar-monitoring/rport/share"
)

type SaveToFileTask struct {
	log      *chshare.Logger
	csr      *chserver.ClientSessionRepository
	fileName string
}

// NewSaveToFileTask returns a task to save Client Session Repository to a given file on disk.
func NewSaveToFileTask(log *chshare.Logger, csr *chserver.ClientSessionRepository, fileName string) *SaveToFileTask {
	return &SaveToFileTask{
		log:      log,
		csr:      csr,
		fileName: fileName,
	}
}

func (t *SaveToFileTask) Run() error {
	sessions, err := t.csr.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get client sessions from CSR: %v", err)
	}
	t.log.Debugf("Got %d client sessions from CSR. Writing to file...", len(sessions))

	if err := createOrOverrideFile(t.fileName, sessions); err != nil {
		return fmt.Errorf("failed to write client sessions to CSR file: %v", err)
	}

	return nil
}

func createOrOverrideFile(fileName string, data interface{}) error {
	// create or truncate the file
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to write to file: %v", err)
	}

	return nil
}
