package users

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserCache(t *testing.T) {
	u1 := &User{Username: "u1", Password: "p1"}
	u2 := &User{Username: "u2", Password: "p2"}
	u3 := &User{Username: "u3", Password: "p3"}

	c := NewUserCache([]*User{u1, u2})

	u, err := c.GetByUsername("u1")
	require.NoError(t, err)
	assert.Equal(t, u1, u)

	u, err = c.GetByUsername("u2")
	require.NoError(t, err)
	assert.Equal(t, u2, u)

	u, err = c.GetByUsername("u3")
	require.NoError(t, err)
	assert.Nil(t, u)

	users, err := c.GetAll()
	require.NoError(t, err)
	assert.ElementsMatch(t, []*User{u1, u2}, users)

	c.Load([]*User{u2, u3})

	u, err = c.GetByUsername("u1")
	require.NoError(t, err)
	assert.Nil(t, u)

	u, err = c.GetByUsername("u2")
	require.NoError(t, err)
	assert.Equal(t, u2, u)

	u, err = c.GetByUsername("u3")
	require.NoError(t, err)
	assert.Equal(t, u3, u)

	users, err = c.GetAll()
	require.NoError(t, err)
	assert.ElementsMatch(t, []*User{u2, u3}, users)
}
