package storedtunnels

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/share/query"
)

type SQLiteProvider struct {
	db *sqlx.DB
}

func newSQLiteProvider(db *sqlx.DB) *SQLiteProvider {
	return &SQLiteProvider{
		db: db,
	}
}

func (p *SQLiteProvider) Insert(ctx context.Context, t *StoredTunnel) error {
	_, err := p.db.NamedExecContext(ctx,
		`INSERT INTO stored_tunnels (
			id,
			client_id,
			created_at,
			name,
			scheme,
			remote_ip,
			remote_port,
			acl
		) VALUES (
			:id,
			:client_id,
			:created_at,
			:name,
			:scheme,
			:remote_ip,
			:remote_port,
			:acl
		)`,
		t,
	)

	return err
}

func (p *SQLiteProvider) Update(ctx context.Context, t *StoredTunnel) error {
	_, err := p.db.NamedExecContext(ctx,
		`UPDATE stored_tunnels SET
			name = :name,
			scheme = :scheme,
			remote_ip = :remote_ip,
			remote_port = :remote_port,
			acl = :acl
		WHERE client_id = :client_id AND id = :id`,
		t,
	)

	return err
}

func (p *SQLiteProvider) List(ctx context.Context, clientID string, options *query.ListOptions) ([]*StoredTunnel, error) {
	values := []*StoredTunnel{}

	q := "SELECT * FROM stored_tunnels WHERE client_id = ?"
	params := []interface{}{clientID}

	q, params = query.AppendOptionsToQuery(options, q, params)

	err := p.db.SelectContext(ctx, &values, q, params...)
	if err != nil {
		return values, err
	}

	return values, nil
}

func (p *SQLiteProvider) Count(ctx context.Context, clientID string, options *query.ListOptions) (int, error) {
	var result int

	q := "SELECT COUNT(*) FROM stored_tunnels WHERE client_id = ?"
	params := []interface{}{clientID}

	countOptions := *options
	countOptions.Pagination = nil
	q, params = query.AppendOptionsToQuery(&countOptions, q, params)

	err := p.db.GetContext(ctx, &result, q, params...)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (p *SQLiteProvider) Delete(ctx context.Context, clientID, id string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM stored_tunnels WHERE client_id = ? AND id = ?", clientID, id)
	return err
}
