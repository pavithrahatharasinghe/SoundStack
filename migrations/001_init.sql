CREATE TABLE IF NOT EXISTS tracks (
    id TEXT PRIMARY KEY,
    spotify_track_id TEXT UNIQUE,
    spotify_album_id TEXT,
    title TEXT,
    artists_json TEXT,
    album TEXT,
    release_date TEXT,
    duration_ms INTEGER,
    isrc TEXT,
    mb_recording_id TEXT,
    mb_release_id TEXT,
    bucket TEXT,
    mv_expected INTEGER DEFAULT 0,
    mv_evidence TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS downloads (
    id TEXT PRIMARY KEY,
    track_id TEXT,
    type TEXT,
    source TEXT,
    source_ref TEXT,
    quality_score INTEGER,
    status TEXT,
    file_path TEXT,
    file_size INTEGER,
    checksum TEXT,
    error_text TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(track_id) REFERENCES tracks(id)
);
CREATE INDEX IF NOT EXISTS idx_downloads_track ON downloads(track_id);

CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    track_id TEXT,
    stage TEXT,
    status TEXT,
    attempts INTEGER,
    next_run_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(track_id) REFERENCES tracks(id)
);
CREATE INDEX IF NOT EXISTS idx_jobs_track ON jobs(track_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);

