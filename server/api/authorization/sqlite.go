package authorization

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
func (p *SqliteProvider) GetAll(ctx context.Context) ([]*APIToken, error) {
	var result []*APIToken
	err := p.db.SelectContext(
		ctx, &result,
		"SELECT * FROM api_token WHERE DATETIME(expires_at) >= DATETIME(?)",
		time.Now(),
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

/*
upsert in sqlite:

CREATE TABLE phonebook2(

	name TEXT PRIMARY KEY,
	phonenumber TEXT,
	validDate DATE

);
INSERT INTO phonebook2(name,phonenumber,validDate)

	VALUES('Alice','704-555-1212','2018-05-08')
	ON CONFLICT(name) DO UPDATE SET
	  phonenumber=EXCLUDED.phonenumber,
	  validDate=EXCLUDED.validDate
	WHERE EXCLUDED.validDate>phonebook2.validDate;
*/
func (p *SqliteProvider) save(ctx context.Context, tokenLine *APIToken) (err error) {
	_, err = p.db.NamedExecContext(
		ctx,
		"INSERT INTO"+
			" `api_token` (`username`, `prefix`, `expires_at`, `scope`, `token`)"+
			"      VALUES (:username, :prefix, :expires_at, :scope, :token)"+
			" 	ON CONFLICT(username, prefix) DO UPDATE SET"+
			"		expires_at=EXCLUDED.expires_at,"+
			"		scope=EXCLUDED.scope,"+
			"		token=EXCLUDED.token"+
			"	WHERE EXCLUDED.username = api_token.username AND"+
			"	       EXCLUDED.prefix = api_token.prefix",
		tokenLine,
	)
	if err != nil {
		return fmt.Errorf("unable to create api token: %w", err)
	}

	return nil
}

func (p *SqliteProvider) Delete(ctx context.Context, username, prefix string) error {
	_, err := p.db.ExecContext(
		ctx,
		"DELETE FROM api_token WHERE username = ? AND prefix = ?",
		username,
		prefix,
	)
	if err != nil {
		return fmt.Errorf("unable to delete api token: %w", err)
	}

	return nil
}

func (p *SqliteProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}

	return nil
}

// func (p *SqliteProvider) DeleteAllByUser(ctx context.Context, username string) (err error) {
// 	// _, err = p.db.ExecContext(
// 	// 	ctx,
// 	// 	"DELETE FROM api_token WHERE username = ?",
// 	// 	username,
// 	// )
// 	// if err != nil {
// 	// 	return err
// 	// }

// 	return nil
// }

// func (p *SqliteProvider) DeleteByID(ctx context.Context, username string, sessionID int64) (err error) {
// 	// _, err = p.db.ExecContext(
// 	// 	ctx,
// 	// 	"DELETE FROM api_token WHERE username = ? AND session_id = ?",
// 	// 	username,
// 	// 	sessionID,
// 	// )
// 	// if err != nil {
// 	// 	return err
// 	// }

// 	return nil
// }
