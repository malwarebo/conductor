package db

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

type Migration struct {
	Version string
	Name    string
	Up      func(*gorm.DB) error
	Down    func(*gorm.DB) error
}

type Migrator struct {
	db         *gorm.DB
	migrations []Migration
}

func CreateNewMigrator(db *gorm.DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: make([]Migration, 0),
	}
}

func (m *Migrator) AddMigration(version, name string, up, down func(*gorm.DB) error) {
	m.migrations = append(m.migrations, Migration{
		Version: version,
		Name:    name,
		Up:      up,
		Down:    down,
	})
}

func (m *Migrator) LoadMigrationsFromDir(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}

	sort.Strings(files)

	for _, file := range files {
		filename := filepath.Base(file)
		parts := strings.Split(filename, "_")
		if len(parts) < 2 {
			continue
		}

		version := parts[0]
		name := strings.TrimSuffix(strings.Join(parts[1:], "_"), ".sql")

		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		sql := string(content)
		m.AddMigration(version, name, func(db *gorm.DB) error {
			return db.Exec(sql).Error
		}, func(db *gorm.DB) error {
			return nil
		})
	}

	return nil
}

func (m *Migrator) Up() error {
	if err := m.createMigrationsTable(); err != nil {
		return err
	}

	applied, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	for _, migration := range m.migrations {
		if applied[migration.Version] {
			continue
		}

		if err := migration.Up(m.db); err != nil {
			return fmt.Errorf("failed to apply migration %s: %v", migration.Version, err)
		}

		if err := m.recordMigration(migration.Version, migration.Name); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) Down(version string) error {
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	for i := len(m.migrations) - 1; i >= 0; i-- {
		migration := m.migrations[i]
		if migration.Version == version {
			break
		}

		if !applied[migration.Version] {
			continue
		}

		if err := migration.Down(m.db); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %v", migration.Version, err)
		}

		if err := m.removeMigration(migration.Version); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) createMigrationsTable() error {
	return m.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
}

func (m *Migrator) getAppliedMigrations() (map[string]bool, error) {
	var results []struct {
		Version string
	}

	if err := m.db.Table("schema_migrations").Select("version").Find(&results).Error; err != nil {
		return nil, err
	}

	applied := make(map[string]bool)
	for _, result := range results {
		applied[result.Version] = true
	}

	return applied, nil
}

func (m *Migrator) recordMigration(version, name string) error {
	return m.db.Exec(`
		INSERT INTO schema_migrations (version, name)
		VALUES (?, ?)
		ON CONFLICT (version) DO NOTHING
	`, version, name).Error
}

func (m *Migrator) removeMigration(version string) error {
	return m.db.Exec("DELETE FROM schema_migrations WHERE version = ?", version).Error
}

func (m *Migrator) Status() ([]MigrationStatus, error) {
	applied, err := m.getAppliedMigrations()
	if err != nil {
		return nil, err
	}

	var statuses []MigrationStatus
	for _, migration := range m.migrations {
		statuses = append(statuses, MigrationStatus{
			Version: migration.Version,
			Name:    migration.Name,
			Applied: applied[migration.Version],
		})
	}

	return statuses, nil
}

type MigrationStatus struct {
	Version string
	Name    string
	Applied bool
}
