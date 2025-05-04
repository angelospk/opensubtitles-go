package opensubtitles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	// "time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create a mock server and client for tests
func setupTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close) // Ensure server is closed after test

	config := Config{
		ApiKey:    "test-api-key",
		UserAgent: "GoTestClient/1.0",
		BaseURL:   server.URL + "/api/v1", // Point client to mock server, assuming base path
	}
	client, err := NewClient(config)
	require.NoError(t, err, "Failed to create client for test")
	return server, client
}

// String is a helper function to return a pointer to a string.
// Useful for optional string parameters in API request structs.
func String(s string) *string {
	return &s
}

func TestLogin(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		expectedToken := "mock-jwt-token"
		expectedBaseURL := "vip-api.opensubtitles.com" // Test the base URL update
		expectedUserID := 123

		handler := func(w http.ResponseWriter, r *http.Request) {
			// 1. Assert Request Correctness
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/login", r.URL.Path) // Check full path
			assert.Equal(t, "test-api-key", r.Header.Get("Api-Key"))
			assert.Equal(t, "GoTestClient/1.0", r.Header.Get("User-Agent"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			var reqBody LoginRequest
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			require.NoError(t, err)
			assert.Equal(t, "testuser", reqBody.Username)
			assert.Equal(t, "testpass", reqBody.Password)

			// 2. Send Mock Response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := LoginResponse{
				User: LoginUser{
					BaseUserInfo: BaseUserInfo{UserID: expectedUserID, Level: "Tester", AllowedDownloads: 10},
				},
				Token:   expectedToken,
				BaseURL: expectedBaseURL, // Return the VIP base URL
				Status:  http.StatusOK,
			}
			err = json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}

		_, client := setupTestServer(t, handler)

		// 3. Call the method
		loginResp, err := client.Login(context.Background(), LoginRequest{Username: "testuser", Password: "testpass"})

		// 4. Assert Results
		require.NoError(t, err)
		require.NotNil(t, loginResp)
		assert.Equal(t, expectedToken, loginResp.Token)
		assert.Equal(t, expectedBaseURL, loginResp.BaseURL)
		assert.Equal(t, expectedUserID, loginResp.User.UserID)
		assert.Equal(t, http.StatusOK, loginResp.Status)

		// 5. Assert Client State Update
		assert.NotNil(t, client.GetCurrentToken())
		assert.Equal(t, expectedToken, *client.GetCurrentToken())
		// BaseURL stored in client should include scheme and path
		expectedClientBaseURL := "https://" + expectedBaseURL + "/api/v1"
		assert.Equal(t, expectedClientBaseURL, client.GetCurrentBaseURL())
		assert.True(t, client.isAuthenticated())
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/api/v1/login", r.URL.Path)

			// Send 401 Unauthorized
			w.Header().Set("Content-Type", "application/json") // API might still return JSON error
			w.WriteHeader(http.StatusUnauthorized)
			// Simulate potential error body from API (optional)
			_, _ = w.Write([]byte(`{"message": "Invalid username or password", "status": 401}`))
		}

		_, client := setupTestServer(t, handler)

		loginResp, err := client.Login(context.Background(), LoginRequest{Username: "wrong", Password: "bad"})

		require.Error(t, err) // Expect an error
		assert.Nil(t, loginResp)
		assert.Contains(t, err.Error(), "status 401") // Basic check on error message
		assert.Nil(t, client.GetCurrentToken(), "Token should not be set on failed login")
		assert.False(t, client.isAuthenticated())
	})
}

func TestLogout(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		initialToken := "valid-token-to-invalidate"

		handler := func(w http.ResponseWriter, r *http.Request) {
			// Assert Request
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/logout", r.URL.Path)
			assert.Equal(t, "Bearer "+initialToken, r.Header.Get("Authorization"))
			assert.Equal(t, "test-api-key", r.Header.Get("Api-Key"))

			// Send Mock Response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			resp := LogoutResponse{
				Message: "token successfully destroyed",
				Status:  http.StatusOK,
			}
			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)
		}

		_, client := setupTestServer(t, handler)
		serverURL := client.GetCurrentBaseURL() // Get the actual mock server URL

		// Pre-authenticate client for test using the mock server's URL
		err := client.SetAuthToken(initialToken, serverURL) // Use mock server URL
		require.NoError(t, err)
		require.True(t, client.isAuthenticated(), "Client should be authenticated before logout")

		// Call Logout
		logoutResp, err := client.Logout(context.Background())

		// Assert Results
		require.NoError(t, err)
		require.NotNil(t, logoutResp)
		assert.Equal(t, http.StatusOK, logoutResp.Status)
		assert.Equal(t, "token successfully destroyed", logoutResp.Message)

		// Assert Client State
		assert.False(t, client.isAuthenticated(), "Client should not be authenticated after logout")
		assert.Nil(t, client.GetCurrentToken(), "Token should be nil after logout")
	})

	t.Run("RequiresAuth (API Error)", func(t *testing.T) {
		handler := func(w http.ResponseWriter, r *http.Request) {
			// Server checks for Authorization header
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/api/v1/logout", r.URL.Path)

			// Simulate API returning 401 if Authorization is missing or invalid
			if r.Header.Get("Authorization") == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"message": "Authentication required"}`))
				return
			}

			// Should not reach here if auth fails
			t.Fatal("Handler should have returned 401 for missing auth")
		}

		_, client := setupTestServer(t, handler) // Client starts unauthenticated

		// Call Logout without logging in first
		logoutResp, err := client.Logout(context.Background())

		require.Error(t, err) // Expect API error
		assert.Nil(t, logoutResp)
		assert.Contains(t, err.Error(), "status 401") // Check for API 401 error
		assert.False(t, client.isAuthenticated(), "Client should remain unauthenticated")
	})
}

func TestGetUserInfoSuccess(t *testing.T) {
	token := "valid-user-token"
	expectedRemaining := 99

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/infos/user", r.URL.Path)
		assert.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
		assert.Equal(t, "test-api-key", r.Header.Get("Api-Key"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := GetUserInfoResponse{
			Data: UserInfo{
				BaseUserInfo:       BaseUserInfo{UserID: 42, Level: "Tester", AllowedDownloads: 100},
				DownloadsCount:     1,
				RemainingDownloads: expectedRemaining,
			},
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}

	_, client := setupTestServer(t, handler)
	err := client.SetAuthToken(token, "") // Authenticate client
	require.NoError(t, err)

	userInfo, err := client.GetUserInfo(context.Background())

	require.NoError(t, err)
	require.NotNil(t, userInfo)
	assert.Equal(t, 42, userInfo.Data.UserID)                            // Access Data field
	assert.Equal(t, expectedRemaining, userInfo.Data.RemainingDownloads) // Access Data field
}

func TestGetUserInfoRequiresAuth(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/infos/user", r.URL.Path)
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message": "Authentication required"}`))
			return
		}
		t.Fatal("GetUserInfo request should have failed auth")
	}
	_, client := setupTestServer(t, handler) // Uncommented client variable

	// Need client-side check or API failure
	userInfo, err := client.GetUserInfo(context.Background()) // Call the actual method

	require.Error(t, err)
	assert.Nil(t, userInfo)
	assert.Contains(t, err.Error(), "status 401") // Expect API 401
}
