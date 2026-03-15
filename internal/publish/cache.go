package publish

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Default cache TTL: 5 minutes
var cacheTTL = 5 * time.Minute

// cacheEntry wraps cached data with a creation timestamp for TTL expiration.
type cacheEntry[T any] struct {
	Data      T
	CreatedAt time.Time
}

// isExpired returns true if the entry is older than the cache TTL.
func (e *cacheEntry[T]) isExpired() bool {
	return time.Since(e.CreatedAt) > cacheTTL
}

// publishCache holds the in-memory cache for aggregated publish data.
var pubCache = struct {
	mu        sync.RWMutex
	liveCache map[uint]cacheEntry[[]AggregatedChannel]
	epgCache  map[uint]cacheEntry[*AggregatedEPG]
}{
	liveCache: make(map[uint]cacheEntry[[]AggregatedChannel]),
	epgCache:  make(map[uint]cacheEntry[*AggregatedEPG]),
}

// keyMutexes provides per-interface-ID mutexes to prevent cache stampede.
// When multiple goroutines hit a cache miss for the same interface simultaneously,
// only one performs the expensive aggregation while others wait and reuse the result.
var keyMutexes sync.Map // map[string]*sync.Mutex

// getKeyMutex returns the mutex for the given cache key (type + interface ID).
// Creates one if it doesn't exist, using sync.Map for thread safety.
func getKeyMutex(key string) *sync.Mutex {
	val, _ := keyMutexes.LoadOrStore(key, &sync.Mutex{})
	return val.(*sync.Mutex)
}

// getLiveChannels returns the cached live channels for the given interface ID.
// Returns nil and false on cache miss or TTL expiration.
func getLiveChannels(ifaceID uint) ([]AggregatedChannel, bool) {
	pubCache.mu.RLock()
	defer pubCache.mu.RUnlock()

	entry, ok := pubCache.liveCache[ifaceID]
	if !ok || entry.isExpired() {
		return nil, false
	}
	return entry.Data, true
}

// setLiveChannels stores the aggregated live channels for the given interface ID.
func setLiveChannels(ifaceID uint, channels []AggregatedChannel) {
	pubCache.mu.Lock()
	defer pubCache.mu.Unlock()

	pubCache.liveCache[ifaceID] = cacheEntry[[]AggregatedChannel]{
		Data:      channels,
		CreatedAt: time.Now(),
	}
}

// getEPGPrograms returns the cached EPG data for the given interface ID.
// Returns nil and false on cache miss or TTL expiration.
func getEPGPrograms(ifaceID uint) (*AggregatedEPG, bool) {
	pubCache.mu.RLock()
	defer pubCache.mu.RUnlock()

	entry, ok := pubCache.epgCache[ifaceID]
	if !ok || entry.isExpired() {
		return nil, false
	}
	return entry.Data, true
}

// setEPGPrograms stores the aggregated EPG data for the given interface ID.
func setEPGPrograms(ifaceID uint, epg *AggregatedEPG) {
	pubCache.mu.Lock()
	defer pubCache.mu.Unlock()

	pubCache.epgCache[ifaceID] = cacheEntry[*AggregatedEPG]{
		Data:      epg,
		CreatedAt: time.Now(),
	}
}

// LoadOrStoreLiveChannels returns cached live channels, or calls the loader to
// populate the cache on a miss. Uses per-key mutex with double-checked locking
// to prevent cache stampede: only one goroutine runs the loader for a given
// interface ID while others wait and reuse the result.
func LoadOrStoreLiveChannels(ifaceID uint, loader func() ([]AggregatedChannel, error)) ([]AggregatedChannel, error) {
	// Fast path: read lock only
	if channels, ok := getLiveChannels(ifaceID); ok {
		return channels, nil
	}

	// Slow path: acquire per-key mutex to prevent stampede
	mu := getKeyMutex(fmt.Sprintf("live:%d", ifaceID))
	mu.Lock()
	defer mu.Unlock()

	// Double-check: another goroutine may have populated the cache while we waited
	if channels, ok := getLiveChannels(ifaceID); ok {
		return channels, nil
	}

	// Actually load from DB
	channels, err := loader()
	if err != nil {
		return nil, err
	}

	setLiveChannels(ifaceID, channels)
	return channels, nil
}

// LoadOrStoreEPGPrograms returns cached EPG data, or calls the loader to
// populate the cache on a miss. Uses per-key mutex with double-checked locking
// to prevent cache stampede.
func LoadOrStoreEPGPrograms(ifaceID uint, loader func() (*AggregatedEPG, error)) (*AggregatedEPG, error) {
	// Fast path: read lock only
	if epg, ok := getEPGPrograms(ifaceID); ok {
		return epg, nil
	}

	// Slow path: acquire per-key mutex to prevent stampede
	mu := getKeyMutex(fmt.Sprintf("epg:%d", ifaceID))
	mu.Lock()
	defer mu.Unlock()

	// Double-check: another goroutine may have populated the cache while we waited
	if epg, ok := getEPGPrograms(ifaceID); ok {
		return epg, nil
	}

	// Actually load from DB
	epg, err := loader()
	if err != nil {
		return nil, err
	}

	setEPGPrograms(ifaceID, epg)
	return epg, nil
}

// InvalidateAll clears all cached publish data.
// Should be called whenever underlying data changes (source sync, logo CRUD,
// rule CRUD, publish interface CRUD, channel detection, etc.).
func InvalidateAll() {
	pubCache.mu.Lock()
	defer pubCache.mu.Unlock()

	pubCache.liveCache = make(map[uint]cacheEntry[[]AggregatedChannel])
	pubCache.epgCache = make(map[uint]cacheEntry[*AggregatedEPG])
	keyMutexes.Clear()
	slog.Debug("Publish cache invalidated")
}
