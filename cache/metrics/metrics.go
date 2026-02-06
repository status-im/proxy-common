package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	DefaultNamespace = "proxy"
	DefaultSubsystem = "cache"
)

// Config defines configuration for cache metrics
type Config struct {
	Namespace string // e.g., "nft_proxy", "eth_rpc_proxy"
	Subsystem string // default: "cache"
}

// CacheMetrics holds all cache-related Prometheus metrics
type CacheMetrics struct {
	namespace      string
	subsystem      string
	allowedMethods map[string]bool

	// Counter metrics
	Requests     *prometheus.CounterVec
	Hits         *prometheus.CounterVec
	Misses       *prometheus.CounterVec
	Sets         *prometheus.CounterVec
	Evictions    *prometheus.CounterVec
	Errors       *prometheus.CounterVec
	BytesRead    *prometheus.CounterVec
	BytesWritten *prometheus.CounterVec

	// Histogram metrics
	OperationDuration *prometheus.HistogramVec
	ItemAge           *prometheus.HistogramVec

	// Gauge metrics
	Keys     *prometheus.GaugeVec
	Capacity *prometheus.GaugeVec
	Used     *prometheus.GaugeVec
}

// New creates a new CacheMetrics instance with the given configuration
func New(cfg Config) *CacheMetrics {
	if cfg.Namespace == "" {
		cfg.Namespace = DefaultNamespace
	}
	if cfg.Subsystem == "" {
		cfg.Subsystem = DefaultSubsystem
	}

	m := &CacheMetrics{
		namespace: cfg.Namespace,
		subsystem: cfg.Subsystem,
	}

	// Initialize counter metrics
	m.Requests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "requests_total",
			Help:      "Total number of cache requests",
		},
		[]string{"cache_type", "level", "network", "rpc_method"},
	)

	m.Hits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "hits_total",
			Help:      "Total number of cache hits",
		},
		[]string{"cache_type", "level", "network", "rpc_method"},
	)

	m.Misses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "misses_total",
			Help:      "Total number of cache misses",
		},
		[]string{"cache_type", "level", "network", "rpc_method"},
	)

	m.Sets = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "sets_total",
			Help:      "Total number of cache set operations",
		},
		[]string{"level", "cache_type", "network"},
	)

	m.Evictions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "evictions_total",
			Help:      "Total number of cache evictions",
		},
		[]string{"level", "cache_type", "network"},
	)

	m.Errors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "errors_total",
			Help:      "Cache errors by kind",
		},
		[]string{"level", "kind"},
	)

	m.BytesRead = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "bytes_read_total",
			Help:      "Bytes read from cache",
		},
		[]string{"level", "cache_type", "network"},
	)

	m.BytesWritten = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "bytes_written_total",
			Help:      "Bytes written to cache",
		},
		[]string{"level", "cache_type", "network"},
	)

	// Initialize histogram metrics
	m.OperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "operation_duration_seconds",
			Help:      "Duration of cache operations",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"operation", "level"}, // operation: get|set, level: l1|l2|multi
	)

	m.ItemAge = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "item_age_seconds",
			Help:      "Age of item at hit time",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120, 300, 600, 1800, 3600}, // up to 1 hour
		},
		[]string{"level", "cache_type"},
	)

	// Initialize gauge metrics
	m.Keys = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "keys",
			Help:      "Current number of keys in cache",
		},
		[]string{"level"},
	)

	m.Capacity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "capacity_bytes",
			Help:      "L1 cache capacity in bytes",
		},
		[]string{"level"}, // only "l1"
	)

	m.Used = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: cfg.Namespace,
			Subsystem: cfg.Subsystem,
			Name:      "used_bytes",
			Help:      "L1 cache used space in bytes",
		},
		[]string{"level"}, // only "l1"
	)

	return m
}

// InitializeAllowedMethods initializes the allowed methods whitelist from cache rules
func (m *CacheMetrics) InitializeAllowedMethods(methods []string) {
	m.allowedMethods = make(map[string]bool)

	// Add all configured methods to whitelist
	for _, method := range methods {
		m.allowedMethods[method] = true
	}
}

