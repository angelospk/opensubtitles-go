package opensubtitles

import "time"

// --- Common Types ---

// LanguageCode represents an ISO 639-1 or 639-2/B language code.
type LanguageCode string

// SortDirection defines the sorting order.
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// FilterInclusion defines include/exclude options.
type FilterInclusion string

const (
	Include FilterInclusion = "include"
	Exclude FilterInclusion = "exclude"
)

// FilterInclusionOnly defines include/exclude/only options.
type FilterInclusionOnly string

const (
	IncludeOnly FilterInclusionOnly = "include"
	ExcludeOnly FilterInclusionOnly = "exclude" // Not used in docs, but for completeness
	Only        FilterInclusionOnly = "only"
)

// FilterTrustedSources defines include/only options for trusted sources.
type FilterTrustedSources string

const (
	IncludeTrusted FilterTrustedSources = "include"
	OnlyTrusted    FilterTrustedSources = "only"
)

// FeatureType defines the type of a feature.
type FeatureType string

const (
	FeatureMovie   FeatureType = "movie"
	FeatureTVShow  FeatureType = "tvshow"
	FeatureEpisode FeatureType = "episode"
	FeatureAll     FeatureType = "all" // For searching
)

// PaginatedResponse defines the structure for paginated API responses.
type PaginatedResponse struct {
	TotalPages int `json:"total_pages"`
	TotalCount int `json:"total_count"`
	PerPage    int `json:"per_page"`
	Page       int `json:"page"`
}

// ApiDataWrapper wraps the common "id", "type", "attributes" structure.
// We'll often embed this or use specific types due to Go's static nature.
type ApiDataWrapper struct {
	ID   string `json:"id"`
	Type string `json:"type"` // e.g., "subtitle", "feature"
}

// UploaderInfo contains details about the subtitle uploader.
type UploaderInfo struct {
	UploaderID *int    `json:"uploader_id"` // Pointer as can be null
	Name       *string `json:"name"`        // Pointer as can be null/empty
	Rank       *string `json:"rank"`        // Pointer as can be null/empty
}

// RelatedLink represents links found in subtitle details.
type RelatedLink struct {
	Label  string  `json:"label"`
	URL    string  `json:"url"`
	ImgURL *string `json:"img_url"` // Pointer as optional
}

// SubtitleCounts maps language codes to subtitle counts.
type SubtitleCounts map[LanguageCode]int

// BaseUserInfo contains common user details.
type BaseUserInfo struct {
	AllowedDownloads int    `json:"allowed_downloads"`
	Level            string `json:"level"`
	UserID           int    `json:"user_id"`
	ExtInstalled     bool   `json:"ext_installed"`
	VIP              bool   `json:"vip"`
}

// --- Auth Types ---

// LoginRequest is the request body for the login endpoint.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginUser extends BaseUserInfo with fields specific to the login response.
type LoginUser struct {
	BaseUserInfo
	AllowedTranslations int `json:"allowed_translations"`
}

// LoginResponse is the response from the login endpoint.
type LoginResponse struct {
	User    LoginUser `json:"user"`
	BaseURL string    `json:"base_url"`
	Token   string    `json:"token"`
	Status  int       `json:"status"`
}

// LogoutResponse is the response from the logout endpoint.
type LogoutResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

// UserInfo contains details from the /infos/user endpoint.
type UserInfo struct {
	BaseUserInfo
	DownloadsCount     int `json:"downloads_count"`
	RemainingDownloads int `json:"remaining_downloads"`
}

// GetUserInfoResponse wraps the UserInfo data.
type GetUserInfoResponse struct {
	Data UserInfo `json:"data"`
}

// --- Feature Types ---

// SearchFeaturesParams defines query parameters for the /features endpoint.
// Use pointers for optional fields. Use `url:"..."` tag for query string encoding.
type SearchFeaturesParams struct {
	FeatureID  *int    `url:"feature_id,omitempty"`
	IMDbID     *string `url:"imdb_id,omitempty"` // String to handle potential leading zeros if needed, API expects number string
	TMDBID     *string `url:"tmdb_id,omitempty"` // String based on example
	Query      *string `url:"query,omitempty"`
	QueryMatch *string `url:"query_match,omitempty"` // "start", "word", "exact"
	FullSearch *bool   `url:"full_search,omitempty"`
	Type       *string `url:"type,omitempty"` // FeatureType or "all" or empty string
	Year       *int    `url:"year,omitempty"`
}

