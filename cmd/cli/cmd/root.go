package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper" // Import viper
	// Import other necessary packages like viper for config later
)

// Define configuration keys
const (
	CfgKeyOSAPIKey      = "opensubtitles.apikey"
	CfgKeyOSToken       = "opensubtitles.token"    // Store JWT token after login
	CfgKeyOSUsername    = "opensubtitles.username" // For XML-RPC login
	CfgKeyOSPassword    = "opensubtitles.password" // For XML-RPC login
	CfgKeyTraktClientID = "trakt.clientid"
	// Add other keys as needed, e.g., for log level, config dir etc.
)

var (
	// Used for flags.
	cfgFile string

	// RootCmd represents the base command when called without any subcommands
	// Exported for use in tests
	RootCmd = &cobra.Command{ // Renamed from rootCmd to RootCmd
		Use:   "osuploadercli",
		Short: "A CLI tool to interact with OpenSubtitles and upload subtitles.",
		Long: `osuploadercli allows you to search for subtitles on OpenSubtitles,
manage an upload queue, and upload subtitles via the command line.`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	err := RootCmd.Execute() // Use exported RootCmd
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig) // Call initConfig on initialization

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.osuploadercli.yaml or ./config.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle") // Example flag removed

	// Add subcommands here later
	// rootCmd.AddCommand(uploadCmd)
	// rootCmd.AddCommand(searchCmd)
	// rootCmd.AddCommand(loginCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err) // Use cobra helper for error checking

		// Search config in home directory OR current directory
		viper.AddConfigPath(home)             // Add home directory
		viper.AddConfigPath(".")              // Add current directory
		viper.SetConfigType("yaml")           // Set config type
		viper.SetConfigName(".osuploadercli") // Look for .osuploadercli.yaml
		viper.SetConfigName("config")         // Also look for config.yaml
	}

	viper.AutomaticEnv()             // read in environment variables that match
	viper.SetEnvPrefix("OSUPLOADER") // Set env prefix, e.g., OSUPLOADER_TRAKT_CLIENTID
	// Example binding specific env vars if needed, but AutomaticEnv + Prefix is usually sufficient
	// viper.BindEnv(CfgKeyOSAPIKey, "OPENSUBTITLES_API_KEY")
	// viper.BindEnv(CfgKeyTraktClientID, "TRAKT_CLIENT_ID")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	} else {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Fprintln(os.Stderr, "Config file not found, relying on ENV variables.")
		} else {
			// Config file was found but another error was produced
			fmt.Fprintf(os.Stderr, "Error reading config file %s: %v\n", viper.ConfigFileUsed(), err)
		}
	}
}

/* // Example of initConfig function (implement later)
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		v viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".osuploadercli" (without extension).
		v viper.AddConfigPath(home)
		v viper.SetConfigType("yaml")
		v viper.SetConfigName(".osuploadercli")
	}

	v viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := v viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", v viper.ConfigFileUsed())
	}
}
*/
