package noop

import (
	"github.com/status-im/proxy-common/cache"
	"github.com/status-im/proxy-common/models"
)

// Ensure NoOpCache implements cache.Cache
var _ cache.Cache = (*NoOpCache)(nil)

// NoOpCache is a no-operation cache implementation for disabled caches
type NoOpCache struct{}

// NewNoOpCache creates a new no-operation cache instance
func NewNoOpCache() cache.Cache {
	return &NoOpCache{}
}

func (n *NoOpCache) Get(key string) (*models.CacheEntry, bool) {
	return nil, false
}

func (n *NoOpCache) GetStale(key string) (*models.CacheEntry, bool) {
	return nil, false
}

func (n *NoOpCache) Set(key string, val []byte, ttl models.TTL) {
}

func (n *NoOpCache) Delete(key string) {
}
