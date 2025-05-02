package cmd_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/angelospk/osuploadergui/cmd/cli/cmd"
	"github.com/stretchr/testify/assert"
)

// TestUploadCommand_NonExistentPath tests if the command handles non-existent paths.
func TestUploadCommand_NonExistentPath(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	errorBuffer := bytes.NewBufferString("")
	cmd.RootCmd.SetOut(outputBuffer)
	cmd.RootCmd.SetErr(errorBuffer)

	nonExistentPath := filepath.Join(t.TempDir(), "does_not_exist")
	cmd.RootCmd.SetArgs([]string{"upload", nonExistentPath})

	// ExecuteC captures the error instead of os.Exit(1)
	_, err := cmd.RootCmd.ExecuteC()

	// Expect error from RunE (or potentially ExecuteC itself if arg parsing failed)
	assert.Error(t, err, "Expected an error for non-existent path")

	// Check the returned error message OR the stderr buffer for the logged message
	if err != nil {
		// Prefer checking the returned error message if available
		assert.Contains(t, err.Error(), "path does not exist:", "Expected error message content")
	} else {
		// Fallback: check stderr buffer if no error was returned (less ideal)
		errorOutput := errorBuffer.String()
		assert.Contains(t, errorOutput, "Error: Path does not exist:", "Expected specific error message in logs")
	}

	// Reset args
	cmd.RootCmd.SetArgs([]string{})
}

// TODO: Add tests for successful execution with mocks for processor/queue
// TODO: Add tests for flag parsing (e.g., --recursive)
