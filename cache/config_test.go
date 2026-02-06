package cache

import (
	"testing"
	"time"
)

func TestBigCacheConfig_ApplyDefaults(t *testing.T) {
	t.Run("applies default values to zero config", func(t *testing.T) {
		config := &BigCacheConfig{}
		config.ApplyDefaults()

		if config.Size != 100 {
			t.Errorf("expected Size to be 100, got %d", config.Size)
		}
		if config.MaxEntrySize != 1048576 {
			t.Errorf("expected MaxEntrySize to be 1048576, got %d", config.MaxEntrySize)
		}
		if config.Shards != 256 {
			t.Errorf("expected Shards to be 256, got %d", config.Shards)
		}
	})

	t.Run("does not override non-zero values", func(t *testing.T) {
		config := &BigCacheConfig{
			Size:         200,
			MaxEntrySize: 2097152,
			Shards:       512,
		}
		config.ApplyDefaults()

		if config.Size != 200 {
			t.Errorf("expected Size to remain 200, got %d", config.Size)
		}
		if config.MaxEntrySize != 2097152 {
			t.Errorf("expected MaxEntrySize to remain 2097152, got %d", config.MaxEntrySize)
		}
		if config.Shards != 512 {
			t.Errorf("expected Shards to remain 512, got %d", config.Shards)
		}
	})

	t.Run("applies defaults to partially configured", func(t *testing.T) {
		config := &BigCacheConfig{
			Size: 150,
			// MaxEntrySize and Shards are zero
		}
		config.ApplyDefaults()

		if config.Size != 150 {
			t.Errorf("expected Size to remain 150, got %d", config.Size)
		}
		if config.MaxEntrySize != 1048576 {
			t.Errorf("expected MaxEntrySize to be default 1048576, got %d", config.MaxEntrySize)
		}
		if config.Shards != 256 {
			t.Errorf("expected Shards to be default 256, got %d", config.Shards)
		}
	})

	t.Run("does not change Enabled flag", func(t *testing.T) {
		config := &BigCacheConfig{Enabled: true}
		config.ApplyDefaults()

		if !config.Enabled {
			t.Error("expected Enabled to remain true")
		}

		config2 := &BigCacheConfig{Enabled: false}
		config2.ApplyDefaults()

		if config2.Enabled {
			t.Error("expected Enabled to remain false")
		}
	})
}

