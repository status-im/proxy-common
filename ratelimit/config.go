package ratelimit

// RateLimit represents a simple rpm + burst pair
type RateLimit struct {
	RateLimitPerMinute int
	Burst              int
}
