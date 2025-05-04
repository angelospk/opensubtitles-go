package opensubtitles

import (
	// Added for future method signatures
	"errors"
	"fmt"
	"net/url"
	"sync" // For thread-safe access to token/baseUrl

	"github.com/angelospk/opensubtitles-go/internal/constants"
	"github.com/angelospk/opensubtitles-go/internal/httpclient"
	// Placeholder import for upload package
	// "github.com/angelospk/opensubtitles-go/upload"
)

// Config holds the configuration for the OpenSubtitles client.
type Config struct {
	ApiKey    string
	UserAgent string
	BaseURL   string // Optional: Override default base URL
}

// Client is the main OpenSubtitles API client.
type Client struct {
	config         Config
	httpClient     *httpclient.Client // Internal HTTP client
	mu             sync.RWMutex       // Protects access to token and currentBaseUrl
	authToken      *string
	currentBaseUrl string
	// Add UploadClient interface here if needed later
	// uploader   upload.Uploader // Example
}

// NewClient creates a new OpenSubtitles API client.
func NewClient(config Config) (*Client, error) {
	if config.ApiKey == "" {
		return nil, errors.New("API key is required")
	}
	if config.UserAgent == "" {
		return nil, errors.New("User-Agent is required")
	}

	baseUrl := constants.DefaultBaseURL
	if config.BaseURL != "" {
		// Validate user-provided base URL slightly
		if _, err := url.ParseRequestURI(config.BaseURL); err != nil {
			return nil, fmt.Errorf("invalid BaseURL provided: %w", err)
		}
		baseUrl = config.BaseURL
	}

	c := &Client{
		config:         config,
		httpClient:     httpclient.New(baseUrl, config.ApiKey, config.UserAgent),
		currentBaseUrl: baseUrl,
	}

	// Initialize the uploader (if needed)
	// var err error
	// c.uploader, err = upload.NewXmlRpcUploader() // Example initialization
	// if err != nil {
	// 	 return nil, fmt.Errorf("failed to initialize uploader: %w", err)
	// }

	return c, nil
}

// SetAuthToken allows manually setting the auth token (e.g., loading from storage).
func (c *Client) SetAuthToken(token string, baseUrl string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if token == "" {
		c.authToken = nil
		c.httpClient.SetAuthToken(nil) // Clear in http client too
		// Optionally reset base URL? Keep it for now.
		return nil
	}

	// Validate and update base URL if provided
	if baseUrl != "" {
		parsedUrl, err := url.ParseRequestURI(baseUrl)
		if err != nil {
			return fmt.Errorf("invalid base URL provided: %w", err)
		}
		// Ensure scheme if just hostname was given (like in login response)
		if parsedUrl.Scheme == "" {
			baseUrl = "https://" + baseUrl
			// Re-parse to get full URL structure if needed by httpclient
			parsedUrl, err = url.ParseRequestURI(baseUrl)
			if err != nil {
				return fmt.Errorf("failed to construct full base URL: %w", err)
			}
		}
		// Assuming httpclient expects the full base URL with potential /api/v1 path
		// If the login response `base_url` is just `vip-api.opensubtitles.com`, adjust.
		if parsedUrl.Host != "" && parsedUrl.Path == "" {
			c.currentBaseUrl = baseUrl + "/api/v1" // Append standard path
		} else {
			c.currentBaseUrl = baseUrl // Assume full URL provided
		}

		c.httpClient.SetBaseURL(c.currentBaseUrl)
	}

	c.authToken = &token
	c.httpClient.SetAuthToken(&token)

	return nil
}

// GetCurrentToken returns the currently stored auth token.
func (c *Client) GetCurrentToken() *string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	// Return a copy to prevent external modification? For string, it's okay.
	return c.authToken
}

// GetCurrentBaseURL returns the base URL currently used by the client.
func (c *Client) GetCurrentBaseURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentBaseUrl
}

// Helper to check if authenticated
func (c *Client) isAuthenticated() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.authToken != nil && *c.authToken != ""
}

// --- Implement API methods in separate files (auth.go, subtitles.go, etc.) ---
// Example (in auth.go):
// func (c *Client) Login(ctx context.Context, params LoginRequest) (*LoginResponse, error) { ... }
