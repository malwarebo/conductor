package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
}

type ConnectionPool struct {
	primary  *gorm.DB
	replicas []*gorm.DB
	config   PoolConfig
	mu       sync.RWMutex
	health   map[string]bool
}

func CreateNewConnectionPool(primaryDSN string, replicaDSNs []string, config PoolConfig) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		config: config,
		health: make(map[string]bool),
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	primary, err := gorm.Open(postgres.Open(primaryDSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary database: %v", err)
	}

	sqlDB, err := primary.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get primary database instance: %v", err)
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	pool.primary = primary
	pool.health["primary"] = true

	for i, replicaDSN := range replicaDSNs {
		replica, err := gorm.Open(postgres.Open(replicaDSN), gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to replica %d: %v", i, err)
		}

		replicaSQLDB, err := replica.DB()
		if err != nil {
			return nil, fmt.Errorf("failed to get replica %d database instance: %v", i, err)
		}

		replicaSQLDB.SetMaxOpenConns(config.MaxOpenConns)
		replicaSQLDB.SetMaxIdleConns(config.MaxIdleConns)
		replicaSQLDB.SetConnMaxLifetime(config.ConnMaxLifetime)
		replicaSQLDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)

		pool.replicas = append(pool.replicas, replica)
		pool.health[fmt.Sprintf("replica_%d", i)] = true
	}

	go pool.startHealthChecks()

	return pool, nil
}

func (p *ConnectionPool) GetPrimary() *gorm.DB {
	return p.primary
}

func (p *ConnectionPool) GetReplica() (*gorm.DB, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for i, replica := range p.replicas {
		if p.health[fmt.Sprintf("replica_%d", i)] {
			return replica, nil
		}
	}

	return p.primary, nil
}

func (p *ConnectionPool) WithTransaction(ctx context.Context, fn func(context.Context, *gorm.DB) error) error {
	return p.primary.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, "tx", tx)
		return fn(txCtx, tx)
	})
}

func (p *ConnectionPool) WithRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(p.config.RetryDelay * time.Duration(attempt)):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("operation failed after %d attempts: %w", p.config.MaxRetries, lastErr)
}

func (p *ConnectionPool) startHealthChecks() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.checkHealth()
	}
}

func (p *ConnectionPool) checkHealth() {
	p.mu.Lock()
	defer p.mu.Unlock()

	sqlDB, err := p.primary.DB()
	if err != nil {
		p.health["primary"] = false
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = sqlDB.PingContext(ctx)
		cancel()
		p.health["primary"] = err == nil
	}

	for i, replica := range p.replicas {
		key := fmt.Sprintf("replica_%d", i)
		sqlDB, err := replica.DB()
		if err != nil {
			p.health[key] = false
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = sqlDB.PingContext(ctx)
			cancel()
			p.health[key] = err == nil
		}
	}
}

func (p *ConnectionPool) GetHealth() map[string]bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	health := make(map[string]bool)
	for k, v := range p.health {
		health[k] = v
	}
	return health
}

func (p *ConnectionPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := PoolStats{
		PrimaryHealthy:  p.health["primary"],
		ReplicaCount:    len(p.replicas),
		HealthyReplicas: 0,
	}

	for i := range p.replicas {
		if p.health[fmt.Sprintf("replica_%d", i)] {
			stats.HealthyReplicas++
		}
	}

	return stats
}

func (p *ConnectionPool) Close() error {
	var errs []error

	if sqlDB, err := p.primary.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close primary: %v", err))
		}
	}

	for i, replica := range p.replicas {
		if sqlDB, err := replica.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close replica %d: %v", i, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing pool: %v", errs)
	}

	return nil
}

type PoolStats struct {
	PrimaryHealthy  bool
	ReplicaCount    int
	HealthyReplicas int
}
