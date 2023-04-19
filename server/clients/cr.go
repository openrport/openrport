package clients

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/realvnc-labs/rport/server/cgroups"
	"github.com/realvnc-labs/rport/share/formatter"
	"github.com/realvnc-labs/rport/share/logger"
	"github.com/realvnc-labs/rport/share/query"
)

type ClientRepository struct {
	// in-memory state
	clientState map[string]*Client
	// db based store
	clientStore ClientStore

	keepDisconnectedClients *time.Duration

	logger *logger.Logger

	mu sync.RWMutex
}

type User interface {
	IsAdmin() bool
	GetGroups() []string
}

// NewClientRepository returns a new thread-safe in-memory cache to store client connections populated with given clients if any.
// keepDisconnectedClients is a duration to keep disconnected clients. If a client was disconnected longer than a given
// duration it will be treated as obsolete.
func NewClientRepository(initClients []*Client, keepDisconnectedClients *time.Duration, logger *logger.Logger) *ClientRepository {
	return NewClientRepositoryWithDB(initClients, keepDisconnectedClients, nil, logger)
}

// NewClientRepositoryWithDB @todo: used for test setup in two separate packages. need to review use as part of the test code refactoring.
func NewClientRepositoryWithDB(initialClients []*Client, keepDisconnectedClients *time.Duration, store ClientStore, logger *logger.Logger) *ClientRepository {
	clients := make(map[string]*Client)
	for i := range initialClients {
		newClientID := initialClients[i].GetID()
		clients[newClientID] = initialClients[i]
	}

	return &ClientRepository{
		clientState:             clients,
		clientStore:             store,
		logger:                  logger,
		keepDisconnectedClients: keepDisconnectedClients,
	}
}

func InitClientRepository(
	ctx context.Context,
	db *sqlx.DB,
	keepDisconnectedClients *time.Duration,
	logger *logger.Logger,
) (*ClientRepository, error) {
	provider := newSqliteProvider(db, keepDisconnectedClients)
	initialClients, err := LoadInitialClients(ctx, provider, logger)
	if err != nil {
		return nil, err
	}

	return NewClientRepositoryWithDB(initialClients, keepDisconnectedClients, provider, logger), nil
}

func (r *ClientRepository) Save(client *Client) error {
	ts := time.Now()

	store := r.getStore()

	if store != nil {
		err := store.Save(context.Background(), client)
		if err != nil {
			return fmt.Errorf("failed to save client: %w", err)
		}
	}

	r.updateClient(client)

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

	store := r.getStore()

	if store != nil {
		err := store.Delete(context.Background(), clientID, client.Log())
		if err != nil {
			return fmt.Errorf("failed to delete a client: %w", err)
		}
	}

	r.removeClient(clientID)
	return nil
}

