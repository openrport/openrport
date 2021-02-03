package clients

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCRWithExpiration(t *testing.T) {
	now = nowMockF

	exp := 2 * time.Hour
	repo := NewClientRepository([]*Client{c1, c2}, &exp)

	assert := assert.New(t)
	assert.NoError(repo.Save(c3))
	assert.NoError(repo.Save(c4))

	gotCount, err := repo.Count()
	assert.NoError(err)
	assert.Equal(3, gotCount)

	gotClients, err := repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*Client{c1, c2, c3}, gotClients)

	// active
	gotClient, err := repo.GetActiveByID(c1.ID)
	assert.NoError(err)
	assert.Equal(c1, gotClient)

	// disconnected
	gotClient, err = repo.GetActiveByID(c2.ID)
	assert.NoError(err)
	assert.Nil(gotClient)

	deleted, err := repo.DeleteObsolete()
	assert.NoError(err)
	require.Len(t, deleted, 1)
	assert.Equal(c4, deleted[0])
	gotClients, err = repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*Client{c1, c2, c3}, gotClients)

	assert.NoError(repo.Delete(c3))
	gotClients, err = repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*Client{c1, c2}, gotClients)
}

func TestCRWithNoExpiration(t *testing.T) {
	now = nowMockF

	repo := NewClientRepository([]*Client{c1, c2, c3}, nil)
	c4Active := shallowCopy(c4)
	c4Active.Disconnected = nil

	assert := assert.New(t)
	assert.NoError(repo.Save(c4Active))

	gotCount, err := repo.Count()
	assert.NoError(err)
	assert.Equal(4, gotCount)

	gotClients, err := repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*Client{c1, c2, c3, c4Active}, gotClients)

	// active
	gotClient, err := repo.GetActiveByID(c1.ID)
	assert.NoError(err)
	assert.Equal(c1, gotClient)

	// disconnected
	gotClient, err = repo.GetActiveByID(c2.ID)
	assert.NoError(err)
	assert.Nil(gotClient)

	deleted, err := repo.DeleteObsolete()
	assert.NoError(err)
	assert.Len(deleted, 0)

	assert.NoError(repo.Delete(c4Active))
	gotClients, err = repo.GetAll()
	assert.NoError(err)
	assert.ElementsMatch([]*Client{c1, c2, c3}, gotClients)
}
