package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

type ClusterConfig struct {
	PrimaryDSN   string
	ReplicaDSNs  []string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
	MaxIdleTime  time.Duration
}

type Cluster struct {
	primary  *gorm.DB
	replicas []*gorm.DB
	config   ClusterConfig
	mu       sync.RWMutex
	health   map[string]bool
}

func NewCluster(config ClusterConfig) (*Cluster, error) {
	cluster := &Cluster{
		config: config,
		health: make(map[string]bool),
	}

	primaryConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	primary, err := gorm.Open(postgres.Open(config.PrimaryDSN), primaryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary database: %v", err)
	}

	sqlDB, err := primary.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get primary database instance: %v", err)
	}

	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(config.MaxLifetime)
	sqlDB.SetConnMaxIdleTime(config.MaxIdleTime)

	cluster.primary = primary
	cluster.health["primary"] = true

	for i, replicaDSN := range config.ReplicaDSNs {
		replica, err := gorm.Open(postgres.Open(replicaDSN), primaryConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to replica %d: %v", i, err)
		}

		replicaSQLDB, err := replica.DB()
		if err != nil {
			return nil, fmt.Errorf("failed to get replica %d database instance: %v", i, err)
		}

		replicaSQLDB.SetMaxOpenConns(config.MaxOpenConns)
		replicaSQLDB.SetMaxIdleConns(config.MaxIdleConns)
		replicaSQLDB.SetConnMaxLifetime(config.MaxLifetime)
		replicaSQLDB.SetConnMaxIdleTime(config.MaxIdleTime)

		cluster.replicas = append(cluster.replicas, replica)
		cluster.health[fmt.Sprintf("replica_%d", i)] = true
	}

	if len(config.ReplicaDSNs) > 0 {
		resolverConfig := dbresolver.Config{}
		for _, replicaDSN := range config.ReplicaDSNs {
			resolverConfig.Replicas = append(resolverConfig.Replicas, postgres.Open(replicaDSN))
		}

		err = cluster.primary.Use(dbresolver.Register(resolverConfig).
			SetConnMaxIdleTime(config.MaxIdleTime).
			SetConnMaxLifetime(config.MaxLifetime).
			SetMaxIdleConns(config.MaxIdleConns).
			SetMaxOpenConns(config.MaxOpenConns))

		if err != nil {
			return nil, fmt.Errorf("failed to configure read replicas: %v", err)
		}
	}

	go cluster.startHealthChecks()

	return cluster, nil
}

func (c *Cluster) GetPrimary() *gorm.DB {
	return c.primary
}

func (c *Cluster) GetReplica() (*gorm.DB, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for i, replica := range c.replicas {
		if c.health[fmt.Sprintf("replica_%d", i)] {
			return replica, nil
		}
	}

	return c.primary, nil
}

func (c *Cluster) WithTransaction(ctx context.Context, fn func(context.Context, *gorm.DB) error) error {
	return c.primary.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, "tx", tx)
		return fn(txCtx, tx)
	})
}

func (c *Cluster) startHealthChecks() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.checkHealth()
	}
}

func (c *Cluster) checkHealth() {
	c.mu.Lock()
	defer c.mu.Unlock()

	sqlDB, err := c.primary.DB()
	if err != nil {
		c.health["primary"] = false
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = sqlDB.PingContext(ctx)
		cancel()
		c.health["primary"] = err == nil
	}

	for i, replica := range c.replicas {
		key := fmt.Sprintf("replica_%d", i)
		sqlDB, err := replica.DB()
		if err != nil {
			c.health[key] = false
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = sqlDB.PingContext(ctx)
			cancel()
			c.health[key] = err == nil
		}
	}
}

func (c *Cluster) GetHealth() map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	health := make(map[string]bool)
	for k, v := range c.health {
		health[k] = v
	}
	return health
}

func (c *Cluster) Close() error {
	var errs []error

	if sqlDB, err := c.primary.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close primary: %v", err))
		}
	}

	for i, replica := range c.replicas {
		if sqlDB, err := replica.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, fmt.Errorf("failed to close replica %d: %v", i, err))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing cluster: %v", errs)
	}

	return nil
}
