package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudradar-monitoring/rport/db/migration/jobs"
	"github.com/cloudradar-monitoring/rport/db/sqlite"
	"github.com/cloudradar-monitoring/rport/server/test/jb"
)

func TestCleanupJobsMultiJobs(t *testing.T) {
	ctx := context.Background()
	jobsDB, err := sqlite.New(":memory:", jobs.AssetNames(), jobs.Asset, DataSourceOptions)
	require.NoError(t, err)
	p := NewSqliteProvider(jobsDB, testLog)
	defer p.Close()

	mj1 := jb.NewMulti(t).Build()
	mj2 := jb.NewMulti(t).Build()
	mj3 := jb.NewMulti(t).Build()
	require.NoError(t, p.SaveMultiJob(mj1))
	require.NoError(t, p.SaveMultiJob(mj2))
	require.NoError(t, p.SaveMultiJob(mj3))

	j1 := jb.New(t).MultiJobID(mj1.JID).StartedAt(time.Now()).Build()
	j2 := jb.New(t).MultiJobID(mj1.JID).StartedAt(time.Now()).Build()
	j3 := jb.New(t).StartedAt(time.Now()).Build()
	j4 := jb.New(t).MultiJobID(mj2.JID).StartedAt(time.Now()).Build()
	j5 := jb.New(t).StartedAt(time.Now()).Build()
	j6 := jb.New(t).MultiJobID(mj2.JID).StartedAt(time.Now()).Build()
	j7 := jb.New(t).MultiJobID(mj3.JID).StartedAt(time.Now()).Build()
	j8 := jb.New(t).MultiJobID(mj3.JID).StartedAt(time.Now()).Build()
	require.NoError(t, p.SaveJob(j1))
	require.NoError(t, p.SaveJob(j2))
	require.NoError(t, p.SaveJob(j3))
	require.NoError(t, p.SaveJob(j4))
	require.NoError(t, p.SaveJob(j5))
	require.NoError(t, p.SaveJob(j6))
	require.NoError(t, p.SaveJob(j7))
	require.NoError(t, p.SaveJob(j8))

	err = p.CleanupJobsMultiJobs(ctx, 3)
	require.NoError(t, err)

	// mj1 and mj2 should be deleted
	mj, err := p.GetMultiJob(ctx, mj1.JID)
	require.NoError(t, err)
	assert.Nil(t, mj)
	mj, err = p.GetMultiJob(ctx, mj2.JID)
	require.NoError(t, err)
	assert.Nil(t, mj)
	mj, err = p.GetMultiJob(ctx, mj3.JID)
	require.NoError(t, err)
	assert.NotNil(t, mj)

	// deleted all except j5, j7 and j8
	j, err := p.GetByJID(j1.ClientID, j1.JID)
	require.NoError(t, err)
	assert.Nil(t, j)
	j, err = p.GetByJID(j2.ClientID, j2.JID)
	require.NoError(t, err)
	assert.Nil(t, j)
	j, err = p.GetByJID(j3.ClientID, j3.JID)
	require.NoError(t, err)
	assert.Nil(t, j)
	j, err = p.GetByJID(j4.ClientID, j4.JID)
	require.NoError(t, err)
	assert.Nil(t, j)
	j, err = p.GetByJID(j5.ClientID, j5.JID)
	require.NoError(t, err)
	assert.NotNil(t, j)
	j, err = p.GetByJID(j6.ClientID, j6.JID)
	require.NoError(t, err)
	assert.Nil(t, j)
	j, err = p.GetByJID(j7.ClientID, j7.JID)
	require.NoError(t, err)
	assert.NotNil(t, j)
	j, err = p.GetByJID(j8.ClientID, j8.JID)
	require.NoError(t, err)
	assert.NotNil(t, j)
}
