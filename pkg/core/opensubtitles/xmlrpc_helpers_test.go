package opensubtitles

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareTryUploadParams_SubOnly_Success(t *testing.T) {
	intent := UserUploadIntent{
		SubtitleFilePath: filepath.Join("testdata", "dummy.srt"),
		SubtitleFileName: "dummy.srt",
		LanguageID:       "eng",
		IMDBID:           "1234567",
	}
	params, err := PrepareTryUploadParams(intent)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if params.SubHash == "" || params.SubFilename != "dummy.srt" {
		t.Errorf("Subtitle hash or filename not set correctly")
	}
	if params.SubLanguageID != "eng" || params.IDMovieImdb != "1234567" {
		t.Errorf("Language or IMDB ID not set correctly")
	}
}

func TestPrepareTryUploadParams_SubOnly_MissingLang(t *testing.T) {
	intent := UserUploadIntent{
		SubtitleFilePath: filepath.Join("testdata", "dummy.srt"),
		SubtitleFileName: "dummy.srt",
		IMDBID:           "1234567",
	}
	_, err := PrepareTryUploadParams(intent)
	if err == nil || err.Error() != "language ID is required if no video file is provided" {
		t.Fatalf("Expected language ID error, got: %v", err)
	}
}

func TestPrepareTryUploadParams_SubOnly_MissingIMDB(t *testing.T) {
	intent := UserUploadIntent{
		SubtitleFilePath: filepath.Join("testdata", "dummy.srt"),
		SubtitleFileName: "dummy.srt",
		LanguageID:       "eng",
	}
	_, err := PrepareTryUploadParams(intent)
	if err == nil || err.Error() != "IMDB ID is required if no video file is provided" {
		t.Fatalf("Expected IMDB ID error, got: %v", err)
	}
}

func TestPrepareTryUploadParams_SubAndVideo_Success(t *testing.T) {
	intent := UserUploadIntent{
		SubtitleFilePath: filepath.Join("testdata", "dummy.srt"),
		SubtitleFileName: "dummy.srt",
		VideoFilePath:    filepath.Join("testdata", "video.mkv"),
		VideoFileName:    "video.mkv",
	}
	params, err := PrepareTryUploadParams(intent)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}
	if params.MovieHash == "" || params.MovieFilename != "video.mkv" {
		t.Errorf("Movie hash or filename not set correctly")
	}
}

func TestPrepareTryUploadParams_MissingSubtitle(t *testing.T) {
	intent := UserUploadIntent{
		SubtitleFilePath: "",
		SubtitleFileName: "dummy.srt",
		LanguageID:       "eng",
		IMDBID:           "1234567",
	}
	_, err := PrepareTryUploadParams(intent)
	if err == nil || err.Error() != "subtitle file path is required" {
		t.Fatalf("Expected subtitle file path error, got: %v", err)
	}
}

func TestReadAndEncodeSubtitle_EncodeDecodeRoundTrip(t *testing.T) {
	path := filepath.Join("testdata", "dummy.srt")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read original dummy.srt: %v", err)
	}

	encoded, err := ReadAndEncodeSubtitle(path)
	if err != nil {
		t.Fatalf("ReadAndEncodeSubtitle failed: %v", err)
	}

	// Decode base64
	gzipped, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("Base64 decode failed: %v", err)
	}

	// Decompress gzip
	gzipReader, err := gzip.NewReader(bytes.NewReader(gzipped))
	if err != nil {
		t.Fatalf("gzip.NewReader failed: %v", err)
	}
	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		t.Fatalf("gzip decompress failed: %v", err)
	}
	_ = gzipReader.Close()

	if !bytes.Equal(original, decompressed) {
		t.Errorf("Decoded+decompressed content does not match original.\nOriginal: %q\nDecoded: %q", original, decompressed)
	}
}
