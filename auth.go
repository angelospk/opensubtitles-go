package opensubtitles

import (
	"context"
)

// Methods related to authentication (Login, Logout, GetUserInfo)

// Login authenticates the user with username and password, retrieving an API token.
// The token and the appropriate base URL (e.g., vip-api.opensubtitles.com) are stored
// internally in the client for subsequent requests.
func (c *Client) Login(ctx context.Context, params LoginRequest) (*LoginResponse, error) {
	var response LoginResponse
	err := c.httpClient.Post(ctx, "/login", params, &response)
	if err != nil {
		// Clear any potentially stale token if login fails
		_ = c.SetAuthToken("", "") // Ignore error during cleanup
		return nil, err
	}

	// Store the token and base URL from the response
	// SetAuthToken handles potential URL scheme/path adjustments
	err = c.SetAuthToken(response.Token, response.BaseURL)
	if err != nil {
		// Should ideally not happen if response.BaseURL is valid
		return nil, err
	}

	return &response, nil
}

// Logout invalidates the current API token.
// It clears the token stored internally in the client.
func (c *Client) Logout(ctx context.Context) (*LogoutResponse, error) {
	// Check if authenticated before attempting logout?
	// The API might return an error anyway if no valid token is provided.
	// Let the httpClient handle adding the token header if it exists.
	// if !c.isAuthenticated() {
	// 	return nil, errors.New("not logged in")
	// }

	var response LogoutResponse
	err := c.httpClient.Delete(ctx, "/logout", &response)
	if err != nil {
		// Don't clear the token if the API call failed,
		// as the token might still be valid.
		return nil, err
	}

	// Clear the internal token on successful logout
	_ = c.SetAuthToken("", "") // Reset token, keep base URL

	return &response, nil
}

// GetUserInfo retrieves information about the currently authenticated user.
// Requires authentication (a valid token must be set in the client).
func (c *Client) GetUserInfo(ctx context.Context) (*GetUserInfoResponse, error) {
	// No client-side check for auth needed here. If token is missing/invalid,
	// the httpclient will make the request without Authorization header (or with invalid one),
	// and the API will return a 401, which httpclient transforms into an error.
	var response GetUserInfoResponse
	err := c.httpClient.Get(ctx, "/infos/user", nil, &response) // No query params or body
	if err != nil {
		return nil, err
	}

	return &response, nil
}
