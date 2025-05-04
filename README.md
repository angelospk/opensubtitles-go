# opensubtitles-go

A Go client library for interacting with the OpenSubtitles REST API and XML-RPC Upload API.

[![Go Reference](https://pkg.go.dev/badge/github.com/angelospk/opensubtitles-go.svg)](https://pkg.go.dev/github.com/angelospk/opensubtitles-go)
<!-- Add other badges if desired (Build status, coverage, etc.) -->

## Features

*   Access to OpenSubtitles REST API endpoints:
    *   Authentication (Login, Logout)
    *   Subtitle Search
    *   Subtitle Download Link Retrieval
    *   Discover (Latest, Popular, Featured)
    *   Utilities (Formats, Languages, User Info)
*   XML-RPC based Subtitle Upload functionality.
*   Type-safe request parameters and response structs.
*   Built-in helpers for common tasks (e.g., hashing - provided by `upload` package).

## Installation

```bash
go get github.com/angelospk/opensubtitles-go
```

*(Remember to run `go mod tidy` in your project)*

## Requirements

*   **Go:** Version 1.18 or higher.
*   **OpenSubtitles API Key:** Obtainable from the [OpenSubtitles website](https://opensubtitles.stoplight.io/).
*   **OpenSubtitles Account:** Required for Login, Download, and Upload operations.

## Usage

### Client Initialization

```go
package main

import (
	"fmt"
	opensubtitles "github.com/angelospk/opensubtitles-go"
)

func main() {
	config := opensubtitles.Config{
		ApiKey:    "YOUR_API_KEY", // Replace with your actual API key
		UserAgent: "YourAppName/1.0", // Replace with your app's user agent
	}

	client, err := opensubtitles.NewClient(config)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}
	fmt.Println("Client initialized successfully!")

	// ... use client methods ...
}
```

### Authentication (Login/Logout)

```go
	// Login (Requires username and password)
	loginParams := opensubtitles.LoginRequest{
		Username: "YOUR_USERNAME",
		Password: "YOUR_PASSWORD",
	}
	loginResp, err := client.Login(context.Background(), loginParams)
	if err != nil {
		// Handle error
	}
	fmt.Printf("Logged in! Token starts with: %s\n", loginResp.Token[:5])

	// ... perform authenticated actions (download, get user info, etc.) ...

	// Logout
	err = client.Logout(context.Background())
	if err != nil {
		// Handle error
	}
	fmt.Println("Logged out.")
```

### Searching Subtitles

```go
	// Search by IMDb ID
	var imdbID = 1371111 // Example: Inception
	searchParams := opensubtitles.SearchSubtitlesParams{
		IMDbID: &imdbID,
		Languages: opensubtitles.String("en"), // Optional: filter by language
	}

	searchResp, err := client.SearchSubtitles(context.Background(), searchParams)
	if err != nil {
		// Handle error
	}

	fmt.Printf("Found %d subtitles.\n", searchResp.TotalCount)
	for _, sub := range searchResp.Data {
		fmt.Printf("- Subtitle ID: %s, Lang: %s, File ID: %d\n",
			sub.ID, sub.Attributes.Language, sub.Attributes.Files[0].FileID)
	}
```

(See `examples/search/main.go` for more search options like movie hash or query string.)

### Requesting Download Link

```go
	// Requires prior login
	var fileID = 1234567 // Replace with a valid File ID from search results
	downloadPayload := opensubtitles.DownloadRequest{
		FileID: fileID,
	}
	downloadResp, err := client.Download(context.Background(), downloadPayload)
	if err != nil {
		// Handle error (e.g., quota exceeded - check for specific error types)
	}

	fmt.Printf("Download Link: %s\n", downloadResp.Link)
	fmt.Printf("Remaining downloads: %d\n", downloadResp.Remaining)
	// IMPORTANT: Use an HTTP client to fetch the subtitle content from downloadResp.Link
```

(See `examples/download/main.go` for a runnable example.)

### Uploading Subtitles (XML-RPC)

Uploading uses the separate XML-RPC endpoint and requires its own login flow using an MD5 hash of the password.

1.  **Create an Uploader instance:**

    ```go
    import "github.com/angelospk/opensubtitles-go/upload"

    uploader, err := upload.NewXmlRpcUploader()
    if err != nil {
        // Handle error
    }
    defer uploader.Close()
    ```

2.  **Login via Uploader:**

    ```go
    import (
        "crypto/md5"
        "encoding/hex"
    )

    username := "YOUR_USERNAME"
    password := "YOUR_PASSWORD"
    userAgent := "YourAppName/1.0"

    hasher := md5.New()
    hasher.Write([]byte(password))
    md5Password := hex.EncodeToString(hasher.Sum(nil))

    // Login using the uploader instance
    err = uploader.Login(username, md5Password, "en", userAgent)
    if err != nil {
        // Handle XML-RPC login error
    }
    ```

3.  **Prepare Upload Intent and Upload:**

    ```go
    import "path/filepath"

    intent := upload.UserUploadIntent{
        SubtitleFilePath: "/path/to/your/subtitle.srt",
        SubtitleFileName: filepath.Base("/path/to/your/subtitle.srt"),
        IMDBID:           "1371111", // Example: Inception
        LanguageID:       "en",
        // Optionally provide VideoFilePath for movie hash calculation
        // VideoFilePath:    "/path/to/movie.mkv",
        // VideoFileName:    filepath.Base("/path/to/movie.mkv"),
    }

    subtitleURL, err := uploader.Upload(intent)
    if err != nil {
        // Handle upload error (e.g., upload.ErrUploadDuplicate)
    }

    fmt.Printf("Subtitle uploaded successfully! URL: %s\n", subtitleURL)
    ```

4.  **Logout via Uploader:**

    ```go
    err = uploader.Logout()
    if err != nil {
        // Handle logout error
    }
    ```

(See `examples/upload/main.go` for a complete, runnable upload example.)

## Examples

Runnable examples can be found in the [`examples/`](./examples/) directory:

*   `examples/search/`: Demonstrates searching for subtitles.
*   `examples/download/`: Demonstrates logging in and requesting a download link.
*   `examples/upload/`: Demonstrates the full XML-RPC upload flow.

To run an example (e.g., search):

```bash
cd examples/search
go mod init example.com/search # Run only once per example directory
go mod tidy
go run main.go
```

## Contributing

Contributions are welcome! Please feel free to submit pull requests or open issues.

## License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details. 