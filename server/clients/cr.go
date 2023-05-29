package clients

import (
	"context"
	"github.com/realvnc-labs/rport/share/simpleops"
	"github.com/realvnc-labs/rport/share/simplestore/kvs/inmemory"
	"log"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/realvnc-labs/rport/server/cgroups"
	"github.com/realvnc-labs/rport/share/dynops/filterer"
	"github.com/realvnc-labs/rport/share/dynops/formatter"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/query"
	"github.com/realvnc-labs/rport/share/simplestore"
)

type ClientRepository struct {

	// db based store
	clientStore ClientStore

	keepDisconnectedClients *time.Duration

	logger *logger.Logger
}

type User interface {
	IsAdmin() bool
	GetGroups() []string
}

// NewClientRepositoryForTestsSetupWithInMemoryCache (for test only) returns a new thread-safe in-memory cache to store client connections populated with given clients if any.
// keepDisconnectedClients is a duration to keep disconnected clients. If a client was disconnected longer than a given
// duration it will be treated as obsolete.
func NewClientRepositoryForTestsSetupWithInMemoryCache(initClients []*Client, keepDisconnectedClients *time.Duration, logger *logger.Logger) *ClientRepository { // test only
	mem := inmemory.NewInMemory()
	ctx := context.Background()
	store, _ := simplestore.NewSimpleStore[Client](ctx, mem)
	provider := NewSimpleClientStore(store, keepDisconnectedClients)
	for _, c := range initClients {
		_ = provider.Save(ctx, c)
	}
	return NewClientRepositoryWithDB(keepDisconnectedClients, provider, logger)
}

// NewClientRepositoryWithDB @todo: used for test setup in two separate packages. need to review use as part of the test code refactoring.
func NewClientRepositoryWithDB(keepDisconnectedClients *time.Duration, store ClientStore, logger *logger.Logger) *ClientRepository {
	return &ClientRepository{
		clientStore:             store,
		logger:                  logger,
		keepDisconnectedClients: keepDisconnectedClients,
	}
}

func (r *ClientRepository) Save(client *Client) error {
	ts := time.Now()

	err := r.clientStore.Save(context.Background(), client)
	if err != nil {
		return err
	}

	r.log().Debugf(
		"saved client: %s status=%s, within %s",
		client.GetID(),
		FormatConnectionState(client),
		time.Since(ts),
	)

	return nil
}

func (r *ClientRepository) Delete(client *Client) error {
	clientID := client.GetID()

	r.log().Debugf("deleting client: %s status=%s", clientID, FormatConnectionState(client))

	return r.clientStore.Delete(context.Background(), clientID, r.logger)
}

func (r *ClientRepository) GetClientsByTag(tags []string, operator string, allowDisconnected bool) (matchingClients []*Client, err error) {
	var availableClients []*Client
	if allowDisconnected {
		availableClients, _ = r.GetAllClients()
	} else {
		availableClients, _ = r.GetAllActiveClients()
	}
	if strings.EqualFold(operator, "AND") {
		matchingClients = findMatchingANDClients(availableClients, tags)
	} else {
		matchingClients = findMatchingORClients(availableClients, tags)
	}

	return matchingClients, nil
}

// this fn doesn't lock the availableClients. please make sure not to use the main clients state array.
// the various GetXXXClient fns will return new client arrays. please use those fns to get a
// clients array copy for this fn to operate on.
func findMatchingANDClients(availableClients []*Client, tags []string) (matchingClients []*Client) {
	matchingClients = make([]*Client, 0, 64)
	for _, cl := range availableClients {
		clientTags := cl.GetTags()

		foundAllTags := true
		for _, tag := range tags {
			foundTag := false
			for _, clTag := range clientTags {
				if tag == clTag {
					foundTag = true
					break
				}
			}
			if !foundTag {
				foundAllTags = false
				break
			}
		}
		if foundAllTags {
			matchingClients = append(matchingClients, cl)
		}

	}
	return matchingClients
}

