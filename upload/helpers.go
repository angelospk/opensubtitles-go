package upload

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
)

// UserUploadIntent holds all the data provided by the user or derived
// before preparing the specific XML-RPC calls.
// This acts as an intermediary structure.
type UserUploadIntent struct {
	VideoFilePath        string // Path to the video file
	SubtitleFilePath     string // Path to the subtitle file
	IMDBID               string // e.g., "tt1234567" or "1234567"
	LanguageID           string // e.g., "eng"
	VideoFileName        string // Basename of the video file
	SubtitleFileName     string // Basename of the subtitle file
	ReleaseName          string
	MovieAka             string
	FPS                  float64
	Frames               int64
	TimeMS               int64
	Comment              string
	Translator           string
	HighDefinition       bool
	HearingImpaired      bool
	AutomaticTranslation bool
	ForeignPartsOnly     bool
}

// boolToXmlRpc converts a boolean to the "1" or "0" string expected by XML-RPC.
func boolToXmlRpc(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// PrepareTryUploadParams creates the parameters needed for the TryUploadSubtitles XML-RPC call
// based on user intent.
func PrepareTryUploadParams(intent UserUploadIntent) (XmlRpcTryUploadParams, error) {
	params := XmlRpcTryUploadParams{
		CDs: make(map[string]XmlRpcTryUploadFileItem),
	}

	// --- Populate global optional parameters for TryUpload ---
	if intent.IMDBID != "" {
		imdbid := intent.IMDBID
		if len(imdbid) > 2 && imdbid[:2] == "tt" {
			imdbid = imdbid[2:]
		}
		params.IDMovieImdb = imdbid
	}
	if intent.LanguageID != "" {
		params.SubLanguageID = intent.LanguageID
	}
	if intent.Comment != "" {
		params.SubAuthorComment = intent.Comment
	}
	if intent.Translator != "" {
		params.SubTranslator = intent.Translator
	}
	if intent.ReleaseName != "" {
		params.MovieReleaseName = intent.ReleaseName
	}
	if intent.MovieAka != "" {
		params.MovieAka = intent.MovieAka
	}
	params.HearingImpaired = boolToXmlRpc(intent.HearingImpaired)
	params.HighDefinition = boolToXmlRpc(intent.HighDefinition)
	params.AutomaticTranslation = boolToXmlRpc(intent.AutomaticTranslation)
	params.ForeignPartsOnly = boolToXmlRpc(intent.ForeignPartsOnly)

	// --- Populate per-file item for "cd1" ---
	fileItem := XmlRpcTryUploadFileItem{}

	// Subtitle Hash & Filename (Mandatory for TryUpload file item)
	if intent.SubtitleFilePath == "" {
		return params, fmt.Errorf("subtitle file path is required")
	}
	subHash, err := CalculateMD5Hash(intent.SubtitleFilePath)
	if err != nil {
		return params, fmt.Errorf("failed to calculate MD5 hash for subtitle: %w", err)
	}
	fileItem.SubHash = subHash
	fileItem.SubFilename = intent.SubtitleFileName // Assume already set
	if fileItem.SubFilename == "" {
		return params, fmt.Errorf("subtitle filename is required")
	}

	// Video Hash & Filename (Mandatory for TryUpload file item if video present)
	// Plus other video-specific fields
	if intent.VideoFilePath != "" {
		movieHash, movieSize, err := CalculateOSDbHash(intent.VideoFilePath)
		if err != nil {
			return params, fmt.Errorf("failed to calculate OSDb hash for video: %w", err)
		}
		fileItem.MovieHash = movieHash
		fileItem.MovieByteSize = strconv.FormatInt(movieSize, 10) // Kept as string for TryUpload
		fileItem.MovieFilename = intent.VideoFileName             // Assume already set
		if fileItem.MovieFilename == "" {
			return params, fmt.Errorf("video filename is required if video file is provided")
		}
	} else {
		// If no video file, MovieHash, MovieByteSize, MovieFilename might be empty or omitted.
		// The API docs state these are mandatory for the subfile struct in TryUpload.
		// This might require clarification if uploading without a video file is intended for TryUpload.
		// For now, we require LanguageID and IMDBID at the global level as per original logic.
		if intent.LanguageID == "" { // This check is now on params.SubLanguageID
			return params, fmt.Errorf("language ID is required if no video file is provided")
		}
		if intent.IMDBID == "" { // This check is now on params.IDMovieImdb
			return params, fmt.Errorf("IMDB ID is required if no video file is provided")
		}
	}

	// Per-file optional fields from intent
	if intent.FPS > 0 {
		fileItem.MovieFPS = fmt.Sprintf("%.3f", intent.FPS) // Kept as string for TryUpload
	}
	if intent.TimeMS > 0 {
		fileItem.MovieTimeMS = strconv.FormatInt(intent.TimeMS, 10) // Kept as string for TryUpload
	}
	// MovieFrames is int in docs, let's convert if available
	if intent.Frames > 0 {
		fileItem.MovieFrames = strconv.FormatInt(intent.Frames, 10) // Kept as string for TryUpload
	}

	params.CDs["cd1"] = fileItem
	return params, nil
}

// readAndEncodeSubtitle reads the subtitle file, GZips it, and returns its Base64 encoded content.
// UPDATE: Removing Gzip step based on server developer feedback - trying only Base64.
func ReadAndEncodeSubtitle(filePath string) (encodedContent string, subHash string, err error) {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read subtitle file content '%s': %w", filePath, err)
	}

	// GZip the content - REMOVED
	// var gzipBuffer bytes.Buffer
	// gzipWriter := gzip.NewWriter(&gzipBuffer)
	// _, err = gzipWriter.Write(contentBytes)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to gzip subtitle content: %w", err)
	// }
	// err = gzipWriter.Close() // Close is important to finalize compression
	// if err != nil {
	// 	return "", fmt.Errorf("failed to close gzip writer: %w", err)
	// }

	// Base64 encode the *raw* content
	encodedContent = base64.StdEncoding.EncodeToString(contentBytes)

	// Calculate the MD5 hash of the content
	subHash, err = CalculateMD5Hash(filePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to calculate MD5 hash for subtitle: %w", err)
	}

	return encodedContent, subHash, nil
}

