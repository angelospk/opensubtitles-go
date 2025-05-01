package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateMD5Hash(t *testing.T) {
	testFilePath := filepath.Join("testdata", "md5_test.txt")
	// Expected hash for "md5 test content" (no newline issues assumed)
	expectedHash := "af3614785b19e97a8cff2d2ae8066242"

	// Ensure the test file exists
	if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
		t.Fatalf("Test file %s does not exist: %v", testFilePath, err)
	}

	hash, err := CalculateMD5Hash(testFilePath)
	if err != nil {
		t.Fatalf("CalculateMD5Hash returned an unexpected error: %v", err)
	}

	if hash != expectedHash {
		t.Errorf("Expected MD5 hash %s, but got %s", expectedHash, hash)
	}
}

func TestCalculateOSDbHash(t *testing.T) {
	testFilePath := filepath.Join("testdata", "dummy.bin") // Use a small binary file
	// IMPORTANT: This expected hash is a placeholder and needs to be calculated
	// based on a reference file and a verified implementation of the algorithm.
	expectedHash := "placeholder_osdb_hash"
	expectedSize := int64(0) // Placeholder

	// Get actual size
	fileInfo, err := os.Stat(testFilePath)
	if err != nil {
		t.Fatalf("Failed to stat test file %s: %v", testFilePath, err)
	}
	expectedSize = fileInfo.Size()

	hash, size, err := CalculateOSDbHash(testFilePath)
	if err != nil {
		t.Fatalf("CalculateOSDbHash returned an unexpected error: %v", err)
	}

	if size != expectedSize {
		t.Errorf("Expected size %d, but got %d", expectedSize, size)
	}

	// TODO: Update expectedHash once a reference is available or implementation is verified.
	if hash != expectedHash {
		t.Logf("NOTE: OSDb hash test uses a placeholder expected hash.")
		t.Errorf("Expected OSDb hash %s, but got %s", expectedHash, hash)
	}
}
