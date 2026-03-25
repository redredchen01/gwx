package skill

import (
	"sync"
	"time"
)

var (
	cachedSkills []*Skill
	cacheTime    time.Time
	cacheMu      sync.RWMutex
	cacheTTL     = 30 * time.Second
)

// CachedLoadAll returns cached skills if fresh, otherwise reloads from disk.
// Thread-safe: multiple goroutines can call this concurrently.
func CachedLoadAll() ([]*Skill, error) {
	cacheMu.RLock()
	if cachedSkills != nil && time.Since(cacheTime) < cacheTTL {
		result := make([]*Skill, len(cachedSkills))
		copy(result, cachedSkills)
		cacheMu.RUnlock()
		return result, nil
	}
	cacheMu.RUnlock()

	// Reload from disk.
	skills, err := LoadAll()
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	cachedSkills = skills
	cacheTime = time.Now()
	cacheMu.Unlock()

	return skills, nil
}

// InvalidateSkillCache forces the next CachedLoadAll call to reload from disk.
func InvalidateSkillCache() {
	cacheMu.Lock()
	cachedSkills = nil
	cacheMu.Unlock()
}