// CalculateSubHash calculates the MD5 hash of a file, returning the hex string.
// This is the same as CalculateMD5Hash, just renamed for clarity in context.
func CalculateSubHash(filePath string) (string, error) {
	return CalculateMD5Hash(filePath)
}

// PrepareUploadSubtitlesParams prepares the parameters for the final UploadSubtitles XML-RPC call.
func PrepareUploadSubtitlesParams(tryParams XmlRpcTryUploadParams, subtitlePath string) (XmlRpcUploadSubtitlesParams, error) {

	base64Content, calculatedSubHash, err := ReadAndEncodeSubtitle(subtitlePath)
	if err != nil {
		return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("failed to read and encode subtitle for upload: %w", err)
	}
	if base64Content == "" {
		return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("base64 subtitle content cannot be empty")
	}

	// Assuming we are working with "cd1" from tryParams for this simplified example
	// In a multi-file scenario, this would need to iterate or select a specific CD.
	cd1TryInfo, ok := tryParams.CDs["cd1"]
	if !ok {
		return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("cd1 data not found in TryUploadParams")
	}

	// Parse string fields from cd1TryInfo to their correct types (float64, int) for UploadSubtitles
	var movieByteSize float64
	if cd1TryInfo.MovieByteSize != "" {
		movieByteSize, err = strconv.ParseFloat(cd1TryInfo.MovieByteSize, 64)
		if err != nil {
			return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("failed to parse MovieByteSize '%s': %w", cd1TryInfo.MovieByteSize, err)
		}
	}

	var movieFPS float64
	if cd1TryInfo.MovieFPS != "" {
		movieFPS, err = strconv.ParseFloat(cd1TryInfo.MovieFPS, 64)
		if err != nil {
			return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("failed to parse MovieFPS '%s': %w", cd1TryInfo.MovieFPS, err)
		}
	}

	var movieTimeMS int
	if cd1TryInfo.MovieTimeMS != "" {
		movieTimeMS64, errConv := strconv.ParseInt(cd1TryInfo.MovieTimeMS, 10, 64)
		if errConv != nil {
			return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("failed to parse MovieTimeMS '%s': %w", cd1TryInfo.MovieTimeMS, errConv)
		}
		movieTimeMS = int(movieTimeMS64)
	}

	var movieFrames int
	if cd1TryInfo.MovieFrames != "" {
		movieFrames64, errConv := strconv.ParseInt(cd1TryInfo.MovieFrames, 10, 64)
		if errConv != nil {
			return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("failed to parse MovieFrames '%s': %w", cd1TryInfo.MovieFrames, errConv)
		}
		movieFrames = int(movieFrames64)
	}

	// Build the final structure
	params := XmlRpcUploadSubtitlesParams{
		BaseInfo: XmlRpcUploadSubtitlesBaseInfo{
			IDMovieImdb:      tryParams.IDMovieImdb, // Reuse global info from tryParams
			SubLanguageID:    tryParams.SubLanguageID,
			MovieReleaseName: tryParams.MovieReleaseName,
			MovieAka:         tryParams.MovieAka,
			SubAuthorComment: tryParams.SubAuthorComment,
			SubTranslator:    tryParams.SubTranslator,
			HearingImpaired:  tryParams.HearingImpaired,
			HighDefinition:   tryParams.HighDefinition,
			ForeignPartsOnly: tryParams.ForeignPartsOnly,
			// SubTranslator, HearingImpaired, etc. are intentionally omitted as per UploadSubtitles baseinfo spec
		},
		CDs: map[string]XmlRpcUploadSubtitlesCD{
			"cd1": {
				SubHash:       calculatedSubHash,      // Use freshly calculated hash of the content being uploaded
				SubFilename:   cd1TryInfo.SubFilename, // Reuse filename from tryParams.CDs["cd1"]
				MovieHash:     cd1TryInfo.MovieHash,
				MovieByteSize: movieByteSize, // Parsed to float64
				MovieTimeMS:   movieTimeMS,   // Parsed to int
				MovieFPS:      movieFPS,      // Parsed to float64
				MovieFrames:   movieFrames,   // Parsed to int
				MovieFilename: cd1TryInfo.MovieFilename,
				SubContent:    base64Content,
			},
		},
	}

	return params, nil
}

