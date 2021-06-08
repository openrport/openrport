package script

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/cloudradar-monitoring/rport/share/query"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
)

type DbProviderMock struct {
	getByIDGiven        string
	getByIDScriptToGive *Script
	getByIDFoundToGive  bool
	getByIDErrorToGive  error

	listOptionInput  *query.ListOptions
	listValuesToGive []Script
	listErrorToGive  error

	saveScriptGiven  *Script
	saveNowDateGiven time.Time
	saveErrorToGive  error
	saveIDToGive     string

	deleteIDGiven     string
	deleteErrorToGive error

	io.Closer

	isClosed bool
}

func (dpm *DbProviderMock) GetByID(ctx context.Context, id string) (val *Script, found bool, err error) {
	dpm.getByIDGiven = id
	return dpm.getByIDScriptToGive, dpm.getByIDFoundToGive, dpm.getByIDErrorToGive
}

func (dpm *DbProviderMock) List(ctx context.Context, lo *query.ListOptions) ([]Script, error) {
	dpm.listOptionInput = lo

	return dpm.listValuesToGive, dpm.listErrorToGive
}

func (dpm *DbProviderMock) Save(ctx context.Context, s *Script, nowDate time.Time) (string, error) {
	dpm.saveScriptGiven = s
	dpm.saveNowDateGiven = nowDate

	return dpm.saveIDToGive, dpm.saveErrorToGive
}

func (dpm *DbProviderMock) Delete(ctx context.Context, id string) error {
	dpm.deleteIDGiven = id
	return dpm.deleteErrorToGive
}

func (dpm *DbProviderMock) Close() error {
	dpm.isClosed = true

	return nil
}

func (dpm *DbProviderMock) GetDbProvider() DbProvider {
	return dpm
}

func TestManagerList(t *testing.T) {
	expectedScripts := []Script{
		{
			ID:        "123",
			CreatedBy: "user1",
			CreatedAt: time.Now(),
			Name:      "some nam",
			Script:    "some script",
		},
	}
	dbProv := &DbProviderMock{
		listValuesToGive: expectedScripts,
	}

	mngr := NewManager(dbProv, nil, testLog)

	inputURL, err := url.Parse("/someu?sort=name&sort=-created_at&filter[name]=some nam")
	require.NoError(t, err)

	req := &http.Request{
		URL: inputURL,
	}

	actualValues, err := mngr.List(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(
		t,
		&query.ListOptions{
			Sorts: []query.SortOption{
				{
					Column: "name",
					IsASC:  true,
				},
				{
					Column: "created_at",
					IsASC:  false,
				},
			},
			Filters: []query.FilterOption{
				{
					Column: "name",
					Values: []string{"some nam"},
				},
			},
		},
		dbProv.listOptionInput,
	)
	assert.Equal(t, expectedScripts, actualValues)

	dbProv = &DbProviderMock{
		listErrorToGive: errors.New("list error"),
	}

	mngr = NewManager(dbProv, nil, testLog)

	_, err = mngr.List(context.Background(), req)
	require.EqualError(t, err, "list error")
}

func TestListWithUnsupportedFilterAndSort(t *testing.T) {
	dbProv := &DbProviderMock{
		listValuesToGive: []Script{},
	}

	mngr := NewManager(dbProv, nil, testLog)

	inputURL, err := url.Parse("/someu?sort=unsupportedSortField&filter[unsupportedFilter]=val1")
	require.NoError(t, err)

	req := &http.Request{
		URL: inputURL,
	}

	_, err = mngr.List(context.Background(), req)
	require.EqualError(t, err, "unsupported sort field 'unsupportedSortField', unsupported filter field 'unsupportedFilter'")
}

func TestManagerClose(t *testing.T) {
	dbProv := &DbProviderMock{
		listValuesToGive: []Script{},
		isClosed:         false,
	}

	mngr := NewManager(dbProv, nil, testLog)
	err := mngr.Close()
	require.NoError(t, err)
	require.True(t, dbProv.isClosed)
}

func TestGetOne(t *testing.T) {
	givenStoredValue := &Script{
		ID:        "123",
		CreatedBy: "guy",
	}
	expectedValue := &Script{
		ID:        "123",
		CreatedBy: "guy",
	}
	dbProv := &DbProviderMock{
		getByIDScriptToGive: givenStoredValue,
		getByIDFoundToGive:  true,
	}

	mngr := NewManager(dbProv, nil, testLog)

	val, found, err := mngr.GetOne(context.Background(), "1")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, expectedValue, val)

	dbProv = &DbProviderMock{
		getByIDFoundToGive: false,
	}

	mngr = NewManager(dbProv, nil, testLog)

	_, found, err = mngr.GetOne(context.Background(), "1")
	require.NoError(t, err)
	assert.False(t, found)

	dbProv = &DbProviderMock{
		getByIDErrorToGive: errors.New("some get id error"),
	}

	mngr = NewManager(dbProv, nil, testLog)

	_, _, err = mngr.GetOne(context.Background(), "1")
	require.EqualError(t, err, "some get id error")
}

