package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

// Open opens (and creates) the SQLite database at the configured path.
func Open(dataDir string) (*sql.DB, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir data dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, "soundstack.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.Exec("PRAGMA busy_timeout = 5000;")
	return db, nil
}

// Migration represents a SQL migration on disk.
type Migration struct {
	Name string
	SQL  string
}

// RunMigrations applies pending migrations located in the migrationsDir.
func RunMigrations(db *sql.DB, migrationsDir string) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		applied, err := isMigrationApplied(db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if err := markMigrationApplied(db, name); err != nil {
			return err
		}
	}
	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at TIMESTAMP
);`)
	if err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}
	return nil
}

func isMigrationApplied(db *sql.DB, name string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(1) FROM schema_migrations WHERE name = ?", name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}
	return count > 0, nil
}

func markMigrationApplied(db *sql.DB, name string) error {
	_, err := db.Exec("INSERT INTO schema_migrations(name, applied_at) VALUES (?, ?)", name, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("mark migration applied %s: %w", name, err)
	}
	return nil
}
