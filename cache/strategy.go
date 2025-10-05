package cache

import (
	"context"
	"fmt"
	"time"
)

type CacheStrategy string

const (
	WriteThrough CacheStrategy = "write_through"
	WriteBack    CacheStrategy = "write_back"
	WriteAround  CacheStrategy = "write_around"
	ReadThrough  CacheStrategy = "read_through"
)

type CacheConfig struct {
	Strategy       CacheStrategy
	TTL            time.Duration
	MaxSize        int
	EvictionPolicy string
}

type CacheManager struct {
	redis  *RedisCache
	local  map[string]CacheItem
	config CacheConfig
	stats  CacheStats
}

type CacheItem struct {
	Value       interface{}
	ExpiresAt   time.Time
	CreatedAt   time.Time
	AccessCount int
}

type CacheStats struct {
	Hits      int64
	Misses    int64
	Sets      int64
	Deletes   int64
	Evictions int64
}

func CreateCacheManager(redis *RedisCache, config CacheConfig) *CacheManager {
	return &CacheManager{
		redis:  redis,
		local:  make(map[string]CacheItem),
		config: config,
		stats:  CacheStats{},
	}
}

func (cm *CacheManager) Get(ctx context.Context, key string) (interface{}, error) {
	if cm.config.Strategy == ReadThrough {
		return cm.readThrough(ctx, key)
	}

	item, exists := cm.local[key]
	if exists && time.Now().Before(item.ExpiresAt) {
		cm.stats.Hits++
		item.AccessCount++
		cm.local[key] = item
		return item.Value, nil
	}

	if exists {
		delete(cm.local, key)
	}

	value, err := cm.redis.Get(ctx, key)
	if err == nil {
		cm.local[key] = CacheItem{
			Value:       value,
			ExpiresAt:   time.Now().Add(cm.config.TTL),
			CreatedAt:   time.Now(),
			AccessCount: 1,
		}
		cm.stats.Hits++
		return value, nil
	}

	cm.stats.Misses++
	return nil, fmt.Errorf("key not found")
}

func (cm *CacheManager) Set(ctx context.Context, key string, value interface{}) error {
	if cm.config.Strategy == WriteThrough {
		return cm.writeThrough(ctx, key, value)
	}

	if cm.config.Strategy == WriteBack {
		return cm.writeBack(ctx, key, value)
	}

	return cm.writeAround(ctx, key, value)
}

func (cm *CacheManager) Delete(ctx context.Context, key string) error {
	delete(cm.local, key)
	err := cm.redis.Delete(ctx, key)
	if err == nil {
		cm.stats.Deletes++
	}
	return err
}

func (cm *CacheManager) readThrough(ctx context.Context, key string) (interface{}, error) {
	item, exists := cm.local[key]
	if exists && time.Now().Before(item.ExpiresAt) {
		cm.stats.Hits++
		return item.Value, nil
	}

	value, err := cm.redis.Get(ctx, key)
	if err != nil {
		cm.stats.Misses++
		return nil, err
	}

	cm.local[key] = CacheItem{
		Value:       value,
		ExpiresAt:   time.Now().Add(cm.config.TTL),
		CreatedAt:   time.Now(),
		AccessCount: 1,
	}
	cm.stats.Hits++
	return value, nil
}

func (cm *CacheManager) writeThrough(ctx context.Context, key string, value interface{}) error {
	err := cm.redis.Set(ctx, key, value)
	if err != nil {
		return err
	}

	cm.local[key] = CacheItem{
		Value:       value,
		ExpiresAt:   time.Now().Add(cm.config.TTL),
		CreatedAt:   time.Now(),
		AccessCount: 1,
	}
	cm.stats.Sets++
	return nil
}

func (cm *CacheManager) writeBack(ctx context.Context, key string, value interface{}) error {
	cm.local[key] = CacheItem{
		Value:       value,
		ExpiresAt:   time.Now().Add(cm.config.TTL),
		CreatedAt:   time.Now(),
		AccessCount: 1,
	}
	cm.stats.Sets++

	if len(cm.local) >= cm.config.MaxSize {
		cm.evict()
	}

	return nil
}

func (cm *CacheManager) writeAround(ctx context.Context, key string, value interface{}) error {
	err := cm.redis.Set(ctx, key, value)
	if err != nil {
		return err
	}

	cm.stats.Sets++
	return nil
}

func (cm *CacheManager) evict() {
	if cm.config.EvictionPolicy == "lru" {
		cm.evictLRU()
	} else {
		cm.evictFIFO()
	}
}

func (cm *CacheManager) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range cm.local {
		if oldestKey == "" || item.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(cm.local, oldestKey)
		cm.stats.Evictions++
	}
}

func (cm *CacheManager) evictFIFO() {
	for key := range cm.local {
		delete(cm.local, key)
		cm.stats.Evictions++
		break
	}
}

func (cm *CacheManager) GetStats() CacheStats {
	return cm.stats
}

func (cm *CacheManager) Flush(ctx context.Context) error {
	cm.local = make(map[string]CacheItem)
	return nil
}

func (cm *CacheManager) GetHitRate() float64 {
	total := cm.stats.Hits + cm.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(cm.stats.Hits) / float64(total)
}
