package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	TTL      time.Duration // Default TTL for cache entries
}

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func CreateRedisCache(config RedisConfig) (*RedisCache, error) {
	// Convert port to string
	portStr := strconv.Itoa(config.Port)

	addr := config.Host + ":" + portStr
	if config.Port == 0 {
		addr = config.Host + ":6379" // Default Redis port
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: config.Password,
		DB:       config.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, err
	}

	ttl := config.TTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
	}, nil
}

func (c *RedisCache) Client() *redis.Client {
	return c.client
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}) error {
	return c.client.Set(ctx, key, value, c.ttl).Err()
}

func (c *RedisCache) SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	return result > 0, err
}