// FeatureBaseAttributes holds common fields for all feature types.
type FeatureBaseAttributes struct {
	FeatureID       string         `json:"feature_id"`
	FeatureType     string         `json:"feature_type"` // "Movie", "Tvshow", "Episode"
	Title           string         `json:"title"`
	OriginalTitle   *string        `json:"original_title"` // Pointer as can be null
	Year            string         `json:"year"`           // String in API response
	IMDbID          *int           `json:"imdb_id"`        // Pointer as can be null
	TMDBID          *int           `json:"tmdb_id"`        // Pointer as can be null
	TitleAKA        []string       `json:"title_aka"`
	URL             string         `json:"url"`
	ImgURL          *string        `json:"img_url"` // Pointer as can be null
	SubtitlesCount  int            `json:"subtitles_count"`
	SubtitlesCounts SubtitleCounts `json:"subtitles_counts"`
}

// TvshowEpisodeStub represents basic episode info within a season list.
type TvshowEpisodeStub struct {
	EpisodeNumber int    `json:"episode_number"`
	Title         string `json:"title"`
	FeatureID     string `json:"feature_id"`
}

// TvshowSeason represents a season with its episodes.
type TvshowSeason struct {
	SeasonNumber int                 `json:"season_number"`
	Episodes     []TvshowEpisodeStub `json:"episodes"`
}

// FeatureMovieAttributes specific fields for movies.
type FeatureMovieAttributes struct {
	FeatureBaseAttributes
	// Potentially null fields that might appear from schema examples
	SeasonsCount    *int    `json:"seasons_count"`
	ParentTitle     *string `json:"parent_title"`
	SeasonNumber    *int    `json:"season_number"`
	EpisodeNumber   *int    `json:"episode_number"`
	ParentIMDbID    *int    `json:"parent_imdb_id"`
	ParentTMDBID    *int    `json:"parent_tmdb_id"`    // Added for consistency
	ParentFeatureID *string `json:"parent_feature_id"` // Added for consistency
}

// FeatureTvshowAttributes specific fields for TV shows.
type FeatureTvshowAttributes struct {
	FeatureBaseAttributes
	SeasonsCount int            `json:"seasons_count"`
	Seasons      []TvshowSeason `json:"seasons"`
}

// FeatureEpisodeAttributes specific fields for episodes.
type FeatureEpisodeAttributes struct {
	FeatureBaseAttributes
	ParentIMDbID    *int    `json:"parent_imdb_id"`    // Pointer as can be null
	ParentTitle     *string `json:"parent_title"`      // Pointer as can be null
	ParentTMDBID    *int    `json:"parent_tmdb_id"`    // Pointer as can be null
	ParentFeatureID *string `json:"parent_feature_id"` // Pointer as can be null
	SeasonNumber    int     `json:"season_number"`
	EpisodeNumber   int     `json:"episode_number"`
	MovieName       *string `json:"movie_name"` // Optional formatted name
}

// Feature represents any feature type returned by the API.
// Use type assertion on Attributes based on FeatureType.
type Feature struct {
	ApiDataWrapper
	Attributes interface{} `json:"attributes"` // Use specific attribute struct after unmarshalling based on FeatureType
}

// SearchFeaturesResponse wraps the list of features.
type SearchFeaturesResponse struct {
	Data []Feature `json:"data"`
}

// --- Subtitle Types ---

