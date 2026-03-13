package cache

import "sync"

type ChunkCache struct {
	mu    sync.RWMutex
	items map[string][]byte
}

func NewChunkCache() *ChunkCache {
	return &ChunkCache{items: map[string][]byte{}}
}

func (c *ChunkCache) Set(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = value
}

func (c *ChunkCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[key]
	return v, ok
}
