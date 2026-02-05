package app

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"soundstack/internal/config"
	"soundstack/internal/db"
	"soundstack/internal/db/models"
	"soundstack/internal/musicbrainz"
	"soundstack/internal/organizer"
	"soundstack/internal/queue"
	"soundstack/internal/slskd"
	"soundstack/internal/spotify"
	"soundstack/internal/tagger"
	"soundstack/internal/verifier"
	"soundstack/internal/ytdlp"
)

type App struct {
	Config *config.Config
	DB     *sql.DB
	Logger *log.Logger
}

func New(cfg *config.Config) (*App, error) {
	dbConn, err := db.Open(cfg.App.DataDir)
	if err != nil {
		return nil, err
	}
	if err := db.RunMigrations(dbConn, "migrations"); err != nil {
		return nil, err
	}
	return &App{
		Config: cfg,
		DB:     dbConn,
		Logger: log.New(os.Stdout, "[soundstack] ", log.LstdFlags|log.Lmicroseconds),
	}, nil
}

func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
}

// InitFilesystem prepares storage directories and writes example config files.
func (a *App) InitFilesystem() error {
	dirs := []string{
		a.Config.App.DataDir,
		filepath.Join(a.Config.App.DataDir, "downloads"),
		filepath.Join(a.Config.App.DataDir, "tmp"),
		filepath.Join(a.Config.App.DataDir, "quarantine"),
		filepath.Join(a.Config.App.DataDir, "cache"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", d, err)
		}
	}
	return nil
}

func (a *App) Doctor(ctx context.Context) error {
	checks := []struct {
		name string
		fn   func() error
	}{
		{"yt-dlp", func() error { return ytdlp.CheckBinary(a.Config.YtDlp.Path) }},
		{"ffprobe", func() error { return verifier.CheckBinary(a.Config.Ffprobe.Path) }},
		{"slskd", func() error { return slskd.Check(ctx, a.Config.Slskd) }},
	}
	for _, c := range checks {
		if err := c.fn(); err != nil {
			a.Logger.Printf("doctor: %s: %v", c.name, err)
			return err
		}
		a.Logger.Printf("doctor: %s OK", c.name)
	}
	return nil
}

func (a *App) SyncPlaylist(ctx context.Context, playlistID string, watch bool) error {
	client := spotify.NewClient(a.Config.Spotify, a.Logger)
	tracks, err := client.FetchPlaylistTracks(ctx, playlistID)
	if err != nil {
		return err
	}
	a.Logger.Printf("found %d tracks", len(tracks))
	for _, t := range tracks {
		trackID, err := queue.UpsertTrack(ctx, a.DB, t)
		if err != nil {
			return err
		}
		if err := queue.CreateJobIfNotExists(ctx, a.DB, trackID, models.JobStageMetadata, time.Now()); err != nil {
			return err
		}
	}
	if watch {
		a.Logger.Printf("watch mode not implemented, exiting after initial sync")
	}
	return nil
}

func (a *App) Run(ctx context.Context, once bool) error {
	for {
		job, err := queue.FetchNextJob(ctx, a.DB)
		if err != nil {
			return err
		}
		if job == nil {
			if once {
				return nil
			}
			time.Sleep(time.Second)
			continue
		}
		if err := a.handleJob(ctx, job); err != nil {
			a.Logger.Printf("job %s failed: %v", job.ID, err)
			queue.MarkJobFailed(ctx, a.DB, job.ID, err.Error(), 30*time.Second)
			continue
		}
		queue.MarkJobDone(ctx, a.DB, job.ID)
	}
}

func (a *App) handleJob(ctx context.Context, job *models.Job) error {
	switch job.Stage {
	case models.JobStageMetadata:
		return a.handleMetadata(ctx, job)
	case models.JobStageMVCheck:
		return a.handleMVCheck(ctx, job)
	case models.JobStageVideo:
		return a.handleVideo(ctx, job)
	case models.JobStageAudio:
		return a.handleAudio(ctx, job)
	case models.JobStageTag:
		return a.handleTag(ctx, job)
	case models.JobStageOrganize:
		return a.handleOrganize(ctx, job)
	case models.JobStageScan:
		return a.handleScan(ctx, job)
	default:
		return fmt.Errorf("unknown stage %s", job.Stage)
	}
}

func (a *App) handleMetadata(ctx context.Context, job *models.Job) error {
	// Stub metadata resolution; future MB lookup can go here.
	return queue.CreateJobIfNotExists(ctx, a.DB, job.TrackID, models.JobStageMVCheck, time.Now())
}

func (a *App) handleMVCheck(ctx context.Context, job *models.Job) error {
	client := musicbrainz.NewClient(a.Config.MusicBrainz, a.Logger)
	mvExpected, evidence, err := client.CheckMV(ctx, job.TrackID)
	if err != nil {
		return err
	}
	_, err = a.DB.ExecContext(ctx, `UPDATE tracks SET mv_expected = ?, mv_evidence = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		mvExpected, evidence, job.TrackID)
	if err != nil {
		return err
	}
	if mvExpected || a.Config.Rules.AlwaysTryVideo {
		return queue.CreateJobIfNotExists(ctx, a.DB, job.TrackID, models.JobStageVideo, time.Now())
	}
	return queue.CreateJobIfNotExists(ctx, a.DB, job.TrackID, models.JobStageAudio, time.Now())
}

func (a *App) handleVideo(ctx context.Context, job *models.Job) error {
	downloader := ytdlp.New(a.Config.YtDlp, a.Config.Ffprobe, a.Logger)
	return downloader.DownloadVideo(ctx, job.TrackID, a.DB)
}

func (a *App) handleAudio(ctx context.Context, job *models.Job) error {
	client := slskd.NewClient(a.Config.Slskd, a.Logger)
	return client.DownloadAudio(ctx, job.TrackID, a.DB, a.Config.Rules)
}

func (a *App) handleTag(ctx context.Context, job *models.Job) error {
	t := tagger.New(a.Logger)
	return t.Tag(ctx, a.DB, job.TrackID)
}

func (a *App) handleOrganize(ctx context.Context, job *models.Job) error {
	org := organizer.New(a.Config.Library, a.Config.Rules, a.Logger)
	return org.Place(ctx, a.DB, job.TrackID)
}

func (a *App) handleScan(ctx context.Context, job *models.Job) error {
	// Placeholder scanner
	return nil
}

func (a *App) RetryFailed(ctx context.Context) (int64, error) {
	return queue.ResetFailedJobs(ctx, a.DB)
}

func (a *App) JobCounts(ctx context.Context) (map[string]int64, error) {
	return queue.JobCounts(ctx, a.DB)
}
