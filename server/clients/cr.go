package clients

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/server/cgroups"
	"github.com/cloudradar-monitoring/rport/share/logger"
	"github.com/cloudradar-monitoring/rport/share/query"
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
		availableClients, _ = r.GetAllActiveClients()
	}
	if strings.EqualFold(operator, "AND") {
		matchingClients = findMatchingANDClients(availableClients, tags)
	} else {
		matchingClients = findMatchingORClients(availableClients, tags)
	}

	return matchingClients, nil
}

// this fn doesn't lock the availableClients. please make sure not to use the main clients array.
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

	var deleted []*Client
	r.mu.RLock()
	for _, client := range r.getClients() {
		r.mu.RUnlock()
		clientID := client.GetID()

		if client.Obsolete(r.GetKeepDisconnectedClients()) {
			r.log().Debugf("deleting obsolete client: %s status=%s", clientID, FormatConnectionState(client))

			r.removeClient(clientID)

			deleted = append(deleted, client)
		}
		r.mu.RLock()
	}
	r.mu.RUnlock()
	return deleted, nil
}

// Count returns a number of non-obsolete active and disconnected clients.
func (r *ClientRepository) Count() int {
	_, count := r.getNonObsoleteClients()
	return count
}

// CountActive returns a number of active clients.
func (r *ClientRepository) CountActive() (count int) {
	_, count = r.GetAllActiveClients()
	return count
}

// CountDisconnected returns a number of disconnected clients.
func (r *ClientRepository) CountDisconnected() (int, error) {
	availableClients, _ := r.getNonObsoleteClients()

	var n int
	// uses copy of clients array returned by getNonObsoleteClients
	for _, cur := range availableClients {
		if cur.GetDisconnectedAt() != nil {
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

	if client != nil && client.GetDisconnectedAt() != nil {
		return nil, nil
	}
	return client, nil
}

// GetAllByClientAuthID @todo: make it consistent with others whether to return an error. In general it's just a cache, so should not return an err.
func (r *ClientRepository) GetAllByClientAuthID(clientAuthID string) []*Client {
	availableClients := r.GetAllClients()
	var matchingClients []*Client
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
	availableClients, _ := r.getNonObsoleteClients()
	return availableClients
}

// GetUserClients returns all non-obsolete active and disconnected clients that current user has access to
func (r *ClientRepository) GetUserClients(user User, groups []*cgroups.ClientGroup) ([]*Client, error) {
	return r.getNonObsoleteClientsByUser(user, groups)
}

// GetFilteredUserClients returns all non-obsolete active and disconnected clients that current user has access to, filtered by parameters
func (r *ClientRepository) GetFilteredUserClients(user User, filterOptions []query.FilterOption, groups []*cgroups.ClientGroup) ([]*CalculatedClient, error) {
	clients, err := r.getNonObsoleteClientsByUser(user, groups)
	if err != nil {
		return nil, err
	}

	matchingClients := make([]*CalculatedClient, 0, len(clients))

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

// GetAllActiveClients returns a new client array that can be used without locks (assuming not shared)
func (r *ClientRepository) GetAllActiveClients() (matchingClients []*Client, count int) {
	count = 0
	clients := r.getClients()

	r.mu.RLock()
	for _, client := range clients {
		r.mu.RUnlock()
		if client.GetDisconnectedAt() == nil {
			matchingClients = append(matchingClients, client)
			count++
		}
		r.mu.RLock()
	}
	r.mu.RUnlock()

	return matchingClients, count
}

// getNonObsoleteClients returns a new client array that can be used without locks (assuming not shared)
func (r *ClientRepository) getNonObsoleteClients() (matchingClients []*Client, count int) {
	count = 0
	clients := r.getClients()

	r.mu.RLock()
	matchingClients = make([]*Client, 0, len(clients))
	for _, client := range clients {
		r.mu.RUnlock()
		if !client.Obsolete(r.GetKeepDisconnectedClients()) {
			matchingClients = append(matchingClients, client)
			count++
		}
		r.mu.RLock()
	}
	r.mu.RUnlock()
	return matchingClients, count
}

// getNonObsoleteByUser return connected clients the user has access to either by user group or by client group.
// returns a new client array that can be used without locks (assuming not shared)
func (r *ClientRepository) getNonObsoleteClientsByUser(user User, clientGroups []*cgroups.ClientGroup) ([]*Client, error) {
	clients := r.getClients()
	userGroups := user.GetGroups()

	r.mu.RLock()
	matchingClients := make([]*Client, 0, len(clients))
	for _, client := range clients {
		r.mu.RUnlock()
		if !client.Obsolete(r.GetKeepDisconnectedClients()) {
			if user.IsAdmin() || client.HasAccessViaUserGroups(userGroups) || client.UserGroupHasAccessViaClientGroup(userGroups, clientGroups) {
				matchingClients = append(matchingClients, client)
			}
		}
		r.mu.RLock()
	}
	r.mu.RUnlock()
	return matchingClients, nil
}

func (r *ClientRepository) getStore() (store ClientStore) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clientStore
}

func (r *ClientRepository) log() (l *logger.Logger) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.logger
}

// getClients returns the primary map of clients. accessing the return map must use locks.
func (r *ClientRepository) getClients() (clients map[string]*Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clientState
}

func (r *ClientRepository) getClient(clientID string) (client *Client) {
	r.mu.RLock()
	client = r.clientState[clientID]
	r.mu.RUnlock()
	return client
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

func (r *ClientRepository) GetKeepDisconnectedClients() (keep *time.Duration) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.keepDisconnectedClients
}
