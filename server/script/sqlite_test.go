package script

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/share/query"

	"github.com/jmoiron/sqlx"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/test"
)

var testLog = chshare.NewLogger("script", chshare.LogOutput{File: os.Stdout}, chshare.LogLevelDebug)

var demoData = []Script{
	{
		ID:          1,
		Name:        "some name",
		CreatedBy:   "user1",
		CreatedAt:   time.Date(2001, 1, 1, 1, 0, 0, 0, time.UTC),
		Interpreter: "bash",
		IsSudo:      false,
		Cwd:         "/bin",
		Script:      "ls -la",
	},
	{
		ID:          2,
		Name:        "other name 2",
		CreatedBy:   "user1",
		CreatedAt:   time.Date(2002, 1, 1, 1, 0, 0, 0, time.UTC),
		Interpreter: "sh",
		IsSudo:      true,
		Cwd:         "/bin",
		Script:      "pwd",
	},
}

func TestGetByID(t *testing.T) {
	dbProv, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	val, found, err := dbProv.GetByID(ctx, 1)

	require.NoError(t, err)
	require.True(t, found)
	require.NoError(t, err)
	assert.Equal(t, demoData[0], *val)

	_, found, err = dbProv.GetByID(ctx, -2)
	require.NoError(t, err)
	require.False(t, found)
}

func TestList(t *testing.T) {
	dbProv, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	vals, err := dbProv.List(context.Background(), &query.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, demoData, vals)

	vals, err = dbProv.List(context.Background(), &query.ListOptions{
		Sorts: []query.SortOption{
			{
				Column: "created_at",
				IsASC:  false,
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []Script{demoData[1], demoData[0]}, vals)

	vals, err = dbProv.List(context.Background(), &query.ListOptions{
		Sorts: []query.SortOption{
			{
				Column: "name",
				IsASC:  true,
			},
		},
		Filters: []query.FilterOption{
			{
				Column: "created_by",
				Values: []string{"user1"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []Script{demoData[1], demoData[0]}, vals)

	vals, err = dbProv.List(context.Background(), &query.ListOptions{
		Filters: []query.FilterOption{
			{
				Column: "interpreter",
				Values: []string{"not-existing-interpreter"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []Script{}, vals)

	vals, err = dbProv.List(context.Background(), &query.ListOptions{
		Filters: []query.FilterOption{
			{
				Column: "is_sudo",
				Values: []string{"0"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, []Script{demoData[0]}, vals)

	vals, err = dbProv.List(context.Background(), &query.ListOptions{
		Sorts: []query.SortOption{
			{
				Column: "interpreter",
				IsASC:  true,
			},
		},
		Filters: []query.FilterOption{
			{
				Column: "name",
				Values: []string{"some name", "other name 2"},
			},
			{
				Column: "created_by",
				Values: []string{"user1"},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, demoData, vals)
}

func TestCreate(t *testing.T) {
	dbProv, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	expectedCreatedAt, err := time.Parse("2006-01-02 15:04:05", "2001-01-01 01:00:00")
	require.NoError(t, err)

	ctx := context.Background()
	itemToSave := demoData[0]
	itemToSave.ID = 0
	id, err := dbProv.Save(ctx, &itemToSave, expectedCreatedAt.UTC())
	require.NoError(t, err)
	assert.True(t, id > 0)

	expectedRows := []map[string]interface{}{
		{
			"id":          int64(1),
			"name":        itemToSave.Name,
			"created_at":  itemToSave.CreatedAt,
			"created_by":  itemToSave.CreatedBy,
			"interpreter": itemToSave.Interpreter,
			"is_sudo":     int64(0),
			"cwd":         itemToSave.Cwd,
			"script":      itemToSave.Script,
		},
	}
	q := "SELECT * FROM `scripts`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

func TestUpdate(t *testing.T) {
	dbProv, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	itemToSave := demoData[0]
	itemToSave.Script = "awk"

	id, err := dbProv.Save(
		ctx,
		&itemToSave,
		time.Date(2012, 1, 1, 1, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	assert.Equal(t, itemToSave.ID, id)

	expectedRows := []map[string]interface{}{
		{
			"id":          int64(1),
			"name":        itemToSave.Name,
			"created_at":  itemToSave.CreatedAt,
			"created_by":  itemToSave.CreatedBy,
			"interpreter": itemToSave.Interpreter,
			"is_sudo":     int64(0),
			"cwd":         itemToSave.Cwd,
			"script":      itemToSave.Script,
		},
	}
	q := "SELECT * FROM `scripts` where id = 1"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

func TestDelete(t *testing.T) {
	dbProv, err := NewSqliteProvider(":memory:", testLog)
	require.NoError(t, err)
	defer dbProv.Close()

	ctx := context.Background()

	err = addDemoData(dbProv.db)
	require.NoError(t, err)

	err = dbProv.Delete(ctx, -2)
	assert.EqualError(t, err, "cannot find entry by id -2")

	err = dbProv.Delete(ctx, 2)
	require.NoError(t, err)

	expectedRows := []map[string]interface{}{
		{
			"id":          int64(1),
			"name":        demoData[0].Name,
			"created_at":  demoData[0].CreatedAt,
			"created_by":  demoData[0].CreatedBy,
			"interpreter": demoData[0].Interpreter,
			"is_sudo":     int64(0),
			"cwd":         demoData[0].Cwd,
			"script":      demoData[0].Script,
		},
	}
	q := "SELECT * FROM `scripts`"
	test.AssertRowsEqual(t, dbProv.db, expectedRows, q, []interface{}{})
}

func addDemoData(db *sqlx.DB) error {
	for i := range demoData {
		_, err := db.Exec(
			"INSERT INTO `scripts` (`name`, `created_at`, `created_by`, `interpreter`, `is_sudo`, `cwd`, `script`) VALUES (?,?,?,?,?,?,?)",
			demoData[i].Name,
			demoData[i].CreatedAt.Format(time.RFC3339),
			demoData[i].CreatedBy,
			demoData[i].Interpreter,
			demoData[i].IsSudo,
			demoData[i].Cwd,
			demoData[i].Script,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
