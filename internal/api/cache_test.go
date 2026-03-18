package api

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCache_SetGet(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: 10, DefaultTTL: time.Minute})

	c.Set("key1", "value1", 0)
	val, ok := c.Get("key1")
	if !ok {
		t.Fatal("expected key1 to exist")
	}
	if val != "value1" {
		t.Fatalf("expected value1, got %v", val)
	}

	// Non-existent key
	_, ok = c.Get("missing")
	if ok {
		t.Fatal("expected missing key to return false")
	}
}

func TestCache_TTLExpiry(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: 10, DefaultTTL: time.Minute})

	// Set with very short TTL
	c.Set("short", "data", 20*time.Millisecond)
	val, ok := c.Get("short")
	if !ok || val != "data" {
		t.Fatal("expected short-lived entry to be available immediately")
	}

	time.Sleep(30 * time.Millisecond)

	_, ok = c.Get("short")
	if ok {
		t.Fatal("expected short-lived entry to be expired after TTL")
	}

	// DefaultTTL (ttl=0 means use DefaultTTL)
	c2 := NewCache(CacheConfig{MaxEntries: 10, DefaultTTL: 20 * time.Millisecond})
	c2.Set("def", "val", 0)
	time.Sleep(30 * time.Millisecond)
	_, ok = c2.Get("def")
	if ok {
		t.Fatal("expected default TTL entry to expire")
	}
}

func TestCache_LRUEviction(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: 3, DefaultTTL: time.Minute})

	c.Set("a", 1, 0)
	c.Set("b", 2, 0)
	c.Set("c", 3, 0)

	// Access "a" to make it recently used
	c.Get("a")

	// Add "d" — should evict "b" (least recently used)
	c.Set("d", 4, 0)

	if c.Len() != 3 {
		t.Fatalf("expected 3 entries, got %d", c.Len())
	}

	_, ok := c.Get("b")
	if ok {
		t.Fatal("expected b to be evicted")
	}
	_, ok = c.Get("a")
	if !ok {
		t.Fatal("expected a to still exist (was recently accessed)")
	}
	_, ok = c.Get("c")
	if !ok {
		t.Fatal("expected c to still exist")
	}
	_, ok = c.Get("d")
	if !ok {
		t.Fatal("expected d to exist")
	}
}

func TestCache_MaxEntries(t *testing.T) {
	const max = 5
	c := NewCache(CacheConfig{MaxEntries: max, DefaultTTL: time.Minute})

	for i := 0; i < max+3; i++ {
		c.Set(fmt.Sprintf("key%d", i), i, 0)
	}

	if c.Len() != max {
		t.Fatalf("expected Len()=%d, got %d", max, c.Len())
	}
}

func TestCache_Invalidate(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: 10, DefaultTTL: time.Minute})

	c.Set("x", "foo", 0)
	c.Invalidate("x")

	_, ok := c.Get("x")
	if ok {
		t.Fatal("expected x to be invalidated")
	}

	// Invalidate non-existent key should not panic
	c.Invalidate("nonexistent")
}

func TestCache_InvalidatePrefix(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: 20, DefaultTTL: time.Minute})

	c.Set("gmail:inbox", 1, 0)
	c.Set("gmail:sent", 2, 0)
	c.Set("drive:files", 3, 0)
	c.Set("gmail:drafts", 4, 0)
	c.Set("calendar:events", 5, 0)

	c.InvalidatePrefix("gmail:")

	for _, key := range []string{"gmail:inbox", "gmail:sent", "gmail:drafts"} {
		_, ok := c.Get(key)
		if ok {
			t.Fatalf("expected %q to be invalidated by prefix", key)
		}
	}

	for _, key := range []string{"drive:files", "calendar:events"} {
		_, ok := c.Get(key)
		if !ok {
			t.Fatalf("expected %q to remain after prefix invalidation", key)
		}
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := NewCache(CacheConfig{MaxEntries: 64, DefaultTTL: time.Minute})

	const goroutines = 50
	const ops = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < ops; j++ {
				key := fmt.Sprintf("g%d-k%d", id, j%10)
				c.Set(key, j, 0)
				c.Get(key)
				if j%5 == 0 {
					c.Invalidate(key)
				}
			}
		}(i)
	}

	wg.Wait()
	// No panic = pass; Len() must be within bounds
	if c.Len() > 64 {
		t.Fatalf("cache exceeded MaxEntries: %d", c.Len())
	}
}
