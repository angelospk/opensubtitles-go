package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/google/go-querystring/query"
)

// Client manages making HTTP requests to the API.
type Client struct {
	baseURL    string
	apiKey     string
	userAgent  string
	httpClient *http.Client
	mu         sync.RWMutex // Protects token
	authToken  *string
}

// New creates a new internal HTTP client.
func New(baseURL, apiKey, userAgent string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		userAgent:  userAgent,
		httpClient: &http.Client{}, // Use default client, customize if needed (timeout, transport)
	}
}

// SetBaseURL updates the base URL used for requests.
func (c *Client) SetBaseURL(baseURL string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = baseURL
}

// SetAuthToken updates the authentication token.
func (c *Client) SetAuthToken(token *string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.authToken = token
}

// Get makes a GET request.
func (c *Client) Get(ctx context.Context, path string, params interface{}, target interface{}) error {
	return c.doRequest(ctx, http.MethodGet, path, params, nil, target)
}

// Post makes a POST request.
func (c *Client) Post(ctx context.Context, path string, body interface{}, target interface{}) error {
	return c.doRequest(ctx, http.MethodPost, path, nil, body, target)
}

// Delete makes a DELETE request.
func (c *Client) Delete(ctx context.Context, path string, target interface{}) error {
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil, target)
}

// doRequest performs the actual HTTP request.
func (c *Client) doRequest(ctx context.Context, method, path string, params interface{}, body interface{}, target interface{}) error {
	c.mu.RLock()
	currentBaseURL := c.baseURL
	currentToken := c.authToken
	c.mu.RUnlock()

	fullURL, err := url.Parse(currentBaseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	fullURL.Path += path // Assumes baseURL doesn't end with / and path starts with /

	// Encode query parameters if provided
	if params != nil {
		v, err := query.Values(params)
		if err != nil {
			return fmt.Errorf("failed to encode query parameters: %w", err)
		}
		// TODO: Add logic to sort query parameters alphabetically and lowercase keys?
		// This is tricky with go-querystring directly. May need custom encoding or reflection.
		// For now, encode as is.
		fullURL.RawQuery = v.Encode()
	}

	// Encode request body if provided
	var reqBody io.Reader
	var contentType string
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Api-Key", c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Add Authorization header if token exists
	if currentToken != nil && *currentToken != "" {
		req.Header.Set("Authorization", "Bearer "+*currentToken)
	}

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Attempt to decode error response? Or just return status + body
		// Define custom error types? e.g., APIError
		return fmt.Errorf("api request failed: status %d, body: %s", resp.StatusCode, string(respBodyBytes))
		// Consider creating structured APIError type here
		// var apiErr APIError
		// if json.Unmarshal(respBodyBytes, &apiErr) == nil {
		//    apiErr.StatusCode = resp.StatusCode
		//    return apiErr
		// } else { ... fallback ...}
	}

	// Decode successful response if target is provided
	if target != nil {
		if err := json.Unmarshal(respBodyBytes, target); err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}

	return nil
}