// SubtitleFeatureDetails represents the nested feature info within a subtitle.
type SubtitleFeatureDetails struct {
	FeatureID       int     `json:"feature_id"`
	FeatureType     string  `json:"feature_type"` // "Movie", "Episode"
	Year            int     `json:"year"`
	Title           string  `json:"title"`
	MovieName       string  `json:"movie_name"`
	IMDbID          *int    `json:"imdb_id"`
	TMDBID          *int    `json:"tmdb_id"`
	SeasonNumber    *int    `json:"season_number"`
	EpisodeNumber   *int    `json:"episode_number"`
	ParentIMDbID    *int    `json:"parent_imdb_id"`
	ParentTMDBID    *int    `json:"parent_tmdb_id"`
	ParentTitle     *string `json:"parent_title"`
	ParentFeatureID *int    `json:"parent_feature_id"` // Number in example
}

// SubtitleFile represents a single file within a subtitle entry.
type SubtitleFile struct {
	FileID   int    `json:"file_id"` // **ID needed for download**
	CDNumber int    `json:"cd_number"`
	FileName string `json:"file_name"`
}

// SubtitleAttributes holds the details of a subtitle entry.
type SubtitleAttributes struct {
	SubtitleID        string                 `json:"subtitle_id"`
	Language          LanguageCode           `json:"language"`
	DownloadCount     int                    `json:"download_count"`
	NewDownloadCount  int                    `json:"new_download_count"`
	HearingImpaired   bool                   `json:"hearing_impaired"`
	HD                bool                   `json:"hd"`
	FPS               *float64               `json:"fps"` // Pointer as can be 0 or null
	Votes             int                    `json:"votes"`
	Points            *float64               `json:"points"` // Pointer, legacy?
	Ratings           float64                `json:"ratings"`
	FromTrusted       bool                   `json:"from_trusted"`
	ForeignPartsOnly  bool                   `json:"foreign_parts_only"`
	UploadDate        time.Time              `json:"upload_date"` // Use time.Time for ISO 8601
	AITranslated      bool                   `json:"ai_translated"`
	MachineTranslated bool                   `json:"machine_translated"`
	MoviehashMatch    *bool                  `json:"moviehash_match,omitempty"` // Pointer, only present sometimes
	Release           string                 `json:"release"`
	Comments          *string                `json:"comments"`           // Pointer as can be null
	LegacySubtitleID  *int                   `json:"legacy_subtitle_id"` // Pointer as can be null
	NbCD              *int                   `json:"nb_cd"`              // Pointer, from example
	Slug              *string                `json:"slug"`               // Pointer, from example
	Uploader          UploaderInfo           `json:"uploader"`
	FeatureDetails    SubtitleFeatureDetails `json:"feature_details"`
	URL               string                 `json:"url"`
	RelatedLinks      []RelatedLink          `json:"related_links"`
	Files             []SubtitleFile         `json:"files"`
}

// Subtitle represents a full subtitle entry.
type Subtitle struct {
	ApiDataWrapper
	Attributes SubtitleAttributes `json:"attributes"`
}

// SearchSubtitlesParams defines query parameters for the /subtitles endpoint.
type SearchSubtitlesParams struct {
	ID                *int                  `url:"id,omitempty"` // Feature ID
	IMDbID            *int                  `url:"imdb_id,omitempty"`
	TMDBID            *int                  `url:"tmdb_id,omitempty"`
	ParentIMDbID      *int                  `url:"parent_imdb_id,omitempty"`
	ParentTMDBID      *int                  `url:"parent_tmdb_id,omitempty"`
	ParentFeatureID   *int                  `url:"parent_feature_id,omitempty"`
	Query             *string               `url:"query,omitempty"`
	SeasonNumber      *int                  `url:"season_number,omitempty"`
	EpisodeNumber     *int                  `url:"episode_number,omitempty"`
	Moviehash         *string               `url:"moviehash,omitempty"` // Must match `^[a-f0-9]{16}$`
	Languages         *string               `url:"languages,omitempty"` // Comma-separated, sorted LanguageCodes
	Type              *string               `url:"type,omitempty"`      // "movie", "episode", "all"
	Year              *int                  `url:"year,omitempty"`
	AITranslated      *FilterInclusion      `url:"ai_translated,omitempty"`
	MachineTranslated *FilterInclusion      `url:"machine_translated,omitempty"`
	HearingImpaired   *FilterInclusionOnly  `url:"hearing_impaired,omitempty"`
	ForeignPartsOnly  *FilterInclusionOnly  `url:"foreign_parts_only,omitempty"`
	TrustedSources    *FilterTrustedSources `url:"trusted_sources,omitempty"`
	MoviehashMatch    *string               `url:"moviehash_match,omitempty"` // "include", "only"
	UploaderID        *int                  `url:"uploader_id,omitempty"`
	OrderBy           *string               `url:"order_by,omitempty"` // Field name from allowed list
	OrderDirection    *SortDirection        `url:"order_direction,omitempty"`
	Page              *int                  `url:"page,omitempty"`
}

