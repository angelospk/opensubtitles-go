package opensubtitles

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	reqBodyBytes, err := json.Marshal(loginReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login request: %w", err)
	}

	relURL := "/login"
	u, err := url.Parse(c.baseURL + relURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL %s: %w", c.baseURL+relURL, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(reqBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create login request: %w", err)
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Api-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr != nil {
			// If decoding fails, return a generic error with status code
			return nil, fmt.Errorf("API error: status code %d, unable to parse error response: %w", resp.StatusCode, decErr)
		}
		errResp.Status = resp.StatusCode // Ensure status is set if not in JSON
		return nil, &errResp
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode login response: %w", err)
	}

	// Store the token upon successful login
	c.tokenMu.Lock()
	c.jwtToken = loginResp.Token
	c.tokenMu.Unlock()

	return &loginResp, nil
}

// TODO: Implement Logout()
// TODO: Implement doRequest() helper
// TODO: Implement other API endpoints (GetUserInfo, SearchFeatures, etc.)
