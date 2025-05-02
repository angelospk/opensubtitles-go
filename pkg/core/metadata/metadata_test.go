package metadata_test

import (
	"context"
	"errors" // Import errors for mock error return
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angelospk/osuploadergui/pkg/core/imdb" // Import concrete type
	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles" // Import concrete type
	"github.com/angelospk/osuploadergui/pkg/core/trakt"         // Import concrete type
	"github.com/stretchr/testify/assert"
)

// --- Mocks for API Client Interfaces ---

// MockOpenSubtitlesClient is a mock implementation of metadata.OpenSubtitlesClient.
type MockOpenSubtitlesClient struct {
	SearchFeaturesFunc func(ctx context.Context, params map[string]string) (*opensubtitles.FeaturesResponse, error)
	CalledWithParams   map[string]string
}

func (m *MockOpenSubtitlesClient) SearchFeatures(ctx context.Context, params map[string]string) (*opensubtitles.FeaturesResponse, error) {
	m.CalledWithParams = params // Store called parameters for verification
	if m.SearchFeaturesFunc != nil {
		return m.SearchFeaturesFunc(ctx, params)
	}
	return nil, errors.New("SearchFeaturesFunc not set in mock")
}

// Add SearchSubtitles to satisfy the interface
func (m *MockOpenSubtitlesClient) SearchSubtitles(ctx context.Context, params map[string]string) (*opensubtitles.SubtitleSearchResponse, error) {
	// For metadata tests, this likely won't be called, return dummy data or error
	// If a test *needs* this, it should set a specific Func like SearchFeaturesFunc
	return nil, errors.New("SearchSubtitles not implemented in this specific mock instance")
}

// MockTraktClient is a mock implementation of metadata.TraktClient.
type MockTraktClient struct {
	SearchTraktFunc     func(ctx context.Context, queryType string, query string) ([]trakt.SearchResult, error)
	CalledWithQuery     string
	CalledWithQueryType string
}

func (m *MockTraktClient) SearchTrakt(ctx context.Context, queryType string, query string) ([]trakt.SearchResult, error) {
	m.CalledWithQuery = query
	m.CalledWithQueryType = queryType
	if m.SearchTraktFunc != nil {
		return m.SearchTraktFunc(ctx, queryType, query)
	}
	return nil, errors.New("SearchTraktFunc not set in mock")
}

// MockIMDbClient is a mock implementation of metadata.IMDbClient.
type MockIMDbClient struct {
	SearchIMDBSuggestionsFunc func(ctx context.Context, query string) ([]imdb.IMDBSuggestion, error)
	CalledWithQuery           string
}

func (m *MockIMDbClient) SearchIMDBSuggestions(ctx context.Context, query string) ([]imdb.IMDBSuggestion, error) {
	m.CalledWithQuery = query
	if m.SearchIMDBSuggestionsFunc != nil {
		return m.SearchIMDBSuggestionsFunc(ctx, query)
	}
	return nil, errors.New("SearchIMDBSuggestionsFunc not set in mock")
}

// --- End Mocks ---

func TestDetectSubtitleLanguage(t *testing.T) {
	tests := []struct {
		name     string
		expected string // Expecting OpenSubtitles code now
	}{
		// ISO 639-1 codes
		{"movie.title.year.en.srt", "en"},
		{"movie.title.year.el.srt", "el"},
		{"movie title year.es.srt", "es"},
		{"movie_title_year_fr.srt", "fr"},
		{"movie-title-year-de.srt", "de"},
		{"movie.title.it.720p.srt", "it"},
		// ISO 639-2/3 codes
		{"movie.title.year.eng.srt", "en"},
		{"movie.title.year.gre.srt", "el"},
		{"movie title year spa.srt", "es"},
		{"movie_title_year_fre.srt", "fr"},
		{"movie-title-year-ger.srt", "de"},
		{"movie.title.ita.1080p.srt", "it"},
		// Full names (lowercase)
		{"movie.title.year.english.srt", "en"},
		{"movie.title.year.greek.srt", "el"},
		{"movie title year spanish.srt", "es"},
		{"movie_title_year_french.srt", "fr"},
		{"movie-title-year-german.srt", "de"},
		{"movie.title.italian.bluray.srt", "it"},
		// OpenSubtitles specific codes
		{"movie.title.year.pt-br.srt", "pt-br"},
		{"movie.title.year.zh-cn.srt", "zh-cn"},
		{"movie.title.year.ze.srt", "ze"},
		// Edge cases
		{"movie.title.year.srt", ""}, // No lang code
		{".en.srt", "en"},
		{"eng.srt", "en"},
		{"movie.en", "en"},                   // No extension
		{"movie", ""},                        // No lang, no ext
		{"movie.english", "en"},              // Full name, no ext
		{"movie_ita", "it"},                  // Underscore, no ext
		{"German.Movie.Title.Deu.mkv", "de"}, // Mixed case (should be handled by lowercasing)
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lang := metadata.DetectSubtitleLanguage(tc.name)
			assert.Equal(t, tc.expected, lang)
		})
	}
}

