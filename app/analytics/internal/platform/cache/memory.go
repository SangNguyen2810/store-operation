package cache

import (
	"sync"
	"time"
)

// maxEntries bounds the in-process copy, so a long outage cannot grow memory.
const maxEntries = 500

type entry struct {
	val     []byte
	expires time.Time
}

// memory is the API's own copy of what it recently read from or wrote to the
// redis service. It is served only while that service is unavailable. A full
// map drops the entry closest to expiry.
type memory struct {
	mu  sync.Mutex
	now func() time.Time
	m   map[string]entry
}

func newMemory(now func() time.Time) *memory {
	return &memory{now: now, m: make(map[string]entry)}
}

func (c *memory) set(key string, val []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if _, ok := c.m[key]; !ok && len(c.m) >= maxEntries {
		c.evict(now)
	}
	c.m[key] = entry{val: val, expires: now.Add(ttl)}
}

// evict removes expired entries, or if there are none the one expiring first.
func (c *memory) evict(now time.Time) {
	var oldest string
	var oldestAt time.Time
	removed := false
	for k, e := range c.m {
		if !e.expires.After(now) {
			delete(c.m, k)
			removed = true
			continue
		}
		if oldest == "" || e.expires.Before(oldestAt) {
			oldest, oldestAt = k, e.expires
		}
	}
	if !removed {
		delete(c.m, oldest)
	}
}

func (c *memory) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.m[key]
	if !ok {
		return nil, false
	}
	if !e.expires.After(c.now()) {
		delete(c.m, key)
		return nil, false
	}
	return e.val, true
}

func (c *memory) delete(key string) {
	c.mu.Lock()
	delete(c.m, key)
	c.mu.Unlock()
}

func (c *memory) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.m)
}
