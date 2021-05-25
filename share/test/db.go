package test

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ConvertDbRowsToMapSlice(dbRows *sqlx.Rows) ([]map[string]interface{}, error) {
	dataRows := make([]map[string]interface{}, 0)
	for dbRows.Next() {
		dataRow := make(map[string]interface{})
		err := dbRows.MapScan(dataRow)
		if err != nil {
			return nil, err
		}
		dataRows = append(dataRows, dataRow)
	}

	return dataRows, nil
}

func AssertRowsEqual(t *testing.T, db *sqlx.DB, expectedRows []map[string]interface{}, query string, params []interface{}) {
	dbRows, err := db.Queryx(query, params...)
	require.NoError(t, err)

	actualRows, err := ConvertDbRowsToMapSlice(dbRows)
	require.NoError(t, err)
	assert.Equal(t, expectedRows, actualRows)
}
