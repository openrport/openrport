package clientsauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudradar-monitoring/rport/share/query"

	"github.com/cloudradar-monitoring/rport/share/enums"
)

// FileProvider is file based client provider.
// It is not thread save so should be surrounded by CachedProvider.
type FileProvider struct {
	fileName string
}

var _ Provider = &FileProvider{}

func NewFileProvider(fileName string) *FileProvider {
	return &FileProvider{
		fileName: fileName,
	}
}

// NewMockFileProvider creates a clients auth file for testing and returns the FileProvider
func NewMockFileProvider(clients []*ClientAuth, tempDir string) *FileProvider {
	var authFile = tempDir + "/client-auth.json"
	f, _ := os.Create(authFile)
	defer f.Close()
	clientAuth := make(map[string]string)
	for _, v := range clients {
		clientAuth[v.ID] = v.Password
	}
	cj, _ := json.Marshal(clientAuth)
	if _, err := f.Write(cj); err != nil {
		log.Fatalln("error writing client-auth.json for mock provider", err)
	}
	return &FileProvider{
		fileName: authFile,
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

func (c *FileProvider) GetFiltered(filter *query.ListOptions) ([]*ClientAuth, int, error) {
	var ca []*ClientAuth
	ca, err := c.GetAll()
	if err != nil {
		return nil, 0, err
	}
	if len(filter.Filters) > 0 {
		re := regexp.MustCompile("(?i)^" + strings.Replace(filter.Filters[0].Values[0], "*", ".*?", -1) + "$")
		var filtered = []*ClientAuth{}
		for _, v := range ca {
			if re.MatchString(v.ID) {
				filtered = append(filtered, &ClientAuth{v.ID, v.Password})
			}
		}
		ca = filtered
	}
	iLimit, _ := strconv.Atoi(filter.Pagination.Limit)
	iOffset, _ := strconv.Atoi(filter.Pagination.Offset)
	c.SortByID(ca, false)
	l := len(ca)
	if iLimit > l {
		iLimit = l
	}
	if iOffset > l {
		iOffset = l
	}
	return ca[iOffset:iLimit], l, nil
}

func (c *FileProvider) Get(id string) (*ClientAuth, error) {
	idPswdPairs, err := c.load()
	if err != nil {
		return nil, fmt.Errorf("failed to decode rport clients auth file: %v", err)
	}
	if _, ok := idPswdPairs[id]; ok {
		return &ClientAuth{ID: id, Password: idPswdPairs[id]}, nil
	}
	return nil, nil
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

func (c *FileProvider) Source() enums.ProviderSource {
	return enums.ProviderSourceFile
}

func (c *FileProvider) SortByID(a []*ClientAuth, desc bool) {
	sort.Slice(a, func(i, j int) bool {
		less := a[i].ID < a[j].ID
		if desc {
			return !less
		}
		return less
	})
}
