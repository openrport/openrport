package clients

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	chshare "github.com/cloudradar-monitoring/rport/share"
)

type FileClients struct {
	log      *chshare.Logger
	fileName string
}

func NewFileClients(log *chshare.Logger, fileName string) *FileClients {
	return &FileClients{
		log:      log,
		fileName: fileName,
	}
}

// GetAll returns rport clients from a given file.
func (c *FileClients) GetAll() ([]*Client, error) {
	c.log.Infof("Start to get rport clients from file.")

	b, err := ioutil.ReadFile(c.fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rport clients auth file %q: %s", c.fileName, err)
	}
	c.log.Infof("Parsing rport clients data...")

	res, err := decodeClients(b)
	if err != nil {
		return nil, fmt.Errorf("failed to decode rport clients auth file: %v", err)
	}

	c.log.Infof("Loaded %d rport clients from file.", len(res))
	return res, nil
}

func decodeClients(b []byte) ([]*Client, error) {
	var idPswdPairs map[string]string
	if err := json.Unmarshal(b, &idPswdPairs); err != nil {
		return nil, err
	}

	var res []*Client
	for id, pswd := range idPswdPairs {
		if id == "" || pswd == "" {
			return nil, fmt.Errorf("empty client ID or password is not allowed")
		}
		res = append(res, &Client{ID: id, Password: pswd})
	}

	return res, nil
}
