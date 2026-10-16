package internal

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"
)

// Most tests do not need a background reaper. Keeping it stopped isolates Get's
// expiration checks and avoids starting a permanent goroutine for every case.
func newCacheWithoutReaper() *PokeCache {
	return &PokeCache{
		Entries:      make(map[string]cacheEntry),
		ReapInterval: time.Hour,
		mutex:        &sync.RWMutex{},
	}
}

func TestNewPokeCache(t *testing.T) {
	cache := NewPokeCache(time.Hour)
	if cache == nil {
		t.Fatal("NewPokeCache returned nil")
	}
	if cache.Entries == nil || cache.mutex == nil {
		t.Fatal("constructor must initialize the entries and mutex")
	}
	if cache.ReapInterval != time.Hour {
		t.Fatalf("reap interval = %v, want %v", cache.ReapInterval, time.Hour)
	}
	cache.Add("key", []byte("value"), time.Hour)
	assertCacheValue(t, cache, "key", []byte("value"))
}

func TestPokeCacheAddGet(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value []byte
	}{
		{"URL", "https://example.com/path?page=2", []byte(`{"count":20}`)},
		{"empty key", "", []byte("value")},
		{"Unicode key", "pokémon/水", []byte("value")},
		{"binary data", "binary", []byte{0, 1, 127, 128, 255}},
		{"empty value", "empty", []byte{}},
		{"nil value", "nil", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cache := newCacheWithoutReaper()
			cache.Add(tc.key, tc.value, time.Hour)
			assertCacheValue(t, cache, tc.key, tc.value)
		})
	}
}

func TestPokeCacheMissingKey(t *testing.T) {
	cache := newCacheWithoutReaper()
	cache.Add("present", []byte("value"), time.Hour)
	for _, key := range []string{"missing", "", "Present"} {
		value, ok := cache.Get(key)
		if ok || value != nil {
			t.Errorf("Get(%q) = (%q, %v), want (nil, false)", key, value, ok)
		}
	}
}

func TestPokeCacheIndependentKeys(t *testing.T) {
	cache := newCacheWithoutReaper()
	cache.Add("first", []byte("one"), time.Hour)
	cache.Add("second", []byte("two"), time.Hour)
	cache.Add("first", []byte("updated"), time.Hour)
	assertCacheValue(t, cache, "first", []byte("updated"))
	assertCacheValue(t, cache, "second", []byte("two"))
}

func TestPokeCacheOverwrite(t *testing.T) {
	cases := []struct {
		name   string
		oldTTL time.Duration
		newTTL time.Duration
		wantOK bool
	}{
		{"replace live entry", time.Hour, time.Hour, true},
		{"renew expired entry", -time.Hour, time.Hour, true},
		{"expire live entry", time.Hour, -time.Hour, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cache := newCacheWithoutReaper()
			cache.Add("key", []byte("old"), tc.oldTTL)
			cache.Add("key", []byte("new"), tc.newTTL)
			if tc.wantOK {
				assertCacheValue(t, cache, "key", []byte("new"))
				return
			}
			value, ok := cache.Get("key")
			if ok || value != nil {
				t.Fatalf("Get after expired overwrite = (%q, %v), want (nil, false)", value, ok)
			}
		})
	}
}

func TestPokeCacheExpiredEntryBeforeCleanup(t *testing.T) {
	for _, ttl := range []time.Duration{0, -time.Nanosecond, -time.Hour} {
		t.Run(ttl.String(), func(t *testing.T) {
			cache := newCacheWithoutReaper()
			cache.Add("expired", []byte("old"), ttl)
			if !cacheContains(cache, "expired") {
				t.Fatal("test requires an expired entry still waiting for cleanup")
			}
			value, ok := cache.Get("expired")
			if ok || value != nil {
				t.Fatalf("Get expired entry = (%q, %v), want (nil, false)", value, ok)
			}
		})
	}
}

func TestPokeCacheExpiresOverTime(t *testing.T) {
	cache := newCacheWithoutReaper()
	cache.Add("short-lived", []byte("value"), 20*time.Millisecond)
	waitForCacheCondition(t, "entry to expire without a reaper", func() bool {
		value, ok := cache.Get("short-lived")
		return !ok && value == nil
	})
}

