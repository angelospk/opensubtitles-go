package opensubtitles

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	"compress/gzip"

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

	// Video Hash & Filename (Mandatory for TryUpload)
	if intent.VideoFilePath == "" {
		// Maybe allow TryUpload without video? JS seems to require it.
		// For now, enforce based on JS.
		return nil, fmt.Errorf("video file path is required for TryUpload")
	}
	movieHash, movieSize, err := fileops.CalculateOSDbHash(intent.VideoFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate OSDb hash for video: %w", err)
	}
	params.MovieHash = movieHash
	params.MovieByteSize = strconv.FormatInt(movieSize, 10)
	params.MovieFilename = intent.VideoFileName // Assume already set
	if params.MovieFilename == "" {
		return nil, fmt.Errorf("video filename is required")
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
func ReadAndEncodeSubtitle(subtitlePath string) (string, error) {
	contentBytes, err := os.ReadFile(subtitlePath)
	if err != nil {
		return "", fmt.Errorf("failed to read subtitle file content '%s': %w", subtitlePath, err)
	}

	// GZip the content
	var gzipBuffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&gzipBuffer)
	_, err = gzipWriter.Write(contentBytes)
	if err != nil {
		return "", fmt.Errorf("failed to gzip subtitle content: %w", err)
	}
	err = gzipWriter.Close() // Close is important to finalize compression
	if err != nil {
		return "", fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Base64 encode the *gzipped* content
	encodedContent := base64.StdEncoding.EncodeToString(gzipBuffer.Bytes())
	return encodedContent, nil
}

// prepareUploadSubtitlesParams formats the data for the final UploadSubtitles call,
// including the base64 encoded content.
func PrepareUploadSubtitlesParams(tryParams XmlRpcTryUploadParams, base64Content string) (*XmlRpcUploadSubtitlesParams, error) {
	if base64Content == "" {
		return nil, fmt.Errorf("base64 subtitle content cannot be empty")
	}

	// Map fields from TryUploadParams to the nested structure UploadSubtitles expects
	params := XmlRpcUploadSubtitlesParams{
		BaseInfo: XmlRpcUploadSubtitlesBaseInfo{
			IDMovieImdb:          tryParams.IDMovieImdb,
			MovieReleaseName:     tryParams.MovieReleaseName,
			MovieAka:             tryParams.MovieAka,
			SubLanguageID:        tryParams.SubLanguageID,
			SubAuthorComment:     tryParams.SubAuthorComment,
			HearingImpaired:      tryParams.HearingImpaired,
			HighDefinition:       tryParams.HighDefinition,
			AutomaticTranslation: tryParams.AutomaticTranslation,
			SubTranslator:        tryParams.SubTranslator,
			ForeignPartsOnly:     tryParams.ForeignPartsOnly,
		},
		CD1: XmlRpcUploadSubtitlesCD{
			SubHash:       tryParams.SubHash,
			SubFilename:   tryParams.SubFilename,
			SubContent:    base64Content,
			MovieByteSize: tryParams.MovieByteSize,
			MovieHash:     tryParams.MovieHash,
			MovieFilename: tryParams.MovieFilename,
			MovieFPS:      tryParams.MovieFPS,
			MovieFrames:   tryParams.MovieFrames,
			MovieTimeMS:   tryParams.MovieTimeMS,
		},
	}

	return &params, nil
}
