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
	// in-memory cache
	clients                 map[string]*Client
	mu                      sync.RWMutex
	KeepDisconnectedClients *time.Duration
	// storage
	provider ClientProvider
	logger   *logger.Logger
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

func NewClientRepositoryWithDB(initClients []*Client, keepDisconnectedClients *time.Duration, provider ClientProvider, logger *logger.Logger) *ClientRepository {
	clients := make(map[string]*Client)
	for i := range initClients {
		clients[initClients[i].ID] = initClients[i]
	}
	return &ClientRepository{
		clients:                 clients,
		KeepDisconnectedClients: keepDisconnectedClients,
		provider:                provider,
		logger:                  logger,
	}
}

func InitClientRepository(
	ctx context.Context,
	db *sqlx.DB,
	keepDisconnectedClients *time.Duration,
	logger *logger.Logger,
) (*ClientRepository, error) {
	provider := newSqliteProvider(db, keepDisconnectedClients)
	initClients, err := GetInitState(ctx, provider)
	if err != nil {
		return nil, err
	}

	return NewClientRepositoryWithDB(initClients, keepDisconnectedClients, provider, logger), nil
}

func (s *ClientRepository) Save(client *Client) error {
	if s.provider != nil {
		err := s.provider.Save(context.Background(), client)
		if err != nil {
			return fmt.Errorf("failed to save a client: %w", err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client.ID] = client
	return nil
}

func (s *ClientRepository) Delete(client *Client) error {
	if s.provider != nil {
		err := s.provider.Delete(context.Background(), client.ID)
		if err != nil {
			return fmt.Errorf("failed to delete a client: %w", err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, client.ID)
	return nil
}

func (s *ClientRepository) GetActiveByTags(tags []string, operator string) (matchingClients []*Client, err error) {
	activeClients := s.GetAllActive()
	if strings.ToUpper(operator) == "AND" {
		matchingClients = findMatchingANDClients(activeClients, tags)
	} else {
		matchingClients = findMatchingORClients(activeClients, tags)
	}

	return matchingClients, nil
}

func findMatchingANDClients(activeClients []*Client, tags []string) (matchingClients []*Client) {
	matchingClients = make([]*Client, 0, 64)
	for _, cl := range activeClients {
		clientTags := cl.Tags
		foundCount := 0
		for _, tag := range tags {
			for _, clTag := range clientTags {
				if tag == clTag {
					foundCount++
					continue
				}
			}
		}
		if foundCount == len(tags) {
			matchingClients = append(matchingClients, cl)
		}
	}
	return matchingClients
}

func findMatchingORClients(activeClients []*Client, tags []string) (matchingClients []*Client) {
	matchingClients = make([]*Client, 0, 64)
	for _, cl := range activeClients {
		clientTags := cl.Tags
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
func (s *ClientRepository) DeleteObsolete() ([]*Client, error) {
	if s.provider != nil {
		err := s.provider.DeleteObsolete(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to delete obsolete clients: %w", err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	var deleted []*Client
	for _, client := range s.clients {
		if client.Obsolete(s.KeepDisconnectedClients) {
			delete(s.clients, client.ID)
			deleted = append(deleted, client)
		}
	}
	return deleted, nil
}

// Count returns a number of non-obsolete active and disconnected clients.
func (s *ClientRepository) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clients, err := s.getNonObsolete()
	return len(clients), err
}

// CountActive returns a number of active clients.
func (s *ClientRepository) CountActive() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.GetAllActive()), nil
}

// CountDisconnected returns a number of disconnected clients.
func (s *ClientRepository) CountDisconnected() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all, err := s.getNonObsolete()
	if err != nil {
		return 0, err
	}

	var n int
	for _, cur := range all {
		if cur.DisconnectedAt != nil {
			n++
		}
	}
	return n, nil
}

// GetByID returns non-obsolete active or disconnected client by a given id.
func (s *ClientRepository) GetByID(id string) (*Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client := s.clients[id]
	if client != nil && client.Obsolete(s.KeepDisconnectedClients) {
		return nil, nil
	}
	return client, nil
}

// GetActiveByID returns an active client by a given id.
func (s *ClientRepository) GetActiveByID(id string) (*Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client := s.clients[id]
	if client != nil && client.DisconnectedAt != nil {
		return nil, nil
	}
	return client, nil
}

// TODO(m-terel): make it consistent with others whether to return an error. In general it's just a cache, so should not return an err.
func (s *ClientRepository) GetAllByClientAuthID(clientAuthID string) []*Client {
	all, _ := s.GetAll()
	var res []*Client
	for _, v := range all {
		if v.ClientAuthID == clientAuthID {
			res = append(res, v)
		}
	}
	return res
}

// GetAll returns all non-obsolete active and disconnected client clients.
func (s *ClientRepository) GetAll() ([]*Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getNonObsolete()
}

// GetUserClients returns all non-obsolete active and disconnected clients that current user has access to
func (s *ClientRepository) GetUserClients(user User) ([]*Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getNonObsoleteByUser(user)
}

// GetFilteredUserClients returns all non-obsolete active and disconnected clients that current user has access to, filtered by parameters
func (s *ClientRepository) GetFilteredUserClients(user User, filterOptions []query.FilterOption, groups []*cgroups.ClientGroup) ([]*CalculatedClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients, err := s.getNonObsoleteByUser(user)
	if err != nil {
		return nil, err
	}

	result := make([]*CalculatedClient, 0, len(clients))
	for _, client := range clients {
		calculatedClient := client.ToCalculated(groups)

		matches, err := query.MatchesFilters(calculatedClient, filterOptions)
		if err != nil {
			return result, err
		}

		if matches {
			result = append(result, calculatedClient)
		}
	}

	return result, nil
}

func (s *ClientRepository) GetAllActive() []*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Client
	for _, client := range s.clients {
		if client.DisconnectedAt == nil {
			result = append(result, client)
		}
	}
	return result
}

func (s *ClientRepository) getNonObsolete() ([]*Client, error) {
	result := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		if !client.Obsolete(s.KeepDisconnectedClients) {
			result = append(result, client)
		}
	}
	return result, nil
}

func (s *ClientRepository) getNonObsoleteByUser(user User) ([]*Client, error) {
	isAdmin := user.IsAdmin()
	result := make([]*Client, 0, len(s.clients))
	for _, client := range s.clients {
		if client.Obsolete(s.KeepDisconnectedClients) {
			continue
		}

		if !isAdmin && !client.HasAccess(user.GetGroups()) {
			continue
		}

		result = append(result, client)
	}
	return result, nil
}
