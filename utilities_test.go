package opensubtitles

import (
	"context"
	"encoding/json"
	"net/http"

	// "net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Add tests for utility methods

func TestGuessitSuccess(t *testing.T) {
	expectedFilename := "Stranger.Things.S04E01.Chapter.One.The.Hellfire.Club.1080p.NF.WEBRip.DDP5.1.Atmos.x264-GalaxyTV.mkv"
	expectedTitle := "Stranger Things"
	expectedSeason := 4
	expectedEpisode := 1
	expectedScreenSize := "1080p"
	expectedSource := "WEBRip"
	expectedType := "episode"

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/utilities/guessit", r.URL.Path)
		assert.Equal(t, expectedFilename, r.URL.Query().Get("filename"))

		w.WriteHeader(http.StatusOK)
		resp := GuessitResponse{
			Title:            pstr(expectedTitle),
			Season:           pint(expectedSeason),
			Episode:          pint(expectedEpisode),
			EpisodeTitle:     pstr("Chapter One The Hellfire Club"),
			ScreenSize:       pstr(expectedScreenSize),
			StreamingService: pstr("Netflix"),
			Source:           pstr(expectedSource),
			AudioCodec:       pstr("Dolby Digital Plus"),
			AudioChannels:    pstr("5.1"),
			AudioProfile:     pstr("Atmos"),
			VideoCodec:       pstr("H.264"),
			ReleaseGroup:     pstr("GalaxyTV"),
			Type:             pstr(expectedType),
			// Year, Language etc. would be null in this example response
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}

	_, client := setupTestServer(t, handler)
	params := GuessitParams{Filename: expectedFilename}
	guessResp, err := client.Guessit(context.Background(), params)

	// Assert Results
	require.NoError(t, err)
	require.NotNil(t, guessResp)
	assert.Equal(t, expectedTitle, *guessResp.Title)
	assert.Equal(t, expectedSeason, *guessResp.Season)
	assert.Equal(t, expectedEpisode, *guessResp.Episode)
	assert.Equal(t, "Chapter One The Hellfire Club", *guessResp.EpisodeTitle)
	assert.Equal(t, expectedScreenSize, *guessResp.ScreenSize)
	assert.Equal(t, expectedSource, *guessResp.Source)
	assert.Equal(t, "Dolby Digital Plus", *guessResp.AudioCodec)
	assert.Equal(t, "H.264", *guessResp.VideoCodec)
	assert.Equal(t, "GalaxyTV", *guessResp.ReleaseGroup)
	assert.Equal(t, expectedType, *guessResp.Type)
	assert.Nil(t, guessResp.Year)
	assert.Nil(t, guessResp.Language)
}

func TestGuessitMissingFilename(t *testing.T) {
	// This test assumes the API returns 400 if filename is missing,
	// rather than the client preventing the call (though client-side validation is also good).
	handler := func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("filename") == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message": "filename parameter is required"}`))
		} else {
			w.WriteHeader(http.StatusOK) // Should not happen in this test path
			_, _ = w.Write([]byte(`{}`))
		}
	}

	_, client := setupTestServer(t, handler)
	// Intentionally create params without filename to test API response
	// Note: go-querystring might omit if empty string. Let's assume API checks.
	params := GuessitParams{Filename: ""} // Or omit the field if using pointers
	guessResp, err := client.Guessit(context.Background(), params)

	// Assert Results
	require.Error(t, err)
	assert.Nil(t, guessResp)
	assert.Contains(t, err.Error(), "status 400")
}

func TestGuessitError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	_, client := setupTestServer(t, handler)
	params := GuessitParams{Filename: "some.file.mkv"}
	guessResp, err := client.Guessit(context.Background(), params)

	// Assert Results
	require.Error(t, err)
	assert.Nil(t, guessResp)
	assert.Contains(t, err.Error(), "status 500")
}

// Helpers defined in features_test.go or common test file
// func pint(i int) *int       { return &i }
// func pstr(s string) *string { return &s }
