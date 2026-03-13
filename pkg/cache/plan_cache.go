package cache

import "sync"

type PlanCache struct {
	mu    sync.RWMutex
	items map[string]string
}

func NewPlanCache() *PlanCache {
	return &PlanCache{items: map[string]string{}}
}

func (c *PlanCache) Set(key string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}

func (c *PlanCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[key]
	return v, ok
}
