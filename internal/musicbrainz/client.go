package musicbrainz

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"soundstack/internal/config"
)

type Client struct {
	cfg    config.MusicBrainzConfig
	client *http.Client
	logger *log.Logger
}

func NewClient(cfg config.MusicBrainzConfig, logger *log.Logger) *Client {
	return &Client{
		cfg:    cfg,
		client: &http.Client{Timeout: 10 * time.Second},
		logger: logger,
	}
}

// CheckMV performs a minimal MusicBrainz lookup to determine if an MV might exist.
// This is a stub implementation that can be expanded.
func (c *Client) CheckMV(ctx context.Context, trackID string) (bool, string, error) {
	// For now, simply return false with evidence placeholder.
	evidence := map[string]any{
		"reason": "stubbed mv check",
		"track":  trackID,
	}
	data, _ := json.Marshal(evidence)
	return false, string(data), nil
}

// politeRequest builds a basic MusicBrainz request with required UA.
func (c *Client) politeRequest(ctx context.Context, method, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	return req, nil
}

// Rate limit helper (coarse).
func (c *Client) sleepRateLimit() {
	delay := time.Duration(c.cfg.RateLimitMS) * time.Millisecond
	time.Sleep(delay)
}