// SearchSubtitlesResponse wraps the paginated subtitle results.
type SearchSubtitlesResponse struct {
	PaginatedResponse
	Data []Subtitle `json:"data"`
}

// DownloadRequest is the request body for the /download endpoint.
type DownloadRequest struct {
	FileID        int      `json:"file_id"`
	SubFormat     *string  `json:"sub_format,omitempty"`
	FileName      *string  `json:"file_name,omitempty"`
	InFPS         *float64 `json:"in_fps,omitempty"`         // Pointer as optional
	OutFPS        *float64 `json:"out_fps,omitempty"`        // Pointer as optional
	Timeshift     *float64 `json:"timeshift,omitempty"`      // Pointer as optional
	ForceDownload *bool    `json:"force_download,omitempty"` // Pointer as optional
}

// DownloadResponse is the response from the /download endpoint.
type DownloadResponse struct {
	Link         string    `json:"link"`
	FileName     string    `json:"file_name"`
	Requests     int       `json:"requests"`
	Remaining    int       `json:"remaining"`
	Message      string    `json:"message"`
	ResetTime    string    `json:"reset_time"`
	ResetTimeUTC time.Time `json:"reset_time_utc"` // Use time.Time
}

// --- Discover Types ---

// DiscoverParams defines common query parameters for discover endpoints.
type DiscoverParams struct {
	Language *LanguageCode `url:"language,omitempty"` // Single language code or "all"
	Type     *FeatureType  `url:"type,omitempty"`     // "movie", "tvshow"
}

// DiscoverPopularResponse wraps the list of popular features.
// Note: API returns mixed movie/tvshow features. Need runtime check.
type DiscoverPopularResponse struct {
	Data []Feature `json:"data"` // Contains FeatureMovieAttributes or FeatureTvshowAttributes
}

// DiscoverLatestResponse wraps the list of latest subtitles (fixed count).
type DiscoverLatestResponse struct {
	TotalPages int        `json:"total_pages"` // Should be 1
	TotalCount int        `json:"total_count"` // Should be 60
	Page       int        `json:"page"`        // Should be 1
	Data       []Subtitle `json:"data"`
}

// DiscoverMostDownloadedResponse wraps paginated most downloaded subtitles.
type DiscoverMostDownloadedResponse struct {
	PaginatedResponse
	Data []Subtitle `json:"data"`
}

// --- Utilities Types ---

// GuessitParams defines query parameters for the /utilities/guessit endpoint.
type GuessitParams struct {
	Filename string `url:"filename"` // Required
}

// GuessitResponse is the response from the /utilities/guessit endpoint.
// All fields are pointers as they might be null if not detected.
type GuessitResponse struct {
	Title            *string       `json:"title"`
	Year             *int          `json:"year"`
	Season           *int          `json:"season"`
	Episode          *int          `json:"episode"`
	EpisodeTitle     *string       `json:"episode_title"`
	Language         *LanguageCode `json:"language"`
	SubtitleLanguage *LanguageCode `json:"subtitle_language"`
	ScreenSize       *string       `json:"screen_size"`
	StreamingService *string       `json:"streaming_service"`
	Source           *string       `json:"source"`
	Other            *string       `json:"other"` // Docs example show null, not array
	AudioCodec       *string       `json:"audio_codec"`
	AudioChannels    *string       `json:"audio_channels"`
	AudioProfile     *string       `json:"audio_profile"`
	VideoCodec       *string       `json:"video_codec"`
	ReleaseGroup     *string       `json:"release_group"`
	Type             *string       `json:"type"` // "episode", "movie"
}
