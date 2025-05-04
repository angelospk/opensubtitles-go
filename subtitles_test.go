package opensubtitles

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	// "net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Add tests for subtitle methods

func TestSearchSubtitlesSuccess(t *testing.T) {
	expectedIMDbID := 1371111
	expectedLang := "en"
	expectedSubtitleID := "848343"

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Assert Request
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/subtitles", r.URL.Path) // Assuming setupTestServer adds /api/v1
		assert.Equal(t, "test-api-key", r.Header.Get("Api-Key"))
		assert.Equal(t, "GoTestClient/1.0", r.Header.Get("User-Agent"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		// Assert Query Params
		query := r.URL.Query()
		assert.Equal(t, fmt.Sprintf("%d", expectedIMDbID), query.Get("imdb_id"))
		assert.Equal(t, expectedLang, query.Get("languages"))
		assert.Equal(t, "", query.Get("moviehash")) // Ensure omitted params are not present

		// Send Mock Response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Simplified response for brevity
		resp := SearchSubtitlesResponse{
			PaginatedResponse: PaginatedResponse{TotalCount: 1, TotalPages: 1, Page: 1},
			Data: []Subtitle{
				{
					ApiDataWrapper: ApiDataWrapper{ID: expectedSubtitleID, Type: "subtitle"},
					Attributes: SubtitleAttributes{
						SubtitleID: expectedSubtitleID,
						Language:   LanguageCode(expectedLang),
						Files: []SubtitleFile{
							{FileID: 928281, FileName: "Test.srt"},
						},
						FeatureDetails: SubtitleFeatureDetails{
							IMDbID:      &expectedIMDbID,
							FeatureType: "Movie",
						},
					},
				},
			},
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}

	_, client := setupTestServer(t, handler)

	// Call SearchSubtitles (Needs implementation)
	params := SearchSubtitlesParams{
		IMDbID:    &expectedIMDbID,
		Languages: String(expectedLang), // Use helper from README example
	}
	searchResp, err := client.SearchSubtitles(context.Background(), params)

	// Assert Results - Placeholder
	require.NoError(t, err)
	require.NotNil(t, searchResp)
	assert.Equal(t, 1, searchResp.TotalCount)
	require.Len(t, searchResp.Data, 1)
	assert.Equal(t, expectedSubtitleID, searchResp.Data[0].ID)
	assert.Equal(t, expectedSubtitleID, searchResp.Data[0].Attributes.SubtitleID)
	assert.Equal(t, LanguageCode(expectedLang), searchResp.Data[0].Attributes.Language)
	require.NotNil(t, searchResp.Data[0].Attributes.FeatureDetails.IMDbID)
	assert.Equal(t, expectedIMDbID, *searchResp.Data[0].Attributes.FeatureDetails.IMDbID)
	require.Len(t, searchResp.Data[0].Attributes.Files, 1)
	assert.Equal(t, 928281, searchResp.Data[0].Attributes.Files[0].FileID)

	// Dummy assertion - REMOVE
	// assert.True(t, true, "Test needs SearchSubtitles implementation")
}

func TestSearchSubtitlesWithParams(t *testing.T) {
	// Test more complex parameter encoding
	expectedHash := "aabbccddeeff0011"
	expectedType := "movie"
	hearingImpairedOnly := Only

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/subtitles", r.URL.Path)

		query := r.URL.Query()
		assert.Equal(t, expectedHash, query.Get("moviehash"))
		assert.Equal(t, expectedType, query.Get("type"))
		assert.Equal(t, string(hearingImpairedOnly), query.Get("hearing_impaired"))
		assert.Equal(t, "", query.Get("imdb_id")) // Check omitted

		w.WriteHeader(http.StatusOK) // Minimal valid response
		_, _ = w.Write([]byte(`{"total_count": 0, "page": 1, "total_pages": 0, "data": []}`))
	}

	_, client := setupTestServer(t, handler)

	params := SearchSubtitlesParams{
		Moviehash:       String(expectedHash),
		Type:            String(expectedType),
		HearingImpaired: &hearingImpairedOnly,
	}
	_, err := client.SearchSubtitles(context.Background(), params)
	require.NoError(t, err)

	// Dummy assertion - REMOVE
	// assert.True(t, true, "Test needs SearchSubtitles implementation")
}

func TestSearchSubtitlesError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // Simulate server error
		_, _ = w.Write([]byte("Internal Server Error"))
	}

	_, client := setupTestServer(t, handler)
	params := SearchSubtitlesParams{} // Minimal params
	searchResp, err := client.SearchSubtitles(context.Background(), params)

	require.Error(t, err)
	assert.Nil(t, searchResp)
	assert.Contains(t, err.Error(), "status 500")

	// Dummy assertion - REMOVE
	// assert.True(t, true, "Test needs SearchSubtitles implementation")
}

