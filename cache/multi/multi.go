package multi

import (
	"github.com/status-im/proxy-common/cache"
	"github.com/status-im/proxy-common/models"
)

// Ensure MultiCache implements cache.Cache and cache.LevelAwareCache
var _ cache.Cache = (*MultiCache)(nil)
var _ cache.LevelAwareCache = (*MultiCache)(nil)

// MultiCache implements a composite cache that tries multiple cache implementations
// It attempts to get/set values through an array of cache interfaces in order
type MultiCache struct {
	caches            []cache.Cache
	logger            cache.Logger
	enablePropagation bool
}

// Option is a functional option for configuring MultiCache
type Option func(*MultiCache)

// WithLogger sets the logger for MultiCache
func WithLogger(logger cache.Logger) Option {
	return func(mc *MultiCache) {
		mc.logger = logger
	}
}

// NewMultiCache creates a new MultiCache instance with provided cache implementations
func NewMultiCache(caches []cache.Cache, enablePropagation bool, opts ...Option) cache.LevelAwareCache {
	mc := &MultiCache{
		caches:            caches,
		logger:            cache.NoopLogger{},
		enablePropagation: enablePropagation,
	}

	for _, opt := range opts {
		opt(mc)
	}

	return mc
}

// Get retrieves value from the first available cache that has the key
func (mc *MultiCache) Get(key string) (*models.CacheEntry, bool) {
	result := mc.GetWithLevel(key)
	return result.Entry, result.Found
}

// GetStale retrieves stale value from the first available cache that has the key
func (mc *MultiCache) GetStale(key string) (*models.CacheEntry, bool) {
	result := mc.GetStaleWithLevel(key)
	return result.Entry, result.Found
}

// Set stores value in all available caches
func (mc *MultiCache) Set(key string, val []byte, ttl models.TTL) {
	if len(mc.caches) == 0 {
		mc.logger.Warn("No caches available for set operation", "key", key)
		return
	}

	for _, c := range mc.caches {
		c.Set(key, val, ttl)
	}
}

// Delete removes entry from all available caches
func (mc *MultiCache) Delete(key string) {
	if len(mc.caches) == 0 {
		mc.logger.Warn("No caches available for delete operation", "key", key)
		return
	}

	for _, c := range mc.caches {
		c.Delete(key)
	}
}

// GetCacheCount returns the number of caches in the multi-cache
func (mc *MultiCache) GetCacheCount() int {
	return len(mc.caches)
}

// GetWithLevel retrieves value from cache with level information
func (mc *MultiCache) GetWithLevel(key string) *models.CacheResult {
	if len(mc.caches) == 0 {
		mc.logger.Warn("No caches available for get operation", "key", key)
		return &models.CacheResult{
			Entry: nil,
			Found: false,
			Level: models.CacheLevelMiss,
		}
	}

	for i, c := range mc.caches {
		entry, found := c.Get(key)
		if found {
			if i > 0 && mc.enablePropagation {
				mc.propagateToEarlierCaches(key, entry, i)
			}

			level := models.CacheLevelFromIndex(i)

			return &models.CacheResult{
				Entry: entry,
				Found: true,
				Level: level,
			}
		}
	}

	return &models.CacheResult{
		Entry: nil,
		Found: false,
		Level: models.CacheLevelMiss,
	}
}

// GetStaleWithLevel retrieves stale value from cache with level information
func (mc *MultiCache) GetStaleWithLevel(key string) *models.CacheResult {
	if len(mc.caches) == 0 {
		return &models.CacheResult{
			Entry: nil,
			Found: false,
			Level: models.CacheLevelMiss,
		}
	}

	for i, c := range mc.caches {
		entry, found := c.GetStale(key)
		if found {
			if i > 0 && mc.enablePropagation {
				mc.propagateToEarlierCaches(key, entry, i)
			}

			level := models.CacheLevelFromIndex(i)

			return &models.CacheResult{
				Entry: entry,
				Found: true,
				Level: level,
			}
		}
	}

	return &models.CacheResult{
		Entry: nil,
		Found: false,
		Level: models.CacheLevelMiss,
	}
}

// propagateToEarlierCaches propagates a cache entry to earlier caches with adjusted TTL
func (mc *MultiCache) propagateToEarlierCaches(key string, entry *models.CacheEntry, foundAtIndex int) {
	if entry == nil || entry.IsExpired() {
		return
	}
	remainingTTL := entry.RemainingTTL()

	// Only propagate if there's meaningful time left
	if remainingTTL.Fresh <= 0 && remainingTTL.Stale <= 0 {
		return
	}

	for i := 0; i < foundAtIndex; i++ {
		mc.caches[i].Set(key, entry.Data, remainingTTL)
	}
}
