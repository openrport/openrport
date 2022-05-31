package clientsauth

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/cloudradar-monitoring/rport/share/query"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"

	"github.com/cloudradar-monitoring/rport/share/enums"
)

const mysqlDuplicateEntryErrorCode = 1062

type DatabaseProvider struct {
	db        *sqlx.DB
	tableName string
}

var _ Provider = &DatabaseProvider{}

func NewDatabaseProvider(DB *sqlx.DB, tableName string) *DatabaseProvider {
	return &DatabaseProvider{
		db:        DB,
		tableName: tableName,
	}
}

func NewDatabaseMockProvider(clients []*ClientAuth) *DatabaseProvider {
	var authDb *sqlx.DB
	authDb, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		log.Fatalln(err)
	}
	if _, err := authDb.Exec(`CREATE TABLE clients_auth (id text,password text)`); err != nil {
		log.Fatalln(err)
	}
	for _, v := range clients {
		if _, err := authDb.Exec("INSERT INTO clients_auth VALUES(?,?)", v.ID, v.Password); err != nil {
			log.Fatalln(err)
		}
	}
	return &DatabaseProvider{
		db:        authDb,
		tableName: "clients_auth",
	}
}

func (c *DatabaseProvider) GetAll() ([]*ClientAuth, error) {
	var result []*ClientAuth
	err := c.db.Select(&result, fmt.Sprintf("SELECT id, password FROM %s", c.tableName))
	return result, err
}

func (c *DatabaseProvider) GetFiltered(filter *query.ListOptions) ([]*ClientAuth, int, error) {
	var result = []*ClientAuth{}
	iLimit, _ := strconv.Atoi(filter.Pagination.Limit)
	iOffset, _ := strconv.Atoi(filter.Pagination.Offset)
	var count = 0
	sql := fmt.Sprintf("FROM %s WHERE id LIKE ? ESCAPE '\\'", c.tableName)
	var match = "%"
	if len(filter.Filters) > 0 {
		match = strings.Replace(filter.Filters[0].Values[0], "%", "\\%", -1)
		match = strings.Replace(match, "*", "%", -1)
	}
	if err := c.db.Get(&count, "SELECT COUNT(id) "+sql, match); err != nil {
		return nil, 0, err
	}
	err := c.db.Select(&result, "SELECT id,password "+sql+fmt.Sprintf(" ORDER BY id ASC LIMIT %d OFFSET %d", iLimit, iOffset), match)
	return result, count, err
}

func (c *DatabaseProvider) Get(id string) (*ClientAuth, error) {
	result := &ClientAuth{}
	err := c.db.Get(result, fmt.Sprintf("SELECT id, password FROM %s WHERE id = ?", c.tableName), id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return result, err
}

func (c *DatabaseProvider) Add(client *ClientAuth) (bool, error) {
	_, err := c.db.NamedExec(fmt.Sprintf("INSERT INTO %s (id, password) VALUES (:id, :password)", c.tableName), client)
	if err != nil {
		// Check for client already exists error
		switch typeErr := err.(type) {
		case sqlite3.Error:
			if typeErr.Code == sqlite3.ErrConstraint {
				return false, nil
			}
		case *mysql.MySQLError:
			if typeErr.Number == mysqlDuplicateEntryErrorCode {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

func (c *DatabaseProvider) Delete(id string) error {
	_, err := c.db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", c.tableName), id)
	return err
}

func (c *DatabaseProvider) IsWriteable() bool {
	return true
}

func (c *DatabaseProvider) Source() enums.ProviderSource {
	return enums.ProviderSourceDB
}
