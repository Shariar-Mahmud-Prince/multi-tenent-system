package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"multi-tenant-domains/internal/config"
)

// NewPostgresDB establishes a connection pool to PostgreSQL with retries and connection limits.
func NewPostgresDB(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	// Retry loop for pinging the database (essential for container startup orchestration)
	maxRetries := 15
	retryDelay := 2 * time.Second
	var pingErr error

	for i := 1; i <= maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		pingErr = db.PingContext(ctx)
		cancel()

		if pingErr == nil {
			log.Printf("[database] Successfully connected to PostgreSQL (attempt %d/%d)", i, maxRetries)
			return db, nil
		}

		log.Printf("[database] Waiting for PostgreSQL (attempt %d/%d): %v", i, maxRetries, pingErr)
		time.Sleep(retryDelay)
	}

	db.Close()
	return nil, fmt.Errorf("failed to ping database after %d attempts: %w", maxRetries, pingErr)
}

// RunMigrations executes all .sql migration files in the specified directory.
func RunMigrations(db *sql.DB, migrationsDir string) error {
	log.Printf("[database] Checking migrations directory: %s", migrationsDir)

	// Ensure schema_migrations table exists to track applied migrations
	createMigrationsTable := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	// Read migration files
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		// If migrations directory cannot be read, attempt fallback relative paths
		altDirs := []string{"./migrations", "../migrations", "../../migrations", "/app/migrations"}
		found := false
		for _, alt := range altDirs {
			if e, errAlt := os.ReadDir(alt); errAlt == nil {
				entries = e
				migrationsDir = alt
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("failed to read migrations directory '%s': %w", migrationsDir, err)
		}
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		var exists bool
		query := `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1);`
		if err := db.QueryRow(query, file).Scan(&exists); err != nil {
			return fmt.Errorf("failed to check migration status for %s: %w", file, err)
		}

		if exists {
			log.Printf("[database] Migration already applied: %s", file)
			continue
		}

		fullPath := filepath.Join(migrationsDir, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", fullPath, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for %s: %w", file, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}

		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES ($1);`, file); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", file, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", file, err)
		}

		log.Printf("[database] Successfully applied migration: %s", file)
	}

	return nil
}
