package metadata

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/angelospk/osuploadergui/pkg/core/fileops"
	"github.com/angelospk/osuploadergui/pkg/core/imdb"
	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles"
	"github.com/angelospk/osuploadergui/pkg/core/trakt"
	ptn "github.com/razsteinmetz/go-ptn"
	log "github.com/sirupsen/logrus"
)

// Regex to detect common part/disk indicators (case-insensitive)
var partIndicatorRegex = regexp.MustCompile(`(?i)[._ \-]part[._ \-]?\d+|[._ \-]cd[._ \-]?\d+|[._ \-]disk[._ \-]?\d+`)

// VideoInfo holds consolidated information about a video file.
type VideoInfo struct {
	FilePath   string `json:"filePath"`
	FileName   string `json:"fileName"`
	FileSize   int64  `json:"fileSize"`
	OSDbHash   string `json:"osdbHash,omitempty"`
	NFO_IMDbID string `json:"nfoImdbId,omitempty"` // IMDb ID extracted from NFO

	Title        string `json:"title,omitempty"`
	Year         int    `json:"year,omitempty"`
	Season       int    `json:"season,omitempty"`
	Episode      int    `json:"episode,omitempty"`
	Resolution   string `json:"resolution,omitempty"` // e.g., "1080p", "720p"
	Source       string `json:"source,omitempty"`     // e.g., "BluRay", "WEB-DL"
	ReleaseGroup string `json:"releaseGroup,omitempty"`

	// Potential data from APIs
	OSDb_IMDbID string `json:"osdbImdbId,omitempty"` // From OpenSubtitles feature search
	TraktID     string `json:"traktId,omitempty"`    // From Trakt search

	// TODO: Add fields from fileops.MediaInfo if needed directly
	// e.g., Duration, VideoCodec, AudioCodec, etc.
	// MediaInfo   *fileops.MediaInfo `json:"mediaInfo,omitempty"`
}

// SubtitleInfo holds consolidated information about a subtitle file.
type SubtitleInfo struct {
	FilePath string `json:"filePath"`
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	MD5Hash  string `json:"md5Hash,omitempty"`

	Language             string `json:"language,omitempty"` // Detected/Assumed language (e.g., "en", "el")
	Format               string `json:"format,omitempty"`   // e.g., "srt", "ass", "sub"
	IsForHearingImpaired bool   `json:"isForHearingImpaired,omitempty"`
	IsForced             bool   `json:"isForced,omitempty"`
	Encoding             string `json:"encoding,omitempty"` // e.g., "UTF-8"
}

