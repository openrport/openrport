package clientsauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"github.com/patrickmn/go-cache"

	"github.com/openrport/openrport/share/enums"
	"github.com/openrport/openrport/share/query"
)

// FileProvider is file based client provider.
// It is not thread save so should be surrounded by CachedProvider.
type FileProvider struct {
	fileName string
	cache    *cache.Cache
}

var _ Provider = &FileProvider{}

func NewFileProvider(fileName string, cache *cache.Cache) *FileProvider {
	return &FileProvider{
		fileName: fileName,
		cache:    cache,
	}
}

// GetAll returns rport clients auth credentials from a given file.
func (c *FileProvider) getAll() ([]*ClientAuth, error) {
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
	ca, err := c.getAll()
	if err != nil {
		return nil, 0, err
	}
	if len(filter.Filters) > 0 {
		var filtered = []*ClientAuth{}
		for _, v := range ca {
			match, err := query.MatchesFilters(v, filter.Filters)
			if err != nil {
				return nil, 0, err
			}
			if match {
				filtered = append(filtered, &ClientAuth{v.ID, v.Password})
			}
		}
		ca = filtered
	}
	c.SortByID(ca, false)
	l := len(ca)
	start, end := filter.Pagination.GetStartEnd(l)
	return ca[start:end], l, nil
}

func (c *FileProvider) Get(id string) (*ClientAuth, error) {
	if val, _ := c.cache.Get(c.CacheKey(id)); val != nil {
		return val.(*ClientAuth), nil
	}
	idPswdPairs, err := c.load()
	if err != nil {
		return nil, fmt.Errorf("failed to decode rport clients auth file: %v", err)
	}
	if _, ok := idPswdPairs[id]; ok {
		ca := &ClientAuth{ID: id, Password: idPswdPairs[id]}
		if err := c.cache.Add(c.CacheKey(id), ca, 60*time.Minute); err != nil {
			return nil, err
		}
		return ca, nil
	}
	return nil, nil
}

func (c *FileProvider) Add(clientAuth *ClientAuth) (bool, error) {
	idPswdPairs, err := c.load()
	if err != nil {
		return false, fmt.Errorf("failed to decode rport clients auth file: %v", err)
	}

	clientID := clientAuth.ID

	if _, ok := idPswdPairs[clientID]; ok {
		return false, nil
	}

	idPswdPairs[clientID] = clientAuth.Password

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
	c.cache.Delete(c.CacheKey(id))

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

func (c *FileProvider) CacheKey(id string) string {
	return "client-auth-" + id
}
