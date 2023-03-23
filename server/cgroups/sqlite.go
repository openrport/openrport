package cgroups

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/realvnc-labs/rport/share/query"
)

type ClientGroupProvider interface {
	Get(ctx context.Context, id string) (*ClientGroup, error)
	GetAll(ctx context.Context) ([]*ClientGroup, error)
	List(ctx context.Context, options *query.ListOptions) ([]*ClientGroup, error)
	Create(ctx context.Context, group *ClientGroup) error
	Update(ctx context.Context, group *ClientGroup) error
	Delete(ctx context.Context, id string) error
	Close() error
}

type SqliteProvider struct {
	db        *sqlx.DB
	converter *query.SQLConverter
}

func NewSqliteProvider(db *sqlx.DB) (*SqliteProvider, error) {
	return &SqliteProvider{
		db:        db,
		converter: query.NewSQLConverter(db.DriverName()),
	}, nil
}

func (p *SqliteProvider) GetAll(ctx context.Context) ([]*ClientGroup, error) {
	var res []*ClientGroup
	err := p.db.SelectContext(
		ctx,
		&res,
		"SELECT * FROM client_groups ORDER BY id COLLATE NOCASE",
	)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (p *SqliteProvider) List(ctx context.Context, options *query.ListOptions) ([]*ClientGroup, error) {
	query := "SELECT * FROM client_groups"
	q, params := p.converter.ConvertListOptionsToQuery(options, query)

	var res []*ClientGroup
	err := p.db.SelectContext(
		ctx,
		&res,
		q,
		params...,
	)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (p *SqliteProvider) Get(ctx context.Context, id string) (*ClientGroup, error) {
	res := &ClientGroup{}
	err := p.db.GetContext(ctx, res, "SELECT * FROM client_groups WHERE id = ?", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return res, nil
}

func (p *SqliteProvider) Create(ctx context.Context, group *ClientGroup) error {
	_, err := p.db.NamedExecContext(
		ctx,
		"INSERT INTO client_groups (id, description, params, allowed_user_groups) VALUES (:id, :description, :params, :allowed_user_groups)",
		group,
	)
	return err
}

func (p *SqliteProvider) Update(ctx context.Context, group *ClientGroup) error {
	_, err := p.db.NamedExecContext(
		ctx,
		"INSERT OR REPLACE INTO client_groups (id, description, params, allowed_user_groups) VALUES (:id, :description, :params, :allowed_user_groups)",
		group,
	)
	return err
}

func (p *SqliteProvider) Delete(ctx context.Context, id string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM client_groups WHERE id = ?", id)
	return err
}

func (p *SqliteProvider) Close() error {
	return p.db.Close()
}
