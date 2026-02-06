package noop

import (
	"testing"
	"time"

	"github.com/status-im/proxy-common/models"
)

func TestNewNoOpCache(t *testing.T) {
	c := NewNoOpCache()

	// Verify it returns a NoOpCache instance
	if _, ok := c.(*NoOpCache); !ok {
		t.Errorf("NewNoOpCache() should return a *NoOpCache instance")
	}
}

func TestNoOpCache_Get(t *testing.T) {
	c := NewNoOpCache()

	// Test with various keys
	testCases := []string{
		"test-key",
		"",
		"very-long-key-with-special-characters-!@#$%^&*()",
		"key-with-numbers-123456789",
	}

	for _, key := range testCases {
		t.Run("key="+key, func(t *testing.T) {
			entry, found := c.Get(key)

			if entry != nil {
				t.Errorf("Get(%q) entry = %v, want nil", key, entry)
			}
			if found {
				t.Errorf("Get(%q) found = %v, want false", key, found)
			}
		})
	}
}

func TestNoOpCache_GetStale(t *testing.T) {
	c := NewNoOpCache()

	// Test with various keys
	testCases := []string{
		"test-key",
		"",
		"very-long-key-with-special-characters-!@#$%^&*()",
		"key-with-numbers-123456789",
	}

	for _, key := range testCases {
		t.Run("key="+key, func(t *testing.T) {
			entry, found := c.GetStale(key)

			if entry != nil {
				t.Errorf("GetStale(%q) entry = %v, want nil", key, entry)
			}
			if found {
				t.Errorf("GetStale(%q) found = %v, want false", key, found)
			}
		})
	}
}

func TestNoOpCache_Set(t *testing.T) {
	c := NewNoOpCache()

	// Test setting various values
	testCases := []struct {
		key string
		val []byte
		ttl models.TTL
	}{
		{"test-key", []byte("test-value"), models.TTL{Fresh: 60 * time.Second, Stale: 120 * time.Second}},
		{"", []byte(""), models.TTL{Fresh: 0, Stale: 0}},
		{"binary-key", []byte{0x01, 0x02, 0x03, 0xFF}, models.TTL{Fresh: 3600 * time.Second, Stale: 7200 * time.Second}},
		{"json-key", []byte(`{"test": "value"}`), models.TTL{Fresh: 300 * time.Second, Stale: 600 * time.Second}},
	}

	for _, tc := range testCases {
		t.Run("key="+tc.key, func(t *testing.T) {
			// Set should not panic and should be a no-op
			c.Set(tc.key, tc.val, tc.ttl)

			// Verify it's still a cache miss after setting
			entry, found := c.Get(tc.key)
			if entry != nil || found {
				t.Errorf("After Set(%q, %v, %v), Get() = (%v, %v), want (nil, false)",
					tc.key, tc.val, tc.ttl, entry, found)
			}
		})
	}
}

func TestNoOpCache_Delete(t *testing.T) {
	c := NewNoOpCache()

	// Test deleting various keys
	testCases := []string{
		"test-key",
		"",
		"non-existent-key",
		"very-long-key-with-special-characters-!@#$%^&*()",
	}

	for _, key := range testCases {
		t.Run("key="+key, func(t *testing.T) {
			// Delete should not panic and should be a no-op
			c.Delete(key)

			// Verify it's still a cache miss after deleting
			entry, found := c.Get(key)
			if entry != nil || found {
				t.Errorf("After Delete(%q), Get() = (%v, %v), want (nil, false)",
					key, entry, found)
			}
		})
	}
}

func TestNoOpCache_InterfaceCompliance(t *testing.T) {
	c := NewNoOpCache()

	// Verify all interface methods work as expected
	key := "test-key"
	value := []byte("test-value")
	ttl := models.TTL{Fresh: 60 * time.Second, Stale: 120 * time.Second}

	// Test the complete workflow
	c.Set(key, value, ttl)

	entry, found := c.Get(key)
	if entry != nil || found {
		t.Errorf("Get() after Set() = (%v, %v), want (nil, false)", entry, found)
	}

	entry, found = c.GetStale(key)
	if entry != nil || found {
		t.Errorf("GetStale() after Set() = (%v, %v), want (nil, false)", entry, found)
	}

	c.Delete(key)

	entry, found = c.Get(key)
	if entry != nil || found {
		t.Errorf("Get() after Delete() = (%v, %v), want (nil, false)", entry, found)
	}
}

func TestNoOpCache_ConcurrentAccess(t *testing.T) {
	c := NewNoOpCache()

	// Test concurrent access to ensure no race conditions
	done := make(chan bool)

	// Start multiple goroutines performing operations
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			key := "concurrent-key"
			value := []byte("concurrent-value")
			ttl := models.TTL{Fresh: 60 * time.Second, Stale: 120 * time.Second}

			// Perform various operations
			c.Set(key, value, ttl)
			c.Get(key)
			c.GetStale(key)
			c.Delete(key)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify cache is still in expected state
	entry, found := c.Get("concurrent-key")
	if entry != nil || found {
		t.Errorf("After concurrent operations, Get() = (%v, %v), want (nil, false)", entry, found)
	}
}
