package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration loaded from YAML/env.
type Config struct {
	App         AppConfig         `yaml:"app"`
	Spotify     SpotifyConfig     `yaml:"spotify"`
	MusicBrainz MusicBrainzConfig `yaml:"musicbrainz"`
	Slskd       SlskdConfig       `yaml:"slskd"`
	YtDlp       ToolConfig        `yaml:"ytdlp"`
	Ffprobe     ToolConfig        `yaml:"ffprobe"`
	Library     LibraryConfig     `yaml:"library"`
	Rules       RulesConfig       `yaml:"rules"`
}

type AppConfig struct {
	DataDir     string `yaml:"data_dir"`
	LogLevel    string `yaml:"log_level"`
	Concurrency int    `yaml:"concurrency"`
	SecretsFile string `yaml:"secrets_file"`
}

type SpotifyConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RedirectURL  string `yaml:"redirect_url"`
	RefreshToken string `yaml:"refresh_token"`
}

type MusicBrainzConfig struct {
	UserAgent   string `yaml:"user_agent"`
	RateLimitMS int    `yaml:"rate_limit_ms"`
	CacheTTL    string `yaml:"cache_ttl"`
	CacheDir    string `yaml:"cache_dir"`
}

type SlskdConfig struct {
	BaseURL  string `yaml:"base_url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type ToolConfig struct {
	Path string `yaml:"path"`
}

type LibraryConfig struct {
	Root               string `yaml:"root"`
	FlacRoot           string `yaml:"flac_root"`
	VideoRoot          string `yaml:"video_root"`
	CreateArtistFolder bool   `yaml:"create_artist_folders"`
}

type RulesConfig struct {
	BucketOverrides    map[string]string `yaml:"bucket_overrides"`
	AlwaysTryVideo     bool              `yaml:"always_try_video"`
	MaxAudioCandidates int               `yaml:"max_audio_candidates"`
	DurationTolerance  int               `yaml:"duration_tolerance_sec"`
	PlaylistBuckets    map[string]string `yaml:"playlist_buckets"`
}

// Load reads configuration from the provided YAML file and optional secrets file.
func Load(path string) (*Config, error) {
	_ = godotenv.Load()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.App.SecretsFile != "" {
		secretPath := cfg.App.SecretsFile
		if !filepath.IsAbs(secretPath) {
			secretPath = filepath.Join(filepath.Dir(path), secretPath)
		}
		if _, err := os.Stat(secretPath); err == nil {
			secretBytes, err := os.ReadFile(secretPath)
			if err != nil {
				return nil, fmt.Errorf("read secrets file: %w", err)
			}
			secretCfg := Config{}
			if err := yaml.Unmarshal(secretBytes, &secretCfg); err != nil {
				return nil, fmt.Errorf("parse secrets file: %w", err)
			}
			mergeConfigs(cfg, &secretCfg)
		}
	}

	applyEnvOverrides(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func mergeConfigs(base, override *Config) {
	if override.Spotify.ClientID != "" {
		base.Spotify.ClientID = override.Spotify.ClientID
	}
	if override.Spotify.ClientSecret != "" {
		base.Spotify.ClientSecret = override.Spotify.ClientSecret
	}
	if override.Spotify.RedirectURL != "" {
		base.Spotify.RedirectURL = override.Spotify.RedirectURL
	}
	if override.Spotify.RefreshToken != "" {
		base.Spotify.RefreshToken = override.Spotify.RefreshToken
	}
	if override.Slskd.BaseURL != "" {
		base.Slskd.BaseURL = override.Slskd.BaseURL
	}
	if override.Slskd.Username != "" {
		base.Slskd.Username = override.Slskd.Username
	}
	if override.Slskd.Password != "" {
		base.Slskd.Password = override.Slskd.Password
	}
	if override.App.DataDir != "" {
		base.App.DataDir = override.App.DataDir
	}
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SPOTIFY_CLIENT_ID"); v != "" {
		cfg.Spotify.ClientID = v
	}
	if v := os.Getenv("SPOTIFY_CLIENT_SECRET"); v != "" {
		cfg.Spotify.ClientSecret = v
	}
	if v := os.Getenv("SPOTIFY_REFRESH_TOKEN"); v != "" {
		cfg.Spotify.RefreshToken = v
	}
	if v := os.Getenv("SLSKD_USERNAME"); v != "" {
		cfg.Slskd.Username = v
	}
	if v := os.Getenv("SLSKD_PASSWORD"); v != "" {
		cfg.Slskd.Password = v
	}
}

// Validate performs basic config validation and sets defaults.
func (c *Config) Validate() error {
	if c.App.DataDir == "" {
		c.App.DataDir = "./storage"
	}
	if c.App.Concurrency <= 0 {
		c.App.Concurrency = 2
	}
	if c.MusicBrainz.UserAgent == "" {
		c.MusicBrainz.UserAgent = "SoundStack/0.1 (contact@example.com)"
	}
	if c.MusicBrainz.RateLimitMS <= 0 {
		c.MusicBrainz.RateLimitMS = 1000
	}
	if c.Rules.MaxAudioCandidates <= 0 {
		c.Rules.MaxAudioCandidates = 8
	}
	if c.Rules.DurationTolerance <= 0 {
		c.Rules.DurationTolerance = 3
	}
	if c.Library.FlacRoot == "" {
		c.Library.FlacRoot = "Music/FLAC"
	}
	if c.Library.VideoRoot == "" {
		c.Library.VideoRoot = "Videos"
	}
	if c.Library.Root == "" {
		return errors.New("library.root is required")
	}
	if _, err := time.ParseDuration(fmt.Sprintf("%dms", c.MusicBrainz.RateLimitMS)); err != nil {
		return fmt.Errorf("invalid musicbrainz rate_limit_ms: %w", err)
	}
	return nil
}

// WriteExample writes an example configuration to the given path.
func WriteExample(path string) error {
	example := Config{
		App: AppConfig{
			DataDir:     "./storage",
			LogLevel:    "info",
			Concurrency: 4,
			SecretsFile: "./configs/secrets.yaml",
		},
		Spotify: SpotifyConfig{
			ClientID:     "",
			ClientSecret: "",
			RedirectURL:  "",
			RefreshToken: "",
		},
		MusicBrainz: MusicBrainzConfig{
			UserAgent:   "SoundStack/0.1 (email@example.com)",
			RateLimitMS: 1000,
			CacheTTL:    "24h",
			CacheDir:    "./storage/cache/musicbrainz",
		},
		Slskd: SlskdConfig{
			BaseURL:  "http://localhost:5030",
			Username: "",
			Password: "",
		},
		YtDlp: ToolConfig{
			Path: "./tools/yt-dlp/yt-dlp",
		},
		Ffprobe: ToolConfig{
			Path: "./tools/ffmpeg/ffprobe",
		},
		Library: LibraryConfig{
			Root:               "D:/ORGANIZED MEDIA/Music",
			FlacRoot:           "Music/FLAC",
			VideoRoot:          "Videos",
			CreateArtistFolder: true,
		},
		Rules: RulesConfig{
			BucketOverrides: map[string]string{
				"福原美穂": "J-POP",
			},
			AlwaysTryVideo:     true,
			MaxAudioCandidates: 8,
			DurationTolerance:  3,
			PlaylistBuckets:    map[string]string{},
		},
	}
	out, err := yaml.Marshal(&example)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