// this fn doesn't lock the availableClients. please make sure not to use the main clients array.
// the various GetXXXClient fns will return new client arrays. please use those fns to Get a
// clients array copy for this fn to operate on.
func findMatchingORClients(availableClients []*Client, tags []string) (matchingClients []*Client) {
	matchingClients = make([]*Client, 0, 64)
	for _, cl := range availableClients {
		clientTags := cl.GetTags()
	nextClientForOR:
		for _, clTag := range clientTags {
			for _, tag := range tags {
				if tag == clTag {
					matchingClients = append(matchingClients, cl)
					break nextClientForOR
				}
			}
		}
	}
	return matchingClients
}

// DeleteObsolete deletes obsolete disconnected clients and returns them.
func (r *ClientRepository) DeleteObsolete() ([]*Client, error) {
	ctx := context.Background()
	r.log().Debugf("deleting obsolete clients")
	if r.keepDisconnectedClients == nil {
		return nil, nil
	}
	all, err := r.clientStore.GetAll(ctx, r.logger)
	if err != nil {
		return nil, err
	}

	cutOff := time.Now().Add(-*r.keepDisconnectedClients)

	deleted := []*Client{}
	for _, c := range all { //nolint:govet
		if c.DisconnectedAt.After(cutOff) {
			r.log().Debugf("deleting obsolete client: %s status=%s", c.ID, FormatConnectionState(c))
			err = r.clientStore.Delete(ctx, c.ID, r.log())
			if err != nil {
				return nil, err
			}
			deleted = append(deleted, c)
		}
	}

	return deleted, nil
}

// CountActive returns a number of active clients.
func (r *ClientRepository) CountActive() (count int) {
	activeClients, _ := r.GetAllActiveClients()
	return len(activeClients)
}

// CountDisconnected returns a number of disconnected clients.
func (r *ClientRepository) CountDisconnected() (int, error) {
	availableClients, err := r.GetAllClients()
	if err != nil {
		return 0, err
	}

	var n int
	// uses copy of clients array returned by GetAllClients
	for _, client := range availableClients {
		if !client.IsConnected() {
			n++
		}
	}
	return n, nil
}

// GetByID returns non-obsolete active or disconnected client by a given id.
func (r *ClientRepository) GetByID(id string) (*Client, error) {
	client, err := r.clientStore.Get(context.Background(), id, r.log())
	if err != nil {
		return nil, err
	}

	if client != nil && client.Obsolete(r.GetKeepDisconnectedClients()) {
		return nil, nil
	}
	return client, nil
}

// GetActiveByID returns an active client by a given id.
func (r *ClientRepository) GetActiveByID(id string) (*Client, error) {
	client, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	if client != nil && !client.IsConnected() {
		return nil, nil
	}
	return client, nil
}

// GetAllByClientAuthID @todo: make it consistent with others whether to return an error. In general it's just a cache, so should not return an err.
func (r *ClientRepository) GetAllByClientAuthID(clientAuthID string) (matchingClients []*Client) {
	matchingClients = make([]*Client, 0, DefaultInitialClientsArraySize)

	availableClients, _ := r.GetAllClients()
	// uses copy of clients array returned by GetAllClients
	for _, c := range availableClients {
		if c.GetClientAuthID() == clientAuthID {
			matchingClients = append(matchingClients, c)
		}
	}
	return matchingClients
}

// GetFilteredUserClients returns all non-obsolete active and disconnected clients that current user has access to, filtered by parameters
func (r *ClientRepository) GetFilteredUserClients(user User, filterOptions []query.FilterOption, groups []*cgroups.ClientGroup) (matchingClients []*CalculatedClient, err error) {
	matchingClients = make([]*CalculatedClient, 0, DefaultInitialClientsArraySize)

	clients, err := r.GetUserClients(user, groups)
	if err != nil {
		return nil, err
	}

	// uses copy of clients array returned by GetUserClients
	for _, client := range clients {
		calculatedClient := client.ToCalculated(groups)

		// we need to lock because MatchesFilters receives an interface and not a client,
		// therefore we lose our ability to lock.
		calculatedClient.flock.RLock()
		matches, err := query.MatchesFilters(calculatedClient, filterOptions)
		calculatedClient.flock.RUnlock()

		if err != nil {
			return matchingClients, err
		}

		if matches {
			matchingClients = append(matchingClients, calculatedClient)
		}

	}

	return matchingClients, nil
}

