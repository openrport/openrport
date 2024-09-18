package clients

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/openrport/openrport/db/sqlite"
	"github.com/openrport/openrport/server/clients/clientdata"
	"github.com/openrport/openrport/server/clients/clienttunnel"
	chshare "github.com/openrport/openrport/share/clientconfig"
	"github.com/openrport/openrport/share/logger"
	"github.com/openrport/openrport/share/models"
)

type ClientStore interface {
	GetAll(ctx context.Context, l *logger.Logger) ([]*clientdata.Client, error)
	Save(ctx context.Context, client *clientdata.Client) error
	DeleteObsolete(ctx context.Context, l *logger.Logger) error
	Delete(ctx context.Context, id string, l *logger.Logger) error
	Close() error
}

type SqliteProvider struct {
	db                      *sqlx.DB
	keepDisconnectedClients *time.Duration
}

func newSqliteProvider(db *sqlx.DB, keepDisconnectedClients *time.Duration) *SqliteProvider {
	return &SqliteProvider{db: db, keepDisconnectedClients: keepDisconnectedClients}
}

func (p *SqliteProvider) GetAll(ctx context.Context, l *logger.Logger) ([]*clientdata.Client, error) {
	var res []*clientSqlite
	err := p.db.SelectContext(
		ctx,
		&res,
		"SELECT * FROM clients WHERE disconnected_at IS NULL OR DATETIME(disconnected_at) >= DATETIME(?) OR ?",
		p.keepDisconnectedClientsStart(),
		p.keepDisconnectedClients == nil,
	)
	if err != nil {
		return nil, err
	}
	return convertClientList(res, l), nil
}

