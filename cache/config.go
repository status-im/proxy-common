package cache

import "time"

// BigCacheConfig represents BigCache (L1) configuration
type BigCacheConfig struct {
	Enabled      bool `yaml:"enabled" json:"enabled"`
	Size         int  `yaml:"size" json:"size"`
	MaxEntrySize int  `yaml:"max_entry_size" json:"max_entry_size"`
	Shards       int  `yaml:"shards" json:"shards"` // must be power of 2
}

func (c *BigCacheConfig) ApplyDefaults() {
	if c.Size == 0 {
		c.Size = 100
	}
	if c.MaxEntrySize == 0 {
		c.MaxEntrySize = 1048576
	}
	if c.Shards == 0 {
		c.Shards = 256 // power of 2
	}
}

// KeyDBConfig represents KeyDB (L2) cache configuration
type KeyDBConfig struct {
	Enabled    bool             `yaml:"enabled" json:"enabled"`
	Connection ConnectionConfig `yaml:"connection" json:"connection"`
	Keepalive  KeepaliveConfig  `yaml:"keepalive" json:"keepalive"`
	Cache      CacheSettings    `yaml:"cache" json:"cache"`
}

func (c *KeyDBConfig) ApplyDefaults() {
	if c.Connection.ConnectTimeout == 0 {
		c.Connection.ConnectTimeout = 1000 * time.Millisecond
	}
	if c.Connection.SendTimeout == 0 {
		c.Connection.SendTimeout = 1000 * time.Millisecond
	}
	if c.Connection.ReadTimeout == 0 {
		c.Connection.ReadTimeout = 1000 * time.Millisecond
	}

	if c.Keepalive.PoolSize == 0 {
		c.Keepalive.PoolSize = 10
	}
	if c.Keepalive.MaxIdleTimeout == 0 {
		c.Keepalive.MaxIdleTimeout = 10000 * time.Millisecond
	}

	if c.Cache.DefaultTTL == 0 {
		c.Cache.DefaultTTL = 3600 * time.Second
	}
	if c.Cache.MaxTTL == 0 {
		c.Cache.MaxTTL = 86400 * time.Second
	}
}

type ConnectionConfig struct {
	ConnectTimeout time.Duration `yaml:"connect_timeout" json:"connect_timeout"`
	SendTimeout    time.Duration `yaml:"send_timeout" json:"send_timeout"`
	ReadTimeout    time.Duration `yaml:"read_timeout" json:"read_timeout"`
}

// KeepaliveConfig represents connection pool settings
type KeepaliveConfig struct {
	PoolSize       int           `yaml:"pool_size" json:"pool_size"` // max connections in pool
	MaxIdleTimeout time.Duration `yaml:"max_idle_timeout" json:"max_idle_timeout"`
}

type CacheSettings struct {
	DefaultTTL time.Duration `yaml:"default_ttl" json:"default_ttl"`
	MaxTTL     time.Duration `yaml:"max_ttl" json:"max_ttl"`
}

type MultiCacheConfig struct {
	EnablePropagation bool `yaml:"enable_propagation" json:"enable_propagation"`
}
