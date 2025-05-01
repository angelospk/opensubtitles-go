package main

import (
	// "context"

	"log"
	// "net/http"
	"os"
	// "strconv"
	// "time"

	"github.com/joho/godotenv"

	// Use the correct module path for your project
	// Alias for our errors
	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Could not load .env file:", err)
	}

	// apiKey := os.Getenv("OS_API_KEY") // NOTE: API Key is for REST API, likely not used by XML-RPC Login
	username := os.Getenv("OS_USERNAME")
	password := os.Getenv("OS_PASSWORD")

	if username == "" || password == "" {
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
	// XML-RPC Login often uses 'en' and a UserAgent string.
	// Let's construct a basic UserAgent.
	// TODO: Get version dynamically later.
	userAgent := "GoOsuploaderGuiExample/0.1"
	err = xmlrpcClient.Login(username, password, "en", userAgent)
	if err != nil {
		log.Fatalf("XML-RPC Login failed: %v", err)
	}
	log.Printf("XML-RPC Login successful!")

	// --- REST Client Setup (Keep for potential future use or comparison) ---
	/*
	   httpClient := &http.Client{
	       Timeout: 30 * time.Second, // Example timeout
	   }
	   restClient := opensubtitles.NewClient(apiKey, httpClient)
	   ctx := context.Background()
	*/

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

	// --- TODO: Implement XML-RPC Upload Test Here ---
	log.Println("\n--- XML-RPC Upload (Not Yet Implemented) ---")
	// Steps:
	// 1. Define TryUploadSubtitles and UploadSubtitles methods in xmlrpc_client.go
	// 2. Define necessary structs for parameters and responses.
	// 3. Implement parameter preparation (hashing, base64, bool->"1"/"0")
	// 4. Call TryUploadSubtitles
	// 5. Parse response, potentially extract imdb id
	// 6. Call UploadSubtitles
	// 7. Check final response
	log.Println("Upload via XML-RPC needs implementation.")

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
