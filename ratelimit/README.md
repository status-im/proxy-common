# ratelimit

Per-key rate limiting using golang.org/x/time/rate.

## Installation

```go
import "github.com/status-im/proxy-common/ratelimit"
```

## Key Types

- `IRateLimiterManager` - Interface for rate limiter management
- `RateLimiterManager` - Manages per-key rate limiters
- `RateLimit` - Rate limit configuration (requests per minute + burst)

## Quick Start

```go
import (
    "context"
    "time"
    
    "github.com/status-im/proxy-common/ratelimit"
)

// Create rate limit config
config := map[string]map[string]ratelimit.RateLimit{
    "primary": {
        "api1": {RPM: 100, Burst: 10}, // 100 req/min, burst 10
        "api2": {RPM: 50, Burst: 5},
    },
    "secondary": {
        "api1": {RPM: 30, Burst: 3},
    },
}

// Create manager
manager := ratelimit.NewRateLimiterManager(config)

// Get limiter for specific key and type
limiter := manager.GetLimiter("my-api-key", "primary")

// Wait for permission (blocks if rate limit exceeded)
ctx := context.Background()
err := limiter.Wait(ctx)
if err != nil {
    // Context cancelled
}

// Or check without blocking
if limiter.Allow() {
    // Request allowed
} else {
    // Rate limit exceeded
}
```

## Dynamic Configuration

Update rate limits without recreating the manager:

```go
newConfig := map[string]map[string]ratelimit.RateLimit{
    "primary": {
        "api1": {RPM: 200, Burst: 20}, // Increased limits
    },
}

manager.SetConfig(newConfig)
// All limiters are rebuilt with new configuration
```

## Rate Limit Configuration

`RateLimit` struct:
- `RPM` - Requests per minute
- `Burst` - Maximum burst size (tokens in bucket)

The rate limiter uses token bucket algorithm from `golang.org/x/time/rate`.

## Usage with API Keys

```go
apiKey := "user-key-123"
keyType := "premium" // or "free", "basic", etc.

limiter := manager.GetLimiter(apiKey, keyType)

if limiter.Allow() {
    // Process request
} else {
    // Return 429 Too Many Requests
}
```
