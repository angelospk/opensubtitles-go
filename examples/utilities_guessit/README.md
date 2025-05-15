# Utilities Guessit Example

This directory contains an example for using the `Guessit` utility method from the `opensubtitles-go` library. The `Guessit` utility parses a video filename and attempts to extract structured information like title, year, season, episode, etc.

## Files

- `main.go`: A runnable Go program that demonstrates the `Guessit` functionality with a sample filename.
- `README.md`: This file.

## How to Run

1.  **Replace Placeholders**: Edit `main.go` and replace `YOUR_API_KEY` in the `getClient()` function with your OpenSubtitles API key. (While `Guessit` itself might not strictly require an API key for its direct functionality according to some API client designs, initializing the client consistently is good practice, and other parts of the library will need it).
2.  **Navigate to Directory**: Open your terminal and change to `examples/utilities_guessit`.
3.  **Run the Program**: Execute `go run main.go`.

## What the Example Does

The `main.go` program demonstrates:

1.  **Client Initialization**: Sets up the OpenSubtitles client.
2.  **Call Guessit**: Calls the `Guessit` method with a sample filename (e.g., "The.Mandalorian.S01E01.Chapter.1.1080p.WEB-DL.DDP5.1.H.264-STAR.mkv").
    *   Prints the parsed information, such as title, year, season, episode, and type (movie/episode).

## Expected Output (Illustrative)

For the filename "The.Mandalorian.S01E01.Chapter.1.1080p.WEB-DL.DDP5.1.H.264-STAR.mkv", the output would be similar to:

```text
[INFO] --- Initializing Client ---
[INFO] Please replace YOUR_API_KEY with your actual OpenSubtitles API key.
[INFO] Client initialized.
[INFO] --- Example: Guessit Utility ---
Guessing info for filename: The.Mandalorian.S01E01.Chapter.1.1080p.WEB-DL.DDP5.1.H.264-STAR.mkv
Guessit Result:
  Title: The Mandalorian
  Year: Not detected
  Season: 1
  Episode: 1
  Episode Title: Chapter 1
  Type: episode
  Screen Size: 1080p
  Source: WEB-DL
  Video Codec: H.264
  Audio Codec: DDP5.1
  Release Group: STAR
-------------------------------------
```

**Notes**:
*   The accuracy and completeness of the parsed information depend on the filename format and the capabilities of the OpenSubtitles `guessit` service.
*   Fields that are not detected by `guessit` will typically be nil or have a zero value (e.g., `Year` in the example above might be nil, so the program prints "Not detected"). 