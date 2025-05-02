package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from the OpenSubtitles REST API",
	Long: `Attempts to log out from the OpenSubtitles REST API session associated
with the currently used authentication token (managed internally by the client based on the API key).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Logout requires the API key to instantiate the client which holds the token
		apiKey := viper.GetString(CfgKeyOSAPIKey)
		if apiKey == "" {
			return errors.New("OpenSubtitles API key not configured. Use config or OSUPLOADER_OPENSUBTITLES_APIKEY")
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Attempting to log out from OpenSubtitles REST API...")

		// Instantiate the client
		client := opensubtitles.NewClient(apiKey, nil)
		ctx := context.Background()

		err := client.Logout(ctx)

		if err != nil {
			return fmt.Errorf("logout failed: %w", err)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Logout successful.")
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(logoutCmd)
}