func TestAnalyzeSubtitleFlags(t *testing.T) {
	tests := []struct {
		name           string
		expectedHI     bool
		expectedForced bool
	}{
		{"movie.title.year.sdh.en.srt", true, false},
		{"movie.title.year.HI.en.srt", true, false},
		{"movie.title.year.hearingimpaired.en.srt", true, false},
		{"movie.title.year.forced.en.srt", false, true},
		{"movie.title.year.en.forced.srt", false, true},
		{"movie.title.year.frc.en.srt", false, true},
		{"movie.title.year.en.srt", false, false},
		{"movie.sdh.forced.en.srt", true, true},
		{"movie.forced.hi.en.srt", true, true},
		{"movie-sdh-frc-en.srt", true, true},
		{"movie_hearingimpaired_en.srt", true, false},
		{"movie forced en.srt", false, true},
		{"SDH.srt", true, false},
		{"forced", false, true}, // No extension
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isHI, isForced := metadata.AnalyzeSubtitleFlags(tc.name)
			assert.Equal(t, tc.expectedHI, isHI, "HI flag mismatch")
			assert.Equal(t, tc.expectedForced, isForced, "Forced flag mismatch")
		})
	}
}

func TestMatchVideoSubtitle(t *testing.T) {
	tests := []struct {
		name     string
		video    string
		subtitle string
		expected bool // Updated expectations based on normalization logic
	}{
		{
			name:     "Identical names",
			video:    "mymovie.mkv",
			subtitle: "mymovie.srt",
			expected: true, // Should match after normalization
		},
		{
			name:     "Identical names different case",
			video:    "MyMovie.avi",
			subtitle: "mymovie.sub",
			expected: true, // Should match after normalization
		},
		{
			name:     "Subtitle with language code",
			video:    "My.Movie.2023.1080p.mkv",
			subtitle: "My.Movie.2023.en.srt", // Normalization should remove '1080p' and 'en'
			expected: true,
		},
		{
			name:     "Subtitle with lang and flags",
			video:    "My.Movie.2023.BluRay.x264-GRP.mkv",
			subtitle: "My.Movie.2023.BluRay.x264-GRP.eng.sdh.forced.srt", // Normalize removes tags, lang, flags
			expected: true,
		},
		{
			name:     "Video with extra tags",
			video:    "My.Movie.2023.1080p.WEB-DL.H264.mkv",
			subtitle: "My.Movie.2023.en.srt", // Normalize removes tags and lang
			expected: true,
		},
		{
			name:     "Completely different names",
			video:    "Another.Movie.2022.mkv",
			subtitle: "My.Movie.2023.en.srt",
			expected: false,
		},
		{
			name:     "Partial match but different core",
			video:    "Movie.Part1.mkv",
			subtitle: "Movie.Part2.en.srt",
			expected: false, // Normalization might result in 'movie part1' vs 'movie part2'
		},
		{
			name:     "Subtitle is prefix",
			video:    "Movie.Title.Extended.Cut.mkv",
			subtitle: "Movie.Title.en.srt", // Normalize removes 'extended cut' and 'en'
			expected: true,
		},
		{
			name:     "Video is prefix",
			video:    "Movie.Title.mkv",
			subtitle: "Movie.Title.Directors.Cut.en.srt", // Normalize removes 'directors cut' and 'en'
			expected: true,
		},
		{
			name:     "Empty video name",
			video:    ".mkv",
			subtitle: "en.srt",
			expected: false, // Normalization results in empty string or just lang
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			match := metadata.MatchVideoSubtitle(tc.video, tc.subtitle)
			assert.Equal(t, tc.expected, match)
		})
	}
}

