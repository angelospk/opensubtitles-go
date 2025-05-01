package opensubtitles

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreErrors "github.com/angelospk/osuploadergui/pkg/core/errors"
)

func TestClient_Login_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/login" {
			t.Errorf("Expected path /api/v1/login, got %s", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("Api-Key") != "test-api-key" {
			t.Errorf("Expected Api-Key header 'test-api-key', got '%s'", r.Header.Get("Api-Key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Check request body
		var reqBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if reqBody["username"] != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", reqBody["username"])
		}
		if reqBody["password"] != "testpass" {
			t.Errorf("Expected password 'testpass', got '%s'", reqBody["password"])
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"user":   map[string]interface{}{"user_id": 123, "level": "Sub VIP", "username": "testuser"},
			"token":  "fake-jwt-token",
			"status": 200,
		})
	}))
	defer server.Close()

	// Create client
	client := NewClient("test-api-key", server.Client())
	client.baseURL = server.URL + "/api/v1" // Point client to mock server

	// Call Login
	loginResp, err := client.Login(context.Background(), "testuser", "testpass")

	// Assertions
	if err != nil {
		t.Fatalf("Login returned an unexpected error: %v", err)
	}

	if loginResp == nil {
		t.Fatal("Login returned nil response")
	}

	if loginResp.Token != "fake-jwt-token" {
		t.Errorf("Expected token 'fake-jwt-token', got '%s'", loginResp.Token)
	}
	if client.jwtToken != "fake-jwt-token" { // Check internal state
		t.Errorf("Client token not set correctly, expected 'fake-jwt-token', got '%s'", client.jwtToken)
	}
	if loginResp.User.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", loginResp.User.Username)
	}
	if loginResp.Status != 200 {
		t.Errorf("Expected status 200, got %d", loginResp.Status)
	}
}

func TestClient_Login_Failure(t *testing.T) {
	// Mock server for 401 Unauthorized
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/login" {
			t.Errorf("Expected path /api/v1/login, got %s", r.URL.Path)
		}

		// Send 401 response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Errors: []string{"Unauthorized"},
			Status: http.StatusUnauthorized,
		})
	}))
	defer server.Close()

	// Create client
	client := NewClient("test-api-key", server.Client())
	client.baseURL = server.URL + "/api/v1" // Point client to mock server

	// Call Login with incorrect credentials
	loginResp, err := client.Login(context.Background(), "testuser", "wrongpass")

	// Assertions
	if err == nil {
		t.Fatal("Login did not return an error on failure")
	}

	if loginResp != nil {
		t.Errorf("Login returned a non-nil response on failure: %+v", loginResp)
	}

	// Check if the error is the expected type (coreErrors.ErrUnauthorized)
	if !errors.Is(err, coreErrors.ErrUnauthorized) {
		t.Fatalf("Expected error coreErrors.ErrUnauthorized, got %T: %v", err, err)
	}

	// Ensure token was not set
	if client.jwtToken != "" {
		t.Errorf("Client token was set on failed login: %s", client.jwtToken)
	}
}

func TestClient_Logout_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != http.MethodDelete {
			t.Errorf("Expected method DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/logout" {
			t.Errorf("Expected path /api/v1/logout, got %s", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("Api-Key") != "test-api-key" {
			t.Errorf("Expected Api-Key header 'test-api-key', got '%s'", r.Header.Get("Api-Key"))
		}
		expectedAuth := "Bearer fake-jwt-token"
		if r.Header.Get("Authorization") != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, r.Header.Get("Authorization"))
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "User logged out", // Example response, adjust if API differs
			"status":  200,
		})
	}))
	defer server.Close()

	// Create client and simulate logged-in state
	client := NewClient("test-api-key", server.Client())
	client.baseURL = server.URL + "/api/v1" // Point client to mock server
	client.jwtToken = "fake-jwt-token"      // Simulate being logged in

	// Call Logout
	err := client.Logout(context.Background())

	// Assertions
	if err != nil {
		t.Fatalf("Logout returned an unexpected error: %v", err)
	}

	// Ensure token was cleared
	if client.jwtToken != "" {
		t.Errorf("Client token was not cleared after logout: %s", client.jwtToken)
	}
}