func (r *ClientRepository) GetClientsByTag(tags []string, operator string, allowDisconnected bool) (matchingClients []*Client, err error) {
	var availableClients []*Client
	if allowDisconnected {
		availableClients = r.GetAllClients()
	} else {
		availableClients = r.GetAllActiveClients()
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
// the various GetXXXClient fns will return new client arrays. please use those fns to get a
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
	r.log().Debugf("deleting obsolete clients")
	store := r.getStore()

	if store != nil {
		err := store.DeleteObsolete(context.Background(), r.log())
		if err != nil {
			return nil, fmt.Errorf("failed to delete obsolete clients: %w", err)
		}
	}

	clientsToDelete := r.queryClients(func(c *Client) (match bool) {
		return c.Obsolete(r.GetKeepDisconnectedClients())
	})

	for _, client := range clientsToDelete {
		clientID := client.GetID()
		r.log().Debugf("deleting obsolete client: %s status=%s", clientID, FormatConnectionState(client))
		r.removeClient(clientID)
	}

	return clientsToDelete, nil
}

// Count returns a number of non-obsolete active and disconnected clients.
func (r *ClientRepository) Count() int {
	availableClients := r.getNonObsoleteClients()
	return len(availableClients)
}

// CountActive returns a number of active clients.
func (r *ClientRepository) CountActive() (count int) {
	activeClients := r.GetAllActiveClients()
	return len(activeClients)
}

// CountDisconnected returns a number of disconnected clients.
func (r *ClientRepository) CountDisconnected() (int, error) {
	availableClients := r.getNonObsoleteClients()

	var n int
	// uses copy of clients array returned by getNonObsoleteClients
	for _, client := range availableClients {
		if !client.IsConnected() {
			n++
		}
	}
	return n, nil
}

// GetByID returns non-obsolete active or disconnected client by a given id.
func (r *ClientRepository) GetByID(id string) (*Client, error) {
	client := r.getClient(id)

	if client != nil && client.Obsolete(r.GetKeepDisconnectedClients()) {
		return nil, nil
	}
	return client, nil
}

// GetActiveByID returns an active client by a given id.
func (r *ClientRepository) GetActiveByID(id string) (*Client, error) {
	client := r.getClient(id)

	if client != nil && !client.IsConnected() {
		return nil, nil
	}
	return client, nil
}

// GetAllByClientAuthID @todo: make it consistent with others whether to return an error. In general it's just a cache, so should not return an err.
func (r *ClientRepository) GetAllByClientAuthID(clientAuthID string) (matchingClients []*Client) {
	matchingClients = make([]*Client, 0, DefaultInitialClientsArraySize)

	availableClients := r.GetAllClients()
	// uses copy of clients array returned by GetAllClients
	for _, c := range availableClients {
		if c.GetClientAuthID() == clientAuthID {
			matchingClients = append(matchingClients, c)
		}
	}
	return matchingClients
}

// GetAll returns all non-obsolete active and disconnected client clients.
func (r *ClientRepository) GetAllClients() []*Client {
	availableClients := r.getNonObsoleteClients()
	return availableClients
}

// GetUserClients returns all non-obsolete active and disconnected clients that current user has access to
func (r *ClientRepository) GetUserClients(user User, groups []*cgroups.ClientGroup) []*Client {
	return r.getNonObsoleteClientsByUser(user, groups)
}

// GetFilteredUserClients returns all non-obsolete active and disconnected clients that current user has access to, filtered by parameters
func (r *ClientRepository) GetFilteredUserClients(user User, filterOptions []query.FilterOption, groups []*cgroups.ClientGroup) (matchingClients []*CalculatedClient, err error) {
	matchingClients = make([]*CalculatedClient, 0, DefaultInitialClientsArraySize)

	clients := r.getNonObsoleteClientsByUser(user, groups)

	// uses copy of clients array returned by getNonObsoleteClientsByUser
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

// GetFilteredUserClients returns all non-obsolete active and disconnected clients that current user has access to, filtered by parameters
func (r *ClientRepository) GetFilteredUserClientsU(user User, filterOptions []query.FilterOption, groups []*cgroups.ClientGroup) (matchingClients []*CalculatedClient, err error) {
	t := time.Now()
	matchingClients = make([]*CalculatedClient, 0, DefaultInitialClientsArraySize)

	insp(t)
	clients := r.getNonObsoleteClientsByUser(user, groups)
	insp(t)

	// uses copy of clients array returned by getNonObsoleteClientsByUser
	for _, client := range clients {
		calculatedClient := client.ToCalculatedU(groups)

		// we need to lock because MatchesFilters receives an interface and not a client,
		// therefore we lose our ability to lock.

		matches, err := true, error(nil)
		//matches, err := query.MatchesFilters(calculatedClient, filterOptions)

		if err != nil {
			return matchingClients, err
		}

		if matches {
			matchingClients = append(matchingClients, calculatedClient)
		}

	}
	insp(t)

	return matchingClients, nil
}

func (r *ClientRepository) GetFilteredUserClientsM(user User, filterOptions []query.FilterOption, groups []*cgroups.ClientGroup) ([]Client, error) {
	t := time.Now()
	insp(t)
	clientsP := r.getNonObsoleteClientsByUser(user, groups)

	insp(t)
	clients := make([]Client, len(clientsP))
	for i, c := range clientsP {
		clients[i] = *c
	}

	if len(filterOptions) == 0 {
		return clients, nil
	}

	insp(t)

	matchingClients := make([]Client, 0, len(clientsP))
	matcher := NewMatcher(clientsP[0], filterOptions[0].Column[0], filterOptions[0].Values[0])

	for _, c := range clients {
		if matcher.Matches(c) {
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

func (r *ClientRepository) getStore() (store ClientStore) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clientStore
}

func (r *ClientRepository) GetKeepDisconnectedClients() (keep *time.Duration) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.keepDisconnectedClients
}

const DefaultInitialClientsArraySize = 64

// GetAllActiveClients returns a new client array that can be used without locks (assuming not shared)
func (r *ClientRepository) GetAllActiveClients() (matchingClients []*Client) {
	matchingClients = r.queryClients(func(c *Client) (match bool) {
		return c.IsConnected()
	})
	return matchingClients
}

// getNonObsoleteClients returns a new client array that can be used without locks (assuming not shared)
func (r *ClientRepository) getNonObsoleteClients() (matchingClients []*Client) {
	matchingClients = r.queryClients(func(c *Client) (match bool) {
		return !c.Obsolete(r.GetKeepDisconnectedClients())
	})
	return matchingClients
}

// getNonObsoleteByUser return connected clients the user has access to either by user group or by client group.
// returns a new client array that can be used without locks (assuming not shared)
func (r *ClientRepository) getNonObsoleteClientsByUser(user User, clientGroups []*cgroups.ClientGroup) (matchingClients []*Client) {
	userGroups := user.GetGroups()

	matchingClients = r.queryClients(func(c *Client) (match bool) {
		if !c.Obsolete(r.GetKeepDisconnectedClients()) {
			if user.IsAdmin() || c.HasAccessViaUserGroups(userGroups) || c.UserGroupHasAccessViaClientGroup(userGroups, clientGroups) {
				return true
			}
		}
		return false
	})

	return matchingClients
}

func (r *ClientRepository) log() (l *logger.Logger) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.logger
}

type ClientQueryFn func(client *Client) (match bool)

// some notes on thread safe looping over a map in the post below
// https://stackoverflow.com/questions/40442846/concurrent-access-to-maps-with-range-in-go

func (r *ClientRepository) queryClients(queryFn ClientQueryFn) (matchingClients []*Client) {
	matchingClients = make([]*Client, 0, DefaultInitialClientsArraySize)

	clients := r.getClients()

	r.mu.RLock()
	for _, c := range clients {
		r.mu.RUnlock()
		match := queryFn(c)
		if match {
			matchingClients = append(matchingClients, c)
		}
		r.mu.RLock()
	}
	r.mu.RUnlock()

	return matchingClients
}

func (r *ClientRepository) getClient(clientID string) (client *Client) {
	r.mu.RLock()
	client = r.clientState[clientID]
	r.mu.RUnlock()
	return client
}

func (r *ClientRepository) getClients() (clients map[string]*Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clientState
}

func (r *ClientRepository) updateClient(client *Client) {
	clientID := client.GetID()

	r.mu.Lock()
	r.clientState[clientID] = client
	r.mu.Unlock()
}

func (r *ClientRepository) removeClient(clientID string) {
	r.mu.Lock()
	delete(r.clientState, clientID)
	r.mu.Unlock()
}
