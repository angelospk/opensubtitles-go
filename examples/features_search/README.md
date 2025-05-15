# Features Search Examples

This directory contains an example demonstrating how to use the `SearchFeatures` method in the `opensubtitles-go` library to find movies, TV shows, or episodes.

## Files

- `main.go`: A runnable Go program showcasing how to search for features and handle the polymorphic `Attributes` field in the response.
- `README.md`: This file.

## How to Run

1.  **Replace Placeholders**: Before running, you **must** edit `main.go` and replace `YOUR_API_KEY` with your actual OpenSubtitles API key in the `getClient()` function.
2.  **Navigate to Directory**: Open your terminal and change to this directory (`examples/features_search`).
3.  **Run the Program**: Execute `go run main.go`.

## What the Example Does

The `main.go` program demonstrates:

1.  **Client Initialization**: Creates an OpenSubtitles client.
2.  **Search Features**: Performs a search for features (e.g., query "game of thrones", type "tvshow").
    *   Prints the ID and type of each feature found.
    *   Demonstrates how to handle the `Attributes` field of a `Feature`:
        *   The `Attributes` field is an `interface{}` because its actual structure depends on the `FeatureType` (movie, tvshow, episode).
        *   The example shows unmarshalling the `Attributes` into a map (`map[string]interface{}`) to inspect its `feature_type`.
        *   Based on the `feature_type`, it then unmarshals the `Attributes` again into the corresponding specific struct (`FeatureMovieAttributes`, `FeatureTvshowAttributes`, or `FeatureEpisodeAttributes`).
        *   Prints some details from the specific attribute structure.

## Expected Output (Illustrative)

If run with a valid API key, the output will be similar to:

```text
[INFO] --- Initializing Client ---
[INFO] Please replace YOUR_API_KEY with your actual OpenSubtitles API key.
[INFO] Client initialized.
[INFO] --- Example: Search Features (TV Show: Game of Thrones) ---
Searching for features with query: "game of thrones", type: "tvshow"
Found 1 features.
--- Feature 1 ---
  ID: 123, Type: feature
  Attempting to unmarshal attributes for feature_type: Tvshow
  Title: Game of Thrones (Year: 2011)
  IMDb ID: 944947, TMDB ID: 1399
  Seasons Count: 8
  URL: https://www.opensubtitles.com/en/search/imdbid-944947
-------------------------------------
[INFO] --- Example: Search Features (Movie: Inception) ---
Searching for features with query: "inception", type: "movie"
Found 1 features.
--- Feature 1 ---
  ID: 456, Type: feature
  Attempting to unmarshal attributes for feature_type: Movie
  Title: Inception (Year: 2010)
  IMDb ID: 1375666, TMDB ID: 27205
  URL: https://www.opensubtitles.com/en/search/imdbid-1375666
-------------------------------------
```

**Notes**:
*   The actual results (IDs, titles, etc.) will vary based on current API data.
*   The key takeaway is the two-step unmarshalling process for the `Attributes` field to access type-specific data.
``` 