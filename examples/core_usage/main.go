package main

import (
	// "context"

	"crypto/md5"   // Import crypto/md5
	"encoding/hex" // Import encoding/hex
	"log"

	// "net/http"
	"os"
	"path/filepath" // Need this for filenames

	// "strconv"
	// "time"

	"github.com/joho/godotenv"

	// Use the correct module path for your project
	// Alias for our errors
	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles"
)

// Helper to get base filename
func getBaseName(path string) string {
	return filepath.Base(path)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Could not load .env file:", err)
	}

	// apiKey := os.Getenv("OS_API_KEY") // NOTE: API Key is for REST API, not used by XML-RPC Login
	username := os.Getenv("OS_USERNAME")
	plainPassword := os.Getenv("OS_PASSWORD") // Read plain password

	// --- Calculate MD5 hash of the password ---
	hash := md5.Sum([]byte(plainPassword))
	md5Password := hex.EncodeToString(hash[:])
	// --- END Calculate MD5 hash ---

	// --- DEBUG: Verify loaded credentials ---
	log.Printf("[DEBUG] Using Username: %s, Password MD5: %s", username, md5Password)
	// --- END DEBUG ---

	if username == "" || plainPassword == "" { // Check plain password presence
		log.Fatal("Error: OS_USERNAME, and OS_PASSWORD must be set either in .env file or environment.")
	}

	// --- Setup XML-RPC Client ---
	log.Println("--- Initializing XML-RPC Client ---")
	xmlrpcClient, err := opensubtitles.NewXmlRpcClient()
	if err != nil {
		log.Fatalf("Failed to create XML-RPC client: %v", err)
	}
	log.Println("XML-RPC Client Initialized.")

	// --- XML-RPC Login ---
	log.Println("--- Logging In (XML-RPC) ---")
	// Use the User-Agent from the original library
	userAgent := "opensubtitles-api v5.1.2"
	// Pass the MD5 HASH of the password, not the plain text
	err = xmlrpcClient.Login(username, md5Password, "en", userAgent)
	if err != nil {
		log.Fatalf("XML-RPC Login failed: %v", err)
	}
	log.Printf("XML-RPC Login successful!")

	// --- REST API Calls (Commented Out) ---
	/*
	   log.Println("--- Logging In (REST - Skipped) ---")
	   // ... REST Login ...

	   log.Println("\n--- Getting User Info (REST - Skipped) ---")
	   // ... REST GetUserInfo ...

	   log.Println("\n--- Searching Features (REST - Skipped) ---")
	   // ... REST SearchFeatures ...

	   log.Println("\n--- Searching Subtitles (REST - Skipped) ---")
	   // ... REST SearchSubtitles ...

	   log.Println("\n--- Requesting Download (REST - Skipped) ---")
	   // ... REST RequestDownload ...

	   log.Println("\n--- Uploading Subtitle (REST - Skipped) --- ")
	   // ... REST Upload (now known to be likely incompatible) ...
	*/

	// --- XML-RPC Upload Process ---
	log.Println("\n--- XML-RPC Upload Process --- ")
	// --- Configurable values for the test upload ---
	videoPathForUpload := `pkg/core/opensubtitles/testdata/video.mkv`    // <-- !!! REPLACE WITH A VALID VIDEO FILE PATH (>= 128KB) !!!
	subtitlePathForUpload := `pkg/core/opensubtitles/testdata/dummy.srt` // Using dummy srt for test
	// Note: IMDB ID is often required by TryUpload if hash is unknown
	imdbIDForUpload := "137523" // IMDB ID for Fight Club
	languageIDForUpload := "eng"
	// --- End Configurable values ---

	log.Printf("Attempting upload for Sub: %s, Video: %s, IMDB: %s, Lang: %s",
		subtitlePathForUpload, videoPathForUpload, imdbIDForUpload, languageIDForUpload)

	// 1. Prepare initial data intent
	intent := opensubtitles.UserUploadIntent{
		VideoFilePath:    videoPathForUpload,
		SubtitleFilePath: subtitlePathForUpload,
		IMDBID:           imdbIDForUpload,
		LanguageID:       languageIDForUpload,
		VideoFileName:    getBaseName(videoPathForUpload),
		SubtitleFileName: getBaseName(subtitlePathForUpload),
		// Add other optional flags if desired
		// HearingImpaired: true,
		// ReleaseName: "Example.Release-GRP",
		// Comment: "Uploaded via Go XML-RPC Example",
	}

	// 2. Prepare TryUpload parameters (Calculates hashes)
	log.Println("Preparing TryUpload parameters...")
	tryParams, err := opensubtitles.PrepareTryUploadParams(intent)
	if err != nil {
		log.Fatalf("Failed to prepare TryUpload params: %v", err)
	}

	// 3. Call TryUploadSubtitles
	log.Println("Calling TryUploadSubtitles...")
	tryResp, err := xmlrpcClient.TryUploadSubtitles(*tryParams)
	if err != nil {
		log.Fatalf("TryUploadSubtitles failed: %v", err)
	}
	log.Printf("TryUploadSubtitles response: Status='%s', AlreadyInDB=%d", tryResp.Status, tryResp.AlreadyInDB)

	// 4. Check if already in DB
	if tryResp.AlreadyInDB == 1 {
		log.Println("Subtitle already exists in the database according to TryUploadSubtitles. Skipping final upload.")
	} else {
		log.Println("Subtitle not found in DB by TryUploadSubtitles. Proceeding with UploadSubtitles...")

		// 5. Read and Base64 encode subtitle content
		log.Println("Reading and encoding subtitle content...")
		base64Content, err := opensubtitles.ReadAndEncodeSubtitle(intent.SubtitleFilePath)
		if err != nil {
			log.Fatalf("Failed to read/encode subtitle: %v", err)
		}

		// 6. Prepare UploadSubtitles parameters
		log.Println("Preparing UploadSubtitles parameters...")
		uploadParams, err := opensubtitles.PrepareUploadSubtitlesParams(*tryParams, base64Content)
		if err != nil {
			log.Fatalf("Failed to prepare UploadSubtitles params: %v", err)
		}

		// 7. Call UploadSubtitles
		log.Println("Calling UploadSubtitles...")
		uploadResp, err := xmlrpcClient.UploadSubtitles(*uploadParams)
		if err != nil {
			log.Fatalf("UploadSubtitles failed: %v", err)
		}
		log.Printf("UploadSubtitles successful! Status: %s, URL: %s", uploadResp.Status, uploadResp.Data)
	}

	// --- XML-RPC Logout ---
	log.Println("\n--- Logging Out (XML-RPC) ---")
	err = xmlrpcClient.Logout()
	if err != nil {
		log.Printf("XML-RPC Logout failed: %v", err)
	} else {
		log.Println("XML-RPC Logout successful.")
	}

	log.Println("\n--- Example Finished ---")
}
