package opensubtitles

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	"github.com/angelospk/osuploadergui/pkg/core/fileops"
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

// prepareTryUploadParams gathers necessary data (hashes) and formats it
// for the TryUploadSubtitles XML-RPC call.
func PrepareTryUploadParams(intent UserUploadIntent) (*XmlRpcTryUploadParams, error) {
	params := XmlRpcTryUploadParams{}

	// Subtitle Hash & Filename (Mandatory for TryUpload)
	if intent.SubtitleFilePath == "" {
		return nil, fmt.Errorf("subtitle file path is required")
	}
	subHash, err := fileops.CalculateMD5Hash(intent.SubtitleFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate MD5 hash for subtitle: %w", err)
	}
	params.SubHash = subHash
	params.SubFilename = intent.SubtitleFileName // Assume already set
	if params.SubFilename == "" {
		return nil, fmt.Errorf("subtitle filename is required")
	}

	// Video Hash & Filename (Optional for TryUpload)
	if intent.VideoFilePath != "" {
		movieHash, movieSize, err := fileops.CalculateOSDbHash(intent.VideoFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate OSDb hash for video: %w", err)
		}
		params.MovieHash = movieHash
		params.MovieByteSize = strconv.FormatInt(movieSize, 10)
		params.MovieFilename = intent.VideoFileName // Assume already set
		if params.MovieFilename == "" {
			return nil, fmt.Errorf("video filename is required if video file is provided")
		}
	} else {
		// If no video, require both LanguageID and IMDBID for best results
		if intent.LanguageID == "" {
			return nil, fmt.Errorf("language ID is required if no video file is provided")
		}
		if intent.IMDBID == "" {
			return nil, fmt.Errorf("IMDB ID is required if no video file is provided")
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
	if intent.Frames > 0 {
		params.MovieFrames = strconv.FormatInt(intent.Frames, 10)
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

	return &params, nil
}

// readAndEncodeSubtitle reads the subtitle file, GZips it, and returns its Base64 encoded content.
// UPDATE: Removing Gzip step based on server developer feedback - trying only Base64.
func ReadAndEncodeSubtitle(subtitlePath string) (string, error) {
	contentBytes, err := os.ReadFile(subtitlePath)
	if err != nil {
		return "", fmt.Errorf("failed to read subtitle file content '%s': %w", subtitlePath, err)
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
	encodedContent := base64.StdEncoding.EncodeToString(contentBytes)
	return encodedContent, nil
}

// prepareUploadSubtitlesParams formats the data for the final UploadSubtitles call,
// including the base64 encoded content.
func PrepareUploadSubtitlesParams(tryParams XmlRpcTryUploadParams, subtitlePath string) (*XmlRpcUploadSubtitlesParams, error) {
	base64Content, err := ReadAndEncodeSubtitle(subtitlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read and encode subtitle for upload: %w", err)
	}
	if base64Content == "" {
		return nil, fmt.Errorf("base64 subtitle content cannot be empty")
	}

	// Parse string fields to their correct types (float64, int)
	var movieByteSize float64
	if tryParams.MovieByteSize != "" {
		movieByteSize, err = strconv.ParseFloat(tryParams.MovieByteSize, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MovieByteSize '%s': %w", tryParams.MovieByteSize, err)
		}
	}

	var movieFPS float64
	if tryParams.MovieFPS != "" {
		movieFPS, err = strconv.ParseFloat(tryParams.MovieFPS, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MovieFPS '%s': %w", tryParams.MovieFPS, err)
		}
	}

	var movieFrames int
	if tryParams.MovieFrames != "" {
		movieFrames64, err := strconv.ParseInt(tryParams.MovieFrames, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MovieFrames '%s': %w", tryParams.MovieFrames, err)
		}
		movieFrames = int(movieFrames64) // Convert to int
	}

	var movieTimeMS int
	if tryParams.MovieTimeMS != "" {
		movieTimeMS64, err := strconv.ParseInt(tryParams.MovieTimeMS, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse MovieTimeMS '%s': %w", tryParams.MovieTimeMS, err)
		}
		movieTimeMS = int(movieTimeMS64) // Convert to int
	}

	// Map fields from TryUploadParams to the nested structure UploadSubtitles expects
	// Using the CDs map field now
	params := XmlRpcUploadSubtitlesParams{
		BaseInfo: XmlRpcUploadSubtitlesBaseInfo{
			IDMovieImdb:      tryParams.IDMovieImdb, // Keep relevant fields from baseinfo
			MovieReleaseName: tryParams.MovieReleaseName,
			MovieAka:         tryParams.MovieAka,
			SubLanguageID:    tryParams.SubLanguageID,
			SubAuthorComment: tryParams.SubAuthorComment,
			// Remove boolean flags if not present in UploadSubtitles BaseInfo struct
			// HearingImpaired:      tryParams.HearingImpaired,
			// HighDefinition:       tryParams.HighDefinition,
			// AutomaticTranslation: tryParams.AutomaticTranslation,
			// SubTranslator:        tryParams.SubTranslator,
			// ForeignPartsOnly:     tryParams.ForeignPartsOnly,
		},
		CDs: map[string]XmlRpcUploadSubtitlesCD{
			"cd1": {
				SubHash:       tryParams.SubHash,
				SubFilename:   tryParams.SubFilename,
				MovieHash:     tryParams.MovieHash,
				MovieByteSize: movieByteSize, // Use parsed float64
				MovieTimeMS:   movieTimeMS,   // Use parsed int
				MovieFrames:   movieFrames,   // Use parsed int
				MovieFPS:      movieFPS,      // Use parsed float64
				MovieFilename: tryParams.MovieFilename,
				SubContent:    base64Content, // Use the provided encoded content
			},
			// Add cd2 etc. here if needed in the future
		},
	}

	return &params, nil
}
