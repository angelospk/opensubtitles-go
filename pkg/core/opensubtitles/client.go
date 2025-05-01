package opensubtitles

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
)

const (
	DefaultBaseURL   = "https://api.opensubtitles.com/api/v1"
	DefaultUserAgent = "GoOpenSubtitlesUploader/0.1"
)

// Client manages communication with the OpenSubtitles API.
type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	userAgent  string

	tokenMu  sync.RWMutex // Protects access to jwtToken
	jwtToken string
}

// NewClient creates a new OpenSubtitles API client.
func NewClient(apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		httpClient: httpClient,
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		userAgent:  DefaultUserAgent,
	}
}

// --- Structs ---

// LoginRequest represents the request body for the login endpoint.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the successful response from the login endpoint.
type LoginResponse struct {
	User   UserInfo `json:"user"`
	Token  string   `json:"token"`
	Status int      `json:"status"`
}

// UserInfo represents user details provided by the API.
type UserInfo struct {
	AllowedDownloads int    `json:"allowed_downloads"`
	Level            string `json:"level"`
	UserID           int    `json:"user_id"`
	ExtInstalled     bool   `json:"ext_installed"`
	Vip              bool   `json:"vip"`
	DownloadsCount   int    `json:"downloads_count"`
	Username         string `json:"username"`
}

// FeaturesResponse represents the response from the feature search endpoint.
type FeaturesResponse struct {
	TotalPages int       `json:"total_pages"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	Data       []Feature `json:"data"`
}

// Feature represents a single movie or show feature.
type Feature struct {
	ID         string            `json:"id"`   // Usually numeric, but API might use string
	Type       string            `json:"type"` // e.g., "feature"
	Attributes FeatureAttributes `json:"attributes"`
}

// FeatureAttributes contains the details of a feature.
type FeatureAttributes struct {
	Title          string `json:"title"`
	OriginalTitle  string `json:"original_title"`
	ImdbID         int    `json:"imdb_id"`
	TmdbID         int    `json:"tmdb_id"`
	FeatureID      string `json:"feature_id"` // String ID, potentially the same as Feature.ID
	Year           int    `json:"year"`
	SubtitlesCount int    `json:"subtitles_count"`
	SeasonsCount   int    `json:"seasons_count"`
	ParentTitle    string `json:"parent_title"` // For episodes
	SeasonNumber   int    `json:"season_number"`
	EpisodeNumber  int    `json:"episode_number"`
	// Add other relevant fields as needed based on API docs
	// e.g., ImageURL, Ratings, etc.
	PosterPath string `json:"poster_path"` // Example
	URL        string `json:"url"`         // Example
}

// SubtitleSearchResponse represents the response from the subtitle search endpoint.
type SubtitleSearchResponse struct {
	TotalPages int        `json:"total_pages"`
	TotalCount int        `json:"total_count"`
	Page       int        `json:"page"`
	Data       []Subtitle `json:"data"`
}

// Subtitle represents a single subtitle entry.
type Subtitle struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"` // e.g., "subtitle"
	Attributes SubtitleAttributes `json:"attributes"`
}

// SubtitleAttributes contains the details of a subtitle.
type SubtitleAttributes struct {
	SubtitleID       string         `json:"subtitle_id"` // Same as Subtitle.ID?
	Language         string         `json:"language"`
	DownloadCount    int            `json:"download_count"`
	NewDownloadCount int            `json:"new_download_count"`
	HearingImpaired  bool           `json:"hearing_impaired"`
	HD               bool           `json:"hd"`
	Format           string         `json:"format"`
	Votes            int            `json:"votes"`
	Points           float64        `json:"points"` // API might use float
	Ratings          float64        `json:"ratings"`
	FromTrusted      bool           `json:"from_trusted"`
	ForeignPartsOnly bool           `json:"foreign_parts_only"`
	UploadDate       string         `json:"upload_date"` // Consider time.Time parsing?
	Release          string         `json:"release"`
	Comments         string         `json:"comments"`
	LegacySubtitleID int            `json:"legacy_subtitle_id"`
	Uploader         UploaderInfo   `json:"uploader"`
	FeatureDetails   FeatureInfo    `json:"feature_details"`
	URL              string         `json:"url"`
	RelatedLinks     []RelatedLink  `json:"related_links"`
	Files            []SubtitleFile `json:"files"`
	// Add other relevant fields as needed
	MoviehashMatch bool `json:"moviehash_match"`
}

// UploaderInfo contains details about the subtitle uploader.
type UploaderInfo struct {
	UploaderID int    `json:"uploader_id"`
	Name       string `json:"name"`
	Rank       string `json:"rank"`
}

// FeatureInfo contains details about the feature associated with the subtitle.
type FeatureInfo struct {
	FeatureID   int    `json:"feature_id"`
	FeatureType string `json:"feature_type"` // movie, tvshow, episode
	Year        int    `json:"year"`
	Title       string `json:"title"`
	MovieName   string `json:"movie_name"` // Duplicate of title?
	ImdbID      int    `json:"imdb_id"`
	TmdbID      int    `json:"tmdb_id"`
}

