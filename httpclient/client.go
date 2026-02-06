package httpclient

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

// HTTPClientWithRetries wraps an HTTP Client with retry capabilities
type HTTPClientWithRetries struct {
	Client        *http.Client
	Opts          RetryOptions
	StatusHandler IHttpStatusHandler
	// RateLimiter is an optional callback that returns a rate limiter for the request
	// The callback receives the request and should return a rate limiter or nil
	RateLimiter func(*http.Request) *rate.Limiter
}

// NewHTTPClientWithRetries creates a new HTTP Client with retry capabilities
func NewHTTPClientWithRetries(opts RetryOptions, handler IHttpStatusHandler, rateLimiter func(*http.Request) *rate.Limiter) *HTTPClientWithRetries {
	client := &http.Client{
		Timeout: opts.RequestTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: opts.ConnectionTimeout,
			}).DialContext,
		},
	}

	return &HTTPClientWithRetries{
		Client:        client,
		Opts:          opts,
		StatusHandler: handler,
		RateLimiter:   rateLimiter,
	}
}

// SetStatusHandler sets the status handler for this Client
func (c *HTTPClientWithRetries) SetStatusHandler(handler IHttpStatusHandler) {
	c.StatusHandler = handler
}

// ExecuteRequest executes an HTTP request with retry logic
func (c *HTTPClientWithRetries) ExecuteRequest(req *http.Request) (*http.Response, []byte, time.Duration, error) {
	var lastErr error

	for attempt := 0; attempt < c.Opts.MaxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("%s: Retry %d/%d after error: %v",
				c.Opts.LogPrefix, attempt, c.Opts.MaxRetries-1, lastErr)

			if c.StatusHandler != nil {
				c.StatusHandler.OnRetry()
			}

			backoffDuration := CalculateBackoffWithJitter(c.Opts.BaseBackoff, attempt)
			log.Printf("%s: Waiting %.2fs before retry", c.Opts.LogPrefix, backoffDuration.Seconds())
			time.Sleep(backoffDuration)
		}

		requestStart := time.Now()

		// Rate limit before executing the request
		if c.RateLimiter != nil {
			limiter := c.RateLimiter(req)
			if limiter != nil {
				if err := limiter.Wait(req.Context()); err != nil {
					lastErr = fmt.Errorf("rate limiter wait failed: %w", err)
					if c.StatusHandler != nil {
						c.StatusHandler.OnRequest("error")
					}
					break
				}
			}
		}

		// Execute request
		resp, err := c.Client.Do(req)
		requestDuration := time.Since(requestStart)

		if err != nil {
			lastErr = fmt.Errorf("request failed after %.2fs: %v", requestDuration.Seconds(), err)
			if c.StatusHandler != nil {
				c.StatusHandler.OnRequest("error")
			}
			continue
		}

		responseBody, err := processResponse(resp, req, requestDuration)
		if err != nil {
			if isRetryableError(resp.StatusCode) {
				lastErr = err
				_ = resp.Body.Close()
				if c.StatusHandler != nil {
					c.StatusHandler.OnRequest("rate_limited")
				}
				continue
			}

			_ = resp.Body.Close()
			if c.StatusHandler != nil {
				c.StatusHandler.OnRequest("error")
			}
			return nil, nil, requestDuration, err
		}

		if c.StatusHandler != nil {
			c.StatusHandler.OnRequest("success")
		}
		return resp, responseBody, requestDuration, nil
	}

	return nil, nil, 0, fmt.Errorf("all %d attempts failed, last error: %v",
		c.Opts.MaxRetries, lastErr)
}

// processResponse reads and processes the HTTP response
func processResponse(resp *http.Response, req *http.Request, requestDuration time.Duration) ([]byte, error) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			log.Printf("rate limit exceeded (status %d), retry after %s: %s",
				resp.StatusCode, retryAfter, string(body))
			return nil, fmt.Errorf("rate limit exceeded (status %d), retry after %s: %s",
				resp.StatusCode, retryAfter, string(body))
		}

		// Special handling for 414 Request-URI Too Large to include URL length
		if resp.StatusCode == 414 {
			var urlLength int
			if req != nil && req.URL != nil {
				urlLength = len(req.URL.String())
			}
			log.Printf("API request failed with status %d after %.2fs (URL length: %d): %s",
				resp.StatusCode, requestDuration.Seconds(), urlLength, string(body))

			return nil, fmt.Errorf("API request failed with status %d after %.2fs (URL length: %d): %s",
				resp.StatusCode, requestDuration.Seconds(), urlLength, string(body))
		}

		return nil, fmt.Errorf("API request failed with status %d after %.2fs: %s",
			resp.StatusCode, requestDuration.Seconds(), string(body))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	return responseBody, nil
}

// isRetryableError determines if a given HTTP status code should trigger a retry
func isRetryableError(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}
