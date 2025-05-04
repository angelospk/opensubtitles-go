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

// Helper function to unmarshal feature attributes based on type hint
// Note: In a real application, you'd likely use type assertion after getting the response.
// This helper is mainly for testing the response structure flexibility.
func unmarshalFeatureAttributes(t *testing.T, raw json.RawMessage, featureType string) interface{} {
	t.Helper()
	switch featureType {
	case "Movie":
		var attrs FeatureMovieAttributes
		err := json.Unmarshal(raw, &attrs)
		require.NoError(t, err)
		return attrs
	case "Tvshow":
		var attrs FeatureTvshowAttributes
		err := json.Unmarshal(raw, &attrs)
		require.NoError(t, err)
		return attrs
	case "Episode":
		var attrs FeatureEpisodeAttributes
		err := json.Unmarshal(raw, &attrs)
		require.NoError(t, err)
		return attrs
	default:
		t.Fatalf("Unknown feature type for unmarshalling: %s", featureType)
		return nil
	}
}

func TestSearchFeaturesSuccessByID(t *testing.T) {
	expectedIMDbID := "539911" // API expects string without 'tt'
	expectedFeatureID := "1480735"
	expectedFeatureType := "Episode" // Capitalized as in response

	handler := func(w http.ResponseWriter, r *http.Request) {
		// Assert Request
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/features", r.URL.Path)
		assert.Equal(t, "test-api-key", r.Header.Get("Api-Key"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		// Assert Query Params
		query := r.URL.Query()
		assert.Equal(t, expectedIMDbID, query.Get("imdb_id"))
		assert.Equal(t, "", query.Get("query")) // Check omitted

		// Send Mock Response (Episode type)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := SearchFeaturesResponse{
			Data: []Feature{
				{
					ApiDataWrapper: ApiDataWrapper{ID: expectedFeatureID, Type: "feature"},
					Attributes: FeatureEpisodeAttributes{ // Use specific struct for marshaling test data
						FeatureBaseAttributes: FeatureBaseAttributes{
							FeatureID:   expectedFeatureID,
							FeatureType: expectedFeatureType,
							Title:       "the tortelli tort",
							Year:        "1982",
							IMDbID:      pint(539911),
							TMDBID:      pint(7645),
						},
						SeasonNumber:  1,
						EpisodeNumber: 3,
						ParentTitle:   pstr("Cheers"),
						ParentIMDbID:  pint(83399),
					},
				},
			},
		}
		// Marshal the specific struct first, then wrap if needed for dynamic testing,
		// but for sending, just marshal the final structure.
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}

	_, client := setupTestServer(t, handler)

	// Call SearchFeatures (Needs implementation)
	params := SearchFeaturesParams{
		IMDbID: String(expectedIMDbID), // Use helper
	}
	searchResp, err := client.SearchFeatures(context.Background(), params)

	// Assert Results - Placeholder
	require.NoError(t, err)
	require.NotNil(t, searchResp)
	require.Len(t, searchResp.Data, 1)
	assert.Equal(t, expectedFeatureID, searchResp.Data[0].ID)
	assert.Equal(t, "feature", searchResp.Data[0].Type) // Outer type

	// Now, check the dynamic attributes field
	// // 1. Marshal the interface{} back to RawMessage
	rawAttrs, err := json.Marshal(searchResp.Data[0].Attributes)
	require.NoError(t, err)

	// // 2. Unmarshal RawMessage into a map to check feature_type
	var attrMap map[string]interface{}
	err = json.Unmarshal(rawAttrs, &attrMap)
	require.NoError(t, err)
	actualFeatureType, ok := attrMap["feature_type"].(string)
	require.True(t, ok, "feature_type should be a string")
	assert.Equal(t, expectedFeatureType, actualFeatureType)

	// // 3. Unmarshal into the specific type based on feature_type
	if actualFeatureType == expectedFeatureType {
		var episodeAttrs FeatureEpisodeAttributes
		err = json.Unmarshal(rawAttrs, &episodeAttrs)
		require.NoError(t, err)
		assert.Equal(t, expectedFeatureID, episodeAttrs.FeatureID)
		assert.Equal(t, "the tortelli tort", episodeAttrs.Title)
		require.NotNil(t, episodeAttrs.ParentTitle)
		assert.Equal(t, "Cheers", *episodeAttrs.ParentTitle)
	} else {
		t.Fatalf("Expected feature type %s but got %s", expectedFeatureType, actualFeatureType)
	}
}

func TestSearchFeaturesSuccessByQuery(t *testing.T) {
	expectedQuery := "lord rings fellowship"
	expectedType := "movie" // Filter param

	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/features", r.URL.Path)
		query := r.URL.Query()
		assert.Equal(t, expectedQuery, query.Get("query"))
		assert.Equal(t, expectedType, query.Get("type"))
		assert.Equal(t, "", query.Get("query_match")) // Expect empty string due to omitempty

		w.WriteHeader(http.StatusOK)
		// Return a Movie type feature
		resp := SearchFeaturesResponse{
			Data: []Feature{
				{
					ApiDataWrapper: ApiDataWrapper{ID: "514811", Type: "feature"},
					Attributes: FeatureMovieAttributes{
						FeatureBaseAttributes: FeatureBaseAttributes{
							FeatureID:   "514811",
							FeatureType: "Movie",
							Title:       "the lord of the rings: the fellowship of the ring",
							Year:        "2001",
							IMDbID:      pint(120737),
						},
					},
				},
			},
		}
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}

	_, client := setupTestServer(t, handler)
	params := SearchFeaturesParams{
		Query: String(expectedQuery),
		Type:  String(expectedType),
	}
	searchResp, err := client.SearchFeatures(context.Background(), params)

	require.NoError(t, err)
	require.NotNil(t, searchResp)
	require.Len(t, searchResp.Data, 1)

	// Check attributes type dynamically
	rawAttrs, _ := json.Marshal(searchResp.Data[0].Attributes)
	var movieAttrs FeatureMovieAttributes
	err = json.Unmarshal(rawAttrs, &movieAttrs)
	require.NoError(t, err)
	assert.Equal(t, "Movie", movieAttrs.FeatureType)
	assert.Equal(t, "514811", movieAttrs.FeatureID)
}

func TestSearchFeaturesError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // Simulate bad request
		_, _ = w.Write([]byte("Invalid parameters"))
	}

	_, client := setupTestServer(t, handler)
	imdbID := "invalid-id" // Example invalid param
	params := SearchFeaturesParams{IMDbID: String(imdbID)}
	searchResp, err := client.SearchFeatures(context.Background(), params)

	require.Error(t, err)
	assert.Nil(t, searchResp)
	assert.Contains(t, err.Error(), "status 400")
}

// Helper functions to create pointers easily in tests
func pint(i int) *int       { return &i }
func pstr(s string) *string { return &s }

// func pflt(f float64) *float64 { return &f }
// func pbool(b bool) *bool { return &b }

// Helper needed for tests in this file
// func String(s string) *string { return &s }