func TestPokeCacheGetDoesNotRefreshExpiration(t *testing.T) {
	cache := newCacheWithoutReaper()
	cache.Add("key", []byte("value"), time.Hour)
	cache.mutex.RLock()
	deadline := cache.Entries["key"].expiredAt
	cache.mutex.RUnlock()

	for i := 0; i < 3; i++ {
		assertCacheValue(t, cache, "key", []byte("value"))
	}
	cache.mutex.RLock()
	after := cache.Entries["key"].expiredAt
	cache.mutex.RUnlock()
	if !after.Equal(deadline) {
		t.Fatalf("Get changed expiration from %v to %v", deadline, after)
	}
}

func TestPokeCacheReapLoop(t *testing.T) {
	cache := NewPokeCache(2 * time.Millisecond)
	cache.Add("live", []byte("keep"), time.Hour)
	cache.Add("expired-first", []byte("remove"), -time.Hour)
	cache.Add("expired-second", []byte("remove"), -time.Hour)

	// Inspect storage directly: Get alone would also pass if cleanup were broken.
	waitForCacheCondition(t, "expired entries to be deleted", func() bool {
		return !cacheContains(cache, "expired-first") && !cacheContains(cache, "expired-second")
	})
	assertCacheValue(t, cache, "live", []byte("keep"))

	// New work after the first cleanup proves that the worker keeps running.
	cache.Add("expired-later", []byte("remove"), -time.Hour)
	waitForCacheCondition(t, "a later expired entry to be deleted", func() bool {
		return !cacheContains(cache, "expired-later")
	})
	assertCacheValue(t, cache, "live", []byte("keep"))
}

func TestPokeCacheConcurrentAccess(t *testing.T) {
	cache := NewPokeCache(time.Millisecond)
	cache.Add("stable", []byte("keep"), time.Hour)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			<-start
			for i := 0; i < 1000; i++ {
				key := fmt.Sprintf("shared-%d", i%4)
				if worker%2 == 0 {
					ttl := time.Hour
					if i%3 == 0 {
						ttl = -time.Hour
					}
					cache.Add(key, []byte(key), ttl)
				} else {
					value, ok := cache.Get(key)
					if ok && !bytes.Equal(value, []byte(key)) {
						t.Errorf("Get(%q) returned corrupt value %q", key, value)
						return
					}
				}
			}
		}(worker)
	}
	close(start)
	workers.Wait()

	assertCacheValue(t, cache, "stable", []byte("keep"))
	cache.Add("expired-after-workers", []byte("remove"), -time.Hour)
	waitForCacheCondition(t, "cleanup after concurrent access", func() bool {
		return !cacheContains(cache, "expired-after-workers")
	})
	cache.Add("shared-0", []byte("final"), time.Hour)
	assertCacheValue(t, cache, "shared-0", []byte("final"))
}

func FuzzPokeCacheRoundTrip(f *testing.F) {
	f.Add("", []byte{})
	f.Add("https://example.com", []byte(`{"count":20}`))
	f.Add("pokémon/水", []byte{0, 128, 255})
	f.Fuzz(func(t *testing.T, key string, value []byte) {
		cache := newCacheWithoutReaper()
		cache.Add(key, []byte("old"), time.Hour)
		cache.Add(key, value, time.Hour)
		assertCacheValue(t, cache, key, value)
		if got, ok := cache.Get(key + "\x00"); ok || got != nil {
			t.Fatalf("a distinct key returned (%q, %v), want (nil, false)", got, ok)
		}
	})
}

func assertCacheValue(t *testing.T, cache *PokeCache, key string, want []byte) {
	t.Helper()
	got, ok := cache.Get(key)
	if !ok || !bytes.Equal(got, want) {
		t.Fatalf("Get(%q) = (%q, %v), want (%q, true)", key, got, ok, want)
	}
}

func cacheContains(cache *PokeCache, key string) bool {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	_, ok := cache.Entries[key]
	return ok
}

func waitForCacheCondition(t *testing.T, description string, condition func() bool) {
	t.Helper()
	timeout := time.NewTimer(3 * time.Second)
	defer timeout.Stop()
	poll := time.NewTicker(time.Millisecond)
	defer poll.Stop()
	for {
		if condition() {
			return
		}
		select {
		case <-poll.C:
		case <-timeout.C:
			t.Fatalf("timed out waiting for %s", description)
		}
	}
}