// RelatedLink represents a link related to the subtitle.
type RelatedLink struct {
	Label  string `json:"label"`
	URL    string `json:"url"`
	ImgURL string `json:"img_url"`
}

// SubtitleFile represents a file associated with a subtitle entry.
type SubtitleFile struct {
	FileID   int    `json:"file_id"`
	FileName string `json:"file_name"`
	CDNumber int    `json:"cd_number"`
}

// DownloadRequest represents the request body for the download endpoint.
type DownloadRequest struct {
	FileID        int    `json:"file_id"`
	SubFormat     string `json:"sub_format,omitempty"`     // Optional: srt, sub, etc.
	FileName      string `json:"file_name,omitempty"`      // Optional: Desired filename
	ForceDownload bool   `json:"force_download,omitempty"` // Optional
	// Add other optional parameters like 'in_fps', 'out_fps' if needed based on docs
}

// DownloadResponse represents the successful response from the download endpoint.
type DownloadResponse struct {
	Link         string `json:"link"`
	FileName     string `json:"file_name"`
	Remaining    int    `json:"remaining"`
	Message      string `json:"message"`
	ResetTime    string `json:"reset_time"`     // Consider time.Time parsing?
	ResetTimeUTC string `json:"reset_time_utc"` // Consider time.Time parsing?
	Requests     int    `json:"requests"`
	Allowed      int    `json:"allowed"` // Should this be string or int? Docs vary.
	Status       int    `json:"status"`  // Usually 200 OK
	VIP          bool   `json:"vip"`     // User VIP status after download
}

// ErrorResponse represents a standard error response from the API.
type ErrorResponse struct {
	Errors  []string `json:"errors"`
	Status  int      `json:"status"`  // Sometimes status is outside errors
	Message string   `json:"message"` // Used in some error cases like download limit
}

// Error implements the error interface.
func (r *ErrorResponse) Error() string {
	if r.Message != "" {
		return fmt.Sprintf("API Error (Status %d): %s", r.Status, r.Message)
	}
	if len(r.Errors) > 0 {
		return fmt.Sprintf("API Error (Status %d): %v", r.Status, r.Errors)
	}
	return fmt.Sprintf("API Error (Status %d): Unknown error", r.Status)
}

// --- Methods ---

// Login authenticates the user with the OpenSubtitles API.
func (c *Client) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	var loginResp LoginResponse
	err := c.doRequest(ctx, http.MethodPost, "/login", &loginReq, &loginResp, false) // Login does not require auth
	if err != nil {
		return nil, err // doRequest already wraps errors appropriately
	}

	// Store the token upon successful login
	c.tokenMu.Lock()
	c.jwtToken = loginResp.Token
	c.tokenMu.Unlock()

	return &loginResp, nil
}

// Logout invalidates the current user session token.
func (c *Client) Logout(ctx context.Context) error {
	c.tokenMu.RLock()
	tokenBeforeLogout := c.jwtToken
	c.tokenMu.RUnlock()

	if tokenBeforeLogout == "" {
		// Already logged out, treat as success (idempotent).
		return nil
	}

	// Perform the logout request - we don't expect a response body on success.
	err := c.doRequest(ctx, http.MethodDelete, "/logout", nil, nil, true) // Requires auth

	// Always clear the token locally, even if the API call failed
	// (e.g., due to network error or expired token on server-side).
	// The user intent is to be logged out.
	c.tokenMu.Lock()
	c.jwtToken = ""
	c.tokenMu.Unlock()

	if err != nil {
		// Check if the error was specifically because the token was invalid (e.g., 401)
		// In this case, we've achieved the goal (being logged out), so don't return an error.
		if apiErr, ok := err.(*ErrorResponse); ok && apiErr.Status == http.StatusUnauthorized {
			return nil // Effectively logged out
		}
		// Return other errors (network, server issues, etc.)
		return err
	}

	// Logout successful
	return nil
}

