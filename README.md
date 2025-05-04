# opensubtitles-go

A Go library for interacting with the OpenSubtitles.com API (v1).

## Features

*   Authentication (Login/Logout)
*   User Info
*   Subtitle Search
*   Subtitle Download Link Generation
*   Feature Search (Movies, TV Shows, Episodes)
*   Discovery Endpoints (Popular, Latest, Most Downloaded)
*   Utilities (Guessit)
*   Subtitle Upload (via legacy XML-RPC)

## Installation

```bash
go get github.com/angelospk/opensubtitles-go
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	opensubtitles "github.com/angelospk/opensubtitles-go"
)

func main() {
	apiKey := os.Getenv("OPENSUBTITLES_API_KEY")
	if apiKey == "" {
		log.Fatal("Missing OPENSUBTITLES_API_KEY environment variable")
	}

	config := opensubtitles.Config{
		ApiKey:    apiKey,
		UserAgent: "MyTestApp/1.0", // Replace with your application's user agent
	}

	client, err := opensubtitles.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Example: Login (Optional - only needed for user-specific actions like download)
	/*
		username := os.Getenv("OPENSUBTITLES_USERNAME")
		password := os.Getenv("OPENSUBTITLES_PASSWORD")
		if username != "" && password != "" {
			loginResp, err := client.Login(context.Background(), opensubtitles.LoginRequest{
				Username: username,
				Password: password,
			})
			if err != nil {
				log.Printf("Login failed: %v", err)
			} else {
				fmt.Printf("Login successful! Token: %s...\n", loginResp.Token[:10])
			}
		}
	*/

	// Example: Search for subtitles
	fmt.Println("\nSearching subtitles for 'Parasite'...")
	searchParams := opensubtitles.SearchSubtitlesParams{
		Query: opensubtitles.String("Parasite"), // Use helper for pointers
		Languages: opensubtitles.String("en"),
	}
	subsResp, err := client.SearchSubtitles(context.Background(), searchParams)
	if err != nil {
		log.Fatalf("Subtitle search failed: %v", err)
	}

	fmt.Printf("Found %d subtitles on page %d/%d:\n", len(subsResp.Data), subsResp.Page, subsResp.TotalPages)
	for _, sub := range subsResp.Data {
		fmt.Printf("- [%s] %s (Downloads: %d)\n",
			sub.Attributes.Language,
			sub.Attributes.Release,
			sub.Attributes.DownloadCount)
		if len(sub.Attributes.Files) > 0 {
			fmt.Printf("  File ID for download: %d\n", sub.Attributes.Files[0].FileID)
		}
	}

	// TODO: Add more examples (Download, Features, etc.)
}

// Helper function for creating string pointers (useful for optional params)
func String(s string) *string { return &s }
func Int(i int) *int { return &i }
func Float64(f float64) *float64 { return &f }
func Bool(b bool) *bool { return &b }

```

## Development

*   Run tests: `go test ./...`
*   Ensure dependencies are tidy: `go mod tidy`

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This library is distributed under the MIT license. See LICENSE file for details. 