package slskd

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"soundstack/internal/config"
	"soundstack/internal/db/models"
	"soundstack/internal/queue"
)

type Client struct {
	cfg    config.SlskdConfig
	client *http.Client
	logger *log.Logger
}

func NewClient(cfg config.SlskdConfig, logger *log.Logger) *Client {
	return &Client{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

// Check verifies basic connectivity to slskd if configured.
func Check(ctx context.Context, cfg config.SlskdConfig) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("slskd base_url not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.BaseURL+"/api/v0/state", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("slskd unhealthy: status %d", resp.StatusCode)
	}
	return nil
}

// DownloadAudio is a placeholder that records a queued audio download and enqueues tagging.
func (c *Client) DownloadAudio(ctx context.Context, trackID string, db *sql.DB, rules config.RulesConfig) error {
	id := uuid.NewString()
	_, err := db.ExecContext(ctx, `
INSERT INTO downloads (id, track_id, type, source, source_ref, quality_score, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`, id, trackID, models.DownloadTypeAudio, "slskd", "stub", 0, models.JobStatusDone)
	if err != nil {
		return err
	}
	return queue.CreateJobIfNotExists(ctx, db, trackID, models.JobStageTag, time.Now())
}