// doRequest performs the actual HTTP request to the OpenSubtitles API.
// It handles setting common headers, marshaling request bodies, sending the request,
// checking status codes, unmarshaling success responses, and parsing standard API errors.
// If requiresAuth is true, it adds the Authorization header with the stored JWT.
// reqBodyStruct should be a pointer to the struct to be marshaled as JSON body (nil if no body).
// respBodyStruct should be a pointer to the struct to unmarshal the JSON response into (nil if no response body expected).
func (c *Client) doRequest(ctx context.Context, method, relPath string, reqBodyStruct, respBodyStruct interface{}, requiresAuth bool) error {
	u, err := url.Parse(c.baseURL + relPath)
	if err != nil {
		return fmt.Errorf("failed to parse URL %s: %w", c.baseURL+relPath, err)
	}

	var reqBody io.Reader
	if reqBodyStruct != nil {
		jsonBody, err := json.Marshal(reqBodyStruct)
		if err != nil {
			return fmt.Errorf("failed to marshal request body for %s %s: %w", method, relPath, err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request for %s %s: %w", method, relPath, err)
	}

	// Set common headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Api-Key", c.apiKey)
	if reqBodyStruct != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add auth header if required
	if requiresAuth {
		c.tokenMu.RLock()
		token := c.jwtToken
		c.tokenMu.RUnlock()
		if token == "" {
			return fmt.Errorf("authentication required for %s %s, but client is not logged in", method, relPath)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request for %s %s: %w", method, relPath, err)
	}
	defer resp.Body.Close()

	// Check status code and handle errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp ErrorResponse
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr != nil {
			// Attempt to read the body for more context if JSON decoding fails
			bodyBytes, _ := io.ReadAll(resp.Body) // Ignore read error here
			return fmt.Errorf("API error for %s %s: status code %d, unable to parse error response: %w. Body: %s", method, relPath, resp.StatusCode, decErr, string(bodyBytes))
		}
		// Ensure status is set if not in JSON payload (some APIs might not include it)
		if errResp.Status == 0 {
			errResp.Status = resp.StatusCode
		}
		return &errResp
	}

	// Decode success response if a target struct is provided
	if respBodyStruct != nil {
		if err := json.NewDecoder(resp.Body).Decode(respBodyStruct); err != nil {
			return fmt.Errorf("failed to decode success response for %s %s: %w", method, relPath, err)
		}
	}

	return nil // Success
}

// GetUserInfo retrieves information about the currently authenticated user.
func (c *Client) GetUserInfo(ctx context.Context) (*UserInfo, error) {
	var userInfo UserInfo
	err := c.doRequest(ctx, http.MethodGet, "/infos/user", nil, &userInfo, true) // Requires auth, no request body
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
}

// SearchFeatures searches for movies or shows based on the provided parameters.
// Parameters should be provided as a map[string]string (e.g., {"query": "value", "year": "2023"}).
// Refer to the API documentation for available parameters.
func (c *Client) SearchFeatures(ctx context.Context, params map[string]string) (*FeaturesResponse, error) {
	relURL := "/features"
	u, err := url.Parse(c.baseURL + relURL)
	if err != nil {
		// This should ideally not happen if baseURL is valid, but handle defensively.
		return nil, fmt.Errorf("internal error parsing base URL for features: %w", err)
	}

	// Build query string from parameters
	query := u.Query()
	for k, v := range params {
		if v != "" { // Only add non-empty parameters
			query.Set(k, v)
		}
	}
	u.RawQuery = query.Encode()

	// We need to pass the relative path with the query to doRequest
	relPathWithQuery := relURL + "?" + u.RawQuery

	var featuresResp FeaturesResponse
	// Features search does not typically require authentication based on docs/common practice.
	err = c.doRequest(ctx, http.MethodGet, relPathWithQuery, nil, &featuresResp, false)
	if err != nil {
		return nil, err
	}

	return &featuresResp, nil
}

// SearchSubtitles searches for subtitles based on the provided parameters.
// Parameters should be provided as a map[string]string (e.g., {"imdb_id": "123", "languages": "en,fr"}).
// Refer to the API documentation for available parameters.
func (c *Client) SearchSubtitles(ctx context.Context, params map[string]string) (*SubtitleSearchResponse, error) {
	relURL := "/subtitles"
	u, err := url.Parse(c.baseURL + relURL)
	if err != nil {
		return nil, fmt.Errorf("internal error parsing base URL for subtitles: %w", err)
	}

	// Build query string from parameters
	query := u.Query()
	for k, v := range params {
		if v != "" {
			query.Set(k, v)
		}
	}
	u.RawQuery = query.Encode()

	relPathWithQuery := relURL + "?" + u.RawQuery

	var subtitlesResp SubtitleSearchResponse
	// Subtitle search also does not typically require authentication.
	err = c.doRequest(ctx, http.MethodGet, relPathWithQuery, nil, &subtitlesResp, false)
	if err != nil {
		return nil, err
	}

	return &subtitlesResp, nil
}

// RequestDownload requests a temporary download link for a specific subtitle file.
// Authentication is required.
func (c *Client) RequestDownload(ctx context.Context, reqData DownloadRequest) (*DownloadResponse, error) {
	var downloadResp DownloadResponse
	// Pass the request data struct directly to doRequest for marshaling.
	err := c.doRequest(ctx, http.MethodPost, "/download", &reqData, &downloadResp, true) // Requires auth
	if err != nil {
		// Specific error handling for 404 (file not found) or 429 (rate limit)?
		// The generic error handling in doRequest already returns *ErrorResponse.
		// We can check the status code here if needed.
		// if apiErr, ok := err.(*ErrorResponse); ok {
		// 	if apiErr.Status == http.StatusNotFound { ... }
		// }
		return nil, err
	}

	return &downloadResp, nil
}

// TODO: Implement /upload endpoint
// TODO: Consider adding a helper function for the actual file download (GET request on the link)
