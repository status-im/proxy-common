# models

Shared data models and types for cache operations.

## Installation

```go
import "github.com/status-im/proxy-common/models"
```

## Types

### CacheEntry

Represents a cache entry with TTL information:

```go
type CacheEntry struct {
    Data      []byte // Cached data
    ExpiresAt int64  // Unix timestamp when entry expires completely
    StaleAt   int64  // Unix timestamp when entry becomes stale
    CreatedAt int64  // Unix timestamp when entry was created
}
```

Methods:
- `IsExpired() bool` - Check if completely expired
- `IsFresh() bool` - Check if still fresh
- `RemainingTTL() TTL` - Calculate remaining time

### TTL

Time-to-live configuration with fresh/stale periods:

```go
type TTL struct {
    Fresh time.Duration // How long data is fresh
    Stale time.Duration // How long stale data can be served (stale-if-error)
}
```

### CacheResult

Result of a level-aware cache operation:

```go
type CacheResult struct {
    Entry *CacheEntry
    Found bool
    Level CacheLevel // L1, L2, or MISS
}
```

### CacheType

Cache duration category:

```go
type CacheType string

const (
    CacheTypePermanent CacheType = "permanent" // Long-lived data
    CacheTypeShort     CacheType = "short"     // Short-lived data
    CacheTypeMinimal   CacheType = "minimal"   // Very short TTL
    CacheTypeNone      CacheType = "none"      // No caching
)
```

### CacheLevel

Indicates which cache level returned data:

```go
type CacheLevel string

const (
    CacheLevelL1   CacheLevel = "L1"   // In-memory cache
    CacheLevelL2   CacheLevel = "L2"   // Distributed cache
    CacheLevelMiss CacheLevel = "MISS" // Cache miss
)
```

### CacheStatus

Status of a cache request:

```go
type CacheStatus string

const (
    CacheStatusHit    CacheStatus = "HIT"    // Cache hit
    CacheStatusMiss   CacheStatus = "MISS"   // Cache miss
    CacheStatusBypass CacheStatus = "BYPASS" // Cache bypassed
)
```

## Usage Example

```go
import (
    "time"
    "github.com/status-im/proxy-common/models"
)

// Create TTL config
ttl := models.TTL{
    Fresh: 60 * time.Second,  // Fresh for 1 minute
    Stale: 300 * time.Second, // Serve stale for 5 minutes
}

// Check cache entry
if entry.IsFresh() {
    // Use fresh data
} else if !entry.IsExpired() {
    // Use stale data (stale-if-error pattern)
} else {
    // Entry expired, fetch new data
}
```
