package ytdlp

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"soundstack/internal/config"
	"soundstack/internal/db/models"
	"soundstack/internal/queue"
)

type Runner struct {
	cfg    config.ToolConfig
	ffcfg  config.ToolConfig
	logger *log.Logger
}

func New(cfg config.ToolConfig, ff config.ToolConfig, logger *log.Logger) *Runner {
	return &Runner{cfg: cfg, ffcfg: ff, logger: logger}
}

func CheckBinary(path string) error {
	if path == "" {
		return fmt.Errorf("binary path not configured")
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	_, err := exec.LookPath(path)
	return err
}

// DownloadVideo is a stub that records a video download row and queues audio stage.
func (r *Runner) DownloadVideo(ctx context.Context, trackID string, db *sql.DB) error {
	id := uuid.NewString()
	_, err := db.ExecContext(ctx, `
INSERT INTO downloads (id, track_id, type, source, source_ref, quality_score, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`, id, trackID, models.DownloadTypeVideo, "ytdlp", "stub", 0, models.JobStatusDone)
	if err != nil {
		return err
	}
	return queue.CreateJobIfNotExists(ctx, db, trackID, models.JobStageAudio, time.Now())
}
