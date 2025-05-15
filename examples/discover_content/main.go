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

// Helper function to easily get a pointer to a LanguageCode
func LangCodePtr(lc opensubtitles.LanguageCode) *opensubtitles.LanguageCode {
	return &lc
}

// Helper function to easily get a pointer to a FeatureType
func FeatureTypePtr(ft opensubtitles.FeatureType) *opensubtitles.FeatureType {
	return &ft
}

// Helper function to easily get a pointer to an int
func IntPtr(i int) *int {
	return &i
}

// exampleDiscoverPopular demonstrates the DiscoverPopular endpoint.
func exampleDiscoverPopular(client *opensubtitles.Client, lang opensubtitles.LanguageCode, fType opensubtitles.FeatureType) {
	log.Printf("[INFO] --- Example: Discover Popular (%s, %s) ---", fType, lang)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := opensubtitles.DiscoverParams{
		Language: LangCodePtr(lang),
		Type:     FeatureTypePtr(fType),
	}

	popularResp, err := client.DiscoverPopular(ctx, params)
	if err != nil {
		log.Printf("DiscoverPopular failed: %v\n", err)
		return
	}

	fmt.Printf("Found %d popular features (type: %s, lang: %s).\n", len(popularResp.Data), fType, lang)
	for i, feat := range popularResp.Data {
		if i >= 3 { // Limit output for brevity
			fmt.Println("...")
			break
		}
		fmt.Printf("--- Popular Feature %d (%s) ---\n", i+1, fType)
		// Attributes handling is similar to SearchFeatures
		attrBytes, _ := json.Marshal(feat.Attributes)
		var genericAttrMap map[string]interface{}
		_ = json.Unmarshal(attrBytes, &genericAttrMap)
		actualFeatureType, _ := genericAttrMap["feature_type"].(string)

		switch actualFeatureType {
		case "Movie", "movie":
			var movieAttrs opensubtitles.FeatureMovieAttributes
			if json.Unmarshal(attrBytes, &movieAttrs) == nil {
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
			}
		case "Tvshow", "tvshow":
			var tvshowAttrs opensubtitles.FeatureTvshowAttributes
			if json.Unmarshal(attrBytes, &tvshowAttrs) == nil {
				fmt.Printf("  Title: %s (Year: %s)\n", tvshowAttrs.Title, tvshowAttrs.Year)
				if tvshowAttrs.IMDbID != nil {
					fmt.Printf("  IMDb ID: %d", *tvshowAttrs.IMDbID)
				}
				if tvshowAttrs.TMDBID != nil {
					fmt.Printf(", TMDB ID: %d\n", *tvshowAttrs.TMDBID)
				} else {
					fmt.Println()
				}
				fmt.Printf("  URL: %s\n", tvshowAttrs.URL)
			}
		default:
			fmt.Printf("  ID: %s, Type: %s (Attributes: %s)\n", feat.ID, feat.Type, string(attrBytes))
		}
	}
}

// exampleDiscoverLatest demonstrates the DiscoverLatest endpoint.
func exampleDiscoverLatest(client *opensubtitles.Client, lang opensubtitles.LanguageCode, fType opensubtitles.FeatureType) {
	log.Printf("[INFO] --- Example: Discover Latest Subtitles (%s, %s) ---", fType, lang)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := opensubtitles.DiscoverParams{
		Language: LangCodePtr(lang),
		Type:     FeatureTypePtr(fType), // 'all', 'movie', 'tvshow'
	}

	latestResp, err := client.DiscoverLatest(ctx, params)
	if err != nil {
		log.Printf("DiscoverLatest failed: %v\n", err)
		return
	}

	fmt.Printf("Found %d latest subtitles (Page %d/%d, Total: %d)\n", len(latestResp.Data), latestResp.Page, latestResp.TotalPages, latestResp.TotalCount)
	for i, sub := range latestResp.Data {
		if i >= 3 { // Limit output
			fmt.Println("...")
			break
		}
		attr := sub.Attributes
		fmt.Printf("--- Latest Subtitle %d ---\n", i+1)
		fmt.Printf("  ID: %s, Lang: %s, Release: %s\n", sub.ID, attr.Language, attr.Release)
		fmt.Printf("  Feature: %s (Year: %d)\n", attr.FeatureDetails.Title, attr.FeatureDetails.Year)
		fmt.Printf("  URL: %s\n", attr.URL)
	}
}

// exampleDiscoverMostDownloaded demonstrates the DiscoverMostDownloaded endpoint.
func exampleDiscoverMostDownloaded(client *opensubtitles.Client, lang opensubtitles.LanguageCode, fType opensubtitles.FeatureType) {
	log.Printf("[INFO] --- Example: Discover Most Downloaded Subtitles (%s, %s) ---", fType, lang)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := opensubtitles.DiscoverParams{
		Language: LangCodePtr(lang),
		Type:     FeatureTypePtr(fType), // 'all', 'movie', 'tvshow'
	}

	mostDownloadedResp, err := client.DiscoverMostDownloaded(ctx, params)
	if err != nil {
		log.Printf("DiscoverMostDownloaded failed: %v\n", err)
		return
	}
	fmt.Printf("Found %d most downloaded subtitles (Page %d/%d, Total: %d)\n", len(mostDownloadedResp.Data), mostDownloadedResp.Page, mostDownloadedResp.TotalPages, mostDownloadedResp.TotalCount)
	for i, sub := range mostDownloadedResp.Data {
		if i >= 3 { // Limit output
			fmt.Println("...")
			break
		}
		attr := sub.Attributes
		fmt.Printf("--- Most Downloaded Subtitle %d ---\n", i+1)
		fmt.Printf("  ID: %s, Lang: %s, Release: %s\n", sub.ID, attr.Language, attr.Release)
		fmt.Printf("  Feature: %s (Year: %d)\n", attr.FeatureDetails.Title, attr.FeatureDetails.Year)
		fmt.Printf("  Downloads: %d\n", attr.DownloadCount)
		fmt.Printf("  URL: %s\n", attr.URL)
	}
}

func main() {
	log.Println("[INFO] --- Initializing Client ---")
	client, err := getClient()
	if err != nil {
		log.Fatalf("Error in getClient: %v", err)
		return
	}

	english := opensubtitles.LanguageCode("en")
	movieType := opensubtitles.FeatureMovie
	tvShowType := opensubtitles.FeatureTVShow
	// allType := opensubtitles.FeatureAll // For DiscoverLatest and DiscoverMostDownloaded if you want all feature types

	exampleDiscoverPopular(client, english, movieType)
	fmt.Println("-------------------------------------")
	exampleDiscoverPopular(client, english, tvShowType)
	fmt.Println("-------------------------------------")

	exampleDiscoverLatest(client, english, movieType) // Can also use allType or tvShowType
	fmt.Println("-------------------------------------")

	exampleDiscoverMostDownloaded(client, english, movieType) // Can also use allType or tvShowType
	fmt.Println("-------------------------------------")
}
