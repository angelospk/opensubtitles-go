# Download Subtitle File Example

This directory contains an example (`main.go`) demonstrating how to use the `opensubtitles-go` library to first get a download link for a subtitle and then actually download the subtitle file content.

## Files

- `main.go`: A runnable Go program showcasing the subtitle download process.
- `README.md`: This file.

## How to Run

1.  **Replace Placeholders**: 
    *   Edit `main.go` and replace `YOUR_API_KEY` in the `getClient()` function (or equivalent client initialization).
    *   You will likely need to provide valid OpenSubtitles `USERNAME` and `PASSWORD` for login, as downloading usually requires authentication and is subject to quotas.
    *   The example will need a `SUBTITLE_FILE_ID` to download. You might need to run a search example first (like `examples/subtitles_search_download/main.go`) to get a valid File ID, then hardcode it or pass it as input to this example.
2.  **Navigate to Directory**: Open your terminal and change to `examples/download`.
3.  **Run the Program**: Execute `go run main.go`.

## What the Example Does (Assumed Functionality)

The `main.go` program likely performs the following steps:

1.  **Client Initialization**: Sets up the OpenSubtitles client with API key and user agent.
2.  **Login**: Logs in the user with provided credentials to ensure download permissions and correct quota tracking.
3.  **Request Download Link**: Uses a known `SUBTITLE_FILE_ID` to call the `Download` (or `RequestDownload`) method to get a temporary download URL for the subtitle file.
4.  **Fetch Subtitle Content**: Makes an HTTP GET request to the obtained download URL.
5.  **Save or Display Content**: Saves the fetched subtitle content to a local file (e.g., `downloaded_subtitle.srt`) or prints a portion of it to the console.

## Expected Output

If successful, the program should:
*   Log the steps (login, requesting download link, fetching content).
*   Either save a subtitle file (e.g., `.srt`) in the current directory or print some of its content.
*   Indicate success or any errors encountered during the process.

**Example Console Output (Conceptual):**
```text
[INFO] Initializing client...
[INFO] Logging in user YOUR_USERNAME...
Login successful!
[INFO] Requesting download link for file ID: 1234567...
Download link obtained: https://dl.opensubtitles.org/...
[INFO] Downloading subtitle content from link...
[INFO] Subtitle content successfully fetched.
[INFO] Saving subtitle to downloaded_subtitle.srt...
Subtitle saved successfully to downloaded_subtitle.srt.
```

**Note**: This README describes assumed functionality based on a typical download example. The actual implementation in `examples/download/main.go` may vary. Please refer to the `main.go` source code for precise details. 