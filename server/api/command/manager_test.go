package command

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errors2 "github.com/cloudradar-monitoring/rport/server/api/errors"
	"github.com/cloudradar-monitoring/rport/share/query"
)

type DbProviderMock struct {
	getByIDGiven         string
	getByIDCommandToGive *Command
	getByIDFoundToGive   bool
	getByIDErrorToGive   error

	listOptionInput  *query.ListOptions
	listValuesToGive []Command
	listErrorToGive  error

	saveCommandGiven *Command
	saveErrorToGive  error
	saveIDToGive     string

	deleteIDGiven     string
	deleteErrorToGive error

	io.Closer

	isClosed bool
}

func (dpm *DbProviderMock) GetByID(ctx context.Context, id string, ro *query.RetrieveOptions) (val *Command, found bool, err error) {
	dpm.getByIDGiven = id
	return dpm.getByIDCommandToGive, dpm.getByIDFoundToGive, dpm.getByIDErrorToGive
}

func (dpm *DbProviderMock) List(ctx context.Context, lo *query.ListOptions) ([]Command, error) {
	dpm.listOptionInput = lo

	return dpm.listValuesToGive, dpm.listErrorToGive
}

func (dpm *DbProviderMock) Save(ctx context.Context, s *Command) (string, error) {
	dpm.saveCommandGiven = s

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
	now := time.Now()
	expectedCommands := []Command{
		{
			ID:        "123",
			CreatedBy: "user1",
			CreatedAt: &now,
			Name:      "some nam",
			Cmd:       "some command",
		},
	}
	dbProv := &DbProviderMock{
		listValuesToGive: expectedCommands,
	}
	mngr := NewManager(dbProv)

	inputURL, err := url.Parse("/someu?sort=name&sort=-created_at&filter[name]=some nam&fields[commands]=id,name")
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
			Fields: []query.FieldsOption{
				{
					Resource: "commands",
					Fields:   []string{"id", "name"},
				},
			},
		},
		dbProv.listOptionInput,
	)
	assert.Equal(t, expectedCommands, actualValues)

	dbProv = &DbProviderMock{
		listErrorToGive: errors.New("list error"),
	}
	mngr = NewManager(dbProv)

	_, err = mngr.List(context.Background(), req)
	require.EqualError(t, err, "list error")
}

func TestListWithUnsupportedOptions(t *testing.T) {
	dbProv := &DbProviderMock{
		listValuesToGive: []Command{},
	}

	mngr := NewManager(dbProv)

	inputURL, err := url.Parse("/someu?sort=unsupportedSortField&filter[unsupportedFilter]=val1&fields[commands]=nope")
	require.NoError(t, err)

	req := &http.Request{
		URL: inputURL,
	}

	_, err = mngr.List(context.Background(), req)
	require.EqualError(t, err, `unsupported sort field 'unsupportedSortField', unsupported filter field 'unsupportedFilter', unsupported field "nope" for resource "commands"`)
}

func TestManagerClose(t *testing.T) {
	dbProv := &DbProviderMock{
		listValuesToGive: []Command{},
		isClosed:         false,
	}

	mngr := NewManager(dbProv)
	err := mngr.Close()
	require.NoError(t, err)
	require.True(t, dbProv.isClosed)
}

func TestGetOne(t *testing.T) {
	givenStoredValue := &Command{
		ID:        "123",
		CreatedBy: "guy",
	}
	expectedValue := &Command{
		ID:        "123",
		CreatedBy: "guy",
	}
	dbProv := &DbProviderMock{
		getByIDCommandToGive: givenStoredValue,
		getByIDFoundToGive:   true,
	}

	inputURL, err := url.Parse("/commands/id?fields[commands]=id,name")
	require.NoError(t, err)
	req := &http.Request{
		URL: inputURL,
	}

	mngr := NewManager(dbProv)

	val, found, err := mngr.GetOne(context.Background(), req, "1")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, expectedValue, val)

	dbProv = &DbProviderMock{
		getByIDFoundToGive: false,
	}

	mngr = NewManager(dbProv)

	_, found, err = mngr.GetOne(context.Background(), req, "1")
	require.NoError(t, err)
	assert.False(t, found)

	dbProv = &DbProviderMock{
		getByIDErrorToGive: errors.New("some get id error"),
	}

	mngr = NewManager(dbProv)

	_, _, err = mngr.GetOne(context.Background(), req, "1")
	require.EqualError(t, err, "some get id error")
}

func TestStore(t *testing.T) {
	inputValue := &InputCommand{
		Name: "some nam",
		Cmd:  "pwd",
	}

	t.Run("create_success", func(t *testing.T) {
		dbProv := &DbProviderMock{
			saveIDToGive: "123",
		}
		mngr := NewManager(dbProv)

		storedCommand, err := mngr.Create(context.Background(), inputValue, "someuser")
		require.NoError(t, err)
		assert.NotEqual(t, "", storedCommand.ID)
		assert.NotEqual(t, "", dbProv.saveCommandGiven.ID)

		assert.Equal(t, "someuser", dbProv.saveCommandGiven.CreatedBy)
		assert.Equal(t, "someuser", storedCommand.CreatedBy)

		assert.True(t, dbProv.saveCommandGiven.CreatedAt.Equal(time.Now()) || dbProv.saveCommandGiven.CreatedAt.Before(time.Now()))
		assert.True(t, storedCommand.CreatedAt.Equal(time.Now()) || storedCommand.CreatedAt.Before(time.Now()))

		assert.Equal(t, "pwd", dbProv.saveCommandGiven.Cmd)
		assert.Equal(t, "pwd", storedCommand.Cmd)

		assert.Equal(t, "some nam", dbProv.saveCommandGiven.Name)
		assert.Equal(t, "some nam", storedCommand.Name)
	})

	t.Run("update_success", func(t *testing.T) {
		const idToUpdate = "123"
		now := time.Now()
		dbProv := &DbProviderMock{
			getByIDFoundToGive: true,
			getByIDCommandToGive: &Command{
				ID:        "123",
				CreatedBy: "user1",
				CreatedAt: &now,
				Name:      "some nam",
				Cmd:       "some command",
			},
			saveIDToGive: idToUpdate,
		}
		mngr := NewManager(dbProv)

		storedCommand, err := mngr.Update(context.Background(), idToUpdate, inputValue, "someuser")
		require.NoError(t, err)

		assert.Equal(t, idToUpdate, storedCommand.ID)

		assert.Equal(t, "user1", dbProv.saveCommandGiven.CreatedBy)
		assert.Equal(t, "user1", storedCommand.CreatedBy)

		assert.Equal(t, "someuser", dbProv.saveCommandGiven.UpdatedBy)
		assert.Equal(t, "someuser", storedCommand.UpdatedBy)

		assert.Equal(t, "pwd", dbProv.saveCommandGiven.Cmd)
		assert.Equal(t, "pwd", storedCommand.Cmd)

		assert.Equal(t, "some nam", dbProv.saveCommandGiven.Name)
		assert.Equal(t, "some nam", storedCommand.Name)
	})

	t.Run("store_failure_key_exists_update", func(t *testing.T) {
		dbProv := &DbProviderMock{
			listValuesToGive: []Command{
				{
					ID: "2",
				},
			},
			getByIDFoundToGive: true,
		}
		mngr := NewManager(dbProv)

		_, err := mngr.Update(context.Background(), "1", inputValue, "someuser")
		require.EqualError(t, err, "another command with the same name 'some nam' exists")
	})

	t.Run("update_with_invalid_id", func(t *testing.T) {
		dbProv := &DbProviderMock{
			getByIDFoundToGive: false,
		}
		mngr := NewManager(dbProv)

		_, err := mngr.Update(context.Background(), "1", inputValue, "someuser")
		require.EqualError(t, err, "cannot find entry by the provided ID")
	})

	t.Run("store_failure_key_exists_create", func(t *testing.T) {
		dbProv := &DbProviderMock{
			listValuesToGive: []Command{
				{
					ID: "3",
				},
			},
		}
		mngr := NewManager(dbProv)

		_, err := mngr.Create(context.Background(), inputValue, "someuser")
		require.EqualError(t, err, "another command with the same name 'some nam' exists")
	})

	t.Run("update_list_error", func(t *testing.T) {
		dbProv := &DbProviderMock{
			listErrorToGive:    errors.New("failed to find anything"),
			getByIDFoundToGive: true,
		}
		mngr := NewManager(dbProv)

		_, err := mngr.Update(context.Background(), "1", inputValue, "someuser")
		require.EqualError(t, err, "failed to find anything")
	})

	t.Run("invalid_input", func(t *testing.T) {
		dbProv := &DbProviderMock{}
		mngr := NewManager(dbProv)

		_, err := mngr.Update(context.Background(), "1", &InputCommand{}, "someuser")
		require.EqualError(t, err, "name is required, cmd is required")
	})

	t.Run("db_store_error", func(t *testing.T) {
		now := time.Now()
		dbProv := &DbProviderMock{
			saveErrorToGive:    errors.New("failed to save"),
			getByIDFoundToGive: true,
			getByIDCommandToGive: &Command{
				ID:        "123",
				CreatedBy: "user1",
				CreatedAt: &now,
				Name:      "some nam",
				Cmd:       "some command",
			},
		}
		mngr := NewManager(dbProv)

		_, err := mngr.Update(context.Background(), "123", inputValue, "someuser")
		require.EqualError(t, err, "failed to save")
	})
}

func TestDeleteCommand(t *testing.T) {
	t.Run("delete_success", func(t *testing.T) {
		dbProv := &DbProviderMock{
			getByIDFoundToGive: true,
		}
		mngr := NewManager(dbProv)

		err := mngr.Delete(context.Background(), "1")
		require.NoError(t, err)

		assert.Equal(t, "1", dbProv.deleteIDGiven)
	})

	t.Run("db_error", func(t *testing.T) {
		dbProv := &DbProviderMock{
			deleteErrorToGive:  errors.New("cannot delete"),
			getByIDFoundToGive: true,
		}
		mngr := NewManager(dbProv)

		err := mngr.Delete(context.Background(), "1")
		require.EqualError(t, err, "cannot delete")
	})

	t.Run("entryNotFound", func(t *testing.T) {
		dbProv := &DbProviderMock{
			getByIDFoundToGive: false,
		}
		mngr := NewManager(dbProv)

		err := mngr.Delete(context.Background(), "1")
		require.Equal(
			t,
			errors2.APIError{
				Message:    "cannot find this entry by the provided id",
				HTTPStatus: http.StatusNotFound,
			},
			err,
		)
	})

	t.Run("entry read error", func(t *testing.T) {
		readErr := errors.New("cannot read database by id")
		dbProv := &DbProviderMock{
			getByIDErrorToGive: readErr,
		}
		mngr := NewManager(dbProv)

		err := mngr.Delete(context.Background(), "1")
		require.Equal(
			t,
			errors2.APIError{
				Err:        readErr,
				HTTPStatus: http.StatusInternalServerError,
			},
			err,
		)
	})
}