func insp(t time.Time) {
	_, filename, line, _ := runtime.Caller(1)
	log.Printf("[time to here] %s:%d %v", filename, line, time.Since(t))
}

func (r *ClientRepository) GetFilteredUserClientsM(user User, filterOptions []query.FilterOption, groups []*cgroups.ClientGroup) ([]CalculatedClient, error) {
	t := time.Now()
	insp(t)
	clientsP, err := r.GetUserClients(user, groups)

	insp(t)
	clients := make([]CalculatedClient, len(clientsP))
	for i, c := range clientsP {
		clients[i] = *c.ToCalculated(groups)
	}

	if len(filterOptions) == 0 {
		return clients, nil
	}

	insp(t)

	filter, err := filterer.CompileFromQueryListOptions[CalculatedClient](filterOptions)
	if err != nil {
		return nil, err
	}

	matchingClients := make([]CalculatedClient, 0, len(clientsP))
	for _, c := range clients {
		if filter.Run(c) {
			matchingClients = append(matchingClients, c)
		}
	}

	insp(t)

	return matchingClients, nil
}

type Matcher struct {
	prop int
	val  string
}

func (m Matcher) Matches(c Client) bool {
	v := reflect.ValueOf(c)

	return v.Field(m.prop).String() == m.val
}

func NewMatcher(smth any, key, value string) *Matcher {
	tt := formatter.BuildTranslationTable(smth)

	return &Matcher{
		prop: tt[key],
		val:  value,
	}
}

func (r *ClientRepository) GetKeepDisconnectedClients() (keep *time.Duration) {
	return r.keepDisconnectedClients
}

const DefaultInitialClientsArraySize = 64

// GetAllActiveClients returns a new client array that can be used without locks (assuming not shared)
func (r *ClientRepository) GetAllActiveClients() ([]*Client, error) {
	ctx := context.Background()

	matchingClients, err := r.clientStore.GetAll(ctx, r.log())
	if err != nil {
		return nil, err
	}

	simpleops.FilterSlice(matchingClients, func(client *Client) bool {
		return client.IsConnected()
	})

	return matchingClients, nil
}

// GetAllClients returns a new client array that can be used without locks (assuming not shared)
func (r *ClientRepository) GetAllClients() ([]*Client, error) {
	ctx := context.Background()

	return r.clientStore.GetNonObsoleteClients(ctx, r.log())
}

// GetUserClients return connected clients the user has access to either by user group or by client group.
// returns a new client array that can be used without locks (assuming not shared)
func (r *ClientRepository) GetUserClients(user User, clientGroups []*cgroups.ClientGroup) ([]*Client, error) {
	ctx := context.Background()
	userGroups := user.GetGroups()

	allClients, err := r.clientStore.GetAll(ctx, r.log())
	if err != nil {
		return nil, err
	}

	filtered := simpleops.FilterSlice(allClients, func(c *Client) bool {
		if !c.Obsolete(r.GetKeepDisconnectedClients()) {
			if user.IsAdmin() || c.HasAccessViaUserGroups(userGroups) || c.UserGroupHasAccessViaClientGroup(userGroups, clientGroups) {
				return true
			}
		}
		return false
	})

	return filtered, nil
}

func (r *ClientRepository) log() (l *logger.Logger) {
	return r.logger
}

type ClientQueryFn func(client *Client) (match bool)

func (r *ClientRepository) Close() error {
	return r.clientStore.Close()
}
