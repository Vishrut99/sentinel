package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const DashboardStatsCacheKey = "dashboard:stats:v1"

type RedisCache struct {
	client *redis.Client
}

type redisNoopLogger struct{}

func (redisNoopLogger) Printf(_ context.Context, _ string, _ ...any) {}

func NewRedisCache(redisURL string) (*RedisCache, error) {
	if redisDisabled(redisURL) {
		return nil, nil
	}

	redis.SetLogger(redisNoopLogger{})

	opt, err := redis.ParseURL(strings.TrimSpace(redisURL))
	if err != nil {
		return nil, fmt.Errorf("RedisCache.NewRedisCache: %w", err)
	}
	opt.MaxRetries = 0
	opt.DialTimeout = 1 * time.Second
	opt.ReadTimeout = 1 * time.Second
	opt.WriteTimeout = 1 * time.Second

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("RedisCache.NewRedisCache: %w", err)
	}

	return &RedisCache{client: client}, nil
}

func redisDisabled(redisURL string) bool {
	switch strings.ToLower(strings.TrimSpace(redisURL)) {
	case "", "disabled", "off", "false", "none":
		return true
	default:
		return false
	}
}

func (c *RedisCache) Close() error {
	if c == nil || c.client == nil {
		return nil
	}
	if err := c.client.Close(); err != nil {
		return fmt.Errorf("RedisCache.Close: %w", err)
	}
	return nil
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if c == nil || c.client == nil {
		return nil, false, nil
	}

	value, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("RedisCache.Get: %w", err)
	}

	return value, true, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.client == nil {
		return nil
	}

	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("RedisCache.Set: %w", err)
	}

	return nil
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return nil
	}

	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("RedisCache.Delete: %w", err)
	}

	return nil
}
