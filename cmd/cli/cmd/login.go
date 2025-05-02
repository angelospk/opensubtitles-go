package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Verify authentication with OpenSubtitles using your API key",
	Long: `Verifies the OpenSubtitles API key configured via Viper (config file
or OSUPLOADER_OPENSUBTITLES_APIKEY environment variable) by making a test API call.

This command confirms your setup allows authenticated requests. 
It does not persistently store session tokens via the CLI itself;
the API key is used for authentication by subsequent commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := viper.GetString(CfgKeyOSAPIKey)
		if apiKey == "" {
			return fmt.Errorf("OpenSubtitles API key not configured. Set via key '%s' or env OSUPLOADER_OPENSUBTITLES_APIKEY", CfgKeyOSAPIKey)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Verifying OpenSubtitles API key...")

		// Instantiate the client with the API key
		// Passing nil for httpClient uses http.DefaultClient
		client := opensubtitles.NewClient(apiKey, nil)

		// Create a context
		ctx := context.Background()

		// Make a test authenticated call (GetUserInfo)
		userInfo, err := client.GetUserInfo(ctx)

		if err != nil {
			return fmt.Errorf("API key verification failed: %w", err)
		}

		if userInfo == nil {
			return errors.New("API key verification failed: Received empty user info from API")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "API Key verified successfully. Logged in as user: %s (Level: %s)\n", userInfo.Username, userInfo.Level)
		// No token saving needed here via CLI - client handles auth implicitly
		return nil
	},
}

func init() {
	RootCmd.AddCommand(loginCmd)
	// Add flags if needed later, e.g., --verbose
}
