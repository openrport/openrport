package authorization

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cloudradar-monitoring/rport/share/query"
	"github.com/jmoiron/sqlx"
)

type SqliteProvider struct {
	db        *sqlx.DB
	converter *query.SQLConverter
}

func NewSqliteProvider(db *sqlx.DB) *SqliteProvider {
	return &SqliteProvider{
		db:        db,
		converter: query.NewSQLConverter(db.DriverName()),
	}
}

func (p *SqliteProvider) GetAll(ctx context.Context, username string) ([]*APIToken, error) {
	var result []*APIToken
	err := p.db.SelectContext(
		ctx, &result,
		"SELECT * FROM api_token WHERE username = ?", // later? AND DATETIME(expires_at) >= DATETIME(?)",
		username,
		// time.Now(),
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
func (p *SqliteProvider) Save(ctx context.Context, tokenLine *APIToken) (err error) {
	_, err = p.db.NamedExecContext(
		ctx,
		"INSERT INTO"+
			" `api_token` (`username`, `prefix`, `expires_at`, `scope`, `token`)"+
			"      VALUES (:username, :prefix, :expires_at, :scope, :token)"+
			" 	ON CONFLICT(username, prefix) DO UPDATE SET"+
			"		expires_at=EXCLUDED.expires_at"+
			"	WHERE EXCLUDED.username = api_token.username AND"+
			"	       EXCLUDED.prefix = api_token.prefix",
		tokenLine,
	)
	if err != nil {
		return fmt.Errorf("unable to create api token: %w", err)
	}
	fmt.Printf("tried to save %+v", tokenLine)
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
		return fmt.Errorf("cannot find entry by username %s and %s", username, prefix)
	}

	return nil
}

func (p *SqliteProvider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}

	return nil
}
