package queue

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"soundstack/internal/db/models"
	"soundstack/internal/spotify"
)

// UpsertTrack inserts the track if missing and returns its ID.
func UpsertTrack(ctx context.Context, db *sql.DB, t spotify.Track) (string, error) {
	var existingID string
	err := db.QueryRowContext(ctx, "SELECT id FROM tracks WHERE spotify_track_id = ?", t.ID).Scan(&existingID)
	if err == nil && existingID != "" {
		_, err := db.ExecContext(ctx, `
UPDATE tracks SET title = ?, artists_json = ?, album = ?, release_date = ?, duration_ms = ?, spotify_album_id = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?`, t.Title, t.ArtistsJSON, t.Album, t.ReleaseDate, t.DurationMS, t.AlbumID, existingID)
		if err != nil {
			return "", err
		}
		return existingID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	id := uuid.NewString()
	_, err = db.ExecContext(ctx, `
INSERT INTO tracks (id, spotify_track_id, spotify_album_id, title, artists_json, album, release_date, duration_ms, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`, id, t.ID, t.AlbumID, t.Title, t.ArtistsJSON, t.Album, t.ReleaseDate, t.DurationMS)
	if err != nil {
		return "", fmt.Errorf("insert track: %w", err)
	}
	return id, nil
}

// CreateJobIfNotExists enqueues a job for the given track and stage if not present.
func CreateJobIfNotExists(ctx context.Context, db *sql.DB, trackID, stage string, runAt time.Time) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(1) FROM jobs WHERE track_id = ? AND stage = ? AND status IN (?, ?, ?)`, trackID, stage, models.JobStatusQueued, models.JobStatusRunning, models.JobStatusDone).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	id := uuid.NewString()
	_, err := db.ExecContext(ctx, `
INSERT INTO jobs (id, track_id, stage, status, attempts, next_run_at, created_at, updated_at)
VALUES (?, ?, ?, ?, 0, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
`, id, trackID, stage, models.JobStatusQueued, runAt)
	return err
}

// FetchNextJob returns and marks a queued job as running.
func FetchNextJob(ctx context.Context, db *sql.DB) (*models.Job, error) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	job := models.Job{}
	err = tx.QueryRowContext(ctx, `
SELECT id, track_id, stage, status, attempts, next_run_at, last_error, created_at, updated_at
FROM jobs WHERE status = ? AND next_run_at <= CURRENT_TIMESTAMP
ORDER BY next_run_at ASC LIMIT 1
`, models.JobStatusQueued).Scan(&job.ID, &job.TrackID, &job.Stage, &job.Status, &job.Attempts, &job.NextRunAt, &job.LastError, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return nil, fmt.Errorf("query rollback: %v (orig: %w)", rbErr, err)
		}
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE jobs SET status = ?, attempts = attempts + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, models.JobStatusRunning, job.ID)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return nil, fmt.Errorf("update rollback: %v (orig: %w)", rbErr, err)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	job.Status = models.JobStatusRunning
	job.Attempts++
	return &job, nil
}

// MarkJobDone marks a job as completed.
func MarkJobDone(ctx context.Context, db *sql.DB, jobID string) error {
	_, err := db.ExecContext(ctx, `UPDATE jobs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, models.JobStatusDone, jobID)
	return err
}

// MarkJobFailed sets job status to failed with last error.
func MarkJobFailed(ctx context.Context, db *sql.DB, jobID string, errText string, delay time.Duration) error {
	_, err := db.ExecContext(ctx, `UPDATE jobs SET status = ?, last_error = ?, next_run_at = datetime(CURRENT_TIMESTAMP, ?), updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		models.JobStatusFailed, errText, fmt.Sprintf("+%d seconds", int(delay.Seconds())), jobID)
	return err
}

// ResetFailedJobs moves failed jobs back to queued.
func ResetFailedJobs(ctx context.Context, db *sql.DB) (int64, error) {
	res, err := db.ExecContext(ctx, `UPDATE jobs SET status = ?, attempts = 0, updated_at = CURRENT_TIMESTAMP WHERE status = ?`, models.JobStatusQueued, models.JobStatusFailed)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// JobCounts returns counts per status.
func JobCounts(ctx context.Context, db *sql.DB) (map[string]int64, error) {
	rows, err := db.QueryContext(ctx, `SELECT status, COUNT(1) FROM jobs GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, nil
}