func TestFindMatchingSubtitle(t *testing.T) {
	videoFile := "/path/to/My.Movie.2023.1080p.mkv"
	subtitles := []string{
		"/subs/Another.Movie.en.srt",
		"/subs/My.Movie.2023.en.forced.srt", // Match!
		"/subs/My.Movie.2023.Part2.el.srt",
	}
	expectedMatch := "/subs/My.Movie.2023.en.forced.srt"

	match := metadata.FindMatchingSubtitle(videoFile, subtitles)
	assert.Equal(t, expectedMatch, match)
}

func TestFindMatchingSubtitle_NoMatch(t *testing.T) {
	videoFile := "/path/to/My.Movie.2023.1080p.mkv"
	subtitles := []string{
		"/subs/Another.Movie.en.srt",
		"/subs/My.Movie.2023.Part2.el.srt",
	}
	expectedMatch := ""

	match := metadata.FindMatchingSubtitle(videoFile, subtitles)
	assert.Equal(t, expectedMatch, match)
}

func TestFindMatchingVideo(t *testing.T) {
	subtitleFile := "/subs/My.Movie.2023.en.forced.srt"
	videos := []string{
		"/vids/Another.Movie.2022.mkv",
		"/vids/My.Movie.2023.Part2.mkv",
		"/vids/My.Movie.2023.1080p.BluRay.x264.mkv", // Match!
	}
	expectedMatch := "/vids/My.Movie.2023.1080p.BluRay.x264.mkv"

	match := metadata.FindMatchingVideo(subtitleFile, videos)
	assert.Equal(t, expectedMatch, match)
}

func TestFindMatchingVideo_NoMatch(t *testing.T) {
	subtitleFile := "/subs/My.Movie.2023.en.forced.srt"
	videos := []string{
		"/vids/Another.Movie.2022.mkv",
		"/vids/My.Movie.2023.Part2.mkv",
	}
	expectedMatch := ""

	match := metadata.FindMatchingVideo(subtitleFile, videos)
	assert.Equal(t, expectedMatch, match)
}

