package trakt_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/angelospk/osuploadergui/pkg/core/trakt" // Adjust import path if necessary
)

func TestNewClient(t *testing.T) {
	t.Run("Success with ClientID set", func(t *testing.T) {
		// Arrange
		originalClientID := os.Getenv("TRAKT_CLIENT_ID")
		defer os.Setenv("TRAKT_CLIENT_ID", originalClientID) // Restore original value
		os.Setenv("TRAKT_CLIENT_ID", "test-client-id")

		// Act
		client, err := trakt.NewClient()

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("Error when ClientID not set", func(t *testing.T) {
		// Arrange
		originalClientID := os.Getenv("TRAKT_CLIENT_ID")
		defer os.Setenv("TRAKT_CLIENT_ID", originalClientID) // Restore original value
		os.Unsetenv("TRAKT_CLIENT_ID")

		// Act
		client, err := trakt.NewClient()

		// Assert
		require.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "TRAKT_CLIENT_ID environment variable not set")
	})
}

// TestSearchTrakt uses httptest to mock the Trakt API.
func TestSearchTrakt(t *testing.T) {
	// Arrange: Setup Mock Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basic validation of the request path and query
		assert.Contains(t, r.URL.Path, "/search/movie,show")
		assert.Equal(t, "test query", r.URL.Query().Get("query"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "2", r.Header.Get("trakt-api-version"))
		assert.Equal(t, "test-client-id", r.Header.Get("trakt-api-key"))

		// Send mock response
		w.WriteHeader(http.StatusOK)
		mockResponse := `[
			{
				"type": "movie",
				"score": 12.3,
				"movie": {
					"title": "Test Movie",
					"year": 2023,
					"ids": {"trakt": 1, "imdb": "tt123", "tmdb": 456}
				}
			},
			{
				"type": "show",
				"score": 10.1,
				"show": {
					"title": "Test Show",
					"year": 2022,
					"ids": {"trakt": 2, "tvdb": 789}
				}
			},
			{
				"type": "episode",
				"score": 9.5,
				"show": { 
					"title": "Parent Show",
					"year": 2021,
					"ids": {"trakt": 3}
				},
				"episode": {
					"season": 1,
					"number": 2,
					"title": "Test Episode",
					"ids": {"trakt": 4, "tvdb": 101, "imdb": "tt999"}
				}
			}
		]`
		fmt.Fprintln(w, mockResponse)
	}))
	defer mockServer.Close()

	// Arrange: Setup Client to use Mock Server
	// Override the baseURL constant for testing - THIS IS NOT IDEAL.
	// A better approach would be to make the baseURL configurable in the Client.
	// For now, we modify it globally (ensure tests run sequentially if needed).
	originalBaseURL := trakt.SetBaseURLForTesting(mockServer.URL)
	defer trakt.SetBaseURLForTesting(originalBaseURL) // Restore

	// Set Client ID env var for NewClient
	originalClientID := os.Getenv("TRAKT_CLIENT_ID")
	defer os.Setenv("TRAKT_CLIENT_ID", originalClientID)
	os.Setenv("TRAKT_CLIENT_ID", "test-client-id")

	client, err := trakt.NewClient()
	require.NoError(t, err)
	require.NotNil(t, client)

	// Act
	results, err := client.SearchTrakt(context.Background(), "movie,show", "test query")

	// Assert
	require.NoError(t, err)
	require.Len(t, results, 3) // Expect movie, show, and episode

	// Movie
	assert.Equal(t, "movie", results[0].Type)
	assert.Equal(t, "Test Movie", results[0].Title)
	assert.Equal(t, 2023, results[0].Year)
	assert.Equal(t, map[string]string{"trakt": "1", "imdb": "tt123", "tmdb": "456"}, results[0].IDs)

	// Show
	assert.Equal(t, "show", results[1].Type)
	assert.Equal(t, "Test Show", results[1].Title)
	assert.Equal(t, 2022, results[1].Year)
	assert.Equal(t, map[string]string{"trakt": "2", "tvdb": "789"}, results[1].IDs)

	// Episode
	assert.Equal(t, "episode", results[2].Type)
	assert.Equal(t, "Parent Show (2021) - 1x2 - Test Episode", results[2].Title)
	assert.Equal(t, 2021, results[2].Year)
	assert.Equal(t, map[string]string{"trakt": "4", "tvdb": "101", "imdb": "tt999"}, results[2].IDs)
}

// Note: Needs helper functions in trakt package to modify baseURL for testing.
// Add these to trakt.go:
/*
var (
	baseURL      = "https://api.trakt.tv" // Keep original default
	// apiVersion and searchEndpoint remain const
)

func SetBaseURLForTesting(newURL string) string {
	oldURL := baseURL
	baseURL = newURL
	return oldURL
}
*/
