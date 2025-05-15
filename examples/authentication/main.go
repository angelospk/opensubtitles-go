package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/angelospk/opensubtitles-go"
)

// getClient initializes and returns a new OpenSubtitles client.
// Remember to replace YOUR_API_KEY.
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

// exampleLogin demonstrates the login process.
func exampleLogin(client *opensubtitles.Client) {
	log.Println("[INFO] --- Example: Login ---")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	loginParams := opensubtitles.LoginRequest{
		Username: "YOUR_USERNAME", // Replace with actual username
		Password: "YOUR_PASSWORD", // Replace with actual password
	}
	if loginParams.Username == "YOUR_USERNAME" || loginParams.Password == "YOUR_PASSWORD" {
		log.Println("[INFO] Please replace YOUR_USERNAME and YOUR_PASSWORD with actual credentials for the Login example.")
	}

	loginResp, err := client.Login(ctx, loginParams)
	if err != nil {
		log.Printf("Login failed: %v\n", err)
		return
	}

	fmt.Printf("Login successful!\n")
	if len(loginResp.Token) >= 10 {
		fmt.Printf("Token: %s (first 10 chars).....\n", loginResp.Token[:10])
	} else {
		fmt.Printf("Token: %s\n", loginResp.Token)
	}
	fmt.Printf("User ID: %d\n", loginResp.User.UserID)
	fmt.Printf("User Level: %s\n", loginResp.User.Level)
	fmt.Printf("Base URL for subsequent requests: %s\n", loginResp.BaseURL)

	fmt.Printf("Client's current base URL: %s\n", client.GetCurrentBaseURL())
	if client.GetCurrentToken() != nil {
		fmt.Println("Client's token has been set.")
	}
}

// exampleGetUserInfo demonstrates fetching user information.
// This should be called after a successful login for meaningful results.
func exampleGetUserInfo(client *opensubtitles.Client) {
	log.Println("[INFO] --- Example: Get User Info ---")
	if client.GetCurrentToken() == nil {
		log.Println("[WARN] User is not logged in. GetUserInfo might fail or return default data.")
		// To make this a self-contained runnable example that shows success,
		// you might want to call exampleLogin() here with valid credentials first.
		// For now, it proceeds and will likely fail if login hasn't occurred.
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	userInfoResp, err := client.GetUserInfo(ctx)
	if err != nil {
		log.Printf("GetUserInfo failed: %v\n", err)
		return
	}

	fmt.Printf("Fetched User Info:\n")
	fmt.Printf("  User ID: %d\n", userInfoResp.Data.UserID)
	fmt.Printf("  Level: %s\n", userInfoResp.Data.Level)
	fmt.Printf("  Allowed Downloads: %d\n", userInfoResp.Data.AllowedDownloads)
	fmt.Printf("  Downloads Count: %d\n", userInfoResp.Data.DownloadsCount)
	fmt.Printf("  Remaining Downloads: %d\n", userInfoResp.Data.RemainingDownloads)
	fmt.Printf("  VIP: %t\n", userInfoResp.Data.VIP)
}

// exampleLogout demonstrates the logout process.
func exampleLogout(client *opensubtitles.Client) {
	log.Println("[INFO] --- Example: Logout ---")
	if client.GetCurrentToken() == nil {
		log.Println("[INFO] Client token is already nil before Logout call (user likely not logged in or already logged out).")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logoutResp, err := client.Logout(ctx)
	if err != nil {
		log.Printf("Logout API call failed: %v\n", err)
		// The client should clear its token regardless of API error during logout.
	} else {
		fmt.Printf("Logout successful: %s\n", logoutResp.Message)
	}

	if client.GetCurrentToken() == nil {
		fmt.Println("Client's token has been cleared after logout attempt.")
	} else {
		// This case should ideally not be reached if Logout logic in client is correct.
		fmt.Println("[WARN] Client's token is still present after Logout call.")
	}
}

func main() {
	log.Println("[INFO] --- Initializing Client ---")
	client, err := getClient()
	if err != nil {
		log.Fatalf("Error in getClient: %v", err)
		return
	}

	// Run Login example
	exampleLogin(client)
	fmt.Println("-------------------------------------")

	// Run GetUserInfo example (will use token from login if successful)
	// Add a small delay or ensure login is fully processed if there are race conditions with token setting.
	// time.Sleep(1 * time.Second) // Usually not needed with proper client locking
	exampleGetUserInfo(client)
	fmt.Println("-------------------------------------")

	// Run Logout example
	exampleLogout(client)
	fmt.Println("-------------------------------------")

	// Verify token is nil after logout
	log.Println("[INFO] --- Verifying token after all operations ---")
	if client.GetCurrentToken() == nil {
		fmt.Println("Final check: Client token is nil as expected.")
	} else {
		fmt.Println("[WARN] Final check: Client token is NOT nil.")
	}
}
