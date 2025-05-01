package imdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// --- Structs for unofficial IMDB Suggestion API ---

// imdbSuggestionResponse mirrors the top-level structure.
type imdbSuggestionResponse struct {
	Version int                  `json:"v"`
	Query   string               `json:"q"`
	Data    []imdbSuggestionItem `json:"d"`
}

// imdbSuggestionItem mirrors the structure of individual suggestions.
// Note: Field names and existence can be inconsistent.
type imdbSuggestionItem struct {
	Label      string `json:"l"`              // Title
	ID         string `json:"id"`             // IMDb ID (e.g., "tt1234567")
	Starring   string `json:"s,omitempty"`    // Actors
	Year       int    `json:"y,omitempty"`    // Primary year field?
	YearRange  string `json:"yr,omitempty"`   // Sometimes used instead of 'y', format "YYYY-YYYY" or "YYYY"
	ResultType string `json:"q,omitempty"`    // e.g., "feature", "TV series", "short"
	Image      any    `json:"i,omitempty"`    // Image details (array), ignore for now
	LegacyType int    `json:"vt,omitempty"`   // Unknown, ignore
	Rank       int    `json:"rank,omitempty"` // Ranking?
}

// getYear tries to get the year from either 'y' or parses the start year from 'yr'.
func (item *imdbSuggestionItem) getYear() int {
	if item.Year != 0 {
		return item.Year
	}
	if item.YearRange != "" {
		// Try parsing YYYY or YYYY-YYYY
		parts := strings.Split(item.YearRange, "-")
		if len(parts) > 0 {
			year, err := strconv.Atoi(parts[0])
			if err == nil {
				return year
			}
		}
	}
	return 0 // Indicate year unknown/unparseable
}

// --- Client Implementation ---

// imdbBaseURL is variable for testing
var imdbBaseURL = "https://v3.sg.media-imdb.com"

// SetBaseURLForTesting allows tests to temporarily override the API base URL.
// It returns the original URL so it can be restored.
func SetBaseURLForTesting(newURL string) string {
	oldURL := imdbBaseURL
	imdbBaseURL = newURL
	return oldURL
}

// Client handles communication with the unofficial IMDB Suggestion API.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a new IMDB suggestion client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// --- Application Specific Struct ---

// IMDBSuggestion represents a simplified search result for the application.
type IMDBSuggestion struct {
	ID    string // e.g., "tt1234567"
	Title string
	Year  int
}

// --- API Call Method ---

// SearchIMDBSuggestions queries the unofficial IMDB suggestion API.
// WARNING: This endpoint is undocumented and may break without notice.
// It's primarily useful for finding movie IDs (tt*), filtering other types.
func (c *Client) SearchIMDBSuggestions(ctx context.Context, query string) ([]IMDBSuggestion, error) {
	if len(query) == 0 {
		return []IMDBSuggestion{}, nil // No query, no results
	}

	// 1. Prepare URL (e.g., https://v3.sg.media-imdb.com/suggestion/titles/t/tron.json)
	query = strings.ToLower(strings.TrimSpace(query))
	if len(query) == 0 {
		return []IMDBSuggestion{}, nil
	}
	firstLetter := string(query[0])
	encodedQuery := url.PathEscape(query) // Basic encoding for the path segment
	apiURL := fmt.Sprintf("%s/suggestion/titles/%s/%s.json", imdbBaseURL, firstLetter, encodedQuery)

	// 2. Create Request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create imdb suggestion request: %w", err)
	}
	// Note: This API doesn't seem to require specific headers like API keys.
	req.Header.Set("Accept", "application/json") // Good practice

	// 3. Execute Request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Don't treat network errors as fatal, as the endpoint is unstable.
		// Log the error instead? Return empty results?
		fmt.Printf("WARN: IMDB suggestion request failed: %v\n", err) // Log warning
		return []IMDBSuggestion{}, nil                                // Return empty instead of error
	}
	defer resp.Body.Close()

	// 4. Handle Non-200 Responses
	if resp.StatusCode != http.StatusOK {
		// Treat non-OK status as non-fatal as well
		fmt.Printf("WARN: IMDB suggestion request returned non-OK status: %s\n", resp.Status)
		return []IMDBSuggestion{}, nil // Return empty
	}

	// 5. Decode JSON Response
	var imdbResponse imdbSuggestionResponse
	if err := json.NewDecoder(resp.Body).Decode(&imdbResponse); err != nil {
		// Treat decoding errors as non-fatal
		fmt.Printf("WARN: Failed to decode IMDB suggestion response: %v\n", err)
		return []IMDBSuggestion{}, nil // Return empty
	}

	// 6. Map to Application Struct & Filter
	output := make([]IMDBSuggestion, 0, len(imdbResponse.Data))
	for _, item := range imdbResponse.Data {
		// Only include results that:
		// 1. Have a valid title (Label)
		// 2. Have a valid IMDb ID (starts with "tt")
		// 3. Are likely movies (ResultType is "feature" or empty/unknown)
		isLikelyMovie := item.ResultType == "feature" || item.ResultType == ""
		if item.ID != "" && strings.HasPrefix(item.ID, "tt") && item.Label != "" && isLikelyMovie {
			suggestion := IMDBSuggestion{
				ID:    item.ID,
				Title: item.Label,
				Year:  item.getYear(), // Use helper to get year
			}
			output = append(output, suggestion)
		}
	}

	return output, nil
}
