package clientsauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

// FileProvider is file based client provider.
// It is not thread save so should be surrounded by CachedProvider.
type FileProvider struct {
	fileName string
}

func NewFileProvider(fileName string) *FileProvider {
	return &FileProvider{
		fileName: fileName,
	}
}

// GetAll returns rport clients auth credentials from a given file.
func (c *FileProvider) GetAll() ([]*ClientAuth, error) {
	idPswdPairs, err := c.load()
	if err != nil {
		return nil, fmt.Errorf("failed to decode rport clients auth file: %v", err)
	}

	var res []*ClientAuth
	for id, pswd := range idPswdPairs {
		if id == "" || pswd == "" {
			return nil, errors.New("empty client auth ID or password is not allowed")
		}
		res = append(res, &ClientAuth{ID: id, Password: pswd})
	}

	return res, nil
}

func (c *FileProvider) Get(id string) (*ClientAuth, error) {
	idPswdPairs, err := c.load()
	if err != nil {
		return nil, fmt.Errorf("failed to decode rport clients auth file: %v", err)
	}

	return &ClientAuth{ID: id, Password: idPswdPairs[id]}, nil
}

func (c *FileProvider) Add(client *ClientAuth) (bool, error) {
	idPswdPairs, err := c.load()
	if err != nil {
		return false, fmt.Errorf("failed to decode rport clients auth file: %v", err)
	}

	if _, ok := idPswdPairs[client.ID]; ok {
		return false, nil
	}

	idPswdPairs[client.ID] = client.Password

	if err := c.save(idPswdPairs); err != nil {
		return false, fmt.Errorf("failed to encode rport clients auth file: %v", err)
	}

	return true, nil
}

func (c *FileProvider) Delete(id string) error {
	idPswdPairs, err := c.load()
	if err != nil {
		return fmt.Errorf("failed to decode rport clients auth file: %v", err)
	}

	delete(idPswdPairs, id)

	if err := c.save(idPswdPairs); err != nil {
		return fmt.Errorf("failed to encode rport clients auth file: %v", err)
	}

	return nil
}

func (c *FileProvider) IsWriteable() bool {
	return true
}

func (c *FileProvider) load() (map[string]string, error) {
	b, err := ioutil.ReadFile(c.fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rport clients auth file %q: %s", c.fileName, err)
	}

	var idPswdPairs map[string]string
	if err := json.Unmarshal(b, &idPswdPairs); err != nil {
		return nil, err
	}

	return idPswdPairs, nil
}

func (c *FileProvider) save(idPswdPairs map[string]string) error {
	file, err := os.OpenFile(c.fileName, os.O_RDWR|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open rport clients auth file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "	")
	if err := encoder.Encode(idPswdPairs); err != nil {
		return fmt.Errorf("failed to write rport clients auth: %v", err)
	}

	return nil
}
