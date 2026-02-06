package ratelimit

import (
	"math"
	"sync"

	"golang.org/x/time/rate"

	"github.com/status-im/proxy-common/apikeys"
)

// IRateLimiterManager provides a way to get a rate limiter for a specific key
type IRateLimiterManager interface {
	GetLimiter(key string, keyType apikeys.KeyType) *rate.Limiter
	SetConfig(config map[apikeys.KeyType]RateLimit)
}

// RateLimiterManager manages per-key rate limiters
type RateLimiterManager struct {
	mu           sync.RWMutex
	keyToLimiter map[string]*rate.Limiter
	config       map[apikeys.KeyType]RateLimit
}

// NewRateLimiterManager creates a new rate limiter manager
func NewRateLimiterManager(config map[apikeys.KeyType]RateLimit) *RateLimiterManager {
	return &RateLimiterManager{
		keyToLimiter: make(map[string]*rate.Limiter),
		config:       config,
	}
}

// SetConfig applies a new rate limit configuration and rebuilds limiters for types with changed settings
func (m *RateLimiterManager) SetConfig(newConfig map[apikeys.KeyType]RateLimit) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = newConfig

	// Rebuild all limiters with new config
	for key := range m.keyToLimiter {
		delete(m.keyToLimiter, key)
	}
}

// GetLimiter returns a limiter for a given api key and type, creating it if missing
func (m *RateLimiterManager) GetLimiter(key string, keyType apikeys.KeyType) *rate.Limiter {
	mapKey := m.limiterMapKey(key, keyType)

	m.mu.RLock()
	if lim, ok := m.keyToLimiter[mapKey]; ok {
		m.mu.RUnlock()
		return lim
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if lim, ok := m.keyToLimiter[mapKey]; ok {
		return lim
	}

	limit := m.limitForType(keyType)
	burst := m.burstForType(keyType, limit)
	limiter := rate.NewLimiter(limit, burst)
	m.keyToLimiter[mapKey] = limiter
	return limiter
}

func (m *RateLimiterManager) limiterMapKey(key string, keyType apikeys.KeyType) string {
	return string(rune(keyType)) + "|" + key
}

func (m *RateLimiterManager) limitForType(keyType apikeys.KeyType) rate.Limit {
	if cfg, ok := m.config[keyType]; ok && cfg.RateLimitPerMinute > 0 {
		return rate.Limit(float64(cfg.RateLimitPerMinute) / 60.0)
	}
	// Default fallback
	return rate.Limit(30.0 / 60.0) // 30 requests per minute
}

func (m *RateLimiterManager) burstForType(keyType apikeys.KeyType, limit rate.Limit) int {
	if cfg, ok := m.config[keyType]; ok && cfg.Burst > 0 {
		return cfg.Burst
	}
	return defaultBurstForLimit(limit)
}

func defaultBurstForLimit(limit rate.Limit) int {
	if limit <= 1.0 {
		return 1
	}
	return int(math.Ceil(float64(limit)))
}