func TestStore(t *testing.T) {
	inputValue := &InputScript{
		Name:        "some nam",
		Interpreter: "some inter",
		IsSudo:      true,
		Cwd:         "/user/local",
		Script:      "pwd",
	}

	t.Run("create_success", func(t *testing.T) {
		dbProv := &DbProviderMock{
			saveIDToGive: "123",
		}
		mngr := NewManager(dbProv, nil, testLog)

		storedScript, err := mngr.Create(context.Background(), inputValue, "someuser")
		require.NoError(t, err)
		assert.NotEqual(t, "", storedScript.ID)
		assert.NotEqual(t, "", dbProv.saveScriptGiven.ID)

		assert.Equal(t, "someuser", dbProv.saveScriptGiven.CreatedBy)
		assert.Equal(t, "someuser", storedScript.CreatedBy)

		assert.True(t, dbProv.saveScriptGiven.CreatedAt.Equal(time.Now()) || dbProv.saveScriptGiven.CreatedAt.Before(time.Now()))
		assert.True(t, storedScript.CreatedAt.Equal(time.Now()) || storedScript.CreatedAt.Before(time.Now()))

		assert.Equal(t, "pwd", dbProv.saveScriptGiven.Script)
		assert.Equal(t, "pwd", storedScript.Script)

		assert.Equal(t, "/user/local", dbProv.saveScriptGiven.Cwd)
		assert.Equal(t, "/user/local", storedScript.Cwd)

		assert.Equal(t, "some inter", dbProv.saveScriptGiven.Interpreter)
		assert.Equal(t, "some inter", storedScript.Interpreter)

		assert.Equal(t, "some nam", dbProv.saveScriptGiven.Name)
		assert.Equal(t, "some nam", storedScript.Name)

		assert.True(t, dbProv.saveScriptGiven.IsSudo)
		assert.True(t, storedScript.IsSudo)
	})

	t.Run("update_success", func(t *testing.T) {
		const idToUpdate = "123"
		dbProv := &DbProviderMock{
			getByIDFoundToGive: true,
			saveIDToGive:       idToUpdate,
		}
		mngr := NewManager(dbProv, nil, testLog)

		storedScript, err := mngr.Update(context.Background(), idToUpdate, inputValue, "someuser")
		require.NoError(t, err)

		assert.Equal(t, idToUpdate, storedScript.ID)

		assert.Equal(t, "someuser", dbProv.saveScriptGiven.CreatedBy)
		assert.Equal(t, "someuser", storedScript.CreatedBy)

		assert.Equal(t, "pwd", dbProv.saveScriptGiven.Script)
		assert.Equal(t, "pwd", storedScript.Script)

		assert.Equal(t, "/user/local", dbProv.saveScriptGiven.Cwd)
		assert.Equal(t, "/user/local", storedScript.Cwd)

		assert.Equal(t, "some inter", dbProv.saveScriptGiven.Interpreter)
		assert.Equal(t, "some inter", storedScript.Interpreter)

		assert.Equal(t, "some nam", dbProv.saveScriptGiven.Name)
		assert.Equal(t, "some nam", storedScript.Name)

		assert.True(t, dbProv.saveScriptGiven.IsSudo)
		assert.True(t, storedScript.IsSudo)
	})

	t.Run("store_failure_key_exists_update", func(t *testing.T) {
		dbProv := &DbProviderMock{
			listValuesToGive: []Script{
				{
					ID: "2",
				},
			},
			getByIDFoundToGive: true,
		}
		mngr := NewManager(dbProv, nil, testLog)

		_, err := mngr.Update(context.Background(), "1", inputValue, "someuser")
		require.EqualError(t, err, "another script with the same name 'some nam' exists")
	})

	t.Run("update_with_invalid_id", func(t *testing.T) {
		dbProv := &DbProviderMock{
			getByIDFoundToGive: false,
		}
		mngr := NewManager(dbProv, nil, testLog)

		_, err := mngr.Update(context.Background(), "1", inputValue, "someuser")
		require.EqualError(t, err, "cannot find entry by the provided ID")
	})

	t.Run("store_failure_key_exists_create", func(t *testing.T) {
		dbProv := &DbProviderMock{
			listValuesToGive: []Script{
				{
					ID: "3",
				},
			},
		}
		mngr := NewManager(dbProv, nil, testLog)

		_, err := mngr.Create(context.Background(), inputValue, "someuser")
		require.EqualError(t, err, "another script with the same name 'some nam' exists")
	})

	t.Run("update_list_error", func(t *testing.T) {
		dbProv := &DbProviderMock{
			listErrorToGive:    errors.New("failed to find anything"),
			getByIDFoundToGive: true,
		}
		mngr := NewManager(dbProv, nil, testLog)

		_, err := mngr.Update(context.Background(), "1", inputValue, "someuser")
		require.EqualError(t, err, "failed to find anything")
	})

	t.Run("invalid_input", func(t *testing.T) {
		dbProv := &DbProviderMock{}
		mngr := NewManager(dbProv, nil, testLog)

		_, err := mngr.Update(context.Background(), "1", &InputScript{}, "someuser")
		require.EqualError(t, err, "name is required, script is required")
	})

	t.Run("db_store_error", func(t *testing.T) {
		dbProv := &DbProviderMock{
			saveErrorToGive:    errors.New("failed to save"),
			getByIDFoundToGive: true,
		}
		mngr := NewManager(dbProv, nil, testLog)

		_, err := mngr.Update(context.Background(), "1", inputValue, "someuser")
		require.EqualError(t, err, "failed to save")
	})
}

