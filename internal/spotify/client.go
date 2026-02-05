package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"log"

	"soundstack/internal/config"
)

type Client struct {
	cfg    config.SpotifyConfig
	logger *log.Logger
	client *http.Client
}

type Track struct {
	ID          string
	AlbumID     string
	Title       string
	ArtistsJSON string
	Album       string
	ReleaseDate string
	DurationMS  int64
	ISRC        string
}

func NewClient(cfg config.SpotifyConfig, logger *log.Logger) *Client {
	return &Client{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) FetchPlaylistTracks(ctx context.Context, playlistID string) ([]Track, error) {
	if c.cfg.ClientID == "" || c.cfg.ClientSecret == "" {
		return nil, errors.New("spotify client credentials are required")
	}
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}
	var tracks []Track
	urlStr := fmt.Sprintf("https://api.spotify.com/v1/playlists/%s/tracks", url.PathEscape(playlistID))
	for urlStr != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("spotify api status %d: %s", resp.StatusCode, string(body))
		}
		var pr playlistResponse
		if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		for _, item := range pr.Items {
			if item.Track.ID == "" {
				continue
			}
			artists := []string{}
			for _, a := range item.Track.Artists {
				artists = append(artists, a.Name)
			}
			artistJSON, _ := json.Marshal(artists)
			tracks = append(tracks, Track{
				ID:          item.Track.ID,
				AlbumID:     item.Track.Album.ID,
				Title:       item.Track.Name,
				ArtistsJSON: string(artistJSON),
				Album:       item.Track.Album.Name,
				ReleaseDate: item.Track.Album.ReleaseDate,
				DurationMS:  item.Track.DurationMS,
				ISRC:        item.Track.ExternalIDs.ISRC,
			})
		}
		urlStr = pr.Next
	}
	return tracks, nil
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	cred := base64.StdEncoding.EncodeToString([]byte(c.cfg.ClientID + ":" + c.cfg.ClientSecret))
	req.Header.Set("Authorization", "Basic "+cred)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed: %s", string(body))
	}
	var t struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", err
	}
	return t.AccessToken, nil
}

type playlistResponse struct {
	Items []struct {
		Track struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DurationMS  int64  `json:"duration_ms"`
			ExternalIDs struct {
				ISRC string `json:"isrc"`
			} `json:"external_ids"`
			Album struct {
				ID          string `json:"id"`
				Name        string `json:"name"`
				ReleaseDate string `json:"release_date"`
			} `json:"album"`
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"track"`
	} `json:"items"`
	Next string `json:"next"`
}
