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
	params := XmlRpcTryUploadParams{}

	// Subtitle Hash & Filename (Mandatory for TryUpload)
	if intent.SubtitleFilePath == "" {
		return params, fmt.Errorf("subtitle file path is required")
	}
	subHash, err := CalculateMD5Hash(intent.SubtitleFilePath)
	if err != nil {
		return params, fmt.Errorf("failed to calculate MD5 hash for subtitle: %w", err)
	}
	params.SubHash = subHash
	params.SubFilename = intent.SubtitleFileName // Assume already set
	if params.SubFilename == "" {
		return params, fmt.Errorf("subtitle filename is required")
	}

	// Video Hash & Filename (Optional for TryUpload)
	if intent.VideoFilePath != "" {
		movieHash, movieSize, err := CalculateOSDbHash(intent.VideoFilePath)
		if err != nil {
			return params, fmt.Errorf("failed to calculate OSDb hash for video: %w", err)
		}
		params.MovieHash = movieHash
		params.MovieByteSize = strconv.FormatInt(movieSize, 10)
		params.MovieFilename = intent.VideoFileName // Assume already set
		if params.MovieFilename == "" {
			return params, fmt.Errorf("video filename is required if video file is provided")
		}
	} else {
		// If no video, require both LanguageID and IMDBID for best results
		if intent.LanguageID == "" {
			return params, fmt.Errorf("language ID is required if no video file is provided")
		}
		if intent.IMDBID == "" {
			return params, fmt.Errorf("IMDB ID is required if no video file is provided")
		}
	}

	// Optional fields from intent (mapping names)
	if intent.IMDBID != "" {
		// Remove "tt" prefix if present
		imdbid := intent.IMDBID
		if len(imdbid) > 2 && imdbid[:2] == "tt" {
			imdbid = imdbid[2:]
		}
		params.IDMovieImdb = imdbid
	}
	if intent.LanguageID != "" {
		params.SubLanguageID = intent.LanguageID
	}
	if intent.FPS > 0 {
		params.MovieFPS = fmt.Sprintf("%.3f", intent.FPS)
	}
	if intent.TimeMS > 0 {
		params.MovieTimeMS = strconv.FormatInt(intent.TimeMS, 10)
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
	// Calculate subtitle hash (reuse from tryParams if possible? No, needs fresh calc)
	subHash, err := CalculateSubHash(subtitlePath)
	if err != nil {
		return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("failed to calculate subhash: %w", err)
	}

	base64Content, subHash, err := ReadAndEncodeSubtitle(subtitlePath)
	if err != nil {
		return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("failed to read and encode subtitle for upload: %w", err)
	}
	if base64Content == "" {
		return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("base64 subtitle content cannot be empty")
	}

	// Parse string fields to their correct types (float64, int)
	var movieByteSize float64
	if tryParams.MovieByteSize != "" {
		movieByteSize, err = strconv.ParseFloat(tryParams.MovieByteSize, 64)
		if err != nil {
			return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("failed to parse MovieByteSize '%s': %w", tryParams.MovieByteSize, err)
		}
	}

	var movieFPS float64
	if tryParams.MovieFPS != "" {
		movieFPS, err = strconv.ParseFloat(tryParams.MovieFPS, 64)
		if err != nil {
			return XmlRpcUploadSubtitlesParams{}, fmt.Errorf("failed to parse MovieFPS '%s': %w", tryParams.MovieFPS, err)
		}
	}

	var movieTimeMS string
	if tryParams.MovieTimeMS != "" {
		// Keep as string
		movieTimeMS = tryParams.MovieTimeMS
		// movieTimeMS64, err := strconv.ParseInt(tryParams.MovieTimeMS, 10, 64)
		// if err != nil {
		// 	 return xmlRpcUploadSubtitlesParams{}, fmt.Errorf("failed to parse MovieTimeMS '%s': %w", tryParams.MovieTimeMS, err)
		// }
		// movieTimeMS = int(movieTimeMS64) // Convert to int - Not needed, keep string
	}

	// Build the final structure
	params := XmlRpcUploadSubtitlesParams{
		BaseInfo: XmlRpcUploadSubtitlesBaseInfo{
			IDMovieImdb:      tryParams.IDMovieImdb, // Reuse relevant info from tryParams
			SubLanguageID:    tryParams.SubLanguageID,
			MovieReleaseName: tryParams.MovieReleaseName,
			MovieAka:         tryParams.MovieAka,
			SubAuthorComment: tryParams.SubAuthorComment,
		},
		CDs: map[string]XmlRpcUploadSubtitlesCD{
			"cd1": {
				SubHash:       subHash,               // Calculated hash
				SubFilename:   tryParams.SubFilename, // Reuse filename
				MovieHash:     tryParams.MovieHash,
				MovieByteSize: strconv.FormatFloat(movieByteSize, 'f', -1, 64), // Keep string
				MovieTimeMS:   movieTimeMS,                                     // Keep string
				MovieFPS:      strconv.FormatFloat(movieFPS, 'f', -1, 64),      // Keep string
				SubContent:    base64Content,
			},
		},
	}

	return params, nil
}

// --- Struct Definitions (Internal to upload package) ---

// XmlRpcTryUploadParams holds parameters for the TryUploadSubtitles call.
// Based on usage in tryUploadSubtitles and PrepareTryUploadParams.
type XmlRpcTryUploadParams struct {
	SubHash              string `xmlrpc:"subhash"`
	SubFilename          string `xmlrpc:"subfilename"`
	MovieHash            string `xmlrpc:"moviehash"`
	MovieByteSize        string `xmlrpc:"moviebytesize"` // String in API
	MovieFilename        string `xmlrpc:"moviefilename"`
	IDMovieImdb          string `xmlrpc:"idmovieimdb,omitempty"` // String in API
	SubLanguageID        string `xmlrpc:"sublanguageid,omitempty"`
	MovieFPS             string `xmlrpc:"moviefps,omitempty"`    // String in API
	MovieTimeMS          string `xmlrpc:"movietimems,omitempty"` // String in API
	SubAuthorComment     string `xmlrpc:"subauthorcomment,omitempty"`
	SubTranslator        string `xmlrpc:"subtranslator,omitempty"`
	MovieReleaseName     string `xmlrpc:"moviereleasename,omitempty"`
	MovieAka             string `xmlrpc:"movieaka,omitempty"`
	HearingImpaired      string `xmlrpc:"hearingimpaired,omitempty"`      // "0" or "1"
	HighDefinition       string `xmlrpc:"highdefinition,omitempty"`       // "0" or "1"
	AutomaticTranslation string `xmlrpc:"automatictranslation,omitempty"` // "0" or "1"
	ForeignPartsOnly     string `xmlrpc:"foreignpartsonly,omitempty"`     // "0" or "1"
}

// XmlRpcUploadSubtitlesBaseInfo holds the 'baseinfo' part for UploadSubtitles.
// Based on PrepareUploadSubtitlesParams.
type XmlRpcUploadSubtitlesBaseInfo struct {
	IDMovieImdb      string `xmlrpc:"idmovieimdb,omitempty"`
	SubLanguageID    string `xmlrpc:"sublanguageid,omitempty"`
	MovieReleaseName string `xmlrpc:"moviereleasename,omitempty"`
	MovieAka         string `xmlrpc:"movieaka,omitempty"`
	SubAuthorComment string `xmlrpc:"subauthorcomment,omitempty"`
	SubTranslator    string `xmlrpc:"subtranslator,omitempty"`
	// Added based on TryUpload params that might be relevant here too
	HearingImpaired  string `xmlrpc:"hearingimpaired,omitempty"`
	HighDefinition   string `xmlrpc:"highdefinition,omitempty"`
	ForeignPartsOnly string `xmlrpc:"foreignpartsonly,omitempty"`
}

// XmlRpcUploadSubtitlesCD holds the 'cdX' data for UploadSubtitles.
// Based on PrepareUploadSubtitlesParams.
type XmlRpcUploadSubtitlesCD struct {
	SubHash       string `xmlrpc:"subhash"`
	SubFilename   string `xmlrpc:"subfilename"`
	MovieHash     string `xmlrpc:"moviehash,omitempty"`     // Optional here? API implies it's needed if no imdbid
	MovieByteSize string `xmlrpc:"moviebytesize,omitempty"` // Optional here?
	SubContent    string `xmlrpc:"subcontent"`              // Base64 encoded content
	MovieFPS      string `xmlrpc:"moviefps,omitempty"`
	MovieTimeMS   string `xmlrpc:"movietimems,omitempty"`
}

// XmlRpcUploadSubtitlesParams is the top-level structure for the UploadSubtitles call.
type XmlRpcUploadSubtitlesParams struct {
	BaseInfo XmlRpcUploadSubtitlesBaseInfo      `xmlrpc:"baseinfo"`
	CDs      map[string]XmlRpcUploadSubtitlesCD `xmlrpc:",inline"` // Map "cd1", "cd2" etc. to CD data
}

// --- Helper Functions ---