func TestKeyDBConfig_ApplyDefaults(t *testing.T) {
	t.Run("applies all default values to zero config", func(t *testing.T) {
		config := &KeyDBConfig{}
		config.ApplyDefaults()

		// Connection defaults
		if config.Connection.ConnectTimeout != 1000*time.Millisecond {
			t.Errorf("expected ConnectTimeout to be 1000ms, got %v", config.Connection.ConnectTimeout)
		}
		if config.Connection.SendTimeout != 1000*time.Millisecond {
			t.Errorf("expected SendTimeout to be 1000ms, got %v", config.Connection.SendTimeout)
		}
		if config.Connection.ReadTimeout != 1000*time.Millisecond {
			t.Errorf("expected ReadTimeout to be 1000ms, got %v", config.Connection.ReadTimeout)
		}

		// Keepalive defaults
		if config.Keepalive.PoolSize != 10 {
			t.Errorf("expected PoolSize to be 10, got %d", config.Keepalive.PoolSize)
		}
		if config.Keepalive.MaxIdleTimeout != 10000*time.Millisecond {
			t.Errorf("expected MaxIdleTimeout to be 10000ms, got %v", config.Keepalive.MaxIdleTimeout)
		}

		// Cache defaults
		if config.Cache.DefaultTTL != 3600*time.Second {
			t.Errorf("expected DefaultTTL to be 3600s, got %v", config.Cache.DefaultTTL)
		}
		if config.Cache.MaxTTL != 86400*time.Second {
			t.Errorf("expected MaxTTL to be 86400s, got %v", config.Cache.MaxTTL)
		}
	})

	t.Run("does not override non-zero ConnectionConfig values", func(t *testing.T) {
		config := &KeyDBConfig{
			Connection: ConnectionConfig{
				ConnectTimeout: 2000 * time.Millisecond,
				SendTimeout:    1500 * time.Millisecond,
				ReadTimeout:    2500 * time.Millisecond,
			},
		}
		config.ApplyDefaults()

		if config.Connection.ConnectTimeout != 2000*time.Millisecond {
			t.Errorf("expected ConnectTimeout to remain 2000ms, got %v", config.Connection.ConnectTimeout)
		}
		if config.Connection.SendTimeout != 1500*time.Millisecond {
			t.Errorf("expected SendTimeout to remain 1500ms, got %v", config.Connection.SendTimeout)
		}
		if config.Connection.ReadTimeout != 2500*time.Millisecond {
			t.Errorf("expected ReadTimeout to remain 2500ms, got %v", config.Connection.ReadTimeout)
		}
	})

	t.Run("does not override non-zero KeepaliveConfig values", func(t *testing.T) {
		config := &KeyDBConfig{
			Keepalive: KeepaliveConfig{
				PoolSize:       20,
				MaxIdleTimeout: 20000 * time.Millisecond,
			},
		}
		config.ApplyDefaults()

		if config.Keepalive.PoolSize != 20 {
			t.Errorf("expected PoolSize to remain 20, got %d", config.Keepalive.PoolSize)
		}
		if config.Keepalive.MaxIdleTimeout != 20000*time.Millisecond {
			t.Errorf("expected MaxIdleTimeout to remain 20000ms, got %v", config.Keepalive.MaxIdleTimeout)
		}
	})

	t.Run("does not override non-zero CacheSettings values", func(t *testing.T) {
		config := &KeyDBConfig{
			Cache: CacheSettings{
				DefaultTTL: 7200 * time.Second,
				MaxTTL:     172800 * time.Second,
			},
		}
		config.ApplyDefaults()

		if config.Cache.DefaultTTL != 7200*time.Second {
			t.Errorf("expected DefaultTTL to remain 7200s, got %v", config.Cache.DefaultTTL)
		}
		if config.Cache.MaxTTL != 172800*time.Second {
			t.Errorf("expected MaxTTL to remain 172800s, got %v", config.Cache.MaxTTL)
		}
	})

	t.Run("applies defaults to partially configured connection", func(t *testing.T) {
		config := &KeyDBConfig{
			Connection: ConnectionConfig{
				ConnectTimeout: 2000 * time.Millisecond,
				// SendTimeout and ReadTimeout are zero
			},
		}
		config.ApplyDefaults()

		if config.Connection.ConnectTimeout != 2000*time.Millisecond {
			t.Errorf("expected ConnectTimeout to remain 2000ms, got %v", config.Connection.ConnectTimeout)
		}
		if config.Connection.SendTimeout != 1000*time.Millisecond {
			t.Errorf("expected SendTimeout to be default 1000ms, got %v", config.Connection.SendTimeout)
		}
		if config.Connection.ReadTimeout != 1000*time.Millisecond {
			t.Errorf("expected ReadTimeout to be default 1000ms, got %v", config.Connection.ReadTimeout)
		}
	})

	t.Run("applies defaults to partially configured keepalive", func(t *testing.T) {
		config := &KeyDBConfig{
			Keepalive: KeepaliveConfig{
				PoolSize: 15,
				// MaxIdleTimeout is zero
			},
		}
		config.ApplyDefaults()

		if config.Keepalive.PoolSize != 15 {
			t.Errorf("expected PoolSize to remain 15, got %d", config.Keepalive.PoolSize)
		}
		if config.Keepalive.MaxIdleTimeout != 10000*time.Millisecond {
			t.Errorf("expected MaxIdleTimeout to be default 10000ms, got %v", config.Keepalive.MaxIdleTimeout)
		}
	})

	t.Run("applies defaults to partially configured cache settings", func(t *testing.T) {
		config := &KeyDBConfig{
			Cache: CacheSettings{
				DefaultTTL: 5000 * time.Second,
				// MaxTTL is zero
			},
		}
		config.ApplyDefaults()

		if config.Cache.DefaultTTL != 5000*time.Second {
			t.Errorf("expected DefaultTTL to remain 5000s, got %v", config.Cache.DefaultTTL)
		}
		if config.Cache.MaxTTL != 86400*time.Second {
			t.Errorf("expected MaxTTL to be default 86400s, got %v", config.Cache.MaxTTL)
		}
	})

	t.Run("does not change Enabled flag", func(t *testing.T) {
		config := &KeyDBConfig{Enabled: true}
		config.ApplyDefaults()

		if !config.Enabled {
			t.Error("expected Enabled to remain true")
		}

		config2 := &KeyDBConfig{Enabled: false}
		config2.ApplyDefaults()

		if config2.Enabled {
			t.Error("expected Enabled to remain false")
		}
	})

	t.Run("handles mix of configured and default values", func(t *testing.T) {
		config := &KeyDBConfig{
			Connection: ConnectionConfig{
				ConnectTimeout: 3000 * time.Millisecond,
				// Others zero
			},
			Keepalive: KeepaliveConfig{
				MaxIdleTimeout: 15000 * time.Millisecond,
				// PoolSize zero
			},
			Cache: CacheSettings{
				// Both zero
			},
		}
		config.ApplyDefaults()

		// Configured values should be preserved
		if config.Connection.ConnectTimeout != 3000*time.Millisecond {
			t.Error("configured ConnectTimeout should be preserved")
		}
		if config.Keepalive.MaxIdleTimeout != 15000*time.Millisecond {
			t.Error("configured MaxIdleTimeout should be preserved")
		}

		// Zero values should get defaults
		if config.Connection.SendTimeout != 1000*time.Millisecond {
			t.Error("zero SendTimeout should get default")
		}
		if config.Connection.ReadTimeout != 1000*time.Millisecond {
			t.Error("zero ReadTimeout should get default")
		}
		if config.Keepalive.PoolSize != 10 {
			t.Error("zero PoolSize should get default")
		}
		if config.Cache.DefaultTTL != 3600*time.Second {
			t.Error("zero DefaultTTL should get default")
		}
		if config.Cache.MaxTTL != 86400*time.Second {
			t.Error("zero MaxTTL should get default")
		}
	})
}

func TestMultiCacheConfig(t *testing.T) {
	t.Run("EnablePropagation can be true", func(t *testing.T) {
		config := MultiCacheConfig{EnablePropagation: true}
		if !config.EnablePropagation {
			t.Error("expected EnablePropagation to be true")
		}
	})

	t.Run("EnablePropagation can be false", func(t *testing.T) {
		config := MultiCacheConfig{EnablePropagation: false}
		if config.EnablePropagation {
			t.Error("expected EnablePropagation to be false")
		}
	})
}
