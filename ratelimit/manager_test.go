package ratelimit

import (
	"sync"
	"testing"
	"time"

	"github.com/status-im/proxy-common/apikeys"
	"golang.org/x/time/rate"
)

func TestNewRateLimiterManager(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		config := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 60, Burst: 10},
		}
		mgr := NewRateLimiterManager(config)
		if mgr == nil {
			t.Fatal("expected non-nil manager")
		}
		if mgr.keyToLimiter == nil {
			t.Error("expected keyToLimiter map to be initialized")
		}
		if mgr.config == nil {
			t.Error("expected config to be set")
		}
	})

	t.Run("with empty config", func(t *testing.T) {
		mgr := NewRateLimiterManager(nil)
		if mgr == nil {
			t.Fatal("expected non-nil manager")
		}
		if mgr.keyToLimiter == nil {
			t.Error("expected keyToLimiter map to be initialized")
		}
	})
}

func TestGetLimiter(t *testing.T) {
	t.Run("creates new limiter for new key", func(t *testing.T) {
		config := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 60, Burst: 10},
		}
		mgr := NewRateLimiterManager(config)

		limiter := mgr.GetLimiter("key1", apikeys.KeyType(1))
		if limiter == nil {
			t.Fatal("expected non-nil limiter")
		}

		// Check limiter parameters
		if limiter.Burst() != 10 {
			t.Errorf("expected burst of 10, got %d", limiter.Burst())
		}

		// Verify limit is correct (60 req/min = 1 req/sec)
		expectedLimit := rate.Limit(1.0)
		if limiter.Limit() != expectedLimit {
			t.Errorf("expected limit of %v, got %v", expectedLimit, limiter.Limit())
		}
	})

	t.Run("returns same limiter for same key", func(t *testing.T) {
		mgr := NewRateLimiterManager(nil)

		limiter1 := mgr.GetLimiter("key1", apikeys.KeyType(1))
		limiter2 := mgr.GetLimiter("key1", apikeys.KeyType(1))

		if limiter1 != limiter2 {
			t.Error("expected same limiter instance for same key")
		}
	})

	t.Run("different limiters for different keys", func(t *testing.T) {
		mgr := NewRateLimiterManager(nil)

		limiter1 := mgr.GetLimiter("key1", apikeys.KeyType(1))
		limiter2 := mgr.GetLimiter("key2", apikeys.KeyType(1))

		if limiter1 == limiter2 {
			t.Error("expected different limiter instances for different keys")
		}
	})

	t.Run("different limiters for different key types", func(t *testing.T) {
		config := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 60, Burst: 10},
			apikeys.KeyType(2): {RateLimitPerMinute: 120, Burst: 20},
		}
		mgr := NewRateLimiterManager(config)

		limiter1 := mgr.GetLimiter("key1", apikeys.KeyType(1))
		limiter2 := mgr.GetLimiter("key1", apikeys.KeyType(2))

		if limiter1 == limiter2 {
			t.Error("expected different limiter instances for different key types")
		}

		if limiter1.Burst() != 10 {
			t.Errorf("expected burst of 10 for type 1, got %d", limiter1.Burst())
		}
		if limiter2.Burst() != 20 {
			t.Errorf("expected burst of 20 for type 2, got %d", limiter2.Burst())
		}
	})

	t.Run("default values when no config", func(t *testing.T) {
		mgr := NewRateLimiterManager(nil)

		limiter := mgr.GetLimiter("key1", apikeys.KeyType(1))

		// Default is 30 req/min = 0.5 req/sec
		expectedLimit := rate.Limit(0.5)
		if limiter.Limit() != expectedLimit {
			t.Errorf("expected default limit of %v, got %v", expectedLimit, limiter.Limit())
		}

		// Default burst for 0.5 should be 1 (since limit <= 1.0)
		if limiter.Burst() != 1 {
			t.Errorf("expected default burst of 1, got %d", limiter.Burst())
		}
	})
}

