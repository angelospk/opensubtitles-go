package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/angelospk/opensubtitles-go/upload"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	// --- Credentials ---

	userAgent := "opensubtitles-api v5.1.2"

	fmt.Println("Enter your OpenSubtitles Username:")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Println("Enter your OpenSubtitles Password (will be MD5 hashed):")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// Calculate MD5 hash of password
	hasher := md5.New()
	hasher.Write([]byte(password))
	md5Password := hex.EncodeToString(hasher.Sum(nil))

	// --- Uploader Initialization & Login ---
	// Get the uploader instance (Assuming NewClient initialized it correctly)
	// We actually need to create a *new* Uploader instance based on the current Uploader interface.
	// The Uploader is stateful (login token) and separate from the REST client state.
	uploader, err := upload.NewXmlRpcUploader() // Create a dedicated XML-RPC uploader
	if err != nil {
		fmt.Printf("Error creating XML-RPC uploader: %v\n", err)
		return
	}
	defer uploader.Close() // Ensure connection is closed

	fmt.Println("\nLogging in via XML-RPC Uploader...")
	// XML-RPC login requires username, MD5(password), language, user agent
	err = uploader.Login(username, md5Password, "en", userAgent)
	if err != nil {
		fmt.Printf("Error logging in via XML-RPC: %v\n", err)
		return
	}
	fmt.Println("XML-RPC Login successful!")

	// --- Upload Details ---
	fmt.Println("\n--- Subtitle Upload ---")
	fmt.Println("Enter the full path to the subtitle file you want to upload:")
	subtitlePath, _ := reader.ReadString('\n')
	subtitlePath = strings.TrimSpace(subtitlePath)

	// Check if subtitle file exists
	if _, err := os.Stat(subtitlePath); os.IsNotExist(err) {
		fmt.Printf("Subtitle file not found: %s\n", subtitlePath)
		logoutUploaderAndExit(uploader)
		return
	}

	fmt.Println("Enter the IMDb ID -ONLY NUMBERS- of the movie/show (e.g., 1371111):")
	imdbIDStr, _ := reader.ReadString('\n')
	imdbIDStr = strings.TrimSpace(imdbIDStr)
	// Validate IMDb ID format (basic check)
	if _, err := strconv.Atoi(imdbIDStr); err != nil {
		fmt.Printf("Invalid IMDb ID format: %s\n", imdbIDStr)
		logoutUploaderAndExit(uploader)
		return
	}

	fmt.Println("Enter the language ID (e.g., en, pob, fre):")
	languageID, _ := reader.ReadString('\n')
	languageID = strings.TrimSpace(languageID)

	// Optionally prompt for video file path to calculate movie hash/size
	// fmt.Println("Enter the full path to the corresponding video file (optional, press Enter to skip):")
	// videoPath, _ := reader.ReadString('\n')
	// videoPath = strings.TrimSpace(videoPath)
	videoPath := "" // Keep it simple for this example

	// --- Prepare Upload Intent ---
	fmt.Println("\nPreparing upload intent...")

	intent := upload.UserUploadIntent{
		SubtitleFilePath: subtitlePath,
		SubtitleFileName: filepath.Base(subtitlePath),
		IMDBID:           imdbIDStr,
		LanguageID:       languageID,
		VideoFilePath:    videoPath, // Optional
		HearingImpaired:  false,
		HighDefinition:   true,
		FPS:              25.0,
		Translator:       "retail",
		Comment:          "official subs",
		// VideoFileName will be set automatically if videoPath is provided (inside uploader logic)
		// Other fields (FPS, ReleaseName, etc.) can be added if needed
		// HearingImpaired: false, // Example boolean
	}
	if videoPath != "" {
		intent.VideoFileName = filepath.Base(videoPath)
	}

	// --- Perform Upload ---
	fmt.Println("\nUploading subtitle...")
	subtitleURL, err := uploader.Upload(intent)

	if err != nil {
		if errors.Is(err, upload.ErrUploadDuplicate) {
			fmt.Println("Upload failed: This subtitle seems to be already in the database.")
		} else {
			fmt.Printf("Error uploading subtitle: %v\n", err)
		}
		logoutUploaderAndExit(uploader)
		return
	}

	// --- Display Upload Result ---
	fmt.Printf("\nUpload successful!\n")
	fmt.Printf("  Subtitle URL: %s\n", subtitleURL)

	// --- Logout Uploader ---
	fmt.Println("\nLogging out XML-RPC Uploader...")
	logoutUploaderAndExit(uploader)
	fmt.Println("XML-RPC Logout successful.")
}

// Helper function to logout uploader and exit
func logoutUploaderAndExit(uploader upload.Uploader) {
	err := uploader.Logout()
	if err != nil && !errors.Is(err, upload.ErrNotLoggedIn) { // Ignore error if already logged out
		fmt.Printf("Error during XML-RPC logout: %v\n", err)
	}
}
