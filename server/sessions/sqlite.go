package sessions

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/db/migration/client_sessions"
)

type ClientSessionProvider interface {
	GetAll(ctx context.Context) ([]*ClientSession, error)
	Save(ctx context.Context, session *ClientSession) error
	DeleteObsolete(ctx context.Context) error
	Close() error
}

type SqliteProvider struct {
	db              *sqlx.DB
	keepLostClients time.Duration
}

func NewSqliteProvider(dbPath string, keepLostClients time.Duration) (*SqliteProvider, error) {
	db, err := sqlx.Connect("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %v", err)
	}

	s := bindata.Resource(client_sessions.AssetNames(),
		func(name string) ([]byte, error) {
			return client_sessions.Asset(name)
		})
	sourceDriver, err := bindata.WithInstance(s)
	if err != nil {
		return nil, fmt.Errorf("failed to init DB source driver: %v", err)
	}

	dbDriver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to init DB migration driver: %v", err)
	}

	m, err := migrate.NewWithInstance("go-bindata", sourceDriver, "client_sessions.db", dbDriver)
	if err != nil {
		return nil, fmt.Errorf("failed to init DB migration instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("failed to migarte DB to the latest version: %v", err)
	}

	return &SqliteProvider{db: db, keepLostClients: keepLostClients}, nil
}

func (p *SqliteProvider) GetAll(ctx context.Context) ([]*ClientSession, error) {
	var res []*sessionSqlite
	err := p.db.SelectContext(
		ctx,
		&res,
		"SELECT * FROM client_sessions WHERE disconnected IS NULL OR DATETIME(disconnected) >= DATETIME(?)",
		p.keepLostClientsStart(),
	)
	if err != nil {
		return nil, err
	}
	return convertSessionList(res), nil
}

func (p *SqliteProvider) get(ctx context.Context, id string) (*ClientSession, error) {
	res := &sessionSqlite{}
	err := p.db.GetContext(ctx, res, "SELECT * FROM client_sessions WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return res.convert(), nil
}

func (p *SqliteProvider) Save(ctx context.Context, session *ClientSession) error {
	_, err := p.db.NamedExecContext(
		ctx,
		"INSERT OR REPLACE INTO client_sessions (id, client_id, disconnected, details) VALUES (:id, :client_id, :disconnected, :details)",
		convertToSqlite(session),
	)
	return err
}

func (p *SqliteProvider) DeleteObsolete(ctx context.Context) error {
	_, err := p.db.ExecContext(
		ctx,
		"DELETE FROM client_sessions WHERE disconnected IS NOT NULL AND DATETIME(disconnected) < DATETIME(?)",
		p.keepLostClientsStart(),
	)
	return err
}

func (p *SqliteProvider) keepLostClientsStart() time.Time {
	return now().Add(-p.keepLostClients)
}

func convertToSqlite(v *ClientSession) *sessionSqlite {
	if v == nil {
		return nil
	}
	res := &sessionSqlite{
		ID:       v.ID,
		ClientID: v.ClientID,
		Details: &clientSessionDetails{
			Name:     v.Name,
			OS:       v.OS,
			OSArch:   v.OSArch,
			OSFamily: v.OSFamily,
			OSKernel: v.OSKernel,
			Hostname: v.Hostname,
			Version:  v.Version,
			Address:  v.Address,
			IPv4:     v.IPv4,
			IPv6:     v.IPv6,
			Tags:     v.Tags,
			Tunnels:  v.Tunnels,
		},
	}
	if v.Disconnected != nil {
		res.Disconnected = sql.NullTime{Time: *v.Disconnected, Valid: true}
	}
	return res
}

type sessionSqlite struct {
	ID           string                `db:"id"`
	ClientID     string                `db:"client_id"`
	Disconnected sql.NullTime          `db:"disconnected"` // Disconnected is a time when a client session was disconnected. If nil - it's connected.
	Details      *clientSessionDetails `db:"details"`
}

type clientSessionDetails struct {
	Name     string    `json:"name"`
	OS       string    `json:"os"`
	OSArch   string    `json:"os_arch"`
	OSFamily string    `json:"os_family"`
	OSKernel string    `json:"os_kernel"`
	Hostname string    `json:"hostname"`
	Version  string    `json:"version"`
	Address  string    `json:"address"`
	IPv4     []string  `json:"ipv4"`
	IPv6     []string  `json:"ipv6"`
	Tags     []string  `json:"tags"`
	Tunnels  []*Tunnel `json:"tunnels"`
}

func (d *clientSessionDetails) Scan(value interface{}) error {
	if d == nil {
		return errors.New("'details' cannot be nil")
	}
	valueStr, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected to have string, got %T", value)
	}
	err := json.Unmarshal([]byte(valueStr), d)
	if err != nil {
		return fmt.Errorf("failed to decode 'details' field: %v", err)
	}
	return nil
}

func (d *clientSessionDetails) Value() (driver.Value, error) {
	if d == nil {
		return nil, errors.New("'details' cannot be nil")
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("failed to encode 'details' field: %v", err)
	}
	return string(b), nil
}

func (s *sessionSqlite) convert() *ClientSession {
	d := s.Details
	res := &ClientSession{
		ID:       s.ID,
		ClientID: s.ClientID,
		Name:     d.Name,
		OS:       d.OS,
		OSArch:   d.OSArch,
		OSFamily: d.OSFamily,
		OSKernel: d.OSKernel,
		Hostname: d.Hostname,
		IPv4:     d.IPv4,
		IPv6:     d.IPv6,
		Tags:     d.Tags,
		Version:  d.Version,
		Address:  d.Address,
		Tunnels:  d.Tunnels,
	}
	if s.Disconnected.Valid {
		res.Disconnected = &s.Disconnected.Time
	}
	return res
}

func (p *SqliteProvider) Close() error {
	return p.db.Close()
}

func convertSessionList(list []*sessionSqlite) []*ClientSession {
	res := make([]*ClientSession, 0, len(list))
	for _, cur := range list {
		res = append(res, cur.convert())
	}
	return res
}
