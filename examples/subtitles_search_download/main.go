package main

import (
	"context"
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

// exampleLogin attempts to log in the user. This is optional for search but good for download quotas.
func exampleLogin(client *opensubtitles.Client) {
	log.Println("[INFO] --- Attempting Login (Optional for Search, Recommended for Download) ---")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	loginParams := opensubtitles.LoginRequest{
		Username: "YOUR_USERNAME", // Replace with actual username
		Password: "YOUR_PASSWORD", // Replace with actual password
	}
	if loginParams.Username == "YOUR_USERNAME" || loginParams.Password == "YOUR_PASSWORD" {
		log.Println("[INFO] Please replace YOUR_USERNAME and YOUR_PASSWORD with actual credentials for the Login example.")
	}

	_, err := client.Login(ctx, loginParams)
	if err != nil {
		log.Printf("Login failed: %v\n", err)
		return
	}
	fmt.Printf("Login successful! Client token is now set.\n")
}

// exampleSearchSubtitles demonstrates searching for subtitles.
// It returns the FileID of the first found subtitle, if any.
func exampleSearchSubtitles(client *opensubtitles.Client) (int, error) {
	log.Println("[INFO] --- Example: Search Subtitles ---")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := opensubtitles.SearchSubtitlesParams{
		Query:     StringPtr("inception 2010"),
		Languages: StringPtr("en"),
		// ImdbID: StringPtr("tt1375666"), // Example: Search by IMDb ID
		// Type: opensubtitles.SubtitleTypeMovie, // Example: Search for movie subtitles
		Page: IntPtr(1),
	}

	log.Printf("[INFO] Searching subtitles with params: Query:'%s' Languages:'%s' ...\n", *params.Query, *params.Languages)

	subtitlesResp, err := client.SearchSubtitles(ctx, params)
	if err != nil {
		log.Printf("SearchSubtitles failed: %v\n", err)
		return 0, err
	}

	fmt.Printf("Found %d subtitles (Page %d/%d, Total: %d)\n", len(subtitlesResp.Data), subtitlesResp.Page, subtitlesResp.TotalPages, subtitlesResp.TotalCount)

	if len(subtitlesResp.Data) == 0 {
		fmt.Println("No subtitles found for the given criteria.")
		return 0, fmt.Errorf("no subtitles found")
	}

	var firstFileID int
	for i, sub := range subtitlesResp.Data {
		if i < 3 { // Print details for the first few
			fmt.Printf("--- Subtitle %d ---\n", i+1)
			fmt.Printf("  ID: %s, Type: %s\n", sub.ID, sub.Type)
			attr := sub.Attributes
			fmt.Printf("  Lang: %s, Release: %s\n", attr.Language, attr.Release)
			fpsVal := "N/A"
			if attr.FPS != nil {
				fpsVal = fmt.Sprintf("%.3f", *attr.FPS)
			}
			fmt.Printf("  FPS: %s, HD: %t, HI: %t\n", fpsVal, attr.HD, attr.HearingImpaired)
			fmt.Printf("  Downloads: %d, URL: %s\n", attr.DownloadCount, attr.URL)
			if len(attr.Files) > 0 {
				fmt.Printf("  File ID for download: %d, File Name: %s\n", attr.Files[0].FileID, attr.Files[0].FileName)
				if i == 0 {
					firstFileID = attr.Files[0].FileID
				}
			} else {
				fmt.Println("  No files associated with this subtitle entry for download.")
			}
		}
	}

	if firstFileID == 0 && len(subtitlesResp.Data) > 0 && len(subtitlesResp.Data[0].Attributes.Files) > 0 {
		// Fallback if the loop didn't set it (e.g., only 1 result)
		firstFileID = subtitlesResp.Data[0].Attributes.Files[0].FileID
	}

	if firstFileID == 0 {
		return 0, fmt.Errorf("no downloadable files found in search results")
	}

	return firstFileID, nil
}

// exampleRequestDownload demonstrates requesting a download link for a subtitle file.
func exampleRequestDownload(client *opensubtitles.Client, fileID int) {
	log.Printf("[INFO] --- Example: Request Download Link for FileID: %d ---", fileID)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if client.GetCurrentToken() == nil {
		log.Println("[WARN] User is not logged in. Download might fail or be restricted.")
	}

	downloadParams := opensubtitles.DownloadRequest{
		FileID: fileID,
		// SubFormat: opensubtitles.SubtitleFormatSRT, // Optional: specify format
	}

	downloadResp, err := client.Download(ctx, downloadParams)
	if err != nil {
		log.Printf("Download failed for FileID %d: %v\n", fileID, err)
		return
	}

	fmt.Printf("Download Request Successful:\n")
	fmt.Printf("  Download Link: %s\n", downloadResp.Link)
	fmt.Printf("  File Name: %s\n", downloadResp.FileName)
	fmt.Printf("  Remaining Downloads for user: %d\n", downloadResp.Remaining)
	fmt.Printf("  Message: %s\n", downloadResp.Message)
	fmt.Printf("  Rate limit reset time: %s (UTC: %s)\n", downloadResp.ResetTime, downloadResp.ResetTimeUTC)
}

func main() {
	log.Println("[INFO] --- Initializing Client ---")
	client, err := getClient()
	if err != nil {
		log.Fatalf("Error in getClient: %v", err)
		return
	}

	// Attempt login (optional for search, but good for download quotas)
	exampleLogin(client)
	fmt.Println("-------------------------------------")

	// Search for subtitles
	fileID, err := exampleSearchSubtitles(client)
	if err != nil {
		log.Printf("Could not proceed to download example: %v", err)
	} else {
		fmt.Println("-------------------------------------")
		// If search was successful and a file ID was found, request download
		exampleRequestDownload(client, fileID)
	}
	fmt.Println("-------------------------------------")
}
