package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

	fmt.Println("Enter your OpenSubtitles Username:")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Println("Enter your OpenSubtitles Password:")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

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

	// --- Login ---
	fmt.Println("\nLogging in...")
	ctx := context.Background()
	loginParams := opensubtitles.LoginRequest{
		Username: username,
		Password: password,
	}
	loginResp, err := client.Login(ctx, loginParams)
	if err != nil {
		fmt.Printf("Error logging in: %v\n", err)
		return
	}
	fmt.Printf("Login successful! Token: %s..., BaseURL: %s\n", loginResp.Token[:min(10, len(loginResp.Token))], loginResp.BaseURL)

	// --- Download Parameters ---
	fmt.Println("\n--- Subtitle Download ---")
	fmt.Println("Enter the File ID of the subtitle to download (e.g., from search results):")
	fileIDStr, _ := reader.ReadString('\n')
	fileIDStr = strings.TrimSpace(fileIDStr)
	fileID, err := strconv.Atoi(fileIDStr)
	if err != nil {
		fmt.Printf("Invalid File ID format: %s\n", fileIDStr)
		logoutAndExit(ctx, client)
		return
	}

	fmt.Println("Enter the desired local filename (optional, press Enter to use server name):")
	localFilename, _ := reader.ReadString('\n')
	localFilename = strings.TrimSpace(localFilename)

	// --- Prepare Download Request ---

	payload := opensubtitles.DownloadRequest{
		FileID: fileID,
	}
	if localFilename != "" {
		fn := localFilename // Create local var
		payload.FileName = &fn
	}

	// --- Perform Download Request ---
	fmt.Println("\nRequesting download link...")
	downloadResp, err := client.Download(ctx, payload)

	if err != nil {
		fmt.Printf("Error requesting download: %v\n", err)
		logoutAndExit(ctx, client)
		return
	}

	// --- Display Download Info ---
	fmt.Printf("\nDownload request successful!\n")
	fmt.Printf("  Download Link: %s\n", downloadResp.Link)
	fmt.Printf("  Server Filename: %s\n", downloadResp.FileName)
	fmt.Printf("  Remaining Downloads: %d\n", downloadResp.Remaining)
	fmt.Printf("  Quota Reset Time: %s (%s)\n", downloadResp.ResetTimeUTC.Format(time.RFC1123), downloadResp.ResetTime)

	fmt.Println("\nIMPORTANT: You need to use an HTTP client (like Go's net/http)")
	fmt.Println("to actually download the file from the provided link.")
	fmt.Println("This example only retrieves the download metadata.")

	// --- Logout ---
	logoutAndExit(ctx, client)
}

// Helper function min for Go versions < 1.21
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper function to logout and exit
func logoutAndExit(ctx context.Context, client *opensubtitles.Client) {
	fmt.Println("\nLogging out...")
	_, err := client.Logout(ctx) // Ignore the response
	if err != nil {
		fmt.Printf("Error logging out: %v\n", err)
	}
}
