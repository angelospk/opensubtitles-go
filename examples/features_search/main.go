package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/angelospk/opensubtitles-go"
)

// getClient initializes and returns a new OpenSubtitles client.
func getClient() (*opensubtitles.Client, error) {
	apiKey := "YOUR_API_KEY" // Replace with your OpenSubtitles API key
	if apiKey == "YOUR_API_KEY" {
		log.Println("[INFO] Please replace YOUR_API_KEY with your actual OpenSubtitles API key for the examples to work correctly.")
	}
	userAgent := "MyGoExampleApp/1.0"

	cfg := opensubtitles.Config{
		ApiKey:    apiKey,
		UserAgent: userAgent,
	}

	client, err := opensubtitles.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	log.Println("[INFO] Client initialized.")
	return client, nil
}

// Helper function to easily get a pointer to a string
func StringPtr(s string) *string {
	return &s
}

// Helper function to easily get a pointer to an int
func IntPtr(i int) *int {
	return &i
}

// exampleSearchFeatures demonstrates searching for features and handling polymorphic attributes.
func exampleSearchFeatures(client *opensubtitles.Client, query string, featureType opensubtitles.FeatureType) {
	log.Printf("[INFO] --- Example: Search Features (%s: %s) ---", featureType, query)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := opensubtitles.SearchFeaturesParams{
		Query: StringPtr(query),
		Type:  StringPtr(string(featureType)),
	}

	log.Printf("Searching for features with query: \"%s\", type: \"%s\"\n", *params.Query, *params.Type)

	featuresResp, err := client.SearchFeatures(ctx, params)
	if err != nil {
		log.Printf("SearchFeatures failed: %v\n", err)
		return
	}

	fmt.Printf("Found %d features.\n", len(featuresResp.Data))

	for i, feat := range featuresResp.Data {
		fmt.Printf("--- Feature %d ---\n", i+1)
		fmt.Printf("  ID: %s, Type: %s\n", feat.ID, feat.Type)

		// The Attributes field is an interface{}. We need to unmarshal it based on FeatureType.
		// First, let's marshal it back to JSON bytes, then unmarshal into a map to inspect feature_type.
		attrBytes, err := json.Marshal(feat.Attributes)
		if err != nil {
			log.Printf("  Error marshalling attributes: %v\n", err)
			continue
		}

		var genericAttrMap map[string]interface{}
		if err := json.Unmarshal(attrBytes, &genericAttrMap); err != nil {
			log.Printf("  Error unmarshalling attributes to generic map: %v\n", err)
			continue
		}
		actualFeatureType := feat.Type

		log.Printf("  Attempting to unmarshal attributes for feature_type: %s\n", actualFeatureType)

		switch actualFeatureType {
		case "Movie", "movie": // API can return "Movie" or "movie"
			var movieAttrs opensubtitles.FeatureMovieAttributes
			if err := json.Unmarshal(attrBytes, &movieAttrs); err == nil {
				fmt.Printf("  Title: %s (Year: %s)\n", movieAttrs.Title, movieAttrs.Year)
				if movieAttrs.IMDbID != nil {
					fmt.Printf("  IMDb ID: %d", *movieAttrs.IMDbID)
				}
				if movieAttrs.TMDBID != nil {
					fmt.Printf(", TMDB ID: %d\n", *movieAttrs.TMDBID)
				} else {
					fmt.Println()
				}
				fmt.Printf("  URL: %s\n", movieAttrs.URL)
			} else {
				log.Printf("  Error unmarshalling into FeatureMovieAttributes: %v\n", err)
			}
		case "Tvshow", "tvshow": // API can return "Tvshow" or "tvshow"
			var tvshowAttrs opensubtitles.FeatureTvshowAttributes
			if err := json.Unmarshal(attrBytes, &tvshowAttrs); err == nil {
				fmt.Printf("  Title: %s (Year: %s)\n", tvshowAttrs.Title, tvshowAttrs.Year)
				if tvshowAttrs.IMDbID != nil {
					fmt.Printf("  IMDb ID: %d", *tvshowAttrs.IMDbID)
				}
				if tvshowAttrs.TMDBID != nil {
					fmt.Printf(", TMDB ID: %d\n", *tvshowAttrs.TMDBID)
				} else {
					fmt.Println()
				}
				fmt.Printf("  Seasons Count: %d\n", tvshowAttrs.SeasonsCount)
				fmt.Printf("  URL: %s\n", tvshowAttrs.URL)
			} else {
				log.Printf("  Error unmarshalling into FeatureTvshowAttributes: %v\n", err)
			}
		case "Episode", "episode":
			var episodeAttrs opensubtitles.FeatureEpisodeAttributes
			if err := json.Unmarshal(attrBytes, &episodeAttrs); err == nil {
				fmt.Printf("  Title: %s (Season: %d, Episode: %d, Year: %s)\n", episodeAttrs.Title, episodeAttrs.SeasonNumber, episodeAttrs.EpisodeNumber, episodeAttrs.Year)
				if episodeAttrs.ParentTitle != nil {
					fmt.Printf("  Parent Title: %s\n", *episodeAttrs.ParentTitle)
				}
				if episodeAttrs.IMDbID != nil {
					fmt.Printf("  IMDb ID: %d", *episodeAttrs.IMDbID)
				}
				if episodeAttrs.TMDBID != nil {
					fmt.Printf(", TMDB ID: %d\n", *episodeAttrs.TMDBID)
				} else {
					fmt.Println()
				}
				fmt.Printf("  URL: %s\n", episodeAttrs.URL)
			} else {
				log.Printf("  Error unmarshalling into FeatureEpisodeAttributes: %v\n", err)
			}
		default:
			log.Printf("  Unknown or unhandled feature_type: %s\n", actualFeatureType)
			fmt.Printf("  Raw Attributes: %s\n", string(attrBytes))
		}
	}
}

func main() {
	log.Println("[INFO] --- Initializing Client ---")
	client, err := getClient()
	if err != nil {
		log.Fatalf("Error in getClient: %v", err)
		return
	}

	// Example 1: Search for a TV show
	exampleSearchFeatures(client, "game of thrones", opensubtitles.FeatureTVShow)
	fmt.Println("-------------------------------------")

	// Example 2: Search for a movie
	exampleSearchFeatures(client, "inception", opensubtitles.FeatureMovie)
	fmt.Println("-------------------------------------")

	// Example 3: Search for an episode (might need more specific query or feature ID)
	// exampleSearchFeatures(client, "winter is coming", opensubtitles.FeatureEpisode) // This query alone might be ambiguous
	// For episode, often better to get TV show first, then find episode ID, or use specific episode IMDb ID if known.
	// log.Println("[INFO] For a specific episode, usually you would search by IMDb ID or Feature ID of the episode.")
	// params := opensubtitles.SearchFeaturesParams{IMDbID: StringPtr("tt1480055")} // Example: GoT S01E01 IMDb ID
	// ... and then call client.SearchFeatures(ctx, params)

}
