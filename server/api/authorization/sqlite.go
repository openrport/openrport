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
		"SELECT * FROM api_token WHERE username = ?",
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
		"SELECT * FROM api_token WHERE username = ? AND prefix = ?",
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

func (p *SqliteProvider) Save(ctx context.Context, tokenLine *APIToken) (err error) {
	res, err := p.db.NamedExecContext(
		ctx,
		`INSERT INTO api_token (username, prefix, name, expires_at, scope, token)
			      VALUES (:username, :prefix, :name, :expires_at, :scope, :token)
			 	ON CONFLICT(username, prefix) DO UPDATE SET
				 -- the following is per-field logic to update only with non empty value
				 -- (that is to say these fields cannot be blanked/nulled)
				 name=CASE WHEN length(:name) > 0 THEN EXCLUDED.name ELSE api_token.name END,
				 expires_at=CASE WHEN length(:expires_at) > 0 THEN EXCLUDED.expires_at ELSE api_token.expires_at END
				WHERE EXCLUDED.username = api_token.username AND
				       EXCLUDED.prefix = api_token.prefix`,
		tokenLine,
	)

	if err != nil {
		return fmt.Errorf("unable to create api token: %w", err)
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
		"DELETE FROM api_token WHERE username = ? AND prefix = ?",
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