// normalizeRPCMethod returns the method name if it's in the whitelist, otherwise "other"
func (m *CacheMetrics) normalizeRPCMethod(method string) string {
	if m.allowedMethods != nil && m.allowedMethods[method] {
		return method
	}
	return "other"
}

// normalizeNetwork creates a network identifier from chain and network
func normalizeNetwork(chain, network string) string {
	if chain == "" || network == "" {
		return "unknown"
	}
	return chain + ":" + network
}

// RecordCacheHit records a cache hit with enhanced labels and age tracking
func (m *CacheMetrics) RecordCacheHit(cacheType, level, chain, network, rpcMethod string, itemAge time.Duration) {
	normalizedNetwork := normalizeNetwork(chain, network)
	normalizedMethod := m.normalizeRPCMethod(rpcMethod)

	// Record request and hit with proper level
	m.Requests.WithLabelValues(cacheType, level, normalizedNetwork, normalizedMethod).Inc()
	m.Hits.WithLabelValues(cacheType, level, normalizedNetwork, normalizedMethod).Inc()

	// Record item age for TTL effectiveness analysis
	if itemAge > 0 {
		m.ItemAge.WithLabelValues(level, cacheType).Observe(itemAge.Seconds())
	}
}

// RecordCacheMiss records a cache miss with enhanced labels
func (m *CacheMetrics) RecordCacheMiss(cacheType, chain, network, rpcMethod string) {
	normalizedNetwork := normalizeNetwork(chain, network)
	normalizedMethod := m.normalizeRPCMethod(rpcMethod)

	// For miss, we don't know the level, so we use "miss" as level
	// This represents requests that didn't hit any cache level
	m.Requests.WithLabelValues(cacheType, "miss", normalizedNetwork, normalizedMethod).Inc()
	m.Misses.WithLabelValues(cacheType, "miss", normalizedNetwork, normalizedMethod).Inc()
}

// RecordCacheSet records a cache set operation with size tracking
func (m *CacheMetrics) RecordCacheSet(level, cacheType, chain, network string, dataSize int) {
	normalizedNetwork := normalizeNetwork(chain, network)

	m.Sets.WithLabelValues(level, cacheType, normalizedNetwork).Inc()
	if dataSize > 0 {
		m.BytesWritten.WithLabelValues(level, cacheType, normalizedNetwork).Add(float64(dataSize))
	}
}

// RecordCacheEviction records a cache eviction
func (m *CacheMetrics) RecordCacheEviction(level, cacheType, chain, network string) {
	normalizedNetwork := normalizeNetwork(chain, network)
	m.Evictions.WithLabelValues(level, cacheType, normalizedNetwork).Inc()
}

// RecordCacheError records a cache error
func (m *CacheMetrics) RecordCacheError(level, kind string) {
	m.Errors.WithLabelValues(level, kind).Inc()
}

// RecordCacheBytesRead records bytes read from cache
func (m *CacheMetrics) RecordCacheBytesRead(level, cacheType, chain, network string, bytesRead int) {
	if bytesRead > 0 {
		normalizedNetwork := normalizeNetwork(chain, network)
		m.BytesRead.WithLabelValues(level, cacheType, normalizedNetwork).Add(float64(bytesRead))
	}
}

// UpdateL1CacheCapacity updates L1 cache capacity metrics
func (m *CacheMetrics) UpdateL1CacheCapacity(capacity, used int64) {
	m.Capacity.WithLabelValues("l1").Set(float64(capacity))
	m.Used.WithLabelValues("l1").Set(float64(used))
}

// UpdateCacheKeys updates the number of keys in cache
func (m *CacheMetrics) UpdateCacheKeys(level string, count int64) {
	m.Keys.WithLabelValues(level).Set(float64(count))
}

// TimeCacheOperation returns a timer function for measuring cache operation duration
func (m *CacheMetrics) TimeCacheOperation(operation, level string) func() {
	timer := prometheus.NewTimer(m.OperationDuration.WithLabelValues(operation, level))
	return func() {
		timer.ObserveDuration()
	}
}

// TimeCacheGetOperation returns a timer function for measuring cache get operation duration (backward compatibility)
func (m *CacheMetrics) TimeCacheGetOperation(level string) func() {
	return m.TimeCacheOperation("get", level)
}
