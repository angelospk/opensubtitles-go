# Discover Content Examples

This directory contains examples for using the discovery endpoints of the `opensubtitles-go` library: `DiscoverPopular`, `DiscoverLatest`, and `DiscoverMostDownloaded`.

## Files

- `main.go`: A runnable Go program that calls each of the discovery endpoints.
- `README.md`: This file.

## How to Run

1.  **Replace Placeholders**: Edit `main.go` and replace `YOUR_API_KEY` in the `getClient()` function with your OpenSubtitles API key.
2.  **Navigate to Directory**: Open your terminal and change to `examples/discover_content`.
3.  **Run the Program**: Execute `go run main.go`.

## What the Example Does

The `main.go` program demonstrates:

1.  **Client Initialization**: Sets up the OpenSubtitles client.
2.  **Discover Popular Features**: Calls `DiscoverPopular` to get a list of popular movies and TV shows.
    *   Similar to `SearchFeatures`, the `Attributes` field is an `interface{}` and needs to be unmarshalled based on the `feature_type` (movie or tvshow) to access specific details.
    *   Prints basic information for a few popular items.
3.  **Discover Latest Subtitles**: Calls `DiscoverLatest` to get recently uploaded subtitles.
    *   Prints details for a few latest subtitles, including language, release name, and associated feature title.
4.  **Discover Most Downloaded Subtitles**: Calls `DiscoverMostDownloaded` to get subtitles with the highest download counts.
    *   Prints details for a few of the most downloaded subtitles.

## Expected Output (Illustrative)

Output will vary greatly depending on current API data. Here's a conceptual example:

```text
[INFO] --- Initializing Client ---
[INFO] Please replace YOUR_API_KEY with your actual OpenSubtitles API key.
[INFO] Client initialized.
[INFO] --- Example: Discover Popular (Movies, English) ---
Found 10 popular features (type: movie, lang: en).
--- Popular Feature 1 (Movie) ---
  Title: Dune: Part Two (Year: 2024)
  IMDb ID: 1160419, TMDB ID: 693134
  URL: https://www.opensubtitles.com/en/search/imdbid-1160419
...
-------------------------------------
[INFO] --- Example: Discover Latest Subtitles (All types, English) ---
Found 60 latest subtitles (Page 1/1, Total: 60)
--- Latest Subtitle 1 ---
  ID: 1957811234, Lang: en, Release: Some.Movie.2024.1080p.WEB-DL
  Feature: Some Movie (Year: 2024)
  URL: https://www.opensubtitles.com/en/subtitles/1957811234/some-movie-some.movie.2024.1080p.web-dl-en
...
-------------------------------------
[INFO] --- Example: Discover Most Downloaded Subtitles (Movies, English) ---
Found 20 most downloaded subtitles (Page 1/X, Total: YYY)
--- Most Downloaded Subtitle 1 ---
  ID: 1954939103, Lang: en, Release: Inception.2010.1080p.BluRay.x264.YIFY
  Feature: Inception (Year: 2010)
  Downloads: 63111
  URL: https://www.opensubtitles.com/en/subtitles/1954939103/inception-inception.2010.1080p.bluray.x264.yify-en
...
-------------------------------------
```

**Notes**:
*   The number of items printed is limited in the example for brevity.
*   The `DiscoverPopular` endpoint requires the same `Attributes` handling as `SearchFeatures` due to mixed feature types.
``` 