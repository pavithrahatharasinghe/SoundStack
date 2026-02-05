package organizer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"soundstack/internal/config"
)

type Organizer struct {
	lib    config.LibraryConfig
	rules  config.RulesConfig
	logger *log.Logger
}

func New(lib config.LibraryConfig, rules config.RulesConfig, logger *log.Logger) *Organizer {
	return &Organizer{lib: lib, rules: rules, logger: logger}
}

func (o *Organizer) Place(ctx context.Context, db *sql.DB, trackID string) error {
	var title, artistsJSON, bucket string
	err := db.QueryRowContext(ctx, `SELECT title, artists_json, COALESCE(bucket, 'OTHER') FROM tracks WHERE id = ?`, trackID).Scan(&title, &artistsJSON, &bucket)
	if err != nil {
		return err
	}
	destDir := filepath.Join(o.lib.Root, o.lib.FlacRoot, bucket, "Audio Only")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("create dest dir: %w", err)
	}
	o.logger.Printf("organized track %s -> %s", title, destDir)
	_, err = db.ExecContext(ctx, `UPDATE tracks SET updated_at = ? WHERE id = ?`, time.Now(), trackID)
	return err
}
