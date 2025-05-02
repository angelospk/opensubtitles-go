package cmd

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewOSClientFunc allows overriding the OpenSubtitles client creation for testing.
// We return the interface defined in metadata for better decoupling.
var NewOSClientFunc = func(apiKey string) (metadata.OpenSubtitlesClient, error) {
	// Return concrete type satisfying the interface
	client := opensubtitles.NewClient(apiKey, http.DefaultClient)
	return client, nil // NewClient itself doesn't return an error
}

var (
	searchType     string
	searchIMDbID   string
	searchLang     string
	searchQuery    string
	searchSeason   int
	searchEpisode  int
	searchParentID string // For searching within a specific movie/show ID
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for subtitles on OpenSubtitles",
	Long: `Searches for subtitles on OpenSubtitles.com based on various criteria.
Requires at least one of --query, --imdbid, or --parent-id.

Examples:
  osuploadercli search --query "My Movie Title" --lang en
  osuploadercli search --imdbid tt1234567 --lang en
  osuploadercli search --parent-id 12345 --type episode --season 1 --episode 5 --lang fr
  osuploadercli search --query "My TV Show" --type episode --season 2 --episode 3`,
	RunE: runSearch,
}

func init() {
	RootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringVarP(&searchQuery, "query", "q", "", "Search query (movie/show title)")
	searchCmd.Flags().StringVar(&searchType, "type", "movie", "Type of feature to search (movie, episode, tvshow)") // Default to movie
	searchCmd.Flags().StringVar(&searchIMDbID, "imdbid", "", "IMDb ID (e.g., tt1234567)")
	searchCmd.Flags().StringVarP(&searchLang, "lang", "l", "", "Comma-separated list of language codes (e.g., en,el)")
	searchCmd.Flags().IntVarP(&searchSeason, "season", "s", 0, "Season number (for type=episode)")
	searchCmd.Flags().IntVarP(&searchEpisode, "episode", "e", 0, "Episode number (for type=episode)")
	searchCmd.Flags().StringVar(&searchParentID, "parent-id", "", "Parent feature ID (search within movie/tvshow ID)")

	// Mark flags dependent on type=episode
	// searchCmd.MarkFlagsRequiredTogether("type", "season", "episode") // Cobra doesn't have good support for conditional required flags based on *value*
}

func runSearch(cmd *cobra.Command, args []string) error {
	logger := logrus.New() // Consider using a shared logger instance
	logger.SetLevel(logrus.InfoLevel)

	apiKey := viper.GetString(CfgKeyOSAPIKey)
	if apiKey == "" {
		return fmt.Errorf("OpenSubtitles API key not configured. Use --api-key flag, OSUPLOADER_OS_API_KEY env var, or config file")
	}

	// Input validation
	if searchQuery == "" && searchIMDbID == "" && searchParentID == "" {
		return fmt.Errorf("at least one of --query, --imdbid, or --parent-id must be provided")
	}
	validTypes := map[string]bool{"movie": true, "episode": true, "tvshow": true}
	if _, ok := validTypes[searchType]; !ok {
		return fmt.Errorf("invalid --type: %s. Must be one of: movie, episode, tvshow", searchType)
	}
	if searchType == "episode" && (searchSeason <= 0 || searchEpisode <= 0) && searchParentID == "" {
		// Allow episode search without S/E if parent ID is given
		if searchParentID == "" {
			return fmt.Errorf("--season and --episode are required when --type=episode unless --parent-id is provided")
		}
	}
	if searchType != "episode" && (searchSeason > 0 || searchEpisode > 0) {
		logger.Warn("--season and --episode flags are ignored when --type is not 'episode'")
	}

	// Initialize client using the injectable function
	osClient, err := NewOSClientFunc(apiKey)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize OpenSubtitles client")
		return fmt.Errorf("failed to initialize OpenSubtitles client: %w", err)
	}

	// Build search parameters
	params := make(map[string]string)
	if searchQuery != "" {
		params["query"] = searchQuery
	}
	if searchIMDbID != "" {
		// Ensure "tt" prefix is handled correctly if needed, but API seems to accept just digits too
		imdbIDNumeric := strings.TrimPrefix(searchIMDbID, "tt")
		params["imdb_id"] = imdbIDNumeric
	}
	if searchLang != "" {
		params["languages"] = searchLang // API expects comma-separated string
	}
	if searchType != "" {
		params["type"] = searchType
	}
	if searchParentID != "" {
		params["parent_feature_id"] = searchParentID
	}
	if searchType == "episode" {
		if searchSeason > 0 {
			params["season_number"] = strconv.Itoa(searchSeason)
		}
		if searchEpisode > 0 {
			params["episode_number"] = strconv.Itoa(searchEpisode)
		}
	}

	logger.WithFields(logrus.Fields{
		"params": params,
	}).Info("Searching subtitles...")

	ctx := context.Background()
	results, err := osClient.SearchSubtitles(ctx, params)
	if err != nil {
		logger.WithError(err).Error("Subtitle search failed")
		return fmt.Errorf("subtitle search failed: %w", err)
	}

	// Print results
	if len(results.Data) == 0 {
		fmt.Println("No subtitles found matching the criteria.")
		return nil
	}

	fmt.Printf("Found %d subtitles (showing %d):\n", results.TotalCount, len(results.Data))
	fmt.Println("--------------------------------------------------")
	for _, sub := range results.Data {
		fmt.Printf("ID: %s\n", sub.ID)
		fmt.Printf("  File Name: %s\n", sub.Attributes.Files[0].FileName) // Assuming at least one file
		fmt.Printf("  Language: %s\n", sub.Attributes.Language)
		fmt.Printf("  Format: %s\n", sub.Attributes.Format)
		fmt.Printf("  Votes: %d\n", sub.Attributes.Votes)
		fmt.Printf("  Points: %.1f\n", sub.Attributes.Points)
		fmt.Printf("  Ratings: %.1f\n", sub.Attributes.Ratings)
		fmt.Printf("  Downloads: %d\n", sub.Attributes.DownloadCount)
		fmt.Printf("  Feature: %s (ID: %d, Title: %s, Year: %d)\n",
			sub.Attributes.FeatureDetails.FeatureType,
			sub.Attributes.FeatureDetails.FeatureID,
			sub.Attributes.FeatureDetails.Title,
			sub.Attributes.FeatureDetails.Year,
		)
		fmt.Println("--------------------------------------------------")
	}

	if results.TotalPages > 1 {
		fmt.Printf("More results available (Page 1 of %d)\n", results.TotalPages)
	}

	return nil
}