func TestConsolidateMetadata_Parsing(t *testing.T) {
	tests := []struct {
		name          string
		videoFilename string
		wantVideoInfo metadata.VideoInfo // Check only parsed fields
		wantErr       bool
	}{
		{
			name:          "Movie Basic",
			videoFilename: "My.Movie.Title.2023.1080p.BluRay.x264-GROUP.mkv",
			wantVideoInfo: metadata.VideoInfo{
				FileName:     "My.Movie.Title.2023.1080p.BluRay.x264-GROUP.mkv",
				Title:        "My Movie Title",
				Year:         2023,
				Resolution:   "1080p",
				Source:       "BluRay",
				ReleaseGroup: "GROUP",
			},
			wantErr: false,
		},
		{
			name:          "TV Show S01E02",
			videoFilename: "My.Show.S01E02.Episode.Title.720p.HDTV.x265-AnoTHER.mkv",
			wantVideoInfo: metadata.VideoInfo{
				FileName:     "My.Show.S01E02.Episode.Title.720p.HDTV.x265-AnoTHER.mkv",
				Title:        "My Show",
				Season:       1,
				Episode:      2,
				Resolution:   "720p",
				Source:       "HDTV",
				ReleaseGroup: "AnoTHER",
			},
			wantErr: false,
		},
		{
			name:          "Movie Minimal",
			videoFilename: "Another.Movie.2021.mkv",
			wantVideoInfo: metadata.VideoInfo{
				FileName: "Another.Movie.2021.mkv",
				Title:    "Another Movie",
				Year:     2021,
			},
			wantErr: false,
		},
		{
			name:          "TV Show No Year",
			videoFilename: "The.Series.S03E10.WEB-DL.x264.mkv",
			wantVideoInfo: metadata.VideoInfo{
				FileName:     "The.Series.S03E10.WEB-DL.x264.mkv",
				Title:        "The Series",
				Season:       3,
				Episode:      10,
				Source:       "WEB-DL",
				ReleaseGroup: "DL.x264",
			},
			wantErr: false,
		},
		{
			name:          "Unparsable Name",
			videoFilename: "justafile.mkv",
			wantVideoInfo: metadata.VideoInfo{
				FileName: "justafile.mkv",
				Title:    "justafile mkv",
			},
			wantErr: false,
		},
	}

	// Create dummy files needed for ConsolidateMetadata (video and subtitle)
	tempDir := t.TempDir()
	dummySubPath := filepath.Join(tempDir, "dummy.srt")
	if err := os.WriteFile(dummySubPath, []byte("dummy sub"), 0644); err != nil {
		t.Fatalf("Failed to create dummy subtitle file: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create dummy video file for this test case
			dummyVideoPath := filepath.Join(tempDir, tt.videoFilename)
			if err := os.WriteFile(dummyVideoPath, []byte("dummy video"), 0644); err != nil {
				t.Fatalf("Failed to create dummy video file: %v", err)
			}

			// Provide an empty client provider for parsing tests
			clients := metadata.APIClientProvider{}
			videoInfo, _, err := metadata.ConsolidateMetadata(context.Background(), dummyVideoPath, dummySubPath, clients)

			if (err != nil) != tt.wantErr {
				t.Errorf("ConsolidateMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return // No further checks if error was expected
			}

			// Compare only the fields populated by parsing
			if videoInfo.FileName != tt.wantVideoInfo.FileName {
				t.Errorf("ConsolidateMetadata() FileName = %v, want %v", videoInfo.FileName, tt.wantVideoInfo.FileName)
			}
			if videoInfo.Title != tt.wantVideoInfo.Title {
				t.Errorf("ConsolidateMetadata() Title = %v, want %v", videoInfo.Title, tt.wantVideoInfo.Title)
			}
			if videoInfo.Year != tt.wantVideoInfo.Year {
				t.Errorf("ConsolidateMetadata() Year = %v, want %v", videoInfo.Year, tt.wantVideoInfo.Year)
			}
			if videoInfo.Season != tt.wantVideoInfo.Season {
				t.Errorf("ConsolidateMetadata() Season = %v, want %v", videoInfo.Season, tt.wantVideoInfo.Season)
			}
			if videoInfo.Episode != tt.wantVideoInfo.Episode {
				t.Errorf("ConsolidateMetadata() Episode = %v, want %v", videoInfo.Episode, tt.wantVideoInfo.Episode)
			}
			if videoInfo.Resolution != tt.wantVideoInfo.Resolution {
				t.Errorf("ConsolidateMetadata() Resolution = %v, want %v", videoInfo.Resolution, tt.wantVideoInfo.Resolution)
			}
			if videoInfo.Source != tt.wantVideoInfo.Source {
				t.Errorf("ConsolidateMetadata() Source = %v, want %v", videoInfo.Source, tt.wantVideoInfo.Source)
			}
			if videoInfo.ReleaseGroup != tt.wantVideoInfo.ReleaseGroup {
				t.Errorf("ConsolidateMetadata() ReleaseGroup = %v, want %v", videoInfo.ReleaseGroup, tt.wantVideoInfo.ReleaseGroup)
			}
		})
	}
}

