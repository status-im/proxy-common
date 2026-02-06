package httpclient

import (
	"math/rand"
	"time"
)

// CalculateBackoffWithJitter calculates backoff duration with jitter for retries
func CalculateBackoffWithJitter(baseBackoff time.Duration, attempt int) time.Duration {
	if attempt <= 0 {
		return baseBackoff
	}

	multiplier := uint(1) << uint(attempt-1)
	backoff := time.Duration(float64(baseBackoff) * float64(multiplier))
	jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
	return backoff + jitter
}
