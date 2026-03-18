package api

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

// CacheConfig holds configuration for a Cache instance.
type CacheConfig struct {
	MaxEntries int           // LRU capacity; defaults to 256 if zero
	DefaultTTL time.Duration // TTL used when Set is called with ttl=0; defaults to 5 minutes
}

// cacheEntry is the value stored inside the doubly-linked list.
type cacheEntry struct {
	key       string
	value     interface{}
	expiresAt time.Time
}

// Cache is a thread-safe LRU cache with per-entry TTL support.
// It uses container/list as the eviction list and a map for O(1) lookups.
type Cache struct {
	mu       sync.Mutex
	entries  map[string]*list.Element
	eviction *list.List
	config   CacheConfig
}

// NewCache creates a Cache with the given configuration.
// Zero values in cfg are replaced with sensible defaults:
//   - MaxEntries 0 → 256
//   - DefaultTTL 0 → 5 minutes
func NewCache(cfg CacheConfig) *Cache {
	if cfg.MaxEntries <= 0 {
		cfg.MaxEntries = 256
	}
	if cfg.DefaultTTL <= 0 {
		cfg.DefaultTTL = 5 * time.Minute
	}
	return &Cache{
		entries:  make(map[string]*list.Element),
		eviction: list.New(),
		config:   cfg,
	}
}

// Get returns the value for key and true if it exists and has not expired.
// Expired entries are lazily removed on access.
// On a cache hit the entry is promoted to the front of the LRU list.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	entry := elem.Value.(*cacheEntry)
	if time.Now().After(entry.expiresAt) {
		// Lazy expiry
		c.eviction.Remove(elem)
		delete(c.entries, key)
		return nil, false
	}

	// Promote to front (most recently used)
	c.eviction.MoveToFront(elem)
	return entry.value, true
}

// Set stores value under key with the given TTL.
// If ttl is 0 the cache's DefaultTTL is used.
// When the cache is at capacity, the least recently used entry is evicted.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	if ttl <= 0 {
		ttl = c.config.DefaultTTL
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Update existing entry
	if elem, ok := c.entries[key]; ok {
		entry := elem.Value.(*cacheEntry)
		entry.value = value
		entry.expiresAt = time.Now().Add(ttl)
		c.eviction.MoveToFront(elem)
		return
	}

	// Evict LRU if at capacity
	if c.eviction.Len() >= c.config.MaxEntries {
		c.evictLRU()
	}

	entry := &cacheEntry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	elem := c.eviction.PushFront(entry)
	c.entries[key] = elem
}

// evictLRU removes the least recently used entry from the cache.
// Must be called with c.mu held.
func (c *Cache) evictLRU() {
	back := c.eviction.Back()
	if back == nil {
		return
	}
	entry := back.Value.(*cacheEntry)
	c.eviction.Remove(back)
	delete(c.entries, entry.key)
}

// Invalidate removes the entry for key, if present.
func (c *Cache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[key]
	if !ok {
		return
	}
	c.eviction.Remove(elem)
	delete(c.entries, key)
}

// InvalidatePrefix removes all entries whose keys begin with prefix.
func (c *Cache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, elem := range c.entries {
		if strings.HasPrefix(key, prefix) {
			c.eviction.Remove(elem)
			delete(c.entries, key)
		}
	}
}

// Len returns the number of entries currently in the cache (including not-yet-expired ones).
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.eviction.Len()
}
