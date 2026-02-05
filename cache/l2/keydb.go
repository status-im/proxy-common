package l2

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/status-im/proxy-common/cache"
	"github.com/status-im/proxy-common/models"
)

// Ensure KeyDBCache implements cache.Cache
var _ cache.Cache = (*KeyDBCache)(nil)

// KeyDBCache implements L2 cache using Redis/KeyDB
type KeyDBCache struct {
	client  cache.KeyDbClient
	cfg     *cache.KeyDBConfig
	logger  cache.Logger
	metrics cache.MetricsRecorder
}

// Option is a functional option for configuring KeyDBCache
type Option func(*KeyDBCache)

// WithLogger sets the logger for KeyDBCache
func WithLogger(logger cache.Logger) Option {
	return func(kc *KeyDBCache) {
		kc.logger = logger
	}
}

// WithMetrics sets the metrics recorder for KeyDBCache
func WithMetrics(metrics cache.MetricsRecorder) Option {
	return func(kc *KeyDBCache) {
		kc.metrics = metrics
	}
}

// NewKeyDBCache creates a new KeyDBCache instance with provided client
func NewKeyDBCache(cfg *cache.KeyDBConfig, client cache.KeyDbClient, opts ...Option) cache.Cache {
	cfg.ApplyDefaults()

	kc := &KeyDBCache{
		client:  client,
		cfg:     cfg,
		logger:  cache.NoopLogger{},
		metrics: cache.NoopMetrics{},
	}

	for _, opt := range opts {
		opt(kc)
	}

	return kc
}

// Get retrieves value from KeyDB cache with freshness information
func (kc *KeyDBCache) Get(key string) (*models.CacheEntry, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), kc.cfg.Connection.ReadTimeout)
	defer cancel()

	data, err := kc.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false
		}
		kc.logger.Warn("L2 cache get failed", "key", key, "error", err)
		return nil, false
	}

	var entry models.CacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		kc.logger.Error("Failed to unmarshal L2 cache entry", "key", key, "error", err)
		kc.metrics.RecordCacheError("l2", "decode")
		kc.client.Del(context.Background(), key)
		return nil, false
	}

	if entry.IsExpired() {
		kc.client.Del(context.Background(), key)
		return nil, false
	}

	return &entry, true
}

// GetStale retrieves value from KeyDB cache regardless of freshness
func (kc *KeyDBCache) GetStale(key string) (*models.CacheEntry, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), kc.cfg.Connection.ReadTimeout)
	defer cancel()

	data, err := kc.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, false
		}
		kc.logger.Warn("L2 cache stale get failed", "key", key, "error", err)
		return nil, false
	}

	var entry models.CacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		kc.logger.Error("Failed to unmarshal L2 cache entry for stale get", "key", key, "error", err)
		kc.client.Del(context.Background(), key)
		return nil, false
	}

	if entry.IsExpired() {
		kc.client.Del(context.Background(), key)
		return nil, false
	}

	return &entry, true
}

// Set stores value in KeyDB cache with TTL
func (kc *KeyDBCache) Set(key string, val []byte, ttl models.TTL) {
	ctx, cancel := context.WithTimeout(context.Background(), kc.cfg.Connection.SendTimeout)
	defer cancel()

	now := time.Now().Unix()

	entry := models.CacheEntry{
		Data:      val,
		CreatedAt: now,
		StaleAt:   now + int64(ttl.Fresh.Seconds()),
		ExpiresAt: now + int64(ttl.Fresh.Seconds()) + int64(ttl.Stale.Seconds()),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		kc.logger.Error("Failed to marshal L2 cache entry", "key", key, "error", err)
		kc.metrics.RecordCacheError("l2", "encode")
		return
	}

	totalTTL := ttl.Fresh + ttl.Stale
	err = kc.client.Set(ctx, key, data, totalTTL).Err()
	if err != nil {
		kc.logger.Warn("Failed to set L2 cache entry", "key", key, "error", err)
		kc.metrics.RecordCacheError("l2", "redis")
		return
	}
}

// Delete removes entry from KeyDB cache
func (kc *KeyDBCache) Delete(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), kc.cfg.Connection.SendTimeout)
	defer cancel()

	err := kc.client.Del(ctx, key).Err()
	if err != nil {
		kc.logger.Warn("Failed to delete L2 cache entry", "key", key, "error", err)
		return
	}
}

// Close closes the KeyDB connection
func (kc *KeyDBCache) Close() error {
	return kc.client.Close()
}
