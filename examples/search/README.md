# Comprehensive Subtitle Search Example

This directory contains `main.go`, an interactive example program for searching subtitles using the `opensubtitles-go` library. It allows users to input various search parameters like IMDb ID, movie hash, languages, and a query string.

## Files

- `main.go`: The interactive Go program for searching subtitles.
- `README.md`: This file.

## How to Run

1.  **Navigate to Directory**: Open your terminal and change to `examples/search`.
2.  **Run the Program**: Execute `go run main.go`.
3.  **Follow Prompts**:
    *   The program will first ask for your OpenSubtitles API Key and a User Agent string.
    *   Then, it will prompt you to enter search criteria:
        *   IMDb ID (e.g., `1375666` for Inception)
        *   Movie Hash
        *   Languages (comma-separated, e.g., `en,es`)
        *   Query string (e.g., movie title like `Inception`)
    *   You must provide at least an IMDb ID, movie hash, or query string.

## What the Example Does

The `main.go` program:

1.  **Collects Credentials**: Takes API key and User Agent from the user.
2.  **Initializes Client**: Creates an OpenSubtitles client instance.
3.  **Collects Search Parameters**: Interactively asks the user for IMDb ID, movie hash, languages, and a query string.
4.  **Performs Search**: Calls the `SearchSubtitles` method with the provided parameters.
5.  **Displays Results**: 
    *   Prints the total number of subtitles found and pagination info (current page/total pages).
    *   If results are found, it iterates through the subtitles on the current page and displays details for each, such as:
        *   Subtitle ID
        *   Language
        *   Associated Feature Title and IMDb ID
        *   Uploader information
        *   Download count, votes, and ratings
        *   Hearing-impaired status
        *   File ID and filename (if available)

## Expected Output (Illustrative)

After providing credentials and search parameters (e.g., IMDb ID `1375666` and language `en`):

```text
--- OpenSubtitles Credentials ---
Enter your OpenSubtitles API Key:
YOUR_API_KEY
Enter your User Agent (e.g., MyApp v1.0):
MySearchApp/1.0
Client created successfully.

--- Subtitle Search ---
Enter IMDb ID to search for (e.g., 1371111 for Inception), or press Enter to skip:
1375666
Enter movie hash to search for, or press Enter to skip:

Enter languages to search for (comma-separated, e.g., en,es), or press Enter for all:
en
Enter a query string (movie title, etc.), or press Enter to skip:


Searching for subtitles...

Found 97 subtitles (Page 1/5):
--------------------------------------------------
Result 1:
  Subtitle ID: 1954939103
  Language:    en
  Feature:     Inception (IMDb: 1375666)
  Uploader:    YIFY (Rank: administrator)
  Downloads:   63111
  Votes:       123 (Score: 9.50)
  Is Hearing Impaired: false
  File ID:     1954939103 (Inception.2010.1080p.BluRay.x264.YIFY.srt)
--------------------------------------------------
Result 2:
  ...
--------------------------------------------------
```

**Notes**:
*   The example currently only displays the first page of results. To implement pagination, you would need to modify the code to allow users to specify a page number and re-run the search with the `Page` parameter in `SearchSubtitlesParams`.
*   Actual results will vary based on API data and search terms. 