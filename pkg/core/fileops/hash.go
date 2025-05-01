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

// CalculateOSDbHash calculates the OpenSubtitles Movie Hash for a given video file.
// Based on the algorithm described at: http://trac.opensubtitles.org/projects/opensubtitles/wiki/HashSourceCodes
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
	if byteSize < osdbHashChunkSize*2 {
		err = fmt.Errorf("file '%s' is too small for OSDb hashing (size: %d)", filePath, byteSize)
		return
	}

	// Read first chunk
	startBuf := make([]byte, osdbHashChunkSize)
	_, err = file.Read(startBuf)
	if err != nil {
		err = fmt.Errorf("failed to read start chunk from '%s': %w", filePath, err)
		return
	}

	// Read last chunk
	endBuf := make([]byte, osdbHashChunkSize)
	_, err = file.ReadAt(endBuf, byteSize-osdbHashChunkSize)
	if err != nil {
		err = fmt.Errorf("failed to read end chunk from '%s': %w", filePath, err)
		return
	}

	// Calculate hash based on file size and chunks
	var fileHash uint64 = uint64(byteSize) // Initialize with file size

	// Process start chunk (64-bit little-endian numbers)
	for i := 0; i < osdbHashChunkSize; i += 8 {
		if i+8 > len(startBuf) { // Should not happen with correct chunk size read
			break
		}
		val := binary.LittleEndian.Uint64(startBuf[i : i+8])
		fileHash += val
	}

	// Process end chunk (64-bit little-endian numbers)
	for i := 0; i < osdbHashChunkSize; i += 8 {
		if i+8 > len(endBuf) { // Should not happen
			break
		}
		val := binary.LittleEndian.Uint64(endBuf[i : i+8])
		fileHash += val
	}

	hash = fmt.Sprintf("%016x", fileHash)
	return // Return hash, byteSize, nil
}

// TODO: Implement MediaInfo Extraction
// TODO: Implement NFO File Reading