// --- Struct Definitions (Internal to upload package) ---

// XmlRpcTryUploadFileItem holds the per-file parameters for the TryUploadSubtitles call.
type XmlRpcTryUploadFileItem struct {
	SubHash       string `xmlrpc:"subhash"`                 // Mandatory
	SubFilename   string `xmlrpc:"subfilename"`             // Mandatory
	MovieHash     string `xmlrpc:"moviehash,omitempty"`     // Mandatory
	MovieByteSize string `xmlrpc:"moviebytesize,omitempty"` // Mandatory, string in API (doc: "string double")
	MovieTimeMS   string `xmlrpc:"movietimems,omitempty"`   // Optional, string in API (doc: "int")
	MovieFrames   string `xmlrpc:"movieframes,omitempty"`   // Optional, string in API (doc: "int")
	MovieFPS      string `xmlrpc:"moviefps,omitempty"`      // Optional, string in API (doc: "double")
	MovieFilename string `xmlrpc:"moviefilename,omitempty"` // Mandatory
}

// XmlRpcTryUploadParams holds parameters for the TryUploadSubtitles call.
// This struct represents the second argument (data) passed to the XML-RPC method.
type XmlRpcTryUploadParams struct {
	// Global optional parameters for the TryUpload request
	IDMovieImdb          string `xmlrpc:"idmovieimdb,omitempty"`
	SubLanguageID        string `xmlrpc:"sublanguageid,omitempty"`
	SubAuthorComment     string `xmlrpc:"subauthorcomment,omitempty"`
	SubTranslator        string `xmlrpc:"subtranslator,omitempty"`
	MovieReleaseName     string `xmlrpc:"moviereleasename,omitempty"`
	MovieAka             string `xmlrpc:"movieaka,omitempty"`
	HearingImpaired      string `xmlrpc:"hearingimpaired,omitempty"`      // "0" or "1"
	HighDefinition       string `xmlrpc:"highdefinition,omitempty"`       // "0" or "1"
	AutomaticTranslation string `xmlrpc:"automatictranslation,omitempty"` // "0" or "1"
	ForeignPartsOnly     string `xmlrpc:"foreignpartsonly,omitempty"`     // "0" or "1"

	// Map for cd1, cd2, etc. The xmlrpc:",inline" tag merges these into the top-level struct.
	CDs map[string]XmlRpcTryUploadFileItem `xmlrpc:",inline"`
}

// XmlRpcUploadSubtitlesParams is the top-level structure for the UploadSubtitles call.
type XmlRpcUploadSubtitlesParams struct {
	BaseInfo XmlRpcUploadSubtitlesBaseInfo      `xmlrpc:"baseinfo"`
	CDs      map[string]XmlRpcUploadSubtitlesCD `xmlrpc:",inline"` // Map "cd1", "cd2" etc. to CD data
}

// XmlRpcUploadSubtitlesBaseInfo holds the 'baseinfo' part for UploadSubtitles.
type XmlRpcUploadSubtitlesBaseInfo struct {
	IDMovieImdb      string `xmlrpc:"idmovieimdb,omitempty"`
	SubLanguageID    string `xmlrpc:"sublanguageid,omitempty"`
	MovieReleaseName string `xmlrpc:"moviereleasename,omitempty"`
	MovieAka         string `xmlrpc:"movieaka,omitempty"`
	SubAuthorComment string `xmlrpc:"subauthorcomment,omitempty"`
	SubTranslator    string `xmlrpc:"subtranslator,omitempty"`
	HearingImpaired  string `xmlrpc:"hearingimpaired,omitempty"`
	HighDefinition   string `xmlrpc:"highdefinition,omitempty"`
	ForeignPartsOnly string `xmlrpc:"foreignpartsonly,omitempty"`
}

// XmlRpcUploadSubtitlesCD holds the 'cdX' data for UploadSubtitles.
type XmlRpcUploadSubtitlesCD struct {
	SubHash       string  `xmlrpc:"subhash"`
	SubFilename   string  `xmlrpc:"subfilename"`
	MovieHash     string  `xmlrpc:"moviehash,omitempty"`     // Mandatory
	MovieByteSize float64 `xmlrpc:"moviebytesize,omitempty"` // Mandatory, double
	SubContent    string  `xmlrpc:"subcontent"`              // Base64 encoded content
	MovieTimeMS   int     `xmlrpc:"movietimems,omitempty"`   // Optional, int
	MovieFrames   int     `xmlrpc:"movieframes,omitempty"`   // Optional, int
	MovieFPS      float64 `xmlrpc:"moviefps,omitempty"`      // Optional, double
	MovieFilename string  `xmlrpc:"moviefilename,omitempty"` // Mandatory
}

// --- Helper Functions ---
