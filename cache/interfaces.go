package cache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/status-im/proxy-common/models"
)

//go:generate mockgen -package=mock -source=interfaces.go -destination=mock/cache.go

// Cache interface defines the contract for cache implementations
type Cache interface {
	Get(key string) (*models.CacheEntry, bool)
	GetStale(key string) (*models.CacheEntry, bool) // stale-if-error
	Set(key string, val []byte, ttl models.TTL)
	Delete(key string)
}

// LevelAwareCache interface extends Cache with level-aware operations
type LevelAwareCache interface {
	Cache
	GetWithLevel(key string) *models.CacheResult
	GetStaleWithLevel(key string) *models.CacheResult // stale-if-error
}

// KeyDbClient defines the interface for KeyDB/Redis client operations
type KeyDbClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

// Logger defines the interface for logging operations
// This allows users to plug in their own logger (zap, logrus, etc.)
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}

// MetricsRecorder defines the interface for recording cache metrics
// Users can implement this to integrate with their metrics system (Prometheus, etc.)
type MetricsRecorder interface {
	RecordCacheError(level, kind string)
	UpdateL1CacheCapacity(capacity, used int64)
	UpdateCacheKeys(level string, count int64)
	RecordCacheHit(cacheType, level, chain, network, rpcMethod string, itemAge time.Duration)
	RecordCacheMiss(cacheType, chain, network, rpcMethod string)
	RecordCacheSet(level, cacheType, chain, network string, dataSize int)
	RecordCacheBytesRead(level, cacheType, chain, network string, bytesRead int)
	TimeCacheOperation(operation, level string) func()
}

// NoopLogger is a no-operation logger that discards all log messages
type NoopLogger struct{}

func (NoopLogger) Debug(msg string, keysAndValues ...interface{}) {}
func (NoopLogger) Info(msg string, keysAndValues ...interface{})  {}
func (NoopLogger) Warn(msg string, keysAndValues ...interface{})  {}
func (NoopLogger) Error(msg string, keysAndValues ...interface{}) {}

// NoopMetrics is a no-operation metrics recorder that discards all metrics
type NoopMetrics struct{}

func (NoopMetrics) RecordCacheError(level, kind string)        {}
func (NoopMetrics) UpdateL1CacheCapacity(capacity, used int64) {}
func (NoopMetrics) UpdateCacheKeys(level string, count int64)  {}
func (NoopMetrics) RecordCacheHit(cacheType, level, chain, network, rpcMethod string, itemAge time.Duration) {
}
func (NoopMetrics) RecordCacheMiss(cacheType, chain, network, rpcMethod string)                 {}
func (NoopMetrics) RecordCacheSet(level, cacheType, chain, network string, dataSize int)        {}
func (NoopMetrics) RecordCacheBytesRead(level, cacheType, chain, network string, bytesRead int) {}
func (NoopMetrics) TimeCacheOperation(operation, level string) func()                           { return func() {} }
