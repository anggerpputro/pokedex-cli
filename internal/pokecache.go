package internal

import (
	"sync"
	"time"
)

type PokeCache struct {
	Entries      map[string]cacheEntry
	ReapInterval time.Duration
	mutex        *sync.RWMutex
}

type cacheEntry struct {
	createdAt time.Time
	expiredAt time.Time
	val       []byte
}

func NewPokeCache(reapInterval time.Duration) *PokeCache {
	cache := PokeCache{
		Entries:      make(map[string]cacheEntry),
		ReapInterval: reapInterval,
		mutex:        &sync.RWMutex{},
	}
	go cache.reapLoop()
	return &cache
}

func (c *PokeCache) Add(key string, val []byte, duration time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cacheEntry := cacheEntry{
		createdAt: time.Now(),
		expiredAt: time.Now().Add(duration),
		val:       val,
	}
	c.Entries[key] = cacheEntry
}

func (c *PokeCache) Get(key string) ([]byte, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cacheEntry, ok := c.Entries[key]
	// Check if key is exists and current time is AFTER the expiration time
	if ok && time.Now().After(cacheEntry.expiredAt) {
		return nil, false
	}
	return cacheEntry.val, ok
}

func (c *PokeCache) reapLoop() {
	ticker := time.NewTicker(c.ReapInterval)
	defer ticker.Stop()
	for {
		_, ok := <-ticker.C
		if ok {
			// fmt.Println("Current time: ", t)
			c.mutex.Lock()

			for k, v := range c.Entries {
				if time.Now().After(v.expiredAt) {
					delete(c.Entries, k)
				}
			}

			c.mutex.Unlock()
		} else {
			// fmt.Println("Stopping reapLoop")
			return
		}
	}
}