func TestSetConfig(t *testing.T) {
	t.Run("updates config", func(t *testing.T) {
		oldConfig := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 60, Burst: 10},
		}
		mgr := NewRateLimiterManager(oldConfig)

		newConfig := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 120, Burst: 20},
		}
		mgr.SetConfig(newConfig)

		// Get a new limiter and verify it uses new config
		limiter := mgr.GetLimiter("key1", apikeys.KeyType(1))
		if limiter.Burst() != 20 {
			t.Errorf("expected burst of 20 after config update, got %d", limiter.Burst())
		}
	})

	t.Run("clears existing limiters", func(t *testing.T) {
		mgr := NewRateLimiterManager(nil)

		// Create a limiter
		limiter1 := mgr.GetLimiter("key1", apikeys.KeyType(1))

		// Update config
		newConfig := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 120, Burst: 20},
		}
		mgr.SetConfig(newConfig)

		// Get limiter again - should be a new instance
		limiter2 := mgr.GetLimiter("key1", apikeys.KeyType(1))

		if limiter1 == limiter2 {
			t.Error("expected new limiter instance after config update")
		}
	})
}

func TestDefaultBurstForLimit(t *testing.T) {
	tests := []struct {
		name          string
		limit         rate.Limit
		expectedBurst int
	}{
		{
			name:          "limit less than 1",
			limit:         rate.Limit(0.5),
			expectedBurst: 1,
		},
		{
			name:          "limit equal to 1",
			limit:         rate.Limit(1.0),
			expectedBurst: 1,
		},
		{
			name:          "limit greater than 1",
			limit:         rate.Limit(2.5),
			expectedBurst: 3, // ceil(2.5)
		},
		{
			name:          "limit is integer",
			limit:         rate.Limit(5.0),
			expectedBurst: 5,
		},
		{
			name:          "limit with decimal",
			limit:         rate.Limit(10.1),
			expectedBurst: 11, // ceil(10.1)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			burst := defaultBurstForLimit(tt.limit)
			if burst != tt.expectedBurst {
				t.Errorf("expected burst of %d for limit %v, got %d", tt.expectedBurst, tt.limit, burst)
			}
		})
	}
}

func TestLimitForType(t *testing.T) {
	t.Run("returns configured limit", func(t *testing.T) {
		config := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 60, Burst: 10},
		}
		mgr := NewRateLimiterManager(config)

		limit := mgr.limitForType(apikeys.KeyType(1))
		expectedLimit := rate.Limit(1.0) // 60/60
		if limit != expectedLimit {
			t.Errorf("expected limit of %v, got %v", expectedLimit, limit)
		}
	})

	t.Run("returns default for unconfigured type", func(t *testing.T) {
		mgr := NewRateLimiterManager(nil)

		limit := mgr.limitForType(apikeys.KeyType(99))
		expectedLimit := rate.Limit(0.5) // 30/60
		if limit != expectedLimit {
			t.Errorf("expected default limit of %v, got %v", expectedLimit, limit)
		}
	})

	t.Run("returns default for zero rate limit", func(t *testing.T) {
		config := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 0, Burst: 10},
		}
		mgr := NewRateLimiterManager(config)

		limit := mgr.limitForType(apikeys.KeyType(1))
		expectedLimit := rate.Limit(0.5) // default
		if limit != expectedLimit {
			t.Errorf("expected default limit of %v, got %v", expectedLimit, limit)
		}
	})
}

func TestBurstForType(t *testing.T) {
	t.Run("returns configured burst", func(t *testing.T) {
		config := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 60, Burst: 15},
		}
		mgr := NewRateLimiterManager(config)

		burst := mgr.burstForType(apikeys.KeyType(1), rate.Limit(1.0))
		if burst != 15 {
			t.Errorf("expected burst of 15, got %d", burst)
		}
	})

	t.Run("returns default burst for unconfigured type", func(t *testing.T) {
		mgr := NewRateLimiterManager(nil)

		burst := mgr.burstForType(apikeys.KeyType(99), rate.Limit(2.5))
		expectedBurst := 3 // ceil(2.5)
		if burst != expectedBurst {
			t.Errorf("expected default burst of %d, got %d", expectedBurst, burst)
		}
	})

	t.Run("returns default burst for zero burst config", func(t *testing.T) {
		config := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 60, Burst: 0},
		}
		mgr := NewRateLimiterManager(config)

		burst := mgr.burstForType(apikeys.KeyType(1), rate.Limit(1.0))
		if burst != 1 {
			t.Errorf("expected default burst of 1, got %d", burst)
		}
	})
}

