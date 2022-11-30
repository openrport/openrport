package authorization

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	api_token "github.com/cloudradar-monitoring/rport/db/migration/authorization"
	"github.com/jmoiron/sqlx"

	"github.com/cloudradar-monitoring/rport/db/sqlite"
)

type SqliteProvider struct {
	db *sqlx.DB
}

func NewSqliteProvider(dbPath string, dataSourceOptions sqlite.DataSourceOptions) (*SqliteProvider, error) {
	db, err := sqlite.New(dbPath, api_token.AssetNames(), api_token.Asset, dataSourceOptions)
	if err != nil {
		return nil, fmt.Errorf("unable to create api_token DB instance: %w", err)
	}

	return &SqliteProvider{db: db}, nil
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

func (p *SqliteProvider) Get(ctx context.Context, id int64) (*APIToken, error) {
	res := &APIToken{}

	err := p.db.GetContext(ctx,
		res,
		"SELECT * FROM api_token WHERE id = ?",
		id,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("unable to get api token from DB by id: %w", err)
	}

	return res, nil
}

// FROM HERE ON, PATTERNS ONLY

func (p *SqliteProvider) Save(ctx context.Context, session *APIToken) (sessionID int64, err error) {
	return 0, nil
	// if session.SessionID == 0 {
	// 	sessionID, err = p.add(ctx, session)
	// } else {
	// 	sessionID, err = p.update(ctx, session)
	// }

	// if err != nil {
	// 	return -1, err
	// }

	// return sessionID, nil
}

func (p *SqliteProvider) add(ctx context.Context, session *APIToken) (sessionID int64, err error) {
	return 0, nil
	// result, err := p.db.NamedExecContext(
	// 	ctx,
	// 	"INSERT INTO"+
	// 		" api_token (expires_at, username, last_access_at, user_agent, ip_address)"+
	// 		" VALUES (:expires_at, :username, :last_access_at, :user_agent, :ip_address)",
	// 	session,
	// )
	// if err != nil {
	// 	return 0, fmt.Errorf("unable to create api session: %w", err)
	// }

	// sessionID, err = result.LastInsertId()
	// if err != nil {
	// 	return 0, fmt.Errorf("unable to get api session ID for save: %w", err)
	// }

	// return sessionID, nil
}

func (p *SqliteProvider) update(ctx context.Context, session *APIToken) (sessionID int64, err error) {
	return 0, nil
	// _, err = p.db.NamedExecContext(
	// 	ctx,
	// 	"UPDATE api_token"+
	// 		"  SET expires_at=:expires_at,username=:username,"+
	// 		"   last_access_at=:last_access_at, user_agent=:user_agent, ip_address=:ip_address"+
	// 		"  WHERE session_id=:session_id",
	// 	session,
	// )
	// if err != nil {
	// 	return 0, fmt.Errorf("unable to update api session: %w", err)
	// }

	// sessionID = session.SessionID
	// return sessionID, nil
}

func (p *SqliteProvider) Delete(ctx context.Context, sessionID int64) error {
	// _, err := p.db.ExecContext(
	// 	ctx,
	// 	"DELETE FROM api_token WHERE session_id = ?",
	// 	sessionID,
	// )
	// if err != nil {
	// 	return fmt.Errorf("unable to delete api session by token: %w", err)
	// }

	return nil
}

func (p *SqliteProvider) DeleteExpired(ctx context.Context) error {
	// _, err := p.db.ExecContext(
	// 	ctx,
	// 	"DELETE FROM api_token WHERE DATETIME(expires_at) <= DATETIME(?)",
	// 	time.Now(),
	// )
	// if err != nil {
	// 	return fmt.Errorf("unable to delete expired api sessions: %w", err)
	// }

	return nil
}

func (p *SqliteProvider) Close() error {
	return p.db.Close()
}

func (p *SqliteProvider) DeleteAllByUser(ctx context.Context, username string) (err error) {
	// _, err = p.db.ExecContext(
	// 	ctx,
	// 	"DELETE FROM api_token WHERE username = ?",
	// 	username,
	// )
	// if err != nil {
	// 	return err
	// }

	return nil
}

func (p *SqliteProvider) DeleteByID(ctx context.Context, username string, sessionID int64) (err error) {
	// _, err = p.db.ExecContext(
	// 	ctx,
	// 	"DELETE FROM api_token WHERE username = ? AND session_id = ?",
	// 	username,
	// 	sessionID,
	// )
	// if err != nil {
	// 	return err
	// }

	return nil
}
