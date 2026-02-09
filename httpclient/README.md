# httpclient

HTTP client with automatic retries, exponential backoff, and rate limiting support.

## Installation

```go
import "github.com/status-im/proxy-common/httpclient"
```

## Key Types

- `HTTPClientWithRetries` - HTTP client with retry logic
- `RetryOptions` - Retry configuration
- `IHttpStatusHandler` - Interface for handling HTTP request status

## Quick Start

```go
import (
    "context"
    "net/http"
    "time"
    
    "github.com/status-im/proxy-common/httpclient"
)

// Create client with default options
client := httpclient.NewHTTPClientWithRetries(
    httpclient.DefaultRetryOptions(),
    nil, // rate limiter (optional)
    nil, // status handler (optional)
)

// Make a request
req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
ctx := context.Background()

resp, err := client.ExecuteRequest(ctx, req)
if err != nil {
    // Handle error
}
defer resp.Body.Close()

// Use response
```

## Retry Configuration

```go
options := httpclient.RetryOptions{
    MaxRetries:     3,
    InitialBackoff: 1 * time.Second,
    MaxBackoff:     30 * time.Second,
    BackoffFactor:  2.0,
    JitterFactor:   0.1, // 10% jitter
}

client := httpclient.NewHTTPClientWithRetries(options, nil, nil)
```

## Retryable Status Codes

Automatically retries on:
- `429` - Too Many Requests
- `500` - Internal Server Error
- `502` - Bad Gateway
- `503` - Service Unavailable
- `504` - Gateway Timeout

Also retries on network errors and connection timeouts.

## With Rate Limiter

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(rate.Limit(10), 1) // 10 req/sec, burst 1
client := httpclient.NewHTTPClientWithRetries(
    httpclient.DefaultRetryOptions(),
    limiter,
    nil,
)
```

## Status Handler

Implement `IHttpStatusHandler` to customize handling of HTTP responses:

```go
type MyHandler struct{}

func (h *MyHandler) HandleStatus(ctx context.Context, req *http.Request, resp *http.Response) error {
    // Custom logic
    return nil
}

client := httpclient.NewHTTPClientWithRetries(
    httpclient.DefaultRetryOptions(),
    nil,
    &MyHandler{},
)
```
