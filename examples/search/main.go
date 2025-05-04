package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	opensubtitles "github.com/angelospk/opensubtitles-go"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	// --- Credentials ---
	fmt.Println("--- OpenSubtitles Credentials ---")
	fmt.Println("Enter your OpenSubtitles API Key:")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	fmt.Println("Enter your User Agent (e.g., MyApp v1.0):")
	userAgent, _ := reader.ReadString('\n')
	userAgent = strings.TrimSpace(userAgent)

	// --- Client Initialization ---
	config := opensubtitles.Config{
		ApiKey:    apiKey,
		UserAgent: userAgent,
	}
	client, err := opensubtitles.NewClient(config)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}
	fmt.Println("Client created successfully.")

	// --- Search Parameters ---
	fmt.Println("\n--- Subtitle Search ---")
	fmt.Println("Enter IMDb ID to search for (e.g., 1371111 for Inception), or press Enter to skip:")
	imdbIDStr, _ := reader.ReadString('\n')
	imdbIDStr = strings.TrimSpace(imdbIDStr)

	fmt.Println("Enter movie hash to search for, or press Enter to skip:")
	movieHash, _ := reader.ReadString('\n')
	movieHash = strings.TrimSpace(movieHash)

	fmt.Println("Enter languages to search for (comma-separated, e.g., en,es), or press Enter for all:")
	languages, _ := reader.ReadString('\n')
	languages = strings.TrimSpace(languages)

	fmt.Println("Enter a query string (movie title, etc.), or press Enter to skip:")
	query, _ := reader.ReadString('\n')
	query = strings.TrimSpace(query)

	// --- Prepare Search ---
	params := opensubtitles.SearchSubtitlesParams{}

	if imdbIDStr != "" {
		id, err := strconv.Atoi(imdbIDStr)
		if err != nil {
			fmt.Printf("Invalid IMDb ID format: %s\n", imdbIDStr)
		} else {
			params.IMDbID = &id
		}
	}
	if movieHash != "" {
		mh := movieHash // Create local var
		params.Moviehash = &mh
	}
	if languages != "" {
		langs := languages // Create local var
		params.Languages = &langs
	}
	if query != "" {
		q := query // Create local var
		params.Query = &q
	}

	if params.IMDbID == nil && params.Moviehash == nil && params.Query == nil {
		fmt.Println("Error: You must provide at least an IMDb ID, movie hash, or query string to search.")
		return
	}

	// --- Perform Search ---
	fmt.Println("\nSearching for subtitles...")
	ctx := context.Background()
	searchResp, err := client.SearchSubtitles(ctx, params)

	if err != nil {
		fmt.Printf("Error searching subtitles: %v\n", err)
		return
	}

	// --- Display Results ---
	fmt.Printf("\nFound %d subtitles (Page %d/%d):\n", searchResp.TotalCount, searchResp.Page, searchResp.TotalPages)
	if len(searchResp.Data) == 0 {
		fmt.Println("No subtitles found matching your criteria.")
		return
	}

	fmt.Println("--------------------------------------------------")
	for i, sub := range searchResp.Data {
		fmt.Printf("Result %d:\n", i+1)
		fmt.Printf("  Subtitle ID: %s\n", sub.ID)
		fmt.Printf("  Language:    %s\n", sub.Attributes.Language)
		fmt.Printf("  Feature:     %s (IMDb: %v)\n", sub.Attributes.FeatureDetails.Title, *sub.Attributes.FeatureDetails.IMDbID)
		fmt.Printf("  Uploader:    %s (Rank: %s)\n", sub.Attributes.Uploader.Name, sub.Attributes.Uploader.Rank)
		fmt.Printf("  Downloads:   %d\n", sub.Attributes.DownloadCount)
		fmt.Printf("  Votes:       %d (Score: %.2f)\n", sub.Attributes.Votes, sub.Attributes.Ratings)
		fmt.Printf("  Is Hearing Impaired: %t\n", sub.Attributes.HearingImpaired)
		if len(sub.Attributes.Files) > 0 {
			fmt.Printf("  File ID:     %d (%s)\n", sub.Attributes.Files[0].FileID, sub.Attributes.Files[0].FileName)
		}
		fmt.Println("--------------------------------------------------")
	}

	// Note: This example only shows the first page of results.
	// Implement pagination using the params.Page field for more results.
}
