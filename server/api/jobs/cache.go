package jobs

import (
	"sync"

	"github.com/cloudradar-monitoring/rport/share/models"
)

// JobCache is a thread-safe in-memory cache.
type JobCache struct {
	jobs map[string]*models.Job
	mu   sync.RWMutex
}

// NewEmptyJobCache returns a thread-safe empty cache.
func NewEmptyJobCache() *JobCache {
	return &JobCache{jobs: map[string]*models.Job{}}
}

func (c *JobCache) Get(key string) *models.Job {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.jobs[key]
}

func (c *JobCache) GetAll() []*models.Job {
	c.mu.RLock()
	defer c.mu.RUnlock()
	res := make([]*models.Job, 0, len(c.jobs))
	for _, v := range c.jobs {
		res = append(res, v)
	}
	return res
}

func (c *JobCache) Set(key string, job *models.Job) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.jobs[key] = job
}

func (c *JobCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.jobs, key)
}

func (c *JobCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.jobs)
}
