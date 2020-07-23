package chshare

import (
	"sync"
)

type CallFn func() error

// SyncCall executes all functions passed as arguments returning the combined error
func SyncCall(fns ...CallFn) error {
	var wg = sync.WaitGroup{}
	wg.Add(len(fns))
	var errs ErrorCollector

	for _, currFn := range fns {
		fn := currFn
		go func() {
			errs.Add(fn())
		}()
	}

	wg.Wait()

	return errs.Combine()
}
