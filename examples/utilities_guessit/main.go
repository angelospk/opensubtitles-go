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

// Helper to print string value of a pointer or a default message
func printStringPtr(val *string, defaultMsg string) string {
	if val != nil {
		return *val
	}
	return defaultMsg
}

// Helper to print int value of a pointer or a default message
func printIntPtr(val *int, defaultMsg string) string {
	if val != nil {
		return fmt.Sprintf("%d", *val)
	}
	return defaultMsg
}

// Helper to print LanguageCode value of a pointer or a default message
func printLangCodePtr(val *opensubtitles.LanguageCode, defaultMsg string) string {
	if val != nil {
		return string(*val)
	}
	return defaultMsg
}

// exampleGuessit demonstrates the Guessit utility.
func exampleGuessit(client *opensubtitles.Client, filename string) {
	log.Println("[INFO] --- Example: Guessit Utility ---")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	params := opensubtitles.GuessitParams{
		Filename: filename,
	}

	log.Printf("Guessing info for filename: %s\n", filename)

	guessitResp, err := client.Guessit(ctx, params)
	if err != nil {
		log.Printf("Guessit failed for filename \"%s\": %v\n", filename, err)
		return
	}

	fmt.Println("Guessit Result:")
	fmt.Printf("  Title: %s\n", printStringPtr(guessitResp.Title, "Not detected"))
	fmt.Printf("  Year: %s\n", printIntPtr(guessitResp.Year, "Not detected"))
	fmt.Printf("  Season: %s\n", printIntPtr(guessitResp.Season, "Not detected"))
	fmt.Printf("  Episode: %s\n", printIntPtr(guessitResp.Episode, "Not detected"))
	fmt.Printf("  Episode Title: %s\n", printStringPtr(guessitResp.EpisodeTitle, "Not detected"))
	fmt.Printf("  Language: %s\n", printLangCodePtr(guessitResp.Language, "Not detected"))
	fmt.Printf("  Subtitle Language: %s\n", printLangCodePtr(guessitResp.SubtitleLanguage, "Not detected"))
	fmt.Printf("  Screen Size: %s\n", printStringPtr(guessitResp.ScreenSize, "Not detected"))
	fmt.Printf("  Streaming Service: %s\n", printStringPtr(guessitResp.StreamingService, "Not detected"))
	fmt.Printf("  Source: %s\n", printStringPtr(guessitResp.Source, "Not detected"))
	fmt.Printf("  Other: %s\n", printStringPtr(guessitResp.Other, "Not detected"))
	fmt.Printf("  Audio Codec: %s\n", printStringPtr(guessitResp.AudioCodec, "Not detected"))
	fmt.Printf("  Audio Channels: %s\n", printStringPtr(guessitResp.AudioChannels, "Not detected"))
	fmt.Printf("  Audio Profile: %s\n", printStringPtr(guessitResp.AudioProfile, "Not detected"))
	fmt.Printf("  Video Codec: %s\n", printStringPtr(guessitResp.VideoCodec, "Not detected"))
	fmt.Printf("  Release Group: %s\n", printStringPtr(guessitResp.ReleaseGroup, "Not detected"))
	fmt.Printf("  Type: %s\n", printStringPtr(guessitResp.Type, "Not detected"))
}

func main() {
	log.Println("[INFO] --- Initializing Client ---")
	client, err := getClient()
	if err != nil {
		log.Fatalf("Error in getClient: %v", err)
		return
	}

	sampleFilename := "The.Mandalorian.S01E01.Chapter.1.1080p.WEB-DL.DDP5.1.H.264-STAR.mkv"
	exampleGuessit(client, sampleFilename)
	fmt.Println("-------------------------------------")

	sampleFilename2 := "Avengers.Endgame.2019.1080p.BluRay.x264-SPARKS.srt"
	exampleGuessit(client, sampleFilename2)
	fmt.Println("-------------------------------------")
}
