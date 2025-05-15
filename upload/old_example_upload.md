this example **does not** work:
```go
package main

import (
	// "context"

	"crypto/md5"   // Import crypto/md5
	"encoding/hex" // Import encoding/hex
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath" // Need this for filenames
	"time"

	// "net/http"
	// "strconv"
	// "time"

	"github.com/joho/godotenv"


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

	

	// --- XML-RPC Upload Process (Subtitle Only) ---
	log.Println("\n--- XML-RPC Upload Process (Subtitle Only) --- ")
	// --- Configurable values for the test upload ---
	subtitlePathForUpload := `pkg/core/opensubtitles/testdata/the woman who run 2020.srt` 
	imdbIDForUpload := "11697690"                                                         
	languageIDForUpload := "ell"
	// --- End Configurable values ---

	fmt.Printf("Attempting upload for Sub: %s, IMDB: %s, Lang: %s\n",
		subtitlePathForUpload, imdbIDForUpload, languageIDForUpload)

	// 1. Prepare initial data intent (subtitle only)
	intent := opensubtitles.UserUploadIntent{
		SubtitleFilePath: subtitlePathForUpload,
		SubtitleFileName: getBaseName(subtitlePathForUpload),
		IMDBID:           imdbIDForUpload,
		LanguageID:       languageIDForUpload,
		// Optionally: Comment: "Uploaded via Go XML-RPC Example",
	}

	// 2. Prepare TryUpload parameters (Calculates hashes)
	fmt.Println("Preparing TryUpload parameters...")
	tryParams, err := opensubtitles.PrepareTryUploadParams(intent)
	if err != nil {
		log.Fatalf("Error preparing TryUpload params: %v", err)
	}
	fmt.Printf("[DEBUG] TryUpload Params: %+v\n", tryParams)

	// 3. Modify filename just before the call for uniqueness testing
	uniqueSubFilename := fmt.Sprintf("%s_%d.srt", getBaseName(subtitlePathForUpload), time.Now().UnixNano())
	tryParams.SubFilename = uniqueSubFilename // Use the unique filename

	// 4. Call TryUploadSubtitles
	fmt.Println("Calling TryUploadSubtitles...")
	tryResponse, err := xmlrpcClient.TryUploadSubtitles(*tryParams)
	if err != nil {
		if errors.Is(err, coreerrors.ErrUploadDuplicate) {
			log.Println("Subtitle already exists in the database according to TryUploadSubtitles. Skipping final upload.")
		} else {
			log.Printf("TryUploadSubtitles failed: %v", err)
			fmt.Printf("[DEBUG] TryUpload Params on Failure: %+v\n", tryParams)
			return
		}
	} else {
		log.Printf("TryUploadSubtitles response: Status='%s', Data=%v, AlreadyInDB=%d", tryResponse.Status, tryResponse.Data, tryResponse.AlreadyInDB)

		// 5. Check if TryUpload response indicates we should proceed
		if !tryResponse.Data {
			log.Println("TryUpload response indicates duplicate or issue (Data=false). Skipping final upload.")
		} else {
			// --- UploadSubtitles ---
			fmt.Println("Preparing UploadSubtitles parameters...")
			// Pass the original subtitle file path now
			uploadParams, err := opensubtitles.PrepareUploadSubtitlesParams(*tryParams, intent.SubtitleFilePath)
			if err != nil {
				log.Fatalf("Error preparing UploadSubtitles params: %v", err)
			}
			fmt.Printf("[DEBUG] UploadSubtitles Params: %+v\n", uploadParams)

			fmt.Println("Calling UploadSubtitles...")
			uploadResp, err := xmlrpcClient.UploadSubtitles(*uploadParams)
			if err != nil {
				log.Printf("UploadSubtitles failed: %v", err)
				fmt.Printf("[DEBUG] UploadSubtitles Params on Failure: %+v\n", uploadParams)
				return
			}
			log.Printf("UploadSubtitles successful! Status: %s, URL: %s", uploadResp.Status, uploadResp.Data)
		}
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
```