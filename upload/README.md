# OpenSubtitles XML-RPC Upload Client

This directory (`upload/`) contains a Go client implementation for interacting with the OpenSubtitles **XML-RPC API**, specifically for uploading subtitles. This is distinct from the main REST API client provided in the parent `opensubtitles-go` library.

**Warning: The OpenSubtitles XML-RPC API is an older API. While it might still be functional for uploads, the modern REST API (OpenSubtitles.com API v2018) is generally recommended for other operations. This upload client is provided for users who specifically need to use the XML-RPC upload functionality.**

## Files

- `uploader.go`: Contains the main logic for the XML-RPC client, including methods like `Login`, `Logout`, `TryUploadSubtitles`, and `UploadSubtitles`.
- `helpers.go`: Provides helper functions for preparing parameters, calculating hashes, and encoding data for the XML-RPC calls.
- `types.go`: Defines the Go structs that map to the XML-RPC request and response structures for the upload-related methods.
- `README.md`: This file.

## Functionality

The client in this directory aims to provide the following capabilities via XML-RPC:

1.  **Login (`Login`)**: Authenticates a user against the OpenSubtitles XML-RPC service and obtains a session token.
2.  **Try Upload Subtitles (`TryUploadSubtitles` or `tryUploadSubtitles`)**: 
    *   Checks if a subtitle (identified by hashes and filenames) already exists in the database.
    *   Verifies if the provided movie information (IMDb ID, etc.) is known.
    *   This step is typically performed before the actual upload to avoid duplicates and ensure metadata alignment.
3.  **Upload Subtitles (`UploadSubtitles`)**: 
    *   Uploads the actual subtitle file content (base64 encoded) along with all its metadata (movie details, subtitle language, comments, etc.).
    *   This is done after a successful `TryUploadSubtitles` call indicates the subtitle is new or can be updated.
4.  **Logout (`Logout`)**: Invalidates the session token.

## Usage (Conceptual)

Using this XML-RPC client typically involves the following sequence:

```go
package main

import (
	"log"
	"path/filepath"

	"your_project_module_path/upload" // Adjust import path
)

func main() {
	// 1. Initialize the XML-RPC client
	//    (Refer to NewClient in uploader.go for initialization details - typically just a user agent)
	cfg := upload.Config{
		UserAgent: "MyUploaderApp/1.0",
		// Debug: true, // Optional: for verbose logging of XML-RPC requests/responses
	}
	client, err := upload.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create XML-RPC client: %v", err)
	}
	defer client.Close()

	// 2. Login
	token, err := client.Login("your_username", "your_password")
	if err != nil {
		log.Fatalf("XML-RPC Login failed: %v", err)
	}
	log.Printf("Logged in successfully, token: %s...\n", token[:10])

	// 3. Prepare Upload Intent (User-defined struct gathering all necessary info)
	subtitlePath := "path/to/your/subtitle.srt"
	videoPath := "path/to/your/video.mkv" // Optional, but helps with hashing

	intent := upload.UserUploadIntent{
		SubtitleFilePath:   subtitlePath,
		SubtitleFileName:   filepath.Base(subtitlePath),
		VideoFilePath:      videoPath, // Can be empty if video info supplied differently
		VideoFileName:      filepath.Base(videoPath),
		IMDBID:             "tt1375666", // Example: Inception
		LanguageID:         "eng",       // 3-letter ISO 639-2/B code
		MovieReleaseName:   "Inception.2010.1080p.BluRay.x264-YIFY",
		Comment:            "My first upload!",
		HearingImpaired:    false,
		// ... other fields from UserUploadIntent
	}

	// 4. Try Upload (check if subtitle exists, get parameters for actual upload)
	//    The tryUploadSubtitles (unexported) or a similar public method would be called.
	//    Let's assume there's a public wrapper or use the internal flow for concept.
	tryUploadParams, err := upload.PrepareTryUploadParams(intent) // This is a helper, not a direct API call
	if err != nil {
		log.Fatalf("Failed to prepare TryUpload params: %v", err)
	}

	// The actual client.tryUploadSubtitles call would use these params.
	// tryResponse, err := client.tryUploadSubtitles(tryUploadParams)
	// For this conceptual example, we'll assume a flow where `PerformUpload` handles this.

	// 5. Perform Upload (which internally calls TryUpload and then UploadSubtitles)
	subtitleURL, err := client.PerformUpload(intent)
	if err != nil {
		log.Fatalf("PerformUpload failed: %v", err)
	}
	log.Printf("Subtitle uploaded successfully! URL: %s\n", subtitleURL)

	// 6. Logout
	if err := client.Logout(); err != nil {
		log.Printf("XML-RPC Logout failed: %v", err)
	} else {
		log.Println("Logged out successfully.")
	}
}
```

## Important Considerations

*   **Error Handling**: The XML-RPC API can return errors in various ways. Robust error handling is crucial.
*   **Hashing**: Correctly calculating MD5 hashes for subtitle files and (OpenSubtitles) hashes for video files is essential for the `TryUploadSubtitles` step.
*   **Parameter Formatting**: XML-RPC is strict about parameter types and structures. The `types.go` and `helpers.go` files are critical for ensuring correct formatting.
*   **API Rate Limits**: Be mindful of API rate limits, though they might be less strictly enforced on the older XML-RPC API compared to the REST API.
*   **Alternative**: If you are building a new application, consider if uploading via the website or other community tools meets your needs, as direct API upload can be complex.

This client provides a Go-native way to interact with the upload mechanism if programmatic upload via XML-RPC is a requirement. 