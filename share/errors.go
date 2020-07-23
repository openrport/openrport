package chshare

import (
	"errors"
	"sync"
)

// ErrorCollector type allows accumulation of errors in one place
// and operating on them as a single error.
// thread-safe
type ErrorCollector struct {
	errs []error
	lock sync.Mutex
}

// Add adds error to collection if err is not nil
func (c *ErrorCollector) Add(err error) {
	if err != nil {
		c.lock.Lock()
		c.errs = append(c.errs, err)
		c.lock.Unlock()
	}
}

// HasErrors returns true if there were any errors collected
func (c *ErrorCollector) HasErrors() bool {
	return len(c.errs) > 0
}

// Combine returns all collected errors as a single error
func (c *ErrorCollector) Combine() error {
	errStr := c.String()
	if errStr != "" {
		return errors.New(errStr)
	}
	return nil
}

// String returns string representation of ErrorCollector (all errors concatenated)
func (c *ErrorCollector) String() string {
	c.lock.Lock()
	defer c.lock.Unlock()

	result := ""
	for i, err := range c.errs {
		result += err.Error()
		if i != len(c.errs)-1 {
			result += "; "
		}
	}
	return result
}
