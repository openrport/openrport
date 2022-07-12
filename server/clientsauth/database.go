package clientsauth

import (
	"database/sql"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/mattn/go-sqlite3"

	"github.com/cloudradar-monitoring/rport/share/query"

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

func (c *DatabaseProvider) GetFiltered(filter *query.ListOptions) ([]*ClientAuth, int, error) {
	filter.Sorts = append(filter.Sorts, query.SortOption{Column: "id", IsASC: true})
	converter := query.NewSQLConverter()
	converter.SetDbDriverName(c.db.DriverName())
	rQuery, rParams := converter.ConvertListOptionsToQuery(filter, fmt.Sprintf("SELECT id,password FROM %s", c.tableName))
	filter.Pagination = nil
	filter.Sorts = nil
	cQuery, cParams := converter.ConvertListOptionsToQuery(filter, fmt.Sprintf("SELECT COUNT(id) FROM %s", c.tableName))
	var count = 0
	if err := c.db.Get(&count, cQuery, cParams...); err != nil {
		return nil, 0, err
	}
	var result = []*ClientAuth{}
	err := c.db.Select(&result, rQuery, rParams...)
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