func TestDeleteScript(t *testing.T) {
	t.Run("delete_success", func(t *testing.T) {
		dbProv := &DbProviderMock{
			getByIDFoundToGive: true,
		}
		mngr := NewManager(dbProv, nil, testLog)

		err := mngr.Delete(context.Background(), "1")
		require.NoError(t, err)

		assert.Equal(t, "1", dbProv.deleteIDGiven)
	})

	t.Run("db_error", func(t *testing.T) {
		dbProv := &DbProviderMock{
			deleteErrorToGive:  errors.New("cannot delete"),
			getByIDFoundToGive: true,
		}
		mngr := NewManager(dbProv, nil, testLog)

		err := mngr.Delete(context.Background(), "1")
		require.EqualError(t, err, "cannot delete")
	})

	t.Run("entryNotFound", func(t *testing.T) {
		dbProv := &DbProviderMock{
			getByIDFoundToGive: false,
		}
		mngr := NewManager(dbProv, nil, testLog)

		err := mngr.Delete(context.Background(), "1")
		require.Equal(
			t,
			errors2.APIError{
				Message: "cannot find this entry by the provided id",
				Code:    http.StatusNotFound,
			},
			err,
		)
	})

	t.Run("entry read error", func(t *testing.T) {
		readErr := errors.New("cannot read database by id")
		dbProv := &DbProviderMock{
			getByIDErrorToGive: readErr,
		}
		mngr := NewManager(dbProv, nil, testLog)

		err := mngr.Delete(context.Background(), "1")
		require.Equal(
			t,
			errors2.APIError{
				Err:  readErr,
				Code: http.StatusInternalServerError,
			},
			err,
		)
	})
}
