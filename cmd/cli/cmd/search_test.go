package cmd_test

import (
	"bytes"
	"testing"

	clicmd "github.com/angelospk/osuploadergui/cmd/cli/cmd"
	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock OS Client for Search --- //

// Re-using MockOSClient from logout_test.go which now embeds mock.Mock

// --- Search Command Tests --- //

// Helper function to execute search command with mock
func executeSearchCommand(t *testing.T, mockClient *MockOSClient, args []string) (string, string, error) {
	// Store original client creation function
	originalNewClientFunc := clicmd.NewOSClientFunc
	defer func() { clicmd.NewOSClientFunc = originalNewClientFunc }() // Restore original

	// Replace with mock client provider
	clicmd.NewOSClientFunc = func(apiKey string) (metadata.OpenSubtitlesClient, error) {
		assert.NotEmpty(t, apiKey, "API key should not be empty when creating client")
		return mockClient, nil // Return the provided mock client
	}

	outBuf := bytes.NewBufferString("")
	errBuf := bytes.NewBufferString("")
	clicmd.RootCmd.SetOut(outBuf)
	clicmd.RootCmd.SetErr(errBuf)
	clicmd.RootCmd.SetArgs(append([]string{"search"}, args...))

	// Set API key for the test
	originalAPIKey := viper.GetString(clicmd.CfgKeyOSAPIKey)
	// Use GetViper() to ensure we modify the same instance used by the command
	vip := viper.GetViper()
	if vip == nil {
		// Initialize viper if it hasn't been initialized elsewhere (e.g., in root command persistence)
		vip = viper.New()
		// If root command has a persistent pre-run to init viper, this might not be needed
	}
	vip.Set(clicmd.CfgKeyOSAPIKey, "test-api-key")
	defer vip.Set(clicmd.CfgKeyOSAPIKey, originalAPIKey)

	err := clicmd.RootCmd.Execute()

	// Reset args
	clicmd.RootCmd.SetArgs([]string{})
	return outBuf.String(), errBuf.String(), err
}

func TestSearchCommand_Success_Query(t *testing.T) {
	mockClient := new(MockOSClient) // Uses the mock from logout_test.go

	// Setup mock expectation
	expectedParams := map[string]string{"query": "My Test Movie", "type": "movie"}
	// Use mockClient directly now
	mockClient.On("SearchSubtitles", mock.AnythingOfType("*context.emptyCtx"), expectedParams).Return(
		&opensubtitles.SubtitleSearchResponse{
			TotalCount: 1,
			Data: []opensubtitles.Subtitle{
				{
					ID:   "sub1",
					Type: "subtitle",
					Attributes: opensubtitles.SubtitleAttributes{
						Language:      "en",
						DownloadCount: 100,
						Format:        "srt",
						Votes:         5,
						Ratings:       8.5,
						FeatureDetails: opensubtitles.FeatureInfo{
							FeatureType: "movie",
							FeatureID:   123,
							Title:       "My Test Movie",
							Year:        2023,
						},
						Files: []opensubtitles.SubtitleFile{
							{FileName: "My.Test.Movie.srt"},
						},
					},
				},
			},
		}, nil).Once() // Expect call once

	args := []string{"--query", "My Test Movie"}
	output, errOutput, err := executeSearchCommand(t, mockClient, args)

	assert.NoError(t, err)
	assert.Empty(t, errOutput, "StdErr should be empty on success")
	assert.Contains(t, output, "Found 1 subtitles (showing 1):")
	assert.Contains(t, output, "ID: sub1")
	assert.Contains(t, output, "File Name: My.Test.Movie.srt")
	assert.Contains(t, output, "Language: en")
	assert.Contains(t, output, "Feature: movie (ID: 123, Title: My Test Movie, Year: 2023)")

	mockClient.AssertExpectations(t)
}

func TestSearchCommand_Success_IMDbID_Lang(t *testing.T) {
	mockClient := new(MockOSClient)

	expectedParams := map[string]string{
		"imdb_id":   "1234567",
		"languages": "en,el",
		"type":      "movie",
	}
	mockClient.On("SearchSubtitles", mock.AnythingOfType("*context.emptyCtx"), expectedParams).Return(
		&opensubtitles.SubtitleSearchResponse{TotalCount: 0, Data: []opensubtitles.Subtitle{}},
		nil, // No error
	).Once()

	args := []string{"--imdbid", "tt1234567", "--lang", "en,el"}
	output, errOutput, err := executeSearchCommand(t, mockClient, args)

	assert.NoError(t, err)
	assert.Empty(t, errOutput)
	assert.Contains(t, output, "No subtitles found")

	mockClient.AssertExpectations(t)
}

func TestSearchCommand_Success_Episode(t *testing.T) {
	mockClient := new(MockOSClient)

	expectedParams := map[string]string{
		"query":          "My Show",
		"type":           "episode",
		"season_number":  "2",
		"episode_number": "5",
	}
	mockClient.On("SearchSubtitles", mock.AnythingOfType("*context.emptyCtx"), expectedParams).Return(
		&opensubtitles.SubtitleSearchResponse{TotalCount: 0, Data: []opensubtitles.Subtitle{}},
		nil,
	).Once()

	args := []string{"--query", "My Show", "--type", "episode", "-s", "2", "-e", "5"}
	output, errOutput, err := executeSearchCommand(t, mockClient, args)

	assert.NoError(t, err)
	assert.Empty(t, errOutput)
	assert.Contains(t, output, "No subtitles found")

	mockClient.AssertExpectations(t)
}

func TestSearchCommand_Fail_NoQueryOrID(t *testing.T) {
	mockClient := new(MockOSClient) // No calls expected
	args := []string{"--lang", "en"}
	_, _, err := executeSearchCommand(t, mockClient, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one of --query, --imdbid, or --parent-id must be provided")
	mockClient.AssertNotCalled(t, "SearchSubtitles", mock.Anything, mock.Anything)
}

func TestSearchCommand_Fail_InvalidType(t *testing.T) {
	mockClient := new(MockOSClient)
	args := []string{"--query", "test", "--type", "invalid"}
	_, _, err := executeSearchCommand(t, mockClient, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --type: invalid")
	mockClient.AssertNotCalled(t, "SearchSubtitles", mock.Anything, mock.Anything)
}

func TestSearchCommand_Fail_EpisodeMissingArgs(t *testing.T) {
	mockClient := new(MockOSClient)
	args := []string{"--query", "test", "--type", "episode", "-s", "1"} // Missing episode
	_, _, err := executeSearchCommand(t, mockClient, args)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--season and --episode are required when --type=episode")
	mockClient.AssertNotCalled(t, "SearchSubtitles", mock.Anything, mock.Anything)
}

func TestSearchCommand_Fail_APIError(t *testing.T) {
	mockClient := new(MockOSClient)

	expectedParams := map[string]string{"query": "API Fail", "type": "movie"}
	mockClient.On("SearchSubtitles", mock.AnythingOfType("*context.emptyCtx"), expectedParams).Return(
		nil,            // No response body on error
		assert.AnError, // Simulate an API error
	).Once()

	args := []string{"--query", "API Fail"}
	_, _, err := executeSearchCommand(t, mockClient, args)

	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError, "Expected the underlying API error to be wrapped")
	assert.Contains(t, err.Error(), "subtitle search failed:")

	mockClient.AssertExpectations(t)
}

// TODO: Add test for --parent-id flag usage
// TODO: Add test for pagination info display if TotalPages > 1
