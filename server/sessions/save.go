package sessions

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type SaveToFileTask struct {
	log      *chshare.Logger
	csr      *ClientSessionRepository
	fileName string
}

// NewSaveToFileTask returns a task to save Client Session Repository to a given file on disk.
func NewSaveToFileTask(log *chshare.Logger, csr *ClientSessionRepository, fileName string) *SaveToFileTask {
	return &SaveToFileTask{
		log:      log,
		csr:      csr,
		fileName: fileName,
	}
}

func (t *SaveToFileTask) Run() error {
	// create or truncate the file
	file, err := os.Create(t.fileName)
	if err != nil {
		return fmt.Errorf("failed to open CSR file: %v", err)
	}
	defer file.Close()

	return t.getAndSave(file)
}

func (t *SaveToFileTask) getAndSave(w io.Writer) error {
	sessions, err := t.csr.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get client sessions from CSR: %v", err)
	}
	t.log.Debugf("Got %d client sessions from CSR. Writing...", len(sessions))

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(sessions); err != nil {
		return fmt.Errorf("failed to write CSR: %v", err)
	}

	return nil
}
