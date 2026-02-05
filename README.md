# SoundStack

SoundStack is a long-running daemon and CLI that ingests Spotify playlists, records state in SQLite, and orchestrates downloading audio/video, tagging, and organizing your offline library.

## Features (initial slice)
- SQLite-backed state with migrations.
- Job queue with stages for metadata, music video check, video download, audio download, tagging, and organization.
- CLI commands: `init`, `doctor`, `sync playlist`, `run`, `status`, `retry`, `scan`.
- Example configuration and `.env` support.

## Getting Started
1. Copy the example config:
   ```bash
   cp configs/config.example.yaml configs/config.yaml
   cp .env.example .env
   ```
   Fill in Spotify client credentials and paths to yt-dlp/ffprobe.

2. Initialize storage and database:
   ```bash
   go run ./cmd/soundstack init
   ```

3. Verify dependencies:
   ```bash
   go run ./cmd/soundstack doctor
   ```

4. Sync a playlist into the queue:
   ```bash
   go run ./cmd/soundstack sync playlist --id <spotify_playlist_id>
   ```

5. Run workers (exit when queue is empty with `--once`):
   ```bash
   go run ./cmd/soundstack run --once
   ```

6. Check queue status:
   ```bash
   go run ./cmd/soundstack status
   ```

## Folder Layout
- `Music/FLAC/<BUCKET>/<Audio Only|With Video>/...`
- `Videos/<BUCKET>/...`

Buckets are derived from Spotify genres or overrides defined in config.

## Migrations
SQL migrations live in `migrations/` and are applied automatically on startup.

## Notes
This repository includes stub implementations for MusicBrainz checks, Soulseek downloads, tagging, and organization to provide a compilable, resumable pipeline skeleton. Extend these components to add full functionality.
