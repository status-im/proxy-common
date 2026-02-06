package cache

import (
	"testing"
	"time"
)

func TestNoopLogger(t *testing.T) {
	logger := NoopLogger{}

	t.Run("Debug does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Debug panicked: %v", r)
			}
		}()
		logger.Debug("test message")
		logger.Debug("test with values", "key1", "value1", "key2", 123)
	})

	t.Run("Info does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Info panicked: %v", r)
			}
		}()
		logger.Info("test message")
		logger.Info("test with values", "key1", "value1", "key2", 123)
	})

	t.Run("Warn does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Warn panicked: %v", r)
			}
		}()
		logger.Warn("test message")
		logger.Warn("test with values", "key1", "value1", "key2", 123)
	})

	t.Run("Error does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Error panicked: %v", r)
			}
		}()
		logger.Error("test message")
		logger.Error("test with values", "key1", "value1", "key2", 123)
	})

	t.Run("accepts various argument types", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Logger panicked with various types: %v", r)
			}
		}()
		logger.Debug("test", "string", "value", "int", 42, "float", 3.14, "bool", true)
		logger.Info("test", "time", time.Now(), "duration", time.Second)
		logger.Warn("test", "nil", nil, "slice", []string{"a", "b"})
		logger.Error("test", "map", map[string]int{"key": 1})
	})

	t.Run("accepts no key-value pairs", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Logger panicked with no args: %v", r)
			}
		}()
		logger.Debug("just a message")
		logger.Info("just a message")
		logger.Warn("just a message")
		logger.Error("just a message")
	})

	t.Run("accepts empty message", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Logger panicked with empty message: %v", r)
			}
		}()
		logger.Debug("")
		logger.Info("")
		logger.Warn("")
		logger.Error("")
	})
}

func TestNoopMetrics(t *testing.T) {
	metrics := NoopMetrics{}

	t.Run("RecordCacheError does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RecordCacheError panicked: %v", r)
			}
		}()
		metrics.RecordCacheError("l1", "connection")
		metrics.RecordCacheError("l2", "timeout")
		metrics.RecordCacheError("", "")
	})

	t.Run("UpdateL1CacheCapacity does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UpdateL1CacheCapacity panicked: %v", r)
			}
		}()
		metrics.UpdateL1CacheCapacity(1000, 500)
		metrics.UpdateL1CacheCapacity(0, 0)
		metrics.UpdateL1CacheCapacity(-1, -1)
	})

	t.Run("UpdateCacheKeys does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("UpdateCacheKeys panicked: %v", r)
			}
		}()
		metrics.UpdateCacheKeys("l1", 100)
		metrics.UpdateCacheKeys("l2", 200)
		metrics.UpdateCacheKeys("", 0)
	})

	t.Run("RecordCacheHit does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RecordCacheHit panicked: %v", r)
			}
		}()
		metrics.RecordCacheHit("json", "l1", "ethereum", "mainnet", "eth_getBlockByNumber", time.Second)
		metrics.RecordCacheHit("", "", "", "", "", 0)
	})

	t.Run("RecordCacheMiss does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RecordCacheMiss panicked: %v", r)
			}
		}()
		metrics.RecordCacheMiss("json", "ethereum", "mainnet", "eth_getBlockByNumber")
		metrics.RecordCacheMiss("", "", "", "")
	})

	t.Run("RecordCacheSet does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RecordCacheSet panicked: %v", r)
			}
		}()
		metrics.RecordCacheSet("l1", "json", "ethereum", "mainnet", 1024)
		metrics.RecordCacheSet("", "", "", "", 0)
		metrics.RecordCacheSet("l2", "json", "ethereum", "mainnet", -1)
	})

	t.Run("RecordCacheBytesRead does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("RecordCacheBytesRead panicked: %v", r)
			}
		}()
		metrics.RecordCacheBytesRead("l1", "json", "ethereum", "mainnet", 2048)
		metrics.RecordCacheBytesRead("", "", "", "", 0)
		metrics.RecordCacheBytesRead("l2", "json", "ethereum", "mainnet", -1)
	})

	t.Run("TimeCacheOperation returns callable function", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("TimeCacheOperation panicked: %v", r)
			}
		}()

		done := metrics.TimeCacheOperation("get", "l1")
		if done == nil {
			t.Fatal("expected non-nil function from TimeCacheOperation")
		}

		// Calling the returned function should not panic
		done()
	})

	t.Run("TimeCacheOperation with various inputs", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("TimeCacheOperation panicked: %v", r)
			}
		}()

		done1 := metrics.TimeCacheOperation("get", "l1")
		done2 := metrics.TimeCacheOperation("set", "l2")
		done3 := metrics.TimeCacheOperation("", "")

		if done1 == nil || done2 == nil || done3 == nil {
			t.Fatal("expected non-nil functions")
		}

		done1()
		done2()
		done3()
	})

	t.Run("returned function can be called multiple times", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Multiple calls panicked: %v", r)
			}
		}()

		done := metrics.TimeCacheOperation("get", "l1")
		done()
		done()
		done()
	})

	t.Run("all methods work together", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Combined operations panicked: %v", r)
			}
		}()

		// Simulate a cache operation workflow
		done := metrics.TimeCacheOperation("get", "l1")
		metrics.RecordCacheMiss("json", "ethereum", "mainnet", "eth_blockNumber")

		metrics.RecordCacheSet("l1", "json", "ethereum", "mainnet", 512)
		metrics.UpdateCacheKeys("l1", 1)
		metrics.UpdateL1CacheCapacity(10000, 512)

		metrics.RecordCacheHit("json", "l1", "ethereum", "mainnet", "eth_blockNumber", time.Millisecond*100)
		metrics.RecordCacheBytesRead("l1", "json", "ethereum", "mainnet", 512)

		done()
	})
}

func TestNoopImplementsInterfaces(t *testing.T) {
	t.Run("NoopLogger implements Logger interface", func(t *testing.T) {
		var logger Logger = NoopLogger{}
		if logger == nil {
			t.Error("NoopLogger does not implement Logger interface")
		}

		// Test that it can be used as Logger
		logger.Debug("test")
		logger.Info("test")
		logger.Warn("test")
		logger.Error("test")
	})

	t.Run("NoopMetrics implements MetricsRecorder interface", func(t *testing.T) {
		var metrics MetricsRecorder = NoopMetrics{}
		if metrics == nil {
			t.Error("NoopMetrics does not implement MetricsRecorder interface")
		}

		// Test that it can be used as MetricsRecorder
		metrics.RecordCacheError("l1", "test")
		metrics.UpdateL1CacheCapacity(100, 50)
		metrics.UpdateCacheKeys("l1", 10)
		metrics.RecordCacheHit("json", "l1", "chain", "network", "method", time.Second)
		metrics.RecordCacheMiss("json", "chain", "network", "method")
		metrics.RecordCacheSet("l1", "json", "chain", "network", 100)
		metrics.RecordCacheBytesRead("l1", "json", "chain", "network", 100)
		done := metrics.TimeCacheOperation("get", "l1")
		done()
	})
}
