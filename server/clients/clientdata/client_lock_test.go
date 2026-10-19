package clientdata

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/openrport/openrport/server/cgroups"
)

func connectionStateGroup(state string) *cgroups.ClientGroup {
	return &cgroups.ClientGroup{
		ID:     "group-" + state,
		Params: &cgroups.ClientParams{ConnectionState: &cgroups.ParamValues{cgroups.Param(state)}},
	}
}

func TestBelongsToMatchesConnectionState(t *testing.T) {
	connected := NewTestClient("c1", "", "host1", "", nil)
	disconnected := NewTestClient("c2", "", "host2", "", nil)
	disconnected.SetDisconnectedNow()

	assert.True(t, connected.BelongsTo(connectionStateGroup("connected")))
	assert.False(t, connected.BelongsTo(connectionStateGroup("disconnected")))
	assert.True(t, disconnected.BelongsTo(connectionStateGroup("disconnected")))
	assert.False(t, disconnected.BelongsTo(connectionStateGroup("connected")))
}

// BelongsTo holds the client read lock; re-acquiring it while a writer is queued
// deadlocked the whole server (every request touching the client hung forever).
func TestBelongsToDoesNotDeadlockWithConcurrentWriter(t *testing.T) {
	c := NewTestClient("c1", "", "host1", "", nil)
	group := connectionStateGroup("connected")

	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 20000; i++ {
				c.BelongsTo(group)
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 20000; i++ {
				if i%2 == 0 {
					c.SetDisconnectedNow()
				} else {
					c.SetDisconnectedAt(nil)
				}
			}
		}()
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("BelongsTo deadlocked against a concurrent writer")
	}
}