func TestClient_GetUserInfo_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/infos/user" {
			t.Errorf("Expected path /api/v1/infos/user, got %s", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("Api-Key") != "test-api-key" {
			t.Errorf("Expected Api-Key header 'test-api-key', got '%s'", r.Header.Get("Api-Key"))
		}
		expectedAuth := "Bearer fake-jwt-token"
		if r.Header.Get("Authorization") != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "" { // GET requests shouldn't have Content-Type
			t.Errorf("Unexpected Content-Type header for GET request: %s", r.Header.Get("Content-Type"))
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Use the existing UserInfo struct for the response body
		json.NewEncoder(w).Encode(UserInfo{
			AllowedDownloads: 100,
			Level:            "Sub God",
			UserID:           987,
			ExtInstalled:     true,
			Vip:              true,
			DownloadsCount:   50,
			Username:         "logged_in_user",
		})
	}))
	defer server.Close()

	// Create client and simulate logged-in state
	client := NewClient("test-api-key", server.Client())
	client.baseURL = server.URL + "/api/v1" // Point client to mock server
	client.jwtToken = "fake-jwt-token"      // Simulate being logged in

	// Call GetUserInfo
	userInfo, err := client.GetUserInfo(context.Background())

	// Assertions
	if err != nil {
		t.Fatalf("GetUserInfo returned an unexpected error: %v", err)
	}

	if userInfo == nil {
		t.Fatal("GetUserInfo returned nil response")
	}

	// Check returned user info fields
	if userInfo.UserID != 987 {
		t.Errorf("Expected UserID 987, got %d", userInfo.UserID)
	}
	if userInfo.Username != "logged_in_user" {
		t.Errorf("Expected Username 'logged_in_user', got '%s'", userInfo.Username)
	}
	if userInfo.Level != "Sub God" {
		t.Errorf("Expected Level 'Sub God', got '%s'", userInfo.Level)
	}
	if !userInfo.Vip {
		t.Error("Expected Vip to be true")
	}
}

func TestClient_SearchFeatures_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/features" {
			t.Errorf("Expected path /api/v1/features, got %s", r.URL.Path)
		}

		// Check query parameters
		expectedQuery := "query=fight+club"
		if r.URL.RawQuery != expectedQuery {
			t.Errorf("Expected query '%s', got '%s'", expectedQuery, r.URL.RawQuery)
		}

		// Check headers
		if r.Header.Get("Api-Key") != "test-api-key" {
			t.Errorf("Expected Api-Key header 'test-api-key', got '%s'", r.Header.Get("Api-Key"))
		}
		if r.Header.Get("Authorization") != "" { // Should not be authenticated
			t.Errorf("Unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(FeaturesResponse{
			TotalPages: 1,
			TotalCount: 1,
			Page:       1,
			Data: []Feature{
				{
					ID:   "12345",
					Type: "movie", // Type might be "feature" or specific like "movie"
					Attributes: FeatureAttributes{
						Title:     "Fight Club",
						FeatureID: "12345",
						Year:      1999,
						ImdbID:    137523,
					},
				},
			},
		})
	}))
	defer server.Close()

	// Create client
	client := NewClient("test-api-key", server.Client())
	client.baseURL = server.URL + "/api/v1" // Point client to mock server

	// Call SearchFeatures
	params := map[string]string{
		"query": "fight club",
	}
	featuresResp, err := client.SearchFeatures(context.Background(), params)

	// Assertions
	if err != nil {
		t.Fatalf("SearchFeatures returned an unexpected error: %v", err)
	}

	if featuresResp == nil {
		t.Fatal("SearchFeatures returned nil response")
	}

	if featuresResp.TotalCount != 1 {
		t.Errorf("Expected TotalCount 1, got %d", featuresResp.TotalCount)
	}
	if len(featuresResp.Data) != 1 {
		t.Fatalf("Expected 1 feature in Data, got %d", len(featuresResp.Data))
	}

	feature := featuresResp.Data[0]
	if feature.ID != "12345" {
		t.Errorf("Expected feature ID '12345', got '%s'", feature.ID)
	}
	if feature.Attributes.Title != "Fight Club" {
		t.Errorf("Expected feature title 'Fight Club', got '%s'", feature.Attributes.Title)
	}
	if feature.Attributes.Year != 1999 {
		t.Errorf("Expected feature year 1999, got %d", feature.Attributes.Year)
	}
}

