package tagger

import (
	"context"
	"database/sql"
	"log"
	"time"

	"soundstack/internal/db/models"
	"soundstack/internal/queue"
)

type Tagger struct {
	logger *log.Logger
}

func New(logger *log.Logger) *Tagger {
	return &Tagger{logger: logger}
}

// Tag is a placeholder for FLAC tagging and cover art embedding.
func (t *Tagger) Tag(ctx context.Context, db *sql.DB, trackID string) error {
	t.logger.Printf("tagging track %s (stub)", trackID)
	return queue.CreateJobIfNotExists(ctx, db, trackID, models.JobStageOrganize, time.Now())
}
