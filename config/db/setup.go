package db

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

type DB struct {
	*gorm.DB
}

func (db *DB) GetDB() *gorm.DB {
	return db.DB
}

func CreateDB(primaryDSN string, replicaDSNs ...string) (*DB, error) {
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Connect to primary database
	db, err := gorm.Open(postgres.Open(primaryDSN), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary database: %v", err)
	}

	// Configure read replicas if provided
	if len(replicaDSNs) > 0 {
		resolverConfig := dbresolver.Config{}

		// Add all replicas
		for _, replicaDSN := range replicaDSNs {
			resolverConfig.Replicas = append(resolverConfig.Replicas, postgres.Open(replicaDSN))
		}

		// Setup resolver with connection pool settings
		err = db.Use(dbresolver.Register(resolverConfig).
			SetConnMaxIdleTime(time.Hour).
			SetConnMaxLifetime(24 * time.Hour).
			SetMaxIdleConns(10).
			SetMaxOpenConns(100))

		if err != nil {
			return nil, fmt.Errorf("failed to configure read replicas: %v", err)
		}

		log.Printf("Configured %d read replicas successfully", len(replicaDSNs))
	}

	// Get underlying *sql.DB for primary connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %v", err)
	}

	// Set connection pool settings for primary connection
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	log.Println("Connected to database successfully")
	return &DB{db}, nil
}

func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %v", err)
	}
	return sqlDB.Close()
}