// test only
func (p *SqliteProvider) get(ctx context.Context, id string, l *logger.Logger) (*clientdata.Client, error) {
	res := &clientSqlite{}
	err := p.db.GetContext(ctx, res, "SELECT * FROM clients WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return res.convert(l), nil
}

func (p *SqliteProvider) Save(ctx context.Context, client *clientdata.Client) error {

	clientForSQL := convertToSqlite(client)

	_, err := sqlite.WithRetryWhenBusy(func() (result sql.Result, err error) {

		_, err = p.db.NamedExecContext(
			ctx,
			"INSERT OR REPLACE INTO clients (id, client_auth_id, disconnected_at, details) VALUES (:id, :client_auth_id, :disconnected_at, :details)",
			clientForSQL,
		)

		return nil, err
	}, "save", client.Log())

	return err
}

func (p *SqliteProvider) DeleteObsolete(ctx context.Context, l *logger.Logger) error {
	_, err := sqlite.WithRetryWhenBusy(func() (result sql.Result, err error) {

		_, err = p.db.ExecContext(
			ctx,
			"DELETE FROM clients WHERE disconnected_at IS NOT NULL AND DATETIME(disconnected_at) < DATETIME(?) AND ?",
			p.keepDisconnectedClientsStart(),
			p.keepDisconnectedClients != nil,
		)

		return nil, err
	}, "delete obsolete", l)

	return err
}

func (p *SqliteProvider) Delete(ctx context.Context, id string, l *logger.Logger) error {
	_, err := sqlite.WithRetryWhenBusy(func() (result sql.Result, err error) {

		_, err = p.db.ExecContext(ctx, "DELETE FROM clients WHERE id = ?", id)

		return nil, err
	}, "delete", l)

	return err
}

func (p *SqliteProvider) Close() error {
	return p.db.Close()
}

func (p *SqliteProvider) keepDisconnectedClientsStart() time.Time {
	t := clientdata.Now()
	if p.keepDisconnectedClients != nil {
		t = t.Add(-*p.keepDisconnectedClients)
	}
	return t
}

func convertToSqlite(c *clientdata.Client) (res *clientSqlite) {
	if c == nil {
		return nil
	}

	c.GetLock().RLock()
	res = &clientSqlite{
		ID:           c.ID,
		ClientAuthID: c.ClientAuthID,
		Details: &clientDetails{
			Name:                   c.Name,
			OS:                     c.OS,
			OSArch:                 c.OSArch,
			OSFamily:               c.OSFamily,
			OSKernel:               c.OSKernel,
			Hostname:               c.Hostname,
			Version:                c.Version,
			Address:                c.Address,
			OSFullName:             c.OSFullName,
			OSVersion:              c.OSVersion,
			OSVirtualizationSystem: c.OSVirtualizationSystem,
			OSVirtualizationRole:   c.OSVirtualizationRole,
			CPUFamily:              c.CPUFamily,
			CPUModel:               c.CPUModel,
			CPUModelName:           c.CPUModelName,
			CPUVendor:              c.CPUVendor,
			NumCPUs:                c.NumCPUs,
			MemoryTotal:            c.MemoryTotal,
			Timezone:               c.Timezone,
			IPv4:                   c.IPv4,
			IPv6:                   c.IPv6,
			Tags:                   c.Tags,
			Labels:                 c.Labels,
			Tunnels:                c.Tunnels,
			AllowedUserGroups:      c.AllowedUserGroups,
			UpdatesStatus:          c.UpdatesStatus,
			Inventory:              c.Inventory,
			IPAddresses:            c.IPAddresses,
			ClientConfig:           c.ClientConfiguration,
		},
	}
	c.GetLock().RUnlock()
	if !c.IsConnected() {
		res.DisconnectedAt = sql.NullTime{Time: c.GetDisconnectedAtValue(), Valid: true}
	}

	return res
}

type clientSqlite struct {
	ID             string         `db:"id"`
	ClientAuthID   string         `db:"client_auth_id"`
	DisconnectedAt sql.NullTime   `db:"disconnected_at"` // DisconnectedAt is a time when a client was disconnected. If nil - it's connected.
	Details        *clientDetails `db:"details"`
}

type clientDetails struct {
	NumCPUs                int                    `json:"num_cpus"`
	MemoryTotal            uint64                 `json:"mem_total"`
	Name                   string                 `json:"name"`
	OS                     string                 `json:"os"`
	OSArch                 string                 `json:"os_arch"`
	OSFamily               string                 `json:"os_family"`
	OSKernel               string                 `json:"os_kernel"`
	OSFullName             string                 `json:"os_full_name"`
	OSVersion              string                 `json:"os_version"`
	OSVirtualizationSystem string                 `json:"os_virtualization_system"`
	OSVirtualizationRole   string                 `json:"os_virtualization_role"`
	CPUFamily              string                 `json:"cpu_family"`
	CPUModel               string                 `json:"cpu_model"`
	CPUModelName           string                 `json:"cpu_model_name"`
	CPUVendor              string                 `json:"cpu_vendor"`
	Timezone               string                 `json:"timezone"`
	Hostname               string                 `json:"hostname"`
	Version                string                 `json:"version"`
	Address                string                 `json:"address"`
	IPv4                   []string               `json:"ipv4"`
	IPv6                   []string               `json:"ipv6"`
	Tags                   []string               `json:"tags"`
	Labels                 map[string]string      `json:"labels"`
	Tunnels                []*clienttunnel.Tunnel `json:"tunnels"`
	AllowedUserGroups      []string               `json:"allowed_user_groups"`
	UpdatesStatus          *models.UpdatesStatus  `json:"updates_status"`
	Inventory              *models.Inventory      `json:"inventory"`
	IPAddresses            *models.IPAddresses    `json:"ext_ip_addresses"`
	ClientConfig           *chshare.Config        `json:"client_configuration"`
}

func (d *clientDetails) Scan(value interface{}) error {
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

func (d *clientDetails) Value() (driver.Value, error) {
	if d == nil {
		return nil, errors.New("'details' cannot be nil")
	}
	b, err := json.Marshal(d)
	if err != nil {
		return nil, fmt.Errorf("failed to encode 'details' field: %v", err)
	}
	return string(b), nil
}

func (s *clientSqlite) convert(l *logger.Logger) (res *clientdata.Client) {
	d := s.Details
	res = &clientdata.Client{
		ID:                     s.ID,
		ClientAuthID:           s.ClientAuthID,
		Name:                   d.Name,
		OS:                     d.OS,
		OSArch:                 d.OSArch,
		OSFamily:               d.OSFamily,
		OSKernel:               d.OSKernel,
		Hostname:               d.Hostname,
		IPv4:                   d.IPv4,
		IPv6:                   d.IPv6,
		Tags:                   d.Tags,
		Labels:                 d.Labels,
		Version:                d.Version,
		Address:                d.Address,
		Tunnels:                d.Tunnels,
		OSFullName:             d.OSFullName,
		OSVersion:              d.OSVersion,
		OSVirtualizationSystem: d.OSVirtualizationSystem,
		OSVirtualizationRole:   d.OSVirtualizationRole,
		CPUFamily:              d.CPUFamily,
		CPUModel:               d.CPUModel,
		CPUModelName:           d.CPUModelName,
		CPUVendor:              d.CPUVendor,
		NumCPUs:                d.NumCPUs,
		MemoryTotal:            d.MemoryTotal,
		Timezone:               d.Timezone,
		AllowedUserGroups:      d.AllowedUserGroups,
		UpdatesStatus:          d.UpdatesStatus,
		Inventory:              d.Inventory,
		IPAddresses:            d.IPAddresses,
		ClientConfiguration:    d.ClientConfig,
		Logger:                 l,
	}
	if s.DisconnectedAt.Valid {
		res.SetDisconnectedAt(&s.DisconnectedAt.Time)
	}
	return res
}

func convertClientList(list []*clientSqlite, l *logger.Logger) []*clientdata.Client {
	res := make([]*clientdata.Client, 0, len(list))
	for _, cur := range list {
		res = append(res, cur.convert(l))
	}
	return res
}
