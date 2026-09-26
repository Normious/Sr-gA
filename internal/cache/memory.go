package cache

import (
	"sync"
	"time"
)

type entry struct {
	value     []byte
	expiresAt time.Time
}

type MemoryCache struct {
	mu       sync.RWMutex
	items    map[string]entry
	maxItems int
	ttl      time.Duration
}

func New(maxItems int, ttl time.Duration) *MemoryCache {
	c := &MemoryCache{
		items:    make(map[string]entry),
		maxItems: maxItems,
		ttl:      ttl,
	}
	go c.cleanupLoop()
	return c
}

func (c *MemoryCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

func (c *MemoryCache) Put(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxItems {
		for k := range c.items {
			delete(c.items, k)
			break
		}
	}

	c.items[key] = entry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *MemoryCache) Stats() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return map[string]any{
		"size":        len(c.items),
		"max_size":    c.maxItems,
		"ttl_seconds": int(c.ttl.Seconds()),
	}
}

func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, e := range c.items {
			if now.After(e.expiresAt) {
				delete(c.items, k)
			}
		}
		c.mu.Unlock()
	}
}
