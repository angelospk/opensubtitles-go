# Subtitles Search and Download Examples

This directory contains an example demonstrating how to search for subtitles and then request a download link for a found subtitle using the `opensubtitles-go` library.

## Files

- `main.go`: A runnable Go program showcasing subtitles search and download link request.
- `README.md`: This file.

## How to Run

1.  **Replace Placeholders**: Before running, you **must** edit `main.go` and replace `YOUR_API_KEY` with your actual OpenSubtitles API key in the `getClient()` function.
2.  **User Credentials (Optional but Recommended for Download)**: For the download part of the example to reflect actual user quotas, it's best if the client is authenticated. The example attempts to log in using placeholder credentials (`YOUR_USERNAME`, `YOUR_PASSWORD`). Replace these in the `exampleLogin()` function within `main.go` if you want to test authenticated downloads.
3.  **Navigate to Directory**: Open your terminal and change to this directory (`examples/subtitles_search_download`).
4.  **Run the Program**: Execute `go run main.go`.

## What the Example Does

The `main.go` program demonstrates a sequence of operations:

1.  **Client Initialization**: Creates an OpenSubtitles client.
2.  **Login (Attempt)**: Tries to log in with placeholder credentials. Replace these for actual authenticated testing.
3.  **Search Subtitles**: Performs a search for subtitles (e.g., for "inception 2010" in English).
    *   Prints the total number of subtitles found and details for the first few results, including their `FileID` which is crucial for downloading.
4.  **Request Download Link**: If subtitles are found, it takes the `FileID` of the first result and attempts to get a download link for it.
    *   Prints the download link, filename, and any messages from the API (like remaining quota).

## Expected Output (Illustrative)

If run with a valid API key (and optionally, valid user credentials for login), the output will be similar to:

```text
[INFO] --- Initializing Client ---
[INFO] Please replace YOUR_API_KEY with your actual OpenSubtitles API key for the examples to work correctly.
[INFO] Client initialized.
[INFO] --- Attempting Login (Optional for Search, Recommended for Download) ---
[INFO] Please replace YOUR_USERNAME and YOUR_PASSWORD with actual credentials for the Login example.
Login failed: POST /login: 401 Unauthorized (Invalid username or password)
-------------------------------------
[INFO] --- Example: Search Subtitles ---
[INFO] Searching subtitles with params: Query:'inception 2010' Languages:'en' ...
Found 97 subtitles (Page 1/5, Total: 97)
--- Subtitle 1 ---
  ID: 1954939103, Type: subtitle
  Lang: en, Release: Inception.2010.1080p.BluRay.x264.YIFY
  FPS: 23.976, HD: true, HI: false
  Downloads: 63111, URL: https://www.opensubtitles.com/en/subtitles/1954939103/inception-inception.2010.1080p.bluray.x264.yify-en
  File ID for download: 1954939103, File Name: Inception.2010.1080p.BluRay.x264.YIFY.srt
...
-------------------------------------
[INFO] --- Example: Request Download Link for FileID: 1954939103 ---
[WARN] User is not logged in. Download might fail or be restricted.
Download Request Successful:
  Download Link: https://dl.opensubtitles.org/en/download/file/1954939103.srt?token=...
  File Name: Inception.2010.1080p.BluRay.x264.YIFY.srt
  Remaining Downloads for user: 5
  Message: 
  Rate limit reset time: 2024-07-30 12:00:00 +0000 UTC (UTC: 2024-07-30T12:00:00Z)
-------------------------------------
```

**Notes**:
*   The login step is included to show a more complete flow, especially relevant for download quotas. Searching subtitles might not always require login.
*   The actual subtitle results, `FileID`, and download link will vary based on current API data.
*   If login fails due to placeholder credentials, the download request will be made anonymously, which might have different rate limits or quota implications. 