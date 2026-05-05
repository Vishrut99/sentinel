package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

type migrationRecord struct {
	Filename string `gorm:"column:filename;primaryKey"`
}

func (migrationRecord) TableName() string { return "schema_migrations" }

// RunMigrations applies SQL files from migrationsDir in filename order.
// It records applied migrations in schema_migrations to ensure idempotency.
func RunMigrations(database *gorm.DB, migrationsDir string) error {
	if database == nil {
		return fmt.Errorf("db.RunMigrations: database is nil")
	}

	if err := ensureMigrationsTable(database); err != nil {
		return err
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("db.RunMigrations: read dir %s: %w", migrationsDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".sql") {
			continue
		}

		applied, err := migrationApplied(database, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		path := filepath.Join(migrationsDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("db.RunMigrations: read %s: %w", path, err)
		}

		log.Printf("applying migration: %s", name)
		if err := applyMigration(database, name, string(content)); err != nil {
			return err
		}
	}

	return nil
}

func ensureMigrationsTable(database *gorm.DB) error {
	if err := database.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`).Error; err != nil {
		return fmt.Errorf("db.RunMigrations: create schema_migrations: %w", err)
	}
	return nil
}

func migrationApplied(database *gorm.DB, filename string) (bool, error) {
	var count int64
	if err := database.Raw(
		"SELECT COUNT(1) FROM schema_migrations WHERE filename = ?",
		filename,
	).Scan(&count).Error; err != nil {
		return false, fmt.Errorf("db.RunMigrations: check %s: %w", filename, err)
	}
	return count > 0, nil
}

func applyMigration(database *gorm.DB, filename, sql string) error {
	return database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(sql).Error; err != nil {
			return fmt.Errorf("db.RunMigrations: apply %s: %w", filename, err)
		}
		if err := tx.Exec(
			"INSERT INTO schema_migrations (filename) VALUES (?) ON CONFLICT DO NOTHING",
			filename,
		).Error; err != nil {
			return fmt.Errorf("db.RunMigrations: record %s: %w", filename, err)
		}
		return nil
	})
}
