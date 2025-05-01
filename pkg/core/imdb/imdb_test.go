package imdb_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/angelospk/osuploadergui/pkg/core/imdb" // Adjust import path
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchIMDBSuggestions_Success(t *testing.T) {
	// Arrange: Mock Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/suggestion/titles/t/test") // Check path format
		mockResponse := `{
			"v": 1,
			"q": "test query",
			"d": [
				{
					"l": "Test Movie One",
					"id": "tt0000001",
					"y": 2020,
					"q": "feature"
				},
				{
					"l": "Test Movie Two (Range)",
					"id": "tt0000002",
					"yr": "2021-2022", 
					"q": "feature"
				},
				{
					"l": "Test TV Series", 
					"id": "tt0000003",
					"yr": "2019-2020",
					"q": "TV series"
				},
				{
					"l": "Unknown Type Movie",
					"id": "tt0000004", 
					"y": 2023
				},
				{
					"l": "Invalid ID Movie", 
					"id": "nm12345", 
					"y": 2022,
					"q": "feature"
				},
				{
					"id": "tt0000005", 
					"y": 2023,
					"q": "feature"
				}
			]
		}`
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, mockResponse)
	}))
	defer mockServer.Close()

	// Arrange: Client
	originalBaseURL := imdb.SetBaseURLForTesting(mockServer.URL)
	defer imdb.SetBaseURLForTesting(originalBaseURL)

	client := imdb.NewClient()

	// Act
	results, err := client.SearchIMDBSuggestions(context.Background(), "test query")

	// Assert
	require.NoError(t, err)
	// Expect only tt* IDs with titles and likely movie type ("feature" or empty)
	require.Len(t, results, 3)

	// Movie 1 (y)
	assert.Equal(t, "tt0000001", results[0].ID)
	assert.Equal(t, "Test Movie One", results[0].Title)
	assert.Equal(t, 2020, results[0].Year)

	// Movie 2 (yr)
	assert.Equal(t, "tt0000002", results[1].ID)
	assert.Equal(t, "Test Movie Two (Range)", results[1].Title)
	assert.Equal(t, 2021, results[1].Year) // Parsed start year

	// Movie 3 (unknown type)
	assert.Equal(t, "tt0000004", results[2].ID)
	assert.Equal(t, "Unknown Type Movie", results[2].Title)
	assert.Equal(t, 2023, results[2].Year)
}

func TestSearchIMDBSuggestions_APIErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		respBody   string
	}{
		{"Non-OK Status", http.StatusInternalServerError, "Internal Server Error"},
		{"Bad JSON", http.StatusOK, `{"d": [`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange: Mock Server
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				fmt.Fprintln(w, tc.respBody)
			}))
			defer mockServer.Close()

			// Arrange: Client
			originalBaseURL := imdb.SetBaseURLForTesting(mockServer.URL)
			defer imdb.SetBaseURLForTesting(originalBaseURL)
			client := imdb.NewClient()

			// Act
			results, err := client.SearchIMDBSuggestions(context.Background(), "test query")

			// Assert
			require.NoError(t, err)  // Function should handle API errors gracefully
			assert.Empty(t, results) // Should return empty slice on API error
		})
	}
}

// Note: Requires SetBaseURLForTesting helper in imdb.go
/*
var imdbBaseURL = "https://v3.sg.media-imdb.com"

func SetBaseURLForTesting(newURL string) string {
	oldURL := imdbBaseURL
	imdbBaseURL = newURL
	return oldURL
}
*/