// UploadJob represents a pair of video and subtitle files prepared for upload.
type UploadJob struct {
	VideoInfo    *VideoInfo    `json:"videoInfo"`
	SubtitleInfo *SubtitleInfo `json:"subtitleInfo"`

	// Information needed for OpenSubtitles upload API
	OSDbFeatureID  int64  `json:"osdbFeatureId,omitempty"`  // ID of the movie/episode on OpenSubtitles
	UploadLanguage string `json:"uploadLanguage,omitempty"` // Language code for the upload

	// Status tracking
	Status      JobStatus `json:"status"`
	Message     string    `json:"message,omitempty"` // Error or success message
	SubmittedAt time.Time `json:"submittedAt,omitempty"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
}

// JobStatus defines the possible states of an upload job.
type JobStatus string

const (
	StatusPending    JobStatus = "Pending"    // Initial state, waiting for processing
	StatusProcessing JobStatus = "Processing" // Metadata lookup, matching in progress
	StatusReady      JobStatus = "Ready"      // Ready for upload
	StatusUploading  JobStatus = "Uploading"  // Upload in progress
	StatusComplete   JobStatus = "Complete"   // Upload successful
	StatusFailed     JobStatus = "Failed"     // Processing or upload failed
	StatusSkipped    JobStatus = "Skipped"    // User skipped or duplicate found
)

// --- Client Interfaces for Dependency Injection ---

// OpenSubtitlesClient defines the methods needed from the OpenSubtitles client.
type OpenSubtitlesClient interface {
	// SearchFeatures searches for features (movies/episodes) on OpenSubtitles.
	SearchFeatures(ctx context.Context, params map[string]string) (*opensubtitles.FeaturesResponse, error)
	// Add other methods used by ConsolidateMetadata if any (e.g., GetUserInfo?)
}

// TraktClient defines the methods needed from the Trakt client.
type TraktClient interface {
	// SearchTrakt searches Trakt for movies or shows.
	SearchTrakt(ctx context.Context, queryType string, query string) ([]trakt.SearchResult, error)
}

// IMDbClient defines the methods needed from the IMDb client.
type IMDbClient interface {
	// SearchSuggest searches IMDb suggestions.
	SearchIMDBSuggestions(ctx context.Context, query string) ([]imdb.IMDBSuggestion, error)
}

// LanguageInfo holds details for a specific language.
type LanguageInfo struct {
	OSCode string // Code used by OpenSubtitles API (e.g., "en", "pb", "pt-br")
	Code2  string // ISO 639-1 Code (e.g., "en", "pt")
	Code3  string // ISO 639-2/3 Code (e.g., "eng", "por")
	Name   string // English name (e.g., "English", "Portuguese")
}

// languagesDB stores known language information.
// Key: Lowercase version of OSCode, Code2, Code3, or Name for lookup.
// Value: The corresponding LanguageInfo struct.
// We populate this based on OpenSubtitles API and ISO 639 data.
var languagesDB = map[string]LanguageInfo{}

func init() {
	// Populate the database - Add more languages as needed!
	languages := []LanguageInfo{
		// Define with the PREFERRED OSCode first
		{OSCode: "en", Code2: "en", Code3: "eng", Name: "English"},
		{OSCode: "el", Code2: "el", Code3: "gre", Name: "Greek"}, // Note: ISO 639-2/3 uses 'ell'/'gre'
		{OSCode: "es", Code2: "es", Code3: "spa", Name: "Spanish"},
		{OSCode: "fr", Code2: "fr", Code3: "fre", Name: "French"}, // Note: ISO 639-2/3 uses 'fra'/'fre'
		{OSCode: "de", Code2: "de", Code3: "ger", Name: "German"}, // Note: ISO 639-2/3 uses 'deu'/'ger'
		{OSCode: "it", Code2: "it", Code3: "ita", Name: "Italian"},
		// Ensure pt-br gets its own specific mapping if the OSCode is found
		{OSCode: "pt-br", Code2: "pt", Code3: "por", Name: "Portuguese (Brazilian)"}, // OS maps pt-br to pt/por
		{OSCode: "pt-pt", Code2: "pt", Code3: "por", Name: "Portuguese"},             // OS maps pt-pt to pt/por
		{OSCode: "zh-cn", Code2: "zh", Code3: "zho", Name: "Chinese (simplified)"},   // OS maps zh-cn to zh/zho
		{OSCode: "zh-tw", Code2: "zh", Code3: "zho", Name: "Chinese (traditional)"},  // OS maps zh-tw to zh/zho
		{OSCode: "ze", Code2: "zh", Code3: "zho", Name: "Chinese bilingual"},         // Keep OS specific code if needed
		// ... Add ALL languages from the API list and common variations ...
		{OSCode: "af", Code2: "af", Code3: "afr", Name: "Afrikaans"},
		{OSCode: "sq", Code2: "sq", Code3: "sqi", Name: "Albanian"},
		{OSCode: "ar", Code2: "ar", Code3: "ara", Name: "Arabic"},
		{OSCode: "hy", Code2: "hy", Code3: "hye", Name: "Armenian"},
		{OSCode: "eu", Code2: "eu", Code3: "eus", Name: "Basque"},
		{OSCode: "bn", Code2: "bn", Code3: "ben", Name: "Bengali"},
		{OSCode: "bg", Code2: "bg", Code3: "bul", Name: "Bulgarian"},
		{OSCode: "ca", Code2: "ca", Code3: "cat", Name: "Catalan"},
		{OSCode: "hr", Code2: "hr", Code3: "hrv", Name: "Croatian"},
		{OSCode: "cs", Code2: "cs", Code3: "ces", Name: "Czech"},
		{OSCode: "da", Code2: "da", Code3: "dan", Name: "Danish"},
		{OSCode: "nl", Code2: "nl", Code3: "nld", Name: "Dutch"},
		{OSCode: "fi", Code2: "fi", Code3: "fin", Name: "Finnish"},
		{OSCode: "he", Code2: "he", Code3: "heb", Name: "Hebrew"},
		{OSCode: "hi", Code2: "hi", Code3: "hin", Name: "Hindi"},
		{OSCode: "hu", Code2: "hu", Code3: "hun", Name: "Hungarian"},
		{OSCode: "id", Code2: "id", Code3: "ind", Name: "Indonesian"},
		{OSCode: "ja", Code2: "ja", Code3: "jpn", Name: "Japanese"},
		{OSCode: "ko", Code2: "ko", Code3: "kor", Name: "Korean"},
		{OSCode: "lv", Code2: "lv", Code3: "lav", Name: "Latvian"},
		{OSCode: "lt", Code2: "lt", Code3: "lit", Name: "Lithuanian"},
		{OSCode: "mk", Code2: "mk", Code3: "mkd", Name: "Macedonian"},
		{OSCode: "ms", Code2: "ms", Code3: "msa", Name: "Malay"},
		{OSCode: "no", Code2: "no", Code3: "nor", Name: "Norwegian"},
		{OSCode: "fa", Code2: "fa", Code3: "fas", Name: "Persian"},
		{OSCode: "pl", Code2: "pl", Code3: "pol", Name: "Polish"},
		{OSCode: "ro", Code2: "ro", Code3: "ron", Name: "Romanian"},
		{OSCode: "ru", Code2: "ru", Code3: "rus", Name: "Russian"},
		{OSCode: "sr", Code2: "sr", Code3: "srp", Name: "Serbian"},
		{OSCode: "sk", Code2: "sk", Code3: "slk", Name: "Slovak"},
		{OSCode: "sl", Code2: "sl", Code3: "slv", Name: "Slovenian"},
		{OSCode: "sv", Code2: "sv", Code3: "swe", Name: "Swedish"},
		{OSCode: "th", Code2: "th", Code3: "tha", Name: "Thai"},
		{OSCode: "tr", Code2: "tr", Code3: "tur", Name: "Turkish"},
		{OSCode: "uk", Code2: "uk", Code3: "ukr", Name: "Ukrainian"},
		{OSCode: "vi", Code2: "vi", Code3: "vie", Name: "Vietnamese"},
	}

	// Populate the map using various keys for lookup
	for _, lang := range languages {
		// Use a consistent approach: map variations to the LanguageInfo struct
		// Lowercase keys for case-insensitive lookup
		keys := []string{}
		if lang.OSCode != "" {
			keys = append(keys, strings.ToLower(lang.OSCode))
		}
		if lang.Code2 != "" {
			keys = append(keys, strings.ToLower(lang.Code2))
		}
		if lang.Code3 != "" {
			keys = append(keys, strings.ToLower(lang.Code3))
		}
		if lang.Name != "" {
			keys = append(keys, strings.ToLower(lang.Name))
			// Add common variations manually if needed
			if lang.Code3 == "gre" {
				keys = append(keys, "greek")
			}
			if lang.Code3 == "ger" {
				keys = append(keys, "german")
			}
			if lang.Code3 == "fre" {
				keys = append(keys, "french")
			}
			if lang.Code3 == "ita" {
				keys = append(keys, "italian")
			}
			if lang.Code3 == "spa" {
				keys = append(keys, "spanish")
			}
			// Add others if necessary
		}

		for _, key := range keys {
			// Prioritize more specific keys if a general key already exists
			// Specifically, ensure pt-br OSCode maps to the pt-br struct
			// and pt-pt OSCode maps to the pt-pt struct.
			// If the key is a specific OSCode like "pt-br", always map it.
			// If the key is more general like "pt", only map it if it doesn't
			// already exist, preventing overwrite by a less specific entry later.
			if key == "pt-br" && lang.OSCode == "pt-br" {
				languagesDB[key] = lang
				continue // Ensure this specific mapping
			}
			if key == "pt-pt" && lang.OSCode == "pt-pt" {
				languagesDB[key] = lang
				continue // Ensure this specific mapping
			}

			// For general keys (like "pt", "por", "portuguese"), only add if not present
			// This allows the first specific OSCode entry (pt-br or pt-pt) to claim these general keys.
			isGeneralKey := (key == "pt" || key == "por" || strings.Contains(key, "portuguese"))
			if isGeneralKey {
				if _, exists := languagesDB[key]; !exists {
					languagesDB[key] = lang
				}
			} else {
				// For other languages or non-conflicting keys, add/overwrite normally
				// (prioritizing OSCodes defined earlier in the list if keys overlap)
				if _, exists := languagesDB[key]; !exists || languagesDB[key].OSCode == "" { // Allow overwrite if current is empty OSCode? Careful.
					languagesDB[key] = lang
				}
				// Or simply overwrite: last definition wins for a given key
				// languagesDB[key] = lang
			}
			// Let's simplify: The LAST entry in the 'languages' slice for a given key will win.
			// So, putting pt-br before pt-pt means pt-pt will claim 'pt', 'por', 'portuguese'
			// unless explicitly checked. Let's stick to explicit OSCode check and default add.
			_, exists := languagesDB[key]
			isSpecificOSCodeKey := (key == strings.ToLower(lang.OSCode))

			if !exists || isSpecificOSCodeKey {
				languagesDB[key] = lang
			}
		}
	}
}

// --- Metadata Extraction/Analysis Functions ---

// DetectSubtitleLanguage attempts to detect the subtitle language from its filename.
// It checks against known ISO codes (2/3 letter) and full names.
// Returns the matching OpenSubtitles language code (e.g., "en", "pt-br") or empty string.
func DetectSubtitleLanguage(filename string) string {
	// fmt.Printf("DEBUG: DetectSubtitleLanguage called with: %s\n", filename) // DEBUG
	lowerFilename := strings.ToLower(filename)
	base := lowerFilename

	// Define known subtitle extensions
	subtitleExts := []string{".srt", ".sub", ".ass", ".ssa", ".vtt", ".txt"} // Add more if needed

	// Remove known subtitle extension ONLY
	for _, ext := range subtitleExts {
		if strings.HasSuffix(lowerFilename, ext) {
			base = strings.TrimSuffix(lowerFilename, ext)
			break // Stop after finding the first matching extension
		}
	}

	// If no subtitle extension was found, filepath.Ext might have incorrectly removed
	// a language code (like .en). In this case, we should use the original lowerFilename.
	// Let's reconsider: the goal is to split the name *before* the language tag.
	// The current split logic handles `movie.en` if the extension isn't removed first.
	// Reset base to the original lower filename before splitting.
	base = lowerFilename
	// Now remove the *correct* extension if it exists among known sub extensions
	for _, ext := range subtitleExts {
		if strings.HasSuffix(lowerFilename, ext) {
			base = strings.TrimSuffix(lowerFilename, ext)
			break
		}
	}

	// fmt.Printf("DEBUG: base=%s\n", base) // DEBUG

	// Split the potentially extension-less base name into parts
	parts := strings.FieldsFunc(base, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == ' '
	})
	// fmt.Printf("DEBUG: parts=%v\n", parts) // DEBUG

	// Check all parts from right-to-left
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		// fmt.Printf("DEBUG: Checking part: '%s'\n", part) // DEBUG
		// Skip empty parts that might result from multiple separators
		if part == "" {
			// fmt.Println("DEBUG: Skipping empty part") // DEBUG
			continue
		}
		if langInfo, ok := languagesDB[part]; ok {
			// fmt.Printf("DEBUG: Found match for '%s': %v\n", part, langInfo) // DEBUG
			return langInfo.OSCode // Return the specific OpenSubtitles code
		}
	}

	// fmt.Println("DEBUG: No language detected") // DEBUG
	// TODO: Consider adding more sophisticated checks (e.g., multi-word names like "Portuguese Brazilian")
	// TODO: Consider using a text analysis library (like whatlanggo) as a fallback?

	return "" // Language not detected
}

// AnalyzeSubtitleFlags checks the filename for common patterns indicating
// Hearing Impaired (HI/SDH) or Forced subtitles.
func AnalyzeSubtitleFlags(filename string) (isHI bool, isForced bool) {
	lowerFilename := strings.ToLower(filename)
	base := strings.TrimSuffix(lowerFilename, filepath.Ext(lowerFilename))

	// Check for common separators and terms
	// Use a set for efficient lookup of flag terms
	hiTerms := map[string]struct{}{"hi": {}, "sdh": {}, "hearingimpaired": {}}
	forcedTerms := map[string]struct{}{"forced": {}, "frc": {}}

	parts := strings.FieldsFunc(base, func(r rune) bool {
		// Split by common separators like ., _, -, space
		return r == '.' || r == '_' || r == '-' || r == ' '
	})

	for _, part := range parts {
		if _, ok := hiTerms[part]; ok {
			isHI = true
		}
		if _, ok := forcedTerms[part]; ok {
			isForced = true
		}
		// Optimization: if both found, no need to check further
		if isHI && isForced {
			break
		}
	}

	return isHI, isForced
}

// normalizeFilenameForMatching removes extensions, language codes, flags, and common tags
// to get a comparable base filename for matching videos and subtitles.
func normalizeFilenameForMatching(filename string) string {
	// Remove extension
	ext := filepath.Ext(filename)
	base := filename[:len(filename)-len(ext)]

	// Convert to lowercase and replace common separators with spaces
	normalized := strings.ToLower(base)
	normalized = strings.ReplaceAll(normalized, ".", " ")
	normalized = strings.ReplaceAll(normalized, "_", " ")
	normalized = strings.ReplaceAll(normalized, "-", " ")

	// Remove known flags (add more as needed)
	flags := []string{"sdh", "hi", "hearingimpaired", "forced", "frc"}
	for _, flag := range flags {
		normalized = strings.ReplaceAll(normalized, " "+flag+" ", " ") // Surrounded by spaces
		normalized = strings.TrimSuffix(normalized, " "+flag)          // At the end
	}

	// Remove language codes (check all variations from languagesDB)
	// We split by space and check the last part(s)
	parts := strings.Fields(normalized)
	if len(parts) > 0 {
		possibleLangKey := parts[len(parts)-1]
		if _, isLang := languagesDB[possibleLangKey]; isLang {
			normalized = strings.Join(parts[:len(parts)-1], " ")
			parts = parts[:len(parts)-1] // Update parts for potential multi-word lang (like pt br)
		}
	}
	// Handle multi-word codes like pt-br (now pt br)
	if len(parts) > 1 {
		possibleLangKey := parts[len(parts)-2] + " " + parts[len(parts)-1]
		if _, isLang := languagesDB[strings.ReplaceAll(possibleLangKey, " ", "-")]; isLang { // check original format
			normalized = strings.Join(parts[:len(parts)-2], " ")
		}
	}

	// Remove common quality/source/codec/release tags (add more as needed)
	// Order matters slightly - remove multi-word tags first if they might contain single-word ones
	commonTags := []string{
		"directors cut", "extended cut", // Multi-word first
		"web dl", "webrip", "webrip", "web cap", "web", "hdtv", "hdrip", "bdrip", "brrip", "bluray", "dvdrip", "dvdr", // Sources
		"1080p", "720p", "2160p", "4k", "uhd", "sd", // Resolutions
		"x264", "h264", "x265", "h265", "hevc", // Video Codecs
		"aac", "ac3", "eac3", "dts", "truehd", // Audio Codecs
		"remux", "repack", "proper", "internal", "limited", "extended", "uncut", // Release types (single word)
		// Add common release group patterns? Maybe too complex/risky.
	}
	// Sort tags by length descending to ensure longer tags are removed first
	sort.Slice(commonTags, func(i, j int) bool {
		return len(commonTags[i]) > len(commonTags[j])
	})

	// Iterate multiple times? Simple replace might leave parts if tags are adjacent.
	for i := 0; i < 2; i++ { // Run twice for overlapping removals
		for _, tag := range commonTags {
			// Replace tag surrounded by spaces
			normalized = strings.ReplaceAll(normalized, " "+tag+" ", " ")
			// Replace tag at the beginning of the string followed by a space
			if strings.HasPrefix(normalized, tag+" ") {
				normalized = normalized[len(tag)+1:]
			}
			// Replace tag at the end of the string preceded by a space
			if strings.HasSuffix(normalized, " "+tag) {
				normalized = normalized[:len(normalized)-len(tag)-1]
			}
			// Replace tag if it's the only thing left (handles cases like "1080p.mkv")
			if normalized == tag {
				normalized = ""
			}
		}
		// Consolidate whitespace again after replaces
		normalized = strings.Join(strings.Fields(normalized), " ")
	}

	// Remove any remaining single characters or digits that might be leftovers
	parts = strings.Fields(normalized)
	filteredParts := []string{}
	for _, part := range parts {
		if len(part) > 1 || (len(part) == 1 && !unicode.IsDigit(rune(part[0]))) {
			filteredParts = append(filteredParts, part)
		}
	}
	normalized = strings.Join(filteredParts, " ")

	// Debug log
	// fmt.Printf("Normalized '%s' -> '%s'\n", filename, normalized)

	return normalized
}

// MatchVideoSubtitle attempts to determine if a video file and subtitle file
// correspond to the same media content based on their filenames.
func MatchVideoSubtitle(videoFilename, subtitleFilename string) bool {
	if videoFilename == "" || subtitleFilename == "" {
		return false
	}

	normVideo := normalizeFilenameForMatching(videoFilename)
	normSub := normalizeFilenameForMatching(subtitleFilename)

	// Debug log
	// fmt.Printf("Comparing Video: '%s' with Sub: '%s'\n", normVideo, normSub)

	// Basic check: if either normalization failed or resulted in empty, no match
	if normVideo == "" || normSub == "" {
		return false
	}

	// The core logic: Check if the normalized names are equal.
	// This ensures a strict match after removing common variations.
	return normVideo == normSub

	/* // Old Prefix logic:
	return strings.HasPrefix(normVideo, normSub) || strings.HasPrefix(normSub, normVideo)
	*/

	/* // --- Old ptn-based logic (removed) ---
	// ... existing code ...
	*/
}

// FindMatchingSubtitle iterates through a list of subtitle filenames and returns the first
// one that matches the given video filename according to MatchVideoSubtitle.
func FindMatchingSubtitle(videoFilename string, subtitleFilenames []string) string {
	videoBase := filepath.Base(videoFilename) // Use only the filename part
	for _, subPath := range subtitleFilenames {
		subBase := filepath.Base(subPath)
		if MatchVideoSubtitle(videoBase, subBase) {
			return subPath // Return the full path of the matching subtitle
		}
	}
	return ""
}

// FindMatchingVideo takes a subtitle filename and a list of video filenames,
// returning the first video filename that matches according to MatchVideoSubtitle.
// Returns an empty string if no match is found.
func FindMatchingVideo(subtitleFilename string, videoFilenames []string) string {
	subBase := filepath.Base(subtitleFilename)
	for _, videoPath := range videoFilenames {
		videoBase := filepath.Base(videoPath)
		if MatchVideoSubtitle(videoBase, subBase) {
			return videoPath // Return the full path of the matching video
		}
	}
	return ""
}

// --- Metadata Consolidation ---

// APIClientProvider bundles the API clients needed by the metadata package.
type APIClientProvider struct {
	OSClient    OpenSubtitlesClient
	TraktClient TraktClient
	IMDbClient  IMDbClient
}

// ConsolidateMetadata gathers information from filename, file system, NFO, and APIs.
func ConsolidateMetadata(ctx context.Context, videoPath, subtitlePath string, clients APIClientProvider) (*VideoInfo, *SubtitleInfo, error) {
	// ... (Initial videoInfo/subInfo population from files remains the same) ...
	videoInfo := &VideoInfo{FilePath: videoPath}
	subInfo := &SubtitleInfo{FilePath: subtitlePath}

	// Populate from subtitle file
	if subtitlePath != "" {
		subStat, err := os.Stat(subtitlePath)
		if err == nil {
			subInfo.FileName = subStat.Name()
			subInfo.FileSize = subStat.Size()
			subInfo.Format = strings.TrimPrefix(strings.ToLower(filepath.Ext(subInfo.FileName)), ".")
			subInfo.Language = DetectSubtitleLanguage(subInfo.FileName)
			subInfo.IsForHearingImpaired, subInfo.IsForced = AnalyzeSubtitleFlags(subInfo.FileName)
			subInfo.MD5Hash, _ = fileops.CalculateMD5Hash(subtitlePath)
		} else {
			log.Warnf("Failed to stat subtitle file %s: %v", subtitlePath, err)
		}
	}

	// Populate from video file
	if videoPath != "" {
		videoStat, err := os.Stat(videoPath)
		if err == nil {
			videoInfo.FileName = videoStat.Name()
			videoInfo.FileSize = videoStat.Size()
			parsed, errPtn := ptn.Parse(videoInfo.FileName)
			if errPtn == nil {
				videoInfo.Title = parsed.Title
				videoInfo.Year = parsed.Year
				videoInfo.Season = parsed.Season
				videoInfo.Episode = parsed.Episode
				videoInfo.Resolution = parsed.Resolution
				videoInfo.Source = parsed.Quality // Assuming ptn uses Quality for source
				videoInfo.ReleaseGroup = parsed.Group
			} else {
				log.Warnf("Failed to parse video filename '%s': %v", videoInfo.FileName, errPtn)
				baseName := strings.TrimSuffix(videoInfo.FileName, filepath.Ext(videoInfo.FileName))
				videoInfo.Title = strings.ReplaceAll(baseName, ".", " ")
			}
			videoInfo.OSDbHash, _ = fileops.CalculateOSDbHash(videoPath)
			nfoPath := strings.TrimSuffix(videoPath, filepath.Ext(videoPath)) + ".nfo"
			// Pass context to ReadNFO
			// Also, ReadNFO likely returns error if file not found, handle it.
			imdbIDFromNfo, errNfo := fileops.ReadNFO(ctx, nfoPath)
			if errNfo != nil && !os.IsNotExist(errNfo) {
				log.Warnf("Error reading NFO file %s: %v", nfoPath, errNfo)
			} else if errNfo == nil {
				videoInfo.NFO_IMDbID = imdbIDFromNfo
			}
		} else {
			log.Warnf("Failed to stat video file %s: %v", videoPath, err)
		}
	}

	// Fetch additional metadata from APIs
	getMetadataFromAPIs(ctx, videoInfo, clients)

	// Return consolidated info (error handling might need refinement)
	return videoInfo, subInfo, nil
}

// getMetadataFromAPIs fetches data from external APIs based on available info.
func getMetadataFromAPIs(ctx context.Context, videoInfo *VideoInfo, clients APIClientProvider) {
	var primaryIMDbID string = videoInfo.NFO_IMDbID // Start with NFO ID

	// 1. OpenSubtitles Search by Hash
	if clients.OSClient != nil && videoInfo.OSDbHash != "" && primaryIMDbID == "" {
		params := map[string]string{"hash": videoInfo.OSDbHash}
		features, err := clients.OSClient.SearchFeatures(ctx, params)
		if err == nil && features != nil && len(features.Data) > 0 {
			feature := features.Data[0]
			// Check if ImdbID (int) is non-zero and format it
			if feature.Attributes.ImdbID != 0 {
				imdbIDStr := fmt.Sprintf("tt%d", feature.Attributes.ImdbID)
				log.Infof("Found IMDb ID (%s) from OSDb hash for %s", imdbIDStr, videoInfo.FileName)
				primaryIMDbID = imdbIDStr // Update primary ID (string)
				videoInfo.OSDb_IMDbID = primaryIMDbID
			}
		} else if err != nil {
			log.Warnf("OSDb SearchFeatures by hash failed for %s: %v", videoInfo.FileName, err)
		}
	}

	// 2. Trakt Search (if Trakt client available)
	if clients.TraktClient != nil {
		var errTrakt error
		var searchResults []trakt.SearchResult
		searchQuery := ""
		searchType := ""
		idType := "" // For Trakt API: imdb, tmdb, etc.

		// Prioritize searching by existing IMDb ID (from NFO or OSDb)
		if primaryIMDbID != "" {
			searchQuery = primaryIMDbID
			idType = "imdb"
			// Trakt queryType isn't needed when searching by ID
			log.Infof("Searching Trakt by IMDb ID %s for %s", searchQuery, videoInfo.FileName)
			searchResults, errTrakt = clients.TraktClient.SearchTrakt(ctx, idType, searchQuery) // Use idType for queryType when searching by ID
			if errTrakt == nil && len(searchResults) > 0 {
				handleTraktResults(videoInfo, searchResults, &primaryIMDbID)
			} else if errTrakt != nil {
				log.Warnf("Trakt SearchTrakt by IMDb ID failed for %s: %v", videoInfo.FileName, errTrakt)
			}
		} else if videoInfo.Title != "" {
			// Fallback to searching by Title (+ Year if available)
			searchQuery = videoInfo.Title
			if videoInfo.Year > 0 {
				searchQuery = fmt.Sprintf("%s %d", videoInfo.Title, videoInfo.Year)
			}
			if videoInfo.Season > 0 {
				searchType = "show,episode" // Search shows/episodes if season is present
			} else {
				searchType = "movie,show" // Default to movie/show search
			}
			idType = "" // No specific ID type for text search
			log.Infof("Searching Trakt by Title '%s' (type: %s) for %s", searchQuery, searchType, videoInfo.FileName)
			searchResults, errTrakt = clients.TraktClient.SearchTrakt(ctx, searchType, searchQuery)
			if errTrakt == nil && len(searchResults) > 0 {
				handleTraktResults(videoInfo, searchResults, &primaryIMDbID)
			} else if errTrakt != nil {
				log.Warnf("Trakt SearchTrakt by title failed for %s: %v", videoInfo.FileName, errTrakt)
			}
		}
	}

	// 3. IMDb Suggest Search (if client available and STILL no IMDb ID)
	if clients.IMDbClient != nil && primaryIMDbID == "" && videoInfo.Title != "" {
		log.Infof("Searching IMDb suggestions for '%s'", videoInfo.Title)
		results, errIMDb := clients.IMDbClient.SearchIMDBSuggestions(ctx, videoInfo.Title)
		if errIMDb == nil && len(results) > 0 {
			// Logic to select best IMDb result
			bestMatchScore := -1
			bestMatchID := ""
			for _, res := range results {
				score := 0
				// Basic scoring: +1 for year match, +1 if ID looks valid
				if res.Year == videoInfo.Year && videoInfo.Year != 0 {
					score++
				}
				if strings.HasPrefix(res.ID, "tt") { // Check ID prefix
					score++
				}
				// Removed check for res.Type
				if score > bestMatchScore {
					bestMatchScore = score
					bestMatchID = res.ID
				}
			}
			if bestMatchID != "" {
				log.Infof("Found potential IMDb ID (%s) from IMDb Suggest for %s", bestMatchID, videoInfo.FileName)
				primaryIMDbID = bestMatchID
			}
		} else if errIMDb != nil {
			// Logged as WARN within the IMDb client itself due to instability
			// log.Warnf("IMDb SearchSuggest failed for %s: %v", videoInfo.Title, errIMDb)
		}
	}

	// Final assignment: Ensure OSDb_IMDbID holds the best determined ID
	// If NFO had an ID, it took precedence initially.
	// If OSDb or Trakt or IMDb found one later, primaryIMDbID was updated.
	videoInfo.OSDb_IMDbID = primaryIMDbID
	log.Infof("Final determined primary IMDb ID for %s: %s", videoInfo.FileName, videoInfo.OSDb_IMDbID)
}

// handleTraktResults processes the search results from Trakt.
// Accepts a pointer to primaryIMDbID so it can be updated if found via Trakt.
func handleTraktResults(videoInfo *VideoInfo, searchResults []trakt.SearchResult, primaryIMDbID *string) {
	// Logic to select the best Trakt result
	// Simplistic: Check first result that matches basic info (type, year)
	for _, res := range searchResults {
		match := false
		var traktID, imdbID string
		if ids, ok := res.IDs["trakt"]; ok {
			traktID = ids
		}
		if ids, ok := res.IDs["imdb"]; ok {
			imdbID = ids
		}

		if videoInfo.Season > 0 && (res.Type == "show" || res.Type == "episode") {
			// If we are looking for a show/episode, check type and maybe year
			if videoInfo.Year == 0 || res.Year == videoInfo.Year {
				match = true
			}
		} else if videoInfo.Season == 0 && res.Type == "movie" {
			// If we are looking for a movie, check type and maybe year
			if videoInfo.Year == 0 || res.Year == videoInfo.Year {
				match = true
			}
		}

		if match {
			log.Infof("Found Trakt match for %s: Type=%s, Title='%s', Year=%d, TraktID=%s, IMDbID=%s",
				videoInfo.FileName, res.Type, res.Title, res.Year, traktID, imdbID)
			if videoInfo.TraktID == "" && traktID != "" {
				videoInfo.TraktID = traktID
			}
			// If we didn't have a primary IMDb ID yet, use the one from Trakt
			if *primaryIMDbID == "" && imdbID != "" {
				*primaryIMDbID = imdbID
				log.Infof("Using IMDb ID (%s) found via Trakt for %s", imdbID, videoInfo.FileName)
			}
			return // Stop after finding the first plausible match
		}
	}
	log.Warnf("No suitable Trakt match found for %s among %d results.", videoInfo.FileName, len(searchResults))
}
