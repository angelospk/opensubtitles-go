# Authentication Examples

This directory contains examples demonstrating how to use the authentication-related methods (Login, GetUserInfo, Logout) in the `opensubtitles-go` library.

## Files

- `main.go`: A runnable Go program showcasing the authentication flows.
- `README.md`: This file.

## How to Run

1.  **Replace Placeholders**: Before running, you **must** edit `main.go` and replace the following placeholder values:
    *   `YOUR_API_KEY` with your actual OpenSubtitles API key.
    *   `YOUR_USERNAME` and `YOUR_PASSWORD` with your OpenSubtitles account credentials for the login example.
2.  **Navigate to Directory**: Open your terminal and change to this directory (`examples/authentication`).
3.  **Run the Program**: Execute `go run main.go`.

## What the Example Does

The `main.go` program demonstrates:

1.  **Client Initialization**: How to create a new OpenSubtitles client with your API key and a user agent.
2.  **Login**: Attempts to log in using the provided username and password.
    *   Prints selected user information and the received token upon successful login.
    *   Shows how the client internally stores the token and updates its base URL.
3.  **Get User Info**: After a simulated login (or if a token is already present from a previous run within the same client instance life), it attempts to fetch and display information about the authenticated user.
    *   This part will only succeed if the login was successful or a valid token is available.
4.  **Logout**: Attempts to log out the user and clears the client's internal token.

## Expected Output (Illustrative)

If run with valid credentials, the output will be similar to:

```text
[INFO] --- Initializing Client ---
[INFO] Please replace YOUR_API_KEY with your actual OpenSubtitles API key.
[INFO] Client initialized.
[INFO] --- Example: Login ---
[INFO] Please replace YOUR_USERNAME and YOUR_PASSWORD with actual credentials for the Login example.
Login successful!
Token: eyJhbGciOi (first 10 chars).....
User ID: 123456
User Level: Sub leecher
Base URL for subsequent requests: https://vip-api.opensubtitles.com/api/v1
Client's current base URL: https://vip-api.opensubtitles.com/api/v1
Client's token has been set.
[INFO] --- Example: Get User Info (after login) ---
Fetched User Info:
User ID: 123456
Level: Sub leecher
Allowed Downloads: 20
Downloads Count: 5
Remaining Downloads: 15
VIP: false
[INFO] --- Example: Logout ---
Logout successful: User logged out
Client's token has been cleared after logout attempt.
```

**Note**: If placeholder values are not replaced, the login step will fail, and subsequent steps requiring authentication will also likely fail or be skipped. 