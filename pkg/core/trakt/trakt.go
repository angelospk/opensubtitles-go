package trakt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Constants remain const as they are unlikely to change per-test
const (
	apiVersion     = "2"
	searchEndpoint = "/search"
)

// baseURL is now a variable to allow modification during tests.
var baseURL = "https://api.trakt.tv"

// SetBaseURLForTesting allows tests to temporarily override the API base URL.
// It returns the original URL so it can be restored.
func SetBaseURLForTesting(newURL string) string {
	oldURL := baseURL
	baseURL = newURL
	return oldURL
}

// --- Structs to decode Trakt API JSON response ---

// traktSearchResultItem mirrors the structure of items in the Trakt API search results.
type traktSearchResultItem struct {
	Type    string        `json:"type"` // movie, show, episode, person, list
	Score   float64       `json:"score"`
	Movie   *traktMovie   `json:"movie,omitempty"`
	Show    *traktShow    `json:"show,omitempty"`
	Episode *traktEpisode `json:"episode,omitempty"`
	// Person *traktPerson `json:"person,omitempty"`
	// List   *traktList   `json:"list,omitempty"`
}

type traktMovie struct {
	Title string    `json:"title"`
	Year  int       `json:"year"`
	IDs   *traktIDs `json:"ids"`
}

type traktShow struct {
	Title string    `json:"title"`
	Year  int       `json:"year"`
	IDs   *traktIDs `json:"ids"`
}

type traktEpisode struct {
	Season int       `json:"season"`
	Number int       `json:"number"`
	Title  string    `json:"title"`
	IDs    *traktIDs `json:"ids"`
}

type traktIDs struct {
	Trakt  int    `json:"trakt"`
	Slug   string `json:"slug,omitempty"`
	Tvdb   int    `json:"tvdb,omitempty"`
	Imdb   string `json:"imdb,omitempty"`
	Tmdb   int    `json:"tmdb,omitempty"`
	TvRage int    `json:"tvrage,omitempty"`
}

// --- Client Implementation ---

// Client handles communication with the Trakt API.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Trakt API client.
func NewClient() (*Client, error) {
	clientID := os.Getenv("TRAKT_CLIENT_ID")
	if clientID == "" {
		return nil, errors.New("TRAKT_CLIENT_ID environment variable not set")
	}

	return &Client{
		apiKey:     clientID,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}, nil
}

// --- Application Specific Structs & Helpers ---

// SearchResult represents a simplified search result item for the application.
type SearchResult struct {
	Type  string // e.g., "movie", "show", "episode"
	Year  int
	Title string
	IDs   map[string]string // e.g., {"trakt": "123", "imdb": "tt1234567"}
}

// idsToStringMap converts traktIDs to a string map.
func idsToStringMap(ids *traktIDs) map[string]string {
	m := make(map[string]string)
	if ids == nil {
		return m
	}
	if ids.Trakt != 0 {
		m["trakt"] = strconv.Itoa(ids.Trakt)
	}
	if ids.Slug != "" {
		m["slug"] = ids.Slug
	}
	if ids.Imdb != "" {
		m["imdb"] = ids.Imdb
	}
	if ids.Tmdb != 0 {
		m["tmdb"] = strconv.Itoa(ids.Tmdb)
	}
	if ids.Tvdb != 0 {
		m["tvdb"] = strconv.Itoa(ids.Tvdb)
	}
	if ids.TvRage != 0 {
		m["tvrage"] = strconv.Itoa(ids.TvRage)
	}
	return m
}

// --- API Call Methods ---

// SearchTrakt searches for movies, shows, or episodes on Trakt.
func (c *Client) SearchTrakt(ctx context.Context, queryType string, query string) ([]SearchResult, error) {
	// 1. Construct URL
	// Example: https://api.trakt.tv/search/movie,show,episode?query=tron
	endpoint := baseURL + searchEndpoint + "/" + url.PathEscape(queryType)
	reqURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trakt search URL: %w", err)
	}
	q := reqURL.Query()
	q.Set("query", query)
	// Add limit, page, extended info etc. here if needed
	// q.Set("limit", "25")
	reqURL.RawQuery = q.Encode()

	// 2. Create Request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create trakt search request: %w", err)
	}

	// 3. Add Headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("trakt-api-version", apiVersion)
	req.Header.Set("trakt-api-key", c.apiKey)

	// 4. Execute Request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute trakt search request: %w", err)
	}
	defer resp.Body.Close()

	// 5. Handle Non-200 Responses
	if resp.StatusCode != http.StatusOK {
		// TODO: Read body for more detailed error message from Trakt?
		return nil, fmt.Errorf("trakt search returned non-OK status: %s", resp.Status)
	}

	// 6. Decode JSON Response
	var traktResults []traktSearchResultItem
	if err := json.NewDecoder(resp.Body).Decode(&traktResults); err != nil {
		return nil, fmt.Errorf("failed to decode trakt search response: %w", err)
	}

	// 7. Map to Application Struct
	output := make([]SearchResult, 0, len(traktResults))
	for _, item := range traktResults {
		var sr SearchResult
		switch item.Type {
		case "movie":
			if item.Movie != nil {
				sr = SearchResult{
					Type:  item.Type,
					Year:  item.Movie.Year,
					Title: item.Movie.Title,
					IDs:   idsToStringMap(item.Movie.IDs),
				}
			}
		case "show":
			if item.Show != nil {
				sr = SearchResult{
					Type:  item.Type,
					Year:  item.Show.Year,
					Title: item.Show.Title,
					IDs:   idsToStringMap(item.Show.IDs),
				}
			}
		case "episode":
			if item.Episode != nil && item.Show != nil { // Episode context needs show info
				showTitle := item.Show.Title
				showYear := item.Show.Year
				epTitle := item.Episode.Title
				epSeason := item.Episode.Season
				epNumber := item.Episode.Number
				fullTitle := fmt.Sprintf("%s (%d) - %dx%d - %s", showTitle, showYear, epSeason, epNumber, epTitle)
				sr = SearchResult{
					Type:  item.Type,
					Year:  showYear,
					Title: fullTitle,
					IDs:   idsToStringMap(item.Episode.IDs),
				}
			}
		default:
			continue // Ignore person, list types for now
		}

		if sr.Title != "" { // Append only if we successfully mapped it
			output = append(output, sr)
		}
	}

	return output, nil
}
