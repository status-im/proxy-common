package l2

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/status-im/proxy-common/cache"
)

// Ensure RedisKeyDbClient implements cache.KeyDbClient
var _ cache.KeyDbClient = (*RedisKeyDbClient)(nil)

// RedisKeyDbClient wraps redis.Client to implement KeyDbClient interface
type RedisKeyDbClient struct {
	client *redis.Client
	logger cache.Logger
}

// ClientOption is a functional option for configuring RedisKeyDbClient
type ClientOption func(*RedisKeyDbClient)

// WithClientLogger sets the logger for RedisKeyDbClient
func WithClientLogger(logger cache.Logger) ClientOption {
	return func(r *RedisKeyDbClient) {
		r.logger = logger
	}
}

// NewRedisKeyDbClient creates a new RedisKeyDbClient instance
func NewRedisKeyDbClient(cfg *cache.KeyDBConfig, keydbURL string, opts ...ClientOption) (cache.KeyDbClient, error) {
	cfg.ApplyDefaults()

	parsedURL, err := url.Parse(keydbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse KeyDB URL: %w", err)
	}

	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "6379"
	}

	redisOpts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		DialTimeout:  cfg.Connection.ConnectTimeout,
		ReadTimeout:  cfg.Connection.ReadTimeout,
		WriteTimeout: cfg.Connection.SendTimeout,
		PoolSize:     cfg.Keepalive.PoolSize,
		IdleTimeout:  cfg.Keepalive.MaxIdleTimeout,
	}

	if parsedURL.User != nil {
		if password, ok := parsedURL.User.Password(); ok {
			redisOpts.Password = password
		}
	}

	if parsedURL.Path != "" && len(parsedURL.Path) > 1 {
		if db, err := strconv.Atoi(parsedURL.Path[1:]); err == nil {
			redisOpts.DB = db
		}
	}

	client := redis.NewClient(redisOpts)

	r := &RedisKeyDbClient{
		client: client,
		logger: cache.NoopLogger{},
	}

	for _, opt := range opts {
		opt(r)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Connection.ConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect to KeyDB at %s: %w", redisOpts.Addr, err)
	}

	r.logger.Info("Connected to KeyDB",
		"address", redisOpts.Addr,
		"connect_timeout", cfg.Connection.ConnectTimeout,
		"pool_size", cfg.Keepalive.PoolSize)

	return r, nil
}

func (r *RedisKeyDbClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return r.client.Get(ctx, key)
}

func (r *RedisKeyDbClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return r.client.Set(ctx, key, value, expiration)
}

func (r *RedisKeyDbClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return r.client.Del(ctx, keys...)
}

func (r *RedisKeyDbClient) Ping(ctx context.Context) *redis.StatusCmd {
	return r.client.Ping(ctx)
}

func (r *RedisKeyDbClient) Close() error {
	return r.client.Close()
}
