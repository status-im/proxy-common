package l1

import (
	"context"
	"encoding/json"
	"time"

	"github.com/allegro/bigcache/v3"

	"github.com/status-im/proxy-common/cache"
	"github.com/status-im/proxy-common/models"
	"github.com/status-im/proxy-common/scheduler"
)

// Ensure BigCache implements cache.Cache
var _ cache.Cache = (*BigCache)(nil)

// BigCache implements L1 cache using BigCache
type BigCache struct {
	cache            *bigcache.BigCache
	logger           cache.Logger
	metrics          cache.MetricsRecorder
	metricsScheduler *scheduler.Scheduler
	maxEntrySize     int
}

// Option is a functional option for configuring BigCache
type Option func(*BigCache)

// WithLogger sets the logger for BigCache
func WithLogger(logger cache.Logger) Option {
	return func(bc *BigCache) {
		bc.logger = logger
	}
}

// WithMetrics sets the metrics recorder for BigCache
func WithMetrics(metrics cache.MetricsRecorder) Option {
	return func(bc *BigCache) {
		bc.metrics = metrics
	}
}

// NewBigCache creates a new BigCache instance
func NewBigCache(cfg *cache.BigCacheConfig, opts ...Option) (cache.Cache, error) {
	cfg.ApplyDefaults()

	config := bigcache.DefaultConfig(10 * time.Minute)
	config.HardMaxCacheSize = cfg.Size
	config.Verbose = false
	config.MaxEntrySize = cfg.MaxEntrySize
	config.Shards = cfg.Shards

	c, err := bigcache.New(context.Background(), config)
	if err != nil {
		return nil, err
	}

	bc := &BigCache{
		cache:        c,
		logger:       cache.NoopLogger{},
		metrics:      cache.NoopMetrics{},
		maxEntrySize: cfg.MaxEntrySize,
	}

	for _, opt := range opts {
		opt(bc)
	}

	bc.startMetricsCollection()

	return bc, nil
}

// Get retrieves value from cache with freshness information
func (bc *BigCache) Get(key string) (*models.CacheEntry, bool) {
	data, err := bc.cache.Get(key)
	if err != nil {
		return nil, false
	}

	var entry models.CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		bc.logger.Warn("Failed to unmarshal L1 cache entry", "key", key, "error", err)
		bc.metrics.RecordCacheError("l1", "decode")
		_ = bc.cache.Delete(key)
		return nil, false
	}

	if entry.IsExpired() {
		_ = bc.cache.Delete(key)
		return nil, false
	}

	return &entry, true
}

// GetStale retrieves value from cache regardless of freshness (for stale-if-error)
func (bc *BigCache) GetStale(key string) (*models.CacheEntry, bool) {
	data, err := bc.cache.Get(key)
	if err != nil {
		return nil, false
	}

	var entry models.CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		_ = bc.cache.Delete(key)
		return nil, false
	}

	if entry.IsExpired() {
		_ = bc.cache.Delete(key)
		return nil, false
	}

	return &entry, true
}

// Set stores value in cache with TTL
func (bc *BigCache) Set(key string, val []byte, ttl models.TTL) {
	now := time.Now().Unix()

	entry := models.CacheEntry{
		Data:      val,
		CreatedAt: now,
		StaleAt:   now + int64(ttl.Fresh.Seconds()),
		ExpiresAt: now + int64(ttl.Fresh.Seconds()) + int64(ttl.Stale.Seconds()),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		bc.logger.Error("Failed to marshal cache entry", "key", key, "error", err)
		bc.metrics.RecordCacheError("l1", "encode")
		return
	}

	if len(data) > bc.maxEntrySize {
		bc.logger.Warn("Cache entry too large, skipping L1 cache",
			"key", key,
			"size", len(data),
			"max_size", bc.maxEntrySize)
		bc.metrics.RecordCacheError("l1", "entry_too_large")
		return
	}

	err = bc.cache.Set(key, data)
	if err != nil {
		bc.logger.Error("Failed to set cache entry", "key", key, "error", err)
		bc.metrics.RecordCacheError("l1", "upstream")
		return
	}
}

// Delete removes entry from cache
func (bc *BigCache) Delete(key string) {
	_ = bc.cache.Delete(key)
}

// Close closes the cache
func (bc *BigCache) Close() error {
	bc.stopMetricsCollection()

	return bc.cache.Close()
}

// GetStats returns cache statistics for metrics
func (bc *BigCache) GetStats() (capacity, used int64) {
	stats := bc.cache.Stats()
	// BigCache doesn't expose exact capacity, but we can use the configured size
	// Convert from MB to bytes
	capacity = int64(bc.cache.Capacity())   // This returns the configured size in bytes
	used = int64(stats.Hits + stats.Misses) // Approximate usage based on operations

	return capacity, used
}

// startMetricsCollection starts periodic metrics collection
func (bc *BigCache) startMetricsCollection() {
	bc.metricsScheduler = scheduler.New(30*time.Second, bc.updateMetrics)
	bc.metricsScheduler.Start()

	bc.updateMetrics()

	bc.logger.Debug("Started L1 cache metrics collection")
}

// stopMetricsCollection stops periodic metrics collection
func (bc *BigCache) stopMetricsCollection() {
	if bc.metricsScheduler != nil {
		bc.metricsScheduler.Stop()
		bc.logger.Debug("Stopped L1 cache metrics collection")
	}
}

// updateMetrics updates cache metrics
func (bc *BigCache) updateMetrics() {
	capacity, used := bc.GetStats()

	bc.metrics.UpdateL1CacheCapacity(capacity, used)

	stats := bc.cache.Stats()
	totalOps := stats.Hits + stats.Misses
	bc.metrics.UpdateCacheKeys("l1", int64(totalOps))
}
