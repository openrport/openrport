package vault

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDbProviderNotInit(t *testing.T) {
	notInitDbProvider := &DbProviderMock{
		statusToGive: DbStatus{StatusName: "not init db"},
	}
	initDbProvider := &DbProviderMock{
		statusToGive: DbStatus{StatusName: "init db"},
	}
	dbProvFactory := NewStatefulDbProviderFactory(
		func() (DbProvider, error) {
			return initDbProvider, nil
		},
		notInitDbProvider,
	)

	actualDBProvider := dbProvFactory.GetDbProvider()
	actualDBProviderStatus, err := actualDBProvider.GetStatus(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "not init db", actualDBProviderStatus.StatusName)

	err = dbProvFactory.Init()
	require.NoError(t, err)

	actualDBProvider2 := dbProvFactory.GetDbProvider()
	actualDBProviderStatus2, err := actualDBProvider2.GetStatus(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "init db", actualDBProviderStatus2.StatusName)
}

func TestDbProviderBuildError(t *testing.T) {
	dbProvFactory := NewStatefulDbProviderFactory(
		func() (DbProvider, error) {
			return nil, errors.New("some error")
		},
		&DbProviderMock{},
	)

	err := dbProvFactory.Init()
	require.EqualError(t, err, "some error")
}
