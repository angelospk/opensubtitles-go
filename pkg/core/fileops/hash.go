package fileops

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const (
	// osdbHashChunkSize is the size of the chunk read from the start and end of the file.
	osdbHashChunkSize = 65536 // 64 * 1024
)

// CalculateMD5Hash computes the MD5 hash of a file.
func CalculateMD5Hash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for MD5 hashing '%s': %w", filePath, err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to copy file content for MD5 hashing '%s': %w", filePath, err)
	}

	hashBytes := hash.Sum(nil)
	return hex.EncodeToString(hashBytes), nil
}

// checksumBuffer calculates the sum of 64-bit little-endian integers in the buffer.
// This mimics the core loop of the checksum logic in the reference JS.
func checksumBuffer(buf []byte) (sum uint64) {
	// Process buffer in 8-byte chunks (64-bit numbers)
	for i := 0; i+8 <= len(buf); i += 8 {
		val := binary.LittleEndian.Uint64(buf[i : i+8])
		sum += val
	}
	return
}

// CalculateOSDbHash calculates the OpenSubtitles Movie Hash for a given video file.
// Based on the algorithm described at: http://trac.opensubtitles.org/projects/opensubtitles/wiki/HashSourceCodes
// AND refined to match the logic in vankasteelj/opensubtitles-api hash.js
func CalculateOSDbHash(filePath string) (hash string, byteSize int64, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		err = fmt.Errorf("failed to open file for OSDb hashing '%s': %w", filePath, err)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		err = fmt.Errorf("failed to stat file '%s': %w", filePath, err)
		return
	}

	byteSize = stat.Size()
	if byteSize < osdbHashChunkSize*2 { // Keep the size check
		err = fmt.Errorf("file '%s' is too small for OSDb hashing (size: %d)", filePath, byteSize)
		return
	}

	// Read first chunk (64KB)
	startBuf := make([]byte, osdbHashChunkSize)
	_, err = file.Read(startBuf)
	if err != nil {
		err = fmt.Errorf("failed to read start chunk from '%s': %w", filePath, err)
		return
	}

	// Read last chunk (64KB)
	endBuf := make([]byte, osdbHashChunkSize)
	_, err = file.ReadAt(endBuf, byteSize-osdbHashChunkSize)
	if err != nil {
		err = fmt.Errorf("failed to read end chunk from '%s': %w", filePath, err)
		return
	}

	// Calculate checksums of the chunks
	startChecksum := checksumBuffer(startBuf)
	endChecksum := checksumBuffer(endBuf)

	// Calculate final hash by summing file size and chunk checksums
	// Use uint64 arithmetic, overflow is expected/part of the algorithm
	finalHash := uint64(byteSize) + startChecksum + endChecksum

	hash = fmt.Sprintf("%016x", finalHash) // Format as 16-char hex
	return                                 // Return hash, byteSize, nil
}

// TODO: Implement MediaInfo Extraction
// TODO: Implement NFO File Reading
