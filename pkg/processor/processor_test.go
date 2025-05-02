package processor_test

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/angelospk/osuploadergui/pkg/core/imdb"
	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles"
	"github.com/angelospk/osuploadergui/pkg/core/trakt"
	"github.com/angelospk/osuploadergui/pkg/processor"

	// "osuploadergui/pkg/core/fileops" // Add imports as needed
	// "osuploadergui/pkg/core/opensubtitles"
	// "osuploadergui/pkg/core/trakt"
	// "osuploadergui/pkg/core/imdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks for the individual client interfaces --- //

type MockOpenSubtitlesClient struct {
	SearchFeaturesFunc func(ctx context.Context, params map[string]string) (*opensubtitles.FeaturesResponse, error)
}

// Ensure MockOpenSubtitlesClient implements metadata.OpenSubtitlesClient
var _ metadata.OpenSubtitlesClient = (*MockOpenSubtitlesClient)(nil)

func (m *MockOpenSubtitlesClient) SearchFeatures(ctx context.Context, params map[string]string) (*opensubtitles.FeaturesResponse, error) {
	if m.SearchFeaturesFunc != nil {
		return m.SearchFeaturesFunc(ctx, params)
	}
	return &opensubtitles.FeaturesResponse{}, nil // Default mock behavior
}

type MockTraktClient struct {
	SearchTraktFunc func(ctx context.Context, queryType string, query string) ([]trakt.SearchResult, error)
}

// Ensure MockTraktClient implements metadata.TraktClient
var _ metadata.TraktClient = (*MockTraktClient)(nil)

func (m *MockTraktClient) SearchTrakt(ctx context.Context, queryType string, query string) ([]trakt.SearchResult, error) {
	if m.SearchTraktFunc != nil {
		return m.SearchTraktFunc(ctx, queryType, query)
	}
	return []trakt.SearchResult{}, nil // Default mock behavior
}

type MockIMDbClient struct {
	SearchIMDBSuggestionsFunc func(ctx context.Context, query string) ([]imdb.IMDBSuggestion, error)
}

// Ensure MockIMDbClient implements metadata.IMDbClient
var _ metadata.IMDbClient = (*MockIMDbClient)(nil)

func (m *MockIMDbClient) SearchIMDBSuggestions(ctx context.Context, query string) ([]imdb.IMDBSuggestion, error) {
	if m.SearchIMDBSuggestionsFunc != nil {
		return m.SearchIMDBSuggestionsFunc(ctx, query)
	}
	return []imdb.IMDBSuggestion{}, nil // Default mock behavior
}

// --- End Mocks --- //

// TestCreateJobsFromDirectory_Basic tests basic scanning and matching.
// Renamed from TestScanAndProcess_Basic
func TestCreateJobsFromDirectory_Basic(t *testing.T) {
	// --- Setup Test Directory ---
	tmpDir, err := ioutil.TempDir("", "processor_test_*") // Use ioutil.TempDir
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tmpDir) // Clean up after test

	videoFileName := "My.Movie.2023.1080p.mkv"
	subFileName := "My.Movie.2023.1080p.en.srt"
	videoFilePath := filepath.Join(tmpDir, videoFileName)
	subFilePath := filepath.Join(tmpDir, subFileName)

	// Create dummy files
	// Make video large enough for OSDb hash (needs > 128kB total, ~64k start + ~64k end)
	dummyVideoContent := make([]byte, 130*1024) // 130 KiB
	copy(dummyVideoContent[:30], []byte("dummy video content start ..."))
	copy(dummyVideoContent[len(dummyVideoContent)-30:], []byte("... dummy video content end"))
	require.NoError(t, os.WriteFile(videoFilePath, dummyVideoContent, 0644))
	require.NoError(t, os.WriteFile(subFilePath, []byte("dummy subtitle content"), 0644))

	// --- Setup Processor ---
	// Create instances of the individual client mocks
	mockOSClient := &MockOpenSubtitlesClient{}
	mockTraktClient := &MockTraktClient{}
	mockIMDbClient := &MockIMDbClient{}

	// Create the *actual* metadata.APIClientProvider struct required by NewProcessor,
	// populating its fields with our mocks.
	apiProviderStruct := metadata.APIClientProvider{
		OSClient:    mockOSClient,    // This now satisfies the field type metadata.OpenSubtitlesClient
		TraktClient: mockTraktClient, // This now satisfies the field type metadata.TraktClient
		IMDbClient:  mockIMDbClient,  // This now satisfies the field type metadata.IMDbClient
	}

	logger := log.New(ioutil.Discard, "TEST: ", log.LstdFlags)     // Use standard log with ioutil.Discard
	processor := processor.NewProcessor(apiProviderStruct, logger) // Pass the correctly typed struct and logger

	// --- Run CreateJobsFromDirectory ---
	ctx := context.Background()                                 // Create context
	jobs, err := processor.CreateJobsFromDirectory(ctx, tmpDir) // Use correct method name

	// --- Assertions ---
	require.NoError(t, err, "CreateJobsFromDirectory returned an error")
	require.Len(t, jobs, 1, "Expected exactly one job to be created")

	job := jobs[0]
	// Check paths using the correct fields in VideoInfo/SubtitleInfo
	require.NotNil(t, job.VideoInfo, "Job VideoInfo should not be nil")
	assert.Equal(t, videoFilePath, job.VideoInfo.FilePath, "Job has incorrect video file path")
	require.NotNil(t, job.SubtitleInfo, "Job SubtitleInfo should not be nil")
	assert.Equal(t, subFilePath, job.SubtitleInfo.FilePath, "Job has incorrect subtitle file path")

	assert.Equal(t, metadata.StatusPending, job.Status, "Job status should be Pending")

	// Removed assertions checking DetectedTitle/Year/Language as ConsolidateMetadata
	// populates Title/Year/Language directly, and mocks don't guarantee their values here.
	// A more thorough test would mock ConsolidateMetadata or provide API mock responses.
	// assert.Equal(t, "My Movie", job.VideoInfo.Title, "Expected title") // Example check
	// assert.Equal(t, 2023, job.VideoInfo.Year, "Expected year")          // Example check
	// assert.Equal(t, "en", job.SubtitleInfo.Language, "Expected language") // Example check
}

// TODO: Add more test cases:
// - Multiple videos, multiple subs
// - Subtitle without matching video
// - Video without matching subtitle
// - Test cancellation context
// - Test error during file walk
// - Test error during metadata consolidation
// - Test subtitle language detection failure leads to skipped job
