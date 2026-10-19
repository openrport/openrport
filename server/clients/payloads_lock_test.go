package clients

import (
	"sync"
	"testing"
	"time"

	"github.com/openrport/openrport/server/clients/clientdata"
)

// GET /api/v1/clients/{id} used to take the client read lock once per field;
// a writer queued between those locks deadlocked the client and every request on it.
func TestConvertToClientPayloadDoesNotDeadlockWithConcurrentWriter(t *testing.T) {
	c := &clientdata.Client{ID: "c1", Address: "1.2.3.4", Hostname: "host1", ClientAuthID: "auth1"}
	calculated := c.ToCalculated(nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			for i := 0; i < 5000; i++ {
				ConvertToClientPayload(calculated, nil)
			}
		}()
		go func() {
			defer wg.Done()
			for i := 0; i < 5000; i++ {
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
		t.Fatal("ConvertToClientPayload deadlocked against a concurrent writer")
	}
}