func TestClient_SearchSubtitles_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != http.MethodGet {
			t.Errorf("Expected method GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/subtitles" {
			t.Errorf("Expected path /api/v1/subtitles, got %s", r.URL.Path)
		}

		// Check query parameters (order might vary, so check individually)
		query := r.URL.Query()
		if query.Get("imdb_id") != "137523" {
			t.Errorf("Expected query param imdb_id=137523, got '%s'", query.Get("imdb_id"))
		}
		if query.Get("languages") != "en" {
			t.Errorf("Expected query param languages=en, got '%s'", query.Get("languages"))
		}
		if len(query) != 2 {
			t.Errorf("Expected 2 query parameters, got %d: %v", len(query), query)
		}

		// Check headers (similar to SearchFeatures)
		if r.Header.Get("Api-Key") != "test-api-key" {
			t.Errorf("Expected Api-Key header 'test-api-key', got '%s'", r.Header.Get("Api-Key"))
		}
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Unexpected Authorization header: %s", r.Header.Get("Authorization"))
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SubtitleSearchResponse{
			TotalPages: 1,
			TotalCount: 1,
			Page:       1,
			Data: []Subtitle{
				{
					ID:   "sub123",
					Type: "subtitle",
					Attributes: SubtitleAttributes{
						SubtitleID: "sub123",
						Language:   "en",
						Release:    "Fight.Club.1999.BluRay.DTS.x264-CtrlHD",
						Files: []SubtitleFile{
							{FileID: 9876, FileName: "fight_club_eng.srt", CDNumber: 1},
						},
						FeatureDetails: FeatureInfo{
							ImdbID: 137523,
							Title:  "Fight Club",
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	// Create client
	client := NewClient("test-api-key", server.Client())
	client.baseURL = server.URL + "/api/v1" // Point client to mock server

	// Call SearchSubtitles
	params := map[string]string{
		"imdb_id":   "137523",
		"languages": "en",
	}
	subtitlesResp, err := client.SearchSubtitles(context.Background(), params)

	// Assertions
	if err != nil {
		t.Fatalf("SearchSubtitles returned an unexpected error: %v", err)
	}

	if subtitlesResp == nil {
		t.Fatal("SearchSubtitles returned nil response")
	}

	if subtitlesResp.TotalCount != 1 {
		t.Errorf("Expected TotalCount 1, got %d", subtitlesResp.TotalCount)
	}
	if len(subtitlesResp.Data) != 1 {
		t.Fatalf("Expected 1 subtitle in Data, got %d", len(subtitlesResp.Data))
	}

	subtitle := subtitlesResp.Data[0]
	if subtitle.ID != "sub123" {
		t.Errorf("Expected subtitle ID 'sub123', got '%s'", subtitle.ID)
	}
	if subtitle.Attributes.Language != "en" {
		t.Errorf("Expected subtitle language 'en', got '%s'", subtitle.Attributes.Language)
	}
	if len(subtitle.Attributes.Files) != 1 || subtitle.Attributes.Files[0].FileID != 9876 {
		t.Errorf("Expected 1 file with ID 9876, got %v", subtitle.Attributes.Files)
	}
	if subtitle.Attributes.FeatureDetails.ImdbID != 137523 {
		t.Errorf("Expected feature IMDB ID 137523, got %d", subtitle.Attributes.FeatureDetails.ImdbID)
	}
}

func TestClient_RequestDownload_Success(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/download" {
			t.Errorf("Expected path /api/v1/download, got %s", r.URL.Path)
		}

		// Check headers
		if r.Header.Get("Api-Key") != "test-api-key" {
			t.Errorf("Expected Api-Key header 'test-api-key', got '%s'", r.Header.Get("Api-Key"))
		}
		expectedAuth := "Bearer fake-jwt-token"
		if r.Header.Get("Authorization") != expectedAuth {
			t.Errorf("Expected Authorization header '%s', got '%s'", expectedAuth, r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Check request body
		var reqBody DownloadRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if reqBody.FileID != 9876 {
			t.Errorf("Expected FileID 9876 in request body, got %d", reqBody.FileID)
		}
		// Optionally check for other fields if testing with them (e.g., FileName)
		if reqBody.FileName != "custom_name.srt" {
			t.Errorf("Expected FileName 'custom_name.srt' in request body, got '%s'", reqBody.FileName)
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(DownloadResponse{
			Link:      "https://dl.opensubtitles.com/some/temporary/link",
			FileName:  "custom_name.srt",
			Remaining: 99,
			Message:   "Download count successful.",
			Status:    200,
			Allowed:   100,
		})
	}))
	defer server.Close()

	// Create client and simulate logged-in state
	client := NewClient("test-api-key", server.Client())
	client.baseURL = server.URL + "/api/v1" // Point client to mock server
	client.jwtToken = "fake-jwt-token"      // Simulate being logged in

	// Call RequestDownload
	downloadReq := DownloadRequest{
		FileID:   9876,
		FileName: "custom_name.srt", // Include optional param in test
	}
	downloadResp, err := client.RequestDownload(context.Background(), downloadReq)

	// Assertions
	if err != nil {
		t.Fatalf("RequestDownload returned an unexpected error: %v", err)
	}

	if downloadResp == nil {
		t.Fatal("RequestDownload returned nil response")
	}

	if downloadResp.Link == "" {
		t.Error("Expected non-empty download link")
	}
	if downloadResp.Link != "https://dl.opensubtitles.com/some/temporary/link" {
		t.Errorf("Expected Link '%s', got '%s'", "https://dl.opensubtitles.com/some/temporary/link", downloadResp.Link)
	}
	if downloadResp.Remaining != 99 {
		t.Errorf("Expected Remaining 99, got %d", downloadResp.Remaining)
	}
	if downloadResp.FileName != "custom_name.srt" {
		t.Errorf("Expected FileName 'custom_name.srt', got '%s'", downloadResp.FileName)
	}
}

func TestClient_UploadSubtitle_Success(t *testing.T) {
	dummyFilePath := filepath.Join("testdata", "dummy.srt")

	// Mock server - This needs to parse multipart form data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method, path, auth, API key (similar to other tests)
		if r.Method != http.MethodPost {
			t.Errorf("Expected method POST, got %s", r.Method)
			return
		}
		if r.URL.Path != "/api/v1/upload" {
			t.Errorf("Expected path /api/v1/upload, got %s", r.URL.Path)
			return
		}
		if r.Header.Get("Api-Key") != "test-api-key" {
			t.Errorf("Expected Api-Key 'test-api-key', got '%s'", r.Header.Get("Api-Key"))
			return
		}
		if r.Header.Get("Authorization") != "Bearer fake-jwt-token" {
			t.Errorf("Expected Authorization 'Bearer fake-jwt-token', got '%s'", r.Header.Get("Authorization"))
			return
		}

		// Parse the multipart form
		err := r.ParseMultipartForm(32 << 20) // 32MB max memory
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to parse multipart form: %v", err), http.StatusBadRequest)
			return
		}
		defer r.MultipartForm.RemoveAll()

		// --- Verify Metadata Fields --- (Adjust names based on ACTUAL API spec)
		expectedFields := map[string]string{
			"feature_id":       "54321",
			"language":         "en",
			"filename":         "dummy_upload.srt",
			"video_filename":   "movie.mkv",
			"moviehash":        "aabbccddeeff0011",
			"movie_bytesize":   "1234567890",
			"hearing_impaired": "true", // Assuming string booleans for test
		}
		for key, expectedValue := range expectedFields {
			if r.MultipartForm.Value[key][0] != expectedValue {
				t.Errorf("Expected form field '%s' to be '%s', got '%s'", key, expectedValue, r.MultipartForm.Value[key][0])
			}
		}

		// --- Verify File Content --- (Adjust form file key based on ACTUAL API spec)
		formFile, fileHeader, err := r.FormFile("file")
		if err != nil {
			t.Errorf("Expected form file field 'file', but got error: %v", err)
			return
		}
		defer formFile.Close()

		if fileHeader.Filename != "dummy_upload.srt" {
			t.Errorf("Expected uploaded filename to be 'dummy_upload.srt', got '%s'", fileHeader.Filename)
		}

		// Read uploaded content and compare with original dummy file
		uploadedContentBytes, err := io.ReadAll(formFile)
		if err != nil {
			t.Errorf("Failed to read uploaded file content: %v", err)
			return
		}
		originalContentBytes, err := os.ReadFile(dummyFilePath)
		if err != nil {
			t.Fatalf("Failed to read original dummy file for comparison: %v", err)
		}

		if !bytes.Equal(uploadedContentBytes, originalContentBytes) {
			t.Errorf("Uploaded file content does not match original dummy file content")
		}

		// Send successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(UploadResponse{
			Message: "Subtitle uploaded successfully",
			Link:    "https://www.opensubtitles.com/subtitles/123456",
			Data: struct {
				SubtitleID int64 `json:"subtitle_id"`
				FileID     int64 `json:"file_id"`
			}{
				SubtitleID: 654321,
				FileID:     987654,
			},
		})
	}))
	defer server.Close()

	// Create client and simulate logged-in state
	client := NewClient("test-api-key", server.Client())
	client.baseURL = server.URL + "/api/v1" // Point client to mock server
	client.jwtToken = "fake-jwt-token"

	// Prepare upload params
	params := UploadParams{
		FeatureID:       54321,
		Language:        "en",
		FileName:        "dummy_upload.srt", // Use a different name than the source file for testing
		VideoFileName:   "movie.mkv",
		Moviehash:       "aabbccddeeff0011",
		MovieByteSize:   1234567890,
		HearingImpaired: true,
	}

	// Call UploadSubtitle
	uploadResp, err := client.UploadSubtitle(context.Background(), params, dummyFilePath)

	// Assertions
	if err != nil {
		t.Fatalf("UploadSubtitle returned an unexpected error: %v", err)
	}
	if uploadResp == nil {
		t.Fatal("UploadSubtitle returned nil response")
	}

	if uploadResp.Data.SubtitleID != 654321 {
		t.Errorf("Expected SubtitleID 654321, got %d", uploadResp.Data.SubtitleID)
	}
	if uploadResp.Data.FileID != 987654 {
		t.Errorf("Expected FileID 987654, got %d", uploadResp.Data.FileID)
	}
	if !strings.Contains(uploadResp.Message, "success") {
		t.Errorf("Expected success message, got: %s", uploadResp.Message)
	}
}
