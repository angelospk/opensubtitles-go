//go:build integration
// +build integration

package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// Requires the OSUPLOADER_OPENSUBTITLES_APIKEY environment variable to be set.
// Run with: go test -tags=integration ./...

func TestLoginCommand_Integration_Success(t *testing.T) {
	apiKey := os.Getenv("OSUPLOADER_OPENSUBTITLES_APIKEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: OSUPLOADER_OPENSUBTITLES_APIKEY environment variable not set.")
	}

	// Ensure Viper uses the env var for this test run
	viper.Reset() // Reset viper state between tests
	viper.SetEnvPrefix("OSUPLOADER")
	viper.AutomaticEnv()
	// Manually set it just in case AutomaticEnv doesn't pick it up in test context
	viper.Set(CfgKeyOSAPIKey, apiKey)

	log.SetOutput(os.Stderr) // Ensure logs are visible
	log.SetLevel(logrus.DebugLevel)

	t.Logf("Using API Key starting with: %s... for login test", apiKey[:min(5, len(apiKey))])

	// Capture output
	var buf bytes.Buffer
	RootCmd.SetOut(&buf)
	RootCmd.SetErr(&buf)

	// Set args for the login command
	RootCmd.SetArgs([]string{"login"})

	// Execute the command
	err := RootCmd.Execute()

	// Assertions
	assert.NoError(t, err, "Login command execution failed")
	output := buf.String()
	t.Logf("Login command output:\n%s", output)
	assert.Contains(t, output, "Successfully authenticated user:", "Expected success message not found in output")

	viper.Reset() // Clean up viper state
}

func TestLoginCommand_Integration_Failure_BadKey(t *testing.T) {
	// Use a deliberately bad API key
	badAPIKey := "thisisnotavalidapikey"
	viper.Reset()
	viper.Set(CfgKeyOSAPIKey, badAPIKey)

	log.SetOutput(os.Stderr)
	log.SetLevel(logrus.DebugLevel)
	t.Logf("Using dummy API Key: %s for login failure test", badAPIKey)

	var buf bytes.Buffer
	RootCmd.SetOut(&buf)
	RootCmd.SetErr(&buf)
	RootCmd.SetArgs([]string{"login"})

	err := RootCmd.Execute()

	// Expect an error from the command itself OR error logged
	output := buf.String()
	t.Logf("Login command (bad key) output:\n%s", output)

	// The command might return an error or just log it depending on implementation
	// Check for logged error message
	assert.True(t, err != nil || strings.Contains(output, "Authentication failed"), "Expected command error or 'Authentication failed' log message")

	viper.Reset() // Clean up viper state
}

// Helper function (consider moving to a test utility package later)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
