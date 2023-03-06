package authorization

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type SqliteProvider struct {
	db *sqlx.DB
}

func NewSqliteProvider(db *sqlx.DB) *SqliteProvider {
	return &SqliteProvider{
		db: db,
	}
}

func (p *SqliteProvider) GetAll(ctx context.Context, username string) ([]*APIToken, error) {
	var result []*APIToken
	err := p.db.SelectContext(
		ctx, &result,
		"SELECT * FROM api_tokens WHERE username = ?",
		username,
	)
	if err != nil {
		return result, fmt.Errorf("unable to get api_token from DB: %w", err)
	}

	return result, nil
}

func (p *SqliteProvider) Get(ctx context.Context, username, prefix string) (*APIToken, error) {
	res := &APIToken{}

	err := p.db.GetContext(ctx,
		res,
		"SELECT * FROM api_tokens WHERE username = ? AND prefix = ?",
		username,
		prefix,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("unable to get api token from DB: %w", err)
	}

	return res, nil
}

func (p *SqliteProvider) GetByName(ctx context.Context, username, name string) (*APIToken, error) {
	res := &APIToken{}
	err := p.db.GetContext(ctx,
		res,
		"SELECT * FROM api_tokens WHERE username = ? AND name = ?",
		username,
		name,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("unable to get api token from DB: %w", err)
	}
	return res, nil
}

func (p *SqliteProvider) Save(ctx context.Context, tokenLine *APIToken) (err error) {
	res, err := p.db.NamedExecContext(
		ctx,
		`INSERT INTO api_tokens (username, prefix, name, created_at, expires_at, scope, token)
			      VALUES (:username, :prefix, :name, 
					CASE WHEN :created_at IS NOT NULL THEN :created_at ELSE CURRENT_TIMESTAMP END,
					:expires_at, :scope, :token)
			 	ON CONFLICT(username, prefix) DO UPDATE SET
				 expires_at=CASE WHEN :expires_at IS NOT NULL THEN EXCLUDED.expires_at ELSE api_tokens.expires_at END,
				 name=CASE WHEN :name != "" THEN EXCLUDED.name ELSE api_tokens.name END
				WHERE EXCLUDED.username = api_tokens.username AND
				       EXCLUDED.prefix = api_tokens.prefix`,
		tokenLine,
	)

	if err != nil {
		return err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affectedRows == 0 {
		return fmt.Errorf("cannot Insert or Update token %s", tokenLine.Prefix)
	}

	return nil
}

func (p *SqliteProvider) Delete(ctx context.Context, username, prefix string) error {
	res, err := p.db.ExecContext(
		ctx,
		"DELETE FROM api_tokens WHERE username = ? AND prefix = ?",
		username,
		prefix,
	)
	if err != nil {
		return err
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if affectedRows == 0 {
		return fmt.Errorf("cannot find API Token by prefix %s", prefix)
	}

	return nil
}

func (p *SqliteProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}

	return nil
}
