package opensubtitles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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
