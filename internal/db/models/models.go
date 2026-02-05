package models

import (
	"database/sql"
	"time"
)

const (
	JobStageIngest   = "ingest"
	JobStageMetadata = "metadata"
	JobStageMVCheck  = "mv_check"
	JobStageVideo    = "video"
	JobStageAudio    = "audio"
	JobStageTag      = "tag"
	JobStageOrganize = "organize"
	JobStageScan     = "scan"

	JobStatusQueued  = "queued"
	JobStatusRunning = "running"
	JobStatusDone    = "done"
	JobStatusFailed  = "failed"
	JobStatusRetry   = "retry"

	DownloadTypeAudio = "audio"
	DownloadTypeVideo = "video"
)

type Track struct {
	ID             string
	SpotifyTrackID string
	SpotifyAlbumID string
	Title          string
	ArtistsJSON    string
	Album          string
	ReleaseDate    string
	DurationMS     int64
	ISRC           sql.NullString
	MBRecordingID  sql.NullString
	MBReleaseID    sql.NullString
	Bucket         sql.NullString
	MVExpected     bool
	MVEvidence     sql.NullString
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Download struct {
	ID           string
	TrackID      string
	Type         string
	Source       string
	SourceRef    string
	QualityScore int
	Status       string
	FilePath     sql.NullString
	FileSize     sql.NullInt64
	Checksum     sql.NullString
	ErrorText    sql.NullString
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Job struct {
	ID        string
	TrackID   string
	Stage     string
	Status    string
	Attempts  int
	NextRunAt time.Time
	LastError sql.NullString
	CreatedAt time.Time
	UpdatedAt time.Time
}
