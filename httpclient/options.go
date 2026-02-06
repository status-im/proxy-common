package httpclient

import "time"

// RetryOptions configures retry behavior for HTTP requests
type RetryOptions struct {
	MaxRetries        int
	BaseBackoff       time.Duration
	LogPrefix         string
	ConnectionTimeout time.Duration // Timeout for establishing connection
	RequestTimeout    time.Duration // Total request timeout including reading response
}

// DefaultRetryOptions returns default retry options
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxRetries:        3,
		BaseBackoff:       1000 * time.Millisecond,
		LogPrefix:         "HTTP",
		ConnectionTimeout: 10 * time.Second, // Default 10s connection timeout
		RequestTimeout:    30 * time.Second, // Default 30s total request timeout
	}
}
