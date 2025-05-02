package fileops

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	mediainfo "github.com/dreamCodeMan/go-mediainfo"
)

// GetMediaInfo runs the mediainfo CLI tool via the dreamCodeMan/go-mediainfo library
// and returns the structured information provided by the library.
// Requires mediainfo CLI to be installed and in the system PATH.
// The returned struct contains detailed track information (General, Video, Audio, Text etc.).
func GetMediaInfo(ctx context.Context, filePath string) (*mediainfo.MediaInfo, error) {
	// The context isn't directly used by this library call, but kept for API consistency.
	_ = ctx

	info, err := mediainfo.GetMediaInfo(filePath)
	if err != nil {
		// Check if the error indicates mediainfo not found vs. execution error
		if strings.Contains(err.Error(), "executable file not found") || strings.Contains(err.Error(), "LookPath") {
			return nil, fmt.Errorf("mediainfo command not found in PATH: %w. Please install mediainfo", err)
		}
		return nil, fmt.Errorf("go-mediainfo failed for '%s': %w", filePath, err)
	}

	// The library returns a struct value. We don't need a nil check here
	// as an error would have been returned above if something went wrong.
	// We return a pointer to the struct.
	return &info, nil
}

// imdbIDRegex pattern to find IMDb IDs (tt followed by 7 or more digits)
var imdbIDRegex = regexp.MustCompile(`(tt[0-9]{7,})`)

// ReadNFO attempts to read an NFO file and extract the first IMDb ID found.
func ReadNFO(ctx context.Context, filePath string) (string, error) {
	// Context isn't strictly needed for file read, but kept for consistency
	_ = ctx

	// Check if file exists before attempting to read
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Return empty string and nil error if NFO doesn't exist, common case.
		return "", nil
	}

	// Read the entire file content
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read NFO file '%s': %w", filePath, err)
	}

	// Convert content to string (assuming UTF-8 or ASCII compatible)
	// TODO: Consider handling other encodings if necessary (e.g., using iconv)
	content := string(contentBytes)

	// Find the first match for an IMDb ID pattern
	match := imdbIDRegex.FindStringSubmatch(content)

	// If a match is found, return the captured ID (group 1)
	if len(match) > 1 {
		return match[1], nil
	}

	// No IMDb ID found
	return "", nil
}

// CalculateMD5Hash calculates the MD5 hash of a file.
func CalculateMD5Hash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for MD5 hashing '%s': %w", filePath, err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file for MD5 hashing '%s': %w", filePath, err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

const osdbChunkSize = 64 * 1024 // 64KB

// CalculateOSDbHash calculates the OpenSubtitles DB hash for a video file.
// Hash = file size + 64bit checksum of first 64KB + 64bit checksum of last 64KB.
func CalculateOSDbHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for OSDb hashing '%s': %w", filePath, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info for OSDb hashing '%s': %w", filePath, err)
	}

	fileSize := stat.Size()
	if fileSize < osdbChunkSize*2 {
		// OSDb hash is not typically calculated for files smaller than 128KB
		// but the algorithm technically still works. Returning error might be safer.
		return "", fmt.Errorf("file '%s' is too small for OSDb hash calculation (size: %d bytes)", filePath, fileSize)
	}

	// Read first chunk
	firstChunk := make([]byte, osdbChunkSize)
	_, err = file.Read(firstChunk)
	if err != nil {
		return "", fmt.Errorf("failed to read first chunk for OSDb hashing '%s': %w", filePath, err)
	}

	// Read last chunk
	lastChunk := make([]byte, osdbChunkSize)
	_, err = file.ReadAt(lastChunk, fileSize-osdbChunkSize)
	if err != nil {
		return "", fmt.Errorf("failed to read last chunk for OSDb hashing '%s': %w", filePath, err)
	}

	// Calculate checksums (little-endian)
	var firstChecksum uint64
	var lastChecksum uint64

	// Process first chunk
	buf := bytes.NewReader(firstChunk)
	var num uint64
	for {
		errRead := binary.Read(buf, binary.LittleEndian, &num)
		if errRead == io.EOF {
			break
		}
		if errRead != nil {
			return "", fmt.Errorf("failed to calculate first chunk checksum (binary read) '%s': %w", filePath, errRead)
		}
		firstChecksum += num
	}

	// Process last chunk
	buf = bytes.NewReader(lastChunk)
	num = 0 // Reset num just in case
	for {
		errRead := binary.Read(buf, binary.LittleEndian, &num)
		if errRead == io.EOF {
			break
		}
		if errRead != nil {
			return "", fmt.Errorf("failed to calculate last chunk checksum (binary read) '%s': %w", filePath, errRead)
		}
		lastChecksum += num
	}

	// Calculate final hash
	hash := uint64(fileSize) + firstChecksum + lastChecksum

	// Format as hex string
	return fmt.Sprintf("%016x", hash), nil
}
