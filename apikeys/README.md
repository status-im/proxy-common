# apikeys

API key management with rotation, failure tracking, and automatic backoff.

## Installation

```go
import "github.com/status-im/proxy-common/apikeys"
```

## Key Types

- `IAPIKeyManager` - Interface for API key management
- `APIKeyManager` - Manager implementation with backoff support
- `APIKey` - Represents an API key with type and label
- `KeyType` - Type for categorizing API keys
- `KeyProvider` - Interface for providing API keys

## Quick Start

```go
import (
    "context"
    "fmt"
    "time"
    
    "github.com/status-im/proxy-common/apikeys"
)

// Define your API keys
keys := []apikeys.APIKey{
    {Key: "key1", Type: "primary", Label: "Primary Key"},
    {Key: "key2", Type: "secondary", Label: "Backup Key"},
    {Key: "key3", Type: "fallback", Label: "Fallback Key"},
}

// Create provider
provider := &SimpleProvider{keys: keys}

// Create manager with backoff configuration
manager := apikeys.NewAPIKeyManager(
    provider,
    5*time.Minute,  // backoff duration
    10*time.Second, // backoff step
)

// Get available keys (respects priority and backoff)
availableKeys := manager.GetAvailableKeys()
```

## TryWithKeys Pattern

The `TryWithKeys` function attempts a request with multiple keys until one succeeds:

```go
import "net/http"

ctx := context.Background()

result, err := manager.TryWithKeys(ctx, func(ctx context.Context, key apikeys.APIKey) (interface{}, error) {
    // Make request with this key
    req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
    req.Header.Set("Authorization", "Bearer "+key.Key)
    
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode == 429 || resp.StatusCode >= 500 {
        // Will try next key
        return nil, fmt.Errorf("request failed: %d", resp.StatusCode)
    }
    
    return resp, nil
})

if err != nil {
    // All keys failed
    fmt.Println("All keys exhausted:", err)
} else {
    // Success with one of the keys
    fmt.Println("Request succeeded:", result)
}
```

## Failure Tracking

Mark keys as failed to put them in backoff:

```go
key := availableKeys[0]
manager.MarkKeyAsFailed(key.Key, key.Type)

// Key will be unavailable until backoff expires
```

## Key Provider Implementation

```go
type SimpleProvider struct {
    keys []apikeys.APIKey
}

func (p *SimpleProvider) GetKeys() []apikeys.APIKey {
    return p.keys
}
```

The manager handles priority ordering and backoff automatically based on key types and failure tracking.
