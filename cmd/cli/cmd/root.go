package cmd

import (
	"bufio" // Added for reading user input
	"fmt"
	"log"
	"os"
	"path/filepath" // Added for creating config path
	"strings"       // Added for trimming input

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
		// PersistentPreRun ensures configuration is checked before any command runs.
		// We use PersistentPreRun instead of relying solely on initConfig via OnInitialize
		// to ensure the prompt logic runs *after* Viper has loaded everything.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			checkAndPromptAPIKey()
		},
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
	// RootCmd.AddCommand(uploadCmd)
	// RootCmd.AddCommand(searchCmd)
	// RootCmd.AddCommand(loginCmd)
}

// initConfig reads in config file and ENV variables if set.
// This runs *before* PersistentPreRun.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err) // Use cobra helper for error checking

		// --- Simplified Config Path Logic ---
		configDir := filepath.Join(home, ".osuploadercli")
		viper.AddConfigPath(configDir) // Add $HOME/.osuploadercli
		viper.AddConfigPath(".")       // Add current directory as fallback/alternative
		viper.SetConfigType("yaml")    // REQUIRED if the config file does not have the extension in the name
		viper.SetConfigName("config")  // Look for config.yaml (or config)
		// --- End Simplified Logic ---
	}

	viper.AutomaticEnv()             // read in environment variables that match
	viper.SetEnvPrefix("OSUPLOADER") // Set env prefix, e.g., OSUPLOADER_OPENSUBTITLES_APIKEY

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error, checkAndPromptAPIKey will handle it
		} else if os.IsNotExist(err) {
			// Handle case where config directory might not exist yet
		} else {
			// Config file was found but another error was produced
			fmt.Fprintf(os.Stderr, "Error reading config file (%s): %v\n", viper.ConfigFileUsed(), err)
		}
	}
}

// checkAndPromptAPIKey checks if the API key is set and prompts if not.
// This runs via PersistentPreRun after initConfig.
func checkAndPromptAPIKey() {
	apiKey := viper.GetString(CfgKeyOSAPIKey)
	if apiKey == "" {
		fmt.Println("OpenSubtitles API Key not found.")
		fmt.Print("Please enter your API Key: ")

		reader := bufio.NewReader(os.Stdin)
		inputKey, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read API Key: %v", err)
		}
		inputKey = strings.TrimSpace(inputKey)

		if inputKey == "" {
			log.Fatalf("API Key cannot be empty.")
		}

		// Set the key in viper instance for the current run (though we exit)
		viper.Set(CfgKeyOSAPIKey, inputKey)

		// --- Standardized Save Path ---
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Could not get home directory: %v", err)
		}
		configDir := filepath.Join(home, ".osuploadercli")
		configPath := filepath.Join(configDir, "config.yaml") // Standardized filename

		// Create the directory if it doesn't exist
		if err := os.MkdirAll(configDir, 0750); err != nil {
			log.Fatalf("Could not create config directory %s: %v", configDir, err)
		}

		// Ensure the settings map reflects the nested structure before writing
		// viper.Set should handle this, but let's be explicit if issues persist.
		// Example: viper.Set("opensubtitles", map[string]string{"apikey": inputKey})

		// Write the config file using the current viper settings
		// Note: WriteConfigAs saves *all* current viper settings, not just the one we set.
		if err := viper.WriteConfigAs(configPath); err != nil {
			log.Fatalf("Failed to save API Key to %s: %v", configPath, err)
		}

		fmt.Printf("API Key saved successfully to %s\n", configPath)
		fmt.Println("Please re-run your command.")
		os.Exit(0) // Exit after saving
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