func TestLimiterMapKey(t *testing.T) {
	mgr := NewRateLimiterManager(nil)

	key1 := mgr.limiterMapKey("apikey123", apikeys.KeyType(1))
	key2 := mgr.limiterMapKey("apikey123", apikeys.KeyType(2))
	key3 := mgr.limiterMapKey("apikey456", apikeys.KeyType(1))

	// Keys should be different for different types or different keys
	if key1 == key2 {
		t.Error("expected different map keys for different key types")
	}
	if key1 == key3 {
		t.Error("expected different map keys for different api keys")
	}

	// Same input should produce same key
	key1Again := mgr.limiterMapKey("apikey123", apikeys.KeyType(1))
	if key1 != key1Again {
		t.Error("expected same map key for same input")
	}
}

func TestConcurrentGetLimiter(t *testing.T) {
	mgr := NewRateLimiterManager(map[apikeys.KeyType]RateLimit{
		apikeys.KeyType(1): {RateLimitPerMinute: 60, Burst: 10},
	})

	const numGoroutines = 100
	const numKeys = 10

	var wg sync.WaitGroup
	limiters := make(chan *rate.Limiter, numGoroutines)

	// Launch multiple goroutines trying to get the same limiter
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Use modulo to ensure multiple goroutines request same keys
			keyID := id % numKeys
			limiter := mgr.GetLimiter("key"+string(rune(keyID)), apikeys.KeyType(1))
			limiters <- limiter
		}(i)
	}

	wg.Wait()
	close(limiters)

	// Verify all limiters were created successfully
	count := 0
	for limiter := range limiters {
		if limiter == nil {
			t.Error("got nil limiter from concurrent access")
		}
		count++
	}

	if count != numGoroutines {
		t.Errorf("expected %d limiters, got %d", numGoroutines, count)
	}

	// Verify that there are exactly numKeys unique limiters
	uniqueLimiters := make(map[*rate.Limiter]bool)
	for i := 0; i < numKeys; i++ {
		limiter := mgr.GetLimiter("key"+string(rune(i)), apikeys.KeyType(1))
		uniqueLimiters[limiter] = true
	}

	if len(uniqueLimiters) != numKeys {
		t.Errorf("expected %d unique limiters, got %d", numKeys, len(uniqueLimiters))
	}
}

func TestConcurrentSetConfig(t *testing.T) {
	mgr := NewRateLimiterManager(map[apikeys.KeyType]RateLimit{
		apikeys.KeyType(1): {RateLimitPerMinute: 60, Burst: 10},
	})

	var wg sync.WaitGroup
	const numReaders = 50
	const numWriters = 5

	// Launch reader goroutines
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				limiter := mgr.GetLimiter("key"+string(rune(id)), apikeys.KeyType(1))
				if limiter == nil {
					t.Error("got nil limiter")
				}
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	// Launch writer goroutines
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				newConfig := map[apikeys.KeyType]RateLimit{
					apikeys.KeyType(1): {
						RateLimitPerMinute: 60 + id*10,
						Burst:              10 + id,
					},
				}
				mgr.SetConfig(newConfig)
				time.Sleep(time.Microsecond * 10)
			}
		}(i)
	}

	wg.Wait()

	// No panics or data races should occur
	// If we get here, the test passed
}

func TestRateLimiterFunctionality(t *testing.T) {
	t.Run("limiter enforces rate limit", func(t *testing.T) {
		config := map[apikeys.KeyType]RateLimit{
			apikeys.KeyType(1): {RateLimitPerMinute: 60, Burst: 2}, // 1 req/sec, burst 2
		}
		mgr := NewRateLimiterManager(config)
		limiter := mgr.GetLimiter("testkey", apikeys.KeyType(1))

		// Should be able to make burst requests immediately
		if !limiter.Allow() {
			t.Error("expected first request to be allowed")
		}
		if !limiter.Allow() {
			t.Error("expected second request to be allowed (within burst)")
		}

		// Third request should be denied (burst exhausted)
		if limiter.Allow() {
			t.Error("expected third request to be denied")
		}
	})
}
