# cache

Multi-level caching system with L1 (in-memory BigCache) and L2 (distributed KeyDB/Redis) support.

## Installation

```go
import "github.com/status-im/proxy-common/cache"
```

## Interfaces

- `Cache` - Basic cache operations (Get, Set, Delete)
- `LevelAwareCache` - Extended interface with cache level tracking
- `KeyDbClient` - Interface for Redis/KeyDB operations
- `Logger` - Pluggable logging interface
- `MetricsRecorder` - Prometheus metrics interface

## Quick Start

### Multi-Level Cache (L1 + L2)

```go
import (
    "github.com/status-im/proxy-common/cache"
    "github.com/status-im/proxy-common/cache/multi"
    "github.com/status-im/proxy-common/cache/l1"
    "github.com/status-im/proxy-common/cache/l2"
    "github.com/status-im/proxy-common/models"
)

// Create L1 cache (BigCache)
l1Config := cache.BigCacheConfig{
    MaxSize:       1024 * 1024 * 100, // 100MB
    CleanInterval: 60,                // seconds
}
l1Cache, _ := l1.NewBigCache(l1Config, cache.NoopLogger{}, cache.NoopMetrics{})

// Create L2 cache (KeyDB/Redis)
l2Config := cache.KeyDBConfig{
    Address:         "localhost:6379",
    ConnectTimeout:  5,  // seconds
    RequestTimeout:  2,  // seconds
    MaxActiveConns:  10,
    MaxIdleConns:    5,
}
l2Cache, _ := l2.NewKeyDBCache(l2Config, cache.NoopLogger{}, cache.NoopMetrics{})

// Create multi-level cache
multiConfig := cache.MultiCacheConfig{
    PropagateUp: true, // Promote L2 hits to L1
}
multiCache := multi.NewMultiCache(
    []cache.Cache{l1Cache, l2Cache},
    multiConfig,
    cache.NoopLogger{},
    cache.NoopMetrics{},
)

// Use it
ttl := models.TTL{Fresh: 60 * time.Second, Stale: 300 * time.Second}
multiCache.Set("key", []byte("value"), ttl)

entry, found := multiCache.Get("key")
if found {
    // Use entry.Data
}
```

## Cache Levels

- **L1**: In-memory BigCache (fast, limited capacity)
- **L2**: KeyDB/Redis (slower, larger capacity, distributed)

When using `MultiCache`:
1. `Get()` checks L1 first, then L2
2. If found in L2 and `PropagateUp: true`, promotes entry to L1
3. `Set()` writes to all levels

## Configuration

### BigCacheConfig (L1)
- `MaxSize` - Maximum cache size in bytes
- `CleanInterval` - Cleanup interval in seconds

### KeyDBConfig (L2)
- `Address` - Redis/KeyDB address
- `ConnectTimeout` - Connection timeout in seconds
- `RequestTimeout` - Request timeout in seconds
- `MaxActiveConns` - Max active connections
- `MaxIdleConns` - Max idle connections

### MultiCacheConfig
- `PropagateUp` - Promote lower-level hits to higher levels

## Logging and Metrics

Use `NoopLogger{}` and `NoopMetrics{}` for quick start, or implement the interfaces for production use.
