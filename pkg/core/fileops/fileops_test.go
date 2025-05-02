package fileops_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/angelospk/osuploadergui/pkg/core/fileops" // Adjust import path
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadNFO(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir() // Create a temporary directory for test files

	tests := []struct {
		name        string
		content     string
		expectedID  string
		expectError bool
		createFile  bool // Flag to indicate if the file should actually be created
	}{
		{
			name:       "File Not Found",
			content:    "",
			expectedID: "",
			createFile: false, // Don't create the file
		},
		{
			name:       "Valid ID in URL",
			content:    "Some text <imdb>https://www.imdb.com/title/tt1234567/</imdb> more text",
			expectedID: "tt1234567",
			createFile: true,
		},
		{
			name:       "Valid ID Standalone",
			content:    "IMDb ID: tt9876543\nDetails...",
			expectedID: "tt9876543",
			createFile: true,
		},
		{
			name:       "Valid ID with 8 digits",
			content:    "tt12345678 is the ID",
			expectedID: "tt12345678",
			createFile: true,
		},
		{
			name:       "No ID Present",
			content:    "This NFO contains no IMDb link.",
			expectedID: "",
			createFile: true,
		},
		{
			name:       "Empty File",
			content:    "",
			expectedID: "",
			createFile: true,
		},
		{
			name:       "Invalid ID Format",
			content:    "ID is tt12345", // Too short
			expectedID: "",
			createFile: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "test.nfo")

			if tc.createFile {
				err := os.WriteFile(filePath, []byte(tc.content), 0644)
				require.NoError(t, err, "Failed to write test NFO file")
				defer os.Remove(filePath) // Clean up the file afterwards
			} else {
				// Ensure file doesn't exist for the not found test
				os.Remove(filePath)
			}

			id, err := fileops.ReadNFO(ctx, filePath)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedID, id)
			}
		})
	}
}

// TODO: Add tests for CalculateOSDbHash and CalculateMD5Hash

func TestCalculateMD5Hash(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.txt")
	content := "The quick brown fox jumps over the lazy dog"
	expectedMD5 := "9e107d9d372bb6826bd81d3542a419d6"

	err := os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)
	defer os.Remove(filePath)

	// Use context in ReadNFO just for consistency, not actually needed here
	hash, err := fileops.CalculateMD5Hash(filePath)

	assert.NoError(t, err)
	assert.Equal(t, expectedMD5, hash)
}

func TestCalculateOSDbHash(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.bin")

	// Create a file larger than 128KB for a valid OSDb hash
	// We need known content to calculate the expected hash.
	// Let's create 3 chunks of 64KB.
	chunkSize := 64 * 1024
	fileSize := int64(chunkSize * 3)
	content := make([]byte, fileSize)

	// Fill first 64KB with 0x01
	for i := 0; i < chunkSize; i++ {
		content[i] = 0x01
	}
	// Fill middle 64KB with 0x02
	for i := chunkSize; i < chunkSize*2; i++ {
		content[i] = 0x02
	}
	// Fill last 64KB with 0x03
	for i := chunkSize * 2; i < chunkSize*3; i++ {
		content[i] = 0x03
	}

	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)
	defer os.Remove(filePath)

	// Calculate expected checksums (sum of uint64 values in little-endian chunks)
	// First chunk (all 0x0101010101010101)
	var firstChecksum uint64
	val1 := uint64(0x0101010101010101)
	numVals := chunkSize / 8 // Number of uint64 values in the chunk
	for i := 0; i < numVals; i++ {
		firstChecksum += val1
	}

	// Last chunk (all 0x0303030303030303)
	var lastChecksum uint64
	val3 := uint64(0x0303030303030303)
	for i := 0; i < numVals; i++ {
		lastChecksum += val3
	}

	expectedHashVal := uint64(fileSize) + firstChecksum + lastChecksum
	expectedHashStr := fmt.Sprintf("%016x", expectedHashVal)

	// --- Act ---
	hash, err := fileops.CalculateOSDbHash(filePath)

	// --- Assert ---
	assert.NoError(t, err)
	assert.Equal(t, expectedHashStr, hash)
}

func TestCalculateOSDbHash_FileTooSmall(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "small.bin")
	content := make([]byte, 64*1024) // Only 64KB
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err)
	defer os.Remove(filePath)

	_, err = fileops.CalculateOSDbHash(filePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too small for OSDb hash calculation")
}
