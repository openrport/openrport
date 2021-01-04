package hgroups

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/db/migration/host_groups"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
)

type HostGroupProvider interface {
	Get(ctx context.Context, id string) (*HostGroup, error)
	GetAll(ctx context.Context) ([]*HostGroup, error)
	Create(ctx context.Context, group *HostGroup) error
	Update(ctx context.Context, group *HostGroup) error
	Delete(ctx context.Context, id string) error
	Close() error
}

type SqliteProvider struct {
	db *sqlx.DB
}

func NewSqliteProvider(dbPath string) (*SqliteProvider, error) {
	db, err := sqlite.New(dbPath, host_groups.AssetNames(), host_groups.Asset)
	if err != nil {
		return nil, fmt.Errorf("failed to create host_groups DB instance: %v", err)
	}
	return &SqliteProvider{db: db}, nil
}

func (p *SqliteProvider) GetAll(ctx context.Context) ([]*HostGroup, error) {
	var res []*HostGroup
	err := p.db.SelectContext(
		ctx,
		&res,
		"SELECT * FROM host_groups ORDER BY id",
	)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (p *SqliteProvider) Get(ctx context.Context, id string) (*HostGroup, error) {
	res := &HostGroup{}
	err := p.db.GetContext(ctx, res, "SELECT * FROM host_groups WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (p *SqliteProvider) Create(ctx context.Context, group *HostGroup) error {
	_, err := p.db.NamedExecContext(
		ctx,
		"INSERT INTO host_groups (id, description, params) VALUES (:id, :description, :params)",
		group,
	)
	return err
}

func (p *SqliteProvider) Update(ctx context.Context, group *HostGroup) error {
	_, err := p.db.NamedExecContext(
		ctx,
		"INSERT OR REPLACE INTO host_groups (id, description, params) VALUES (:id, :description, :params)",
		group,
	)
	return err
}

func (p *SqliteProvider) Delete(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM host_groups WHERE id = ?", id)
	return err
}

func (p *SqliteProvider) Close() error {
	return p.db.Close()
}