// --- Download Subtitle Tests ---

func TestDownloadSubtitleSuccess(t *testing.T) {
	token := "valid-download-token"
	expectedFileID := 11047023
	expectedLink := "https://dl.opensubtitles.com/download/..."
	expectedResetTimeStr := "2022-04-08T13:03:16Z"
	expectedResetTime, _ := time.Parse(time.RFC3339, expectedResetTimeStr)

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Assert Request
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/download", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("Api-Key"))
		assert.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Assert Body
		var reqBody DownloadRequest
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, expectedFileID, reqBody.FileID)
		require.NotNil(t, reqBody.FileName)
		assert.Equal(t, "MyFile.srt", *reqBody.FileName) // Check optional param

		// Send Mock Response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := DownloadResponse{
			Link:         expectedLink,
			FileName:     "actual_filename.srt",
			Requests:     1,
			Remaining:    99,
			Message:      "Download ok",
			ResetTime:    "07 hours",
			ResetTimeUTC: expectedResetTime,
		}
		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}

	_, client := setupTestServer(t, handler)
	err := client.SetAuthToken(token, "") // Authenticate client
	require.NoError(t, err)

	// Call Download (Needs implementation)
	// fileName := "MyFile.srt"
	payload := DownloadRequest{
		FileID:   expectedFileID,
		FileName: String("MyFile.srt"), // Use String helper directly
	}
	downloadResp, err := client.Download(context.Background(), payload)

	// Assert Results - Placeholder
	require.NoError(t, err)
	require.NotNil(t, downloadResp)
	assert.Equal(t, expectedLink, downloadResp.Link)
	assert.Equal(t, 99, downloadResp.Remaining)
	assert.Equal(t, expectedResetTime, downloadResp.ResetTimeUTC)

	// Dummy assertion - REMOVE
	// assert.True(t, true, "Test needs DownloadSubtitle implementation")
}

func TestDownloadSubtitleRequiresAuth(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Handler SHOULD be called, but should return 401 if auth is missing
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/download", r.URL.Path)
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message": "Authentication required"}`))
			return // Return 401 as expected
		}
		// If auth header *is* present (which it shouldn't be), fail the test
		t.Errorf("DownloadSubtitle request made WITH auth header when it should be missing")
	}
	_, client := setupTestServer(t, handler) // Unauthenticated client
	// setupTestServer(t, handler) // Call setup, ignore client

	payload := DownloadRequest{FileID: 123}
	downloadResp, err := client.Download(context.Background(), payload)

	require.Error(t, err) // Now expect API error 401
	assert.Nil(t, downloadResp)
	assert.Contains(t, err.Error(), "status 401") // Check for API error

	// Dummy assertion - REMOVE
	// assert.True(t, true, "Test needs DownloadSubtitle implementation with auth check") // Keep dummy for now
}

func TestDownloadSubtitleErrorQuota(t *testing.T) {
	token := "valid-token-but-no-quota"
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden) // 403 Forbidden for quota exceeded
		// Simulate API error body
		_, _ = w.Write([]byte(`{"message": "Download quota exceeded", "status": 403}`))
	}

	_, client := setupTestServer(t, handler)
	// setupTestServer(t, handler) // Call setup, ignore client
	err := client.SetAuthToken(token, "") // Comment out auth setup for now
	require.NoError(t, err)               // Keep require here as SetAuthToken should succeed

	payload := DownloadRequest{FileID: 123}
	downloadResp, err := client.Download(context.Background(), payload)

	require.Error(t, err)
	assert.Nil(t, downloadResp)
	assert.Contains(t, err.Error(), "status 403")

	// Dummy assertion - REMOVE
	// assert.True(t, true, "Test needs DownloadSubtitle implementation")
}

func TestDownloadSubtitleErrorFileID(t *testing.T) {
	// token := "valid-token"
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusUnprocessableEntity) // 422 for invalid file_id
		_, _ = w.Write([]byte(`{"message": "Invalid file ID", "status": 422}`))
	}

	_, client := setupTestServer(t, handler)
	// setupTestServer(t, handler)
	// err := client.SetAuthToken(token, "")
	// require.NoError(t, err)

	payload := DownloadRequest{FileID: -1} // Invalid ID
	downloadResp, err := client.Download(context.Background(), payload)

	require.Error(t, err)
	assert.Nil(t, downloadResp)
	assert.Contains(t, err.Error(), "status 422")

	// Dummy assertion - REMOVE
	// assert.True(t, true, "Test needs DownloadSubtitle implementation")
}