// TestConsolidateMetadata_API tests the API integration part of ConsolidateMetadata.
func TestConsolidateMetadata_API(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// --- Test Setup Helper Function ---
	setupTestFiles := func(videoName, subName, nfoContent string) (string, string) {
		videoPath := filepath.Join(tempDir, videoName)
		subPath := filepath.Join(tempDir, subName)
		nfoPath := filepath.Join(tempDir, strings.TrimSuffix(videoName, filepath.Ext(videoName))+".nfo")

		// Create dummy video/sub (content doesn't matter for API tests, but files need to exist)
		// Make video large enough for OSDb hash (needs > 128kB total)
		dummyVideoContent := make([]byte, 150*1024) // Increased size
		copy(dummyVideoContent, []byte("dummy video content start..."))
		// Add some content near the end too for hashing
		copy(dummyVideoContent[len(dummyVideoContent)-100:], []byte("...dummy video content end"))
		os.WriteFile(videoPath, dummyVideoContent, 0644)
		os.WriteFile(subPath, []byte("dummy sub content"), 0644)

		if nfoContent != "" {
			os.WriteFile(nfoPath, []byte(nfoContent), 0644)
		} else {
			os.Remove(nfoPath) // Ensure NFO doesn't exist from previous run
		}
		return videoPath, subPath
	}

	// --- Test Cases ---
	tests := []struct {
		name string
		// Input Setup
		videoName  string
		subName    string
		nfoContent string // Content for .nfo file, empty means no NFO
		// Mock Setup
		mockOS    *MockOpenSubtitlesClient
		mockTrakt *MockTraktClient
		mockIMDb  *MockIMDbClient
		// Expected Results
		wantNFOID       string
		wantOSDbID      string // Note: This is videoInfo.OSDb_IMDbID
		wantTraktID     string
		wantFinalIMDbID string // Expected IMDb ID in videoInfo.OSDb_IMDbID after all precedence
	}{
		{
			name:            "NFO Only",
			videoName:       "Movie.With.NFO.2023.mkv",
			subName:         "Movie.With.NFO.2023.en.srt",
			nfoContent:      "some stuff\nhttps://imdb.com/title/tt0011223/\nmore stuff",
			mockOS:          &MockOpenSubtitlesClient{}, // No calls expected
			mockTrakt:       &MockTraktClient{},         // No calls expected
			mockIMDb:        &MockIMDbClient{},          // No calls expected
			wantNFOID:       "tt0011223",
			wantOSDbID:      "", // Should not be set by API
			wantTraktID:     "",
			wantFinalIMDbID: "tt0011223", // NFO takes precedence
		},
		{
			name:       "OSDb Hash Find",
			videoName:  "Movie.By.Hash.2022.mkv",
			subName:    "Movie.By.Hash.2022.en.srt",
			nfoContent: "", // No NFO
			mockOS: &MockOpenSubtitlesClient{
				SearchFeaturesFunc: func(ctx context.Context, params map[string]string) (*opensubtitles.FeaturesResponse, error) {
					if params["hash"] != "" { // Only return result if called by hash
						return &opensubtitles.FeaturesResponse{
							Data: []opensubtitles.Feature{
								{Attributes: opensubtitles.FeatureAttributes{ImdbID: 1122334}}, // Use int ID
							},
						}, nil
					}
					return nil, nil
				},
			},
			mockTrakt:       &MockTraktClient{}, // No Trakt call expected yet (IMDb found by OS)
			mockIMDb:        &MockIMDbClient{},
			wantNFOID:       "",
			wantOSDbID:      "tt1122334",
			wantTraktID:     "",
			wantFinalIMDbID: "tt1122334",
		},
		{
			name:       "Trakt Find by IMDb ID",
			videoName:  "Movie.For.Trakt.2021.mkv",
			subName:    "Movie.For.Trakt.2021.en.srt",
			nfoContent: "tt2233445",                // Provide IMDb ID via NFO
			mockOS:     &MockOpenSubtitlesClient{}, // No OS call needed
			mockTrakt: &MockTraktClient{
				SearchTraktFunc: func(ctx context.Context, queryType string, query string) ([]trakt.SearchResult, error) {
					if queryType == "imdb" && query == "tt2233445" {
						return []trakt.SearchResult{
							{
								Type: "movie", Year: 2021, Title: "Mock Trakt Movie",
								IDs: map[string]string{"trakt": "9876", "imdb": "tt2233445"},
							},
						}, nil
					}
					return nil, nil
				},
			},
			mockIMDb:        &MockIMDbClient{},
			wantNFOID:       "tt2233445",
			wantOSDbID:      "",
			wantTraktID:     "9876",
			wantFinalIMDbID: "tt2233445", // NFO takes precedence
		},
		{
			name:       "Trakt Find by Title & Year",
			videoName:  "Show.S01E01.Title.Search.mkv", // Parsed Title: Show, S:1 E:1
			subName:    "Show.S01E01.en.srt",
			nfoContent: "",                         // No NFO
			mockOS:     &MockOpenSubtitlesClient{}, // Assume OS hash fails or returns no IMDb
			mockTrakt: &MockTraktClient{
				SearchTraktFunc: func(ctx context.Context, queryType string, query string) ([]trakt.SearchResult, error) {
					if queryType == "show,episode" && query == "Show" {
						return []trakt.SearchResult{
							{
								Type: "show", Title: "Show",
								IDs: map[string]string{"trakt": "5432", "imdb": "tt3344556"},
							},
						}, nil
					}
					return nil, nil
				},
			},
			mockIMDb:        &MockIMDbClient{},
			wantNFOID:       "",
			wantOSDbID:      "tt3344556", // Found via Trakt
			wantTraktID:     "5432",
			wantFinalIMDbID: "tt3344556",
		},
		{
			name:       "IMDb Suggest Find",
			videoName:  "Last.Resort.Movie.2020.mkv",
			subName:    "Last.Resort.Movie.2020.en.srt",
			nfoContent: "",
			mockOS:     &MockOpenSubtitlesClient{}, // Assume OS fails
			mockTrakt:  &MockTraktClient{},         // Assume Trakt fails
			mockIMDb: &MockIMDbClient{
				SearchIMDBSuggestionsFunc: func(ctx context.Context, query string) ([]imdb.IMDBSuggestion, error) {
					if query == "Last Resort Movie" { // Assuming ptn parses title like this
						return []imdb.IMDBSuggestion{
							{ID: "tt4455667", Title: "Last Resort Movie", Year: 2020},
						}, nil
					}
					return nil, nil
				},
			},
			wantNFOID:       "",
			wantOSDbID:      "tt4455667", // Found via IMDb Suggest
			wantTraktID:     "",
			wantFinalIMDbID: "tt4455667",
		},
		{
			name:       "Precedence NFO > OSDb > Trakt",
			videoName:  "Precedence.Clash.2019.mkv",
			subName:    "Precedence.Clash.2019.en.srt",
			nfoContent: "tt0000001", // NFO ID
			mockOS: &MockOpenSubtitlesClient{ // OS would find different ID if called
				SearchFeaturesFunc: func(ctx context.Context, params map[string]string) (*opensubtitles.FeaturesResponse, error) {
					return &opensubtitles.FeaturesResponse{
						Data: []opensubtitles.Feature{
							{Attributes: opensubtitles.FeatureAttributes{ImdbID: 2}},
						},
					}, nil
				},
			},
			mockTrakt: &MockTraktClient{ // Trakt finds yet another ID
				SearchTraktFunc: func(ctx context.Context, queryType string, query string) ([]trakt.SearchResult, error) {
					if queryType == "imdb" && query == "tt0000001" { // Called with NFO ID
						return []trakt.SearchResult{
							{
								Type: "movie", Year: 2019, Title: "Mock Trakt Movie",
								IDs: map[string]string{"trakt": "111", "imdb": "tt0000003"},
							},
						}, nil
					}
					return nil, nil
				},
			},
			mockIMDb:        &MockIMDbClient{},
			wantNFOID:       "tt0000001",
			wantOSDbID:      "", // OSDb hash search shouldn't run because NFO ID exists
			wantTraktID:     "111",
			wantFinalIMDbID: "tt0000001", // NFO wins
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			videoPath, subPath := setupTestFiles(tt.videoName, tt.subName, tt.nfoContent)

			clients := metadata.APIClientProvider{
				OSClient:    tt.mockOS,
				TraktClient: tt.mockTrakt,
				IMDbClient:  tt.mockIMDb,
			}

			videoInfo, _, err := metadata.ConsolidateMetadata(ctx, videoPath, subPath, clients)

			assert.NoError(t, err)
			assert.NotNil(t, videoInfo)

			// Assertions
			assert.Equal(t, tt.wantNFOID, videoInfo.NFO_IMDbID, "NFO_IMDbID mismatch")
			assert.Equal(t, tt.wantTraktID, videoInfo.TraktID, "TraktID mismatch")
			assert.Equal(t, tt.wantFinalIMDbID, videoInfo.OSDb_IMDbID, "Final OSDb_IMDbID mismatch after precedence")

			// Assert mock calls
			if tt.name == "NFO Only" {
				assert.Nil(t, tt.mockOS.CalledWithParams, "OSClient should not be called with NFO")
				// Expect Trakt to be called with the NFO ID if TraktID is missing
				assert.Equal(t, "tt0011223", tt.mockTrakt.CalledWithQuery, "TraktClient should be called with NFO IMDb ID")
				assert.Equal(t, "imdb", tt.mockTrakt.CalledWithQueryType, "TraktClient should be called with type imdb")
				assert.Empty(t, tt.mockIMDb.CalledWithQuery, "IMDbClient should not be called with NFO")
			}
			if tt.name == "OSDb Hash Find" {
				// Expect OSDb to be called with hash
				assert.NotNil(t, tt.mockOS.CalledWithParams, "OSClient should be called")
				assert.NotEmpty(t, tt.mockOS.CalledWithParams["hash"], "OSClient should be called with hash")
				// Expect Trakt to be called with the ID found by OSDb
				assert.Equal(t, "tt1122334", tt.mockTrakt.CalledWithQuery, "TraktClient should be called with OSDb IMDb ID")
				assert.Equal(t, "imdb", tt.mockTrakt.CalledWithQueryType, "TraktClient should be called with type imdb")
				assert.Empty(t, tt.mockIMDb.CalledWithQuery, "IMDbClient should not be called")
			}
			// Add more assertions for other cases if needed

		})
	}
}
