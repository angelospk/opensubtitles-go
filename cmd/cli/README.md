# osuploadercli - OpenSubtitles Uploader CLI

This is the command-line interface for the Go OpenSubtitles Uploader project. It allows users to interact with OpenSubtitles.com to search for subtitles and upload their own, directly from the terminal.

## Building

To build the CLI application, navigate to the project's root directory (`osuploadergui`) in your terminal and run:

```bash
go build -o osuploadercli ./cmd/cli
```

This will create an executable file named `osuploadercli` (or `osuploadercli.exe` on Windows) in the project's root directory. You can move this executable to a directory in your system's PATH for easier access.

Alternatively, you can run the CLI directly without building using:

```bash
go run ./cmd/cli [command] [flags]
```

## Configuration

The CLI uses [Viper](https://github.com/spf13/viper) for configuration management. Configuration is loaded in the following order of precedence (higher items override lower ones):

1.  **Environment Variables:** Set environment variables prefixed with `OSUPLOADER_`.
    *   Example for API Key: `OSUPLOADER_OPENSUBTITLES_APIKEY="YourApiKeyHere"`
2.  **Configuration File:** Create a YAML file named `config.yaml` in one of the following locations:
    *   Your user's home directory, inside a `.osuploadercli` folder (e.g., `$HOME/.osuploadercli/config.yaml` on Linux/macOS, `C:\Users\YourUsername\.osuploadercli\config.yaml` on Windows). This is the **recommended** location.
    *   The current working directory (`.`).
    *   A specific path provided via the `--config` flag.

    The file structure should look like this:
    ```yaml
    # $HOME/.osuploadercli/config.yaml
    opensubtitles:
      apikey: "YourApiKeyHere"
    # Add other top-level keys like 'trakt' as needed
    # trakt:
    #   clientid: "YourTraktClientID"
    ```

**Required Configuration:**

*   **OpenSubtitles API Key:** You **must** provide your OpenSubtitles.com API key for most commands to work. Set it either via:
    *   Environment variable: `OSUPLOADER_OPENSUBTITLES_APIKEY`
    *   Config file key: `opensubtitles.apikey`

**Interactive Setup:**

*   If the OpenSubtitles API key is not found in the environment variables or the configuration file (`$HOME/.osuploadercli/config.yaml` or `./config.yaml`) when you run the CLI, you will be prompted to enter it directly in the terminal.
*   The key you enter will be automatically saved to `$HOME/.osuploadercli/config.yaml` for future use.
*   After saving, you will need to re-run the command you initially intended to execute.

## Usage

The base command is `osuploadercli`. Use `--help` to see available commands and flags.

```bash
# If osuploadercli is in your PATH
osuploadercli --help

# Otherwise, navigate to the directory containing the executable
./osuploadercli --help
```

### Commands

#### `login`

Verifies your OpenSubtitles API key by attempting to fetch user info.

```bash
./osuploadercli login
# Example Output (Success):
# INFO[0000] Successfully authenticated user: YourUsername

# Example Output (Failure):
# ERRO[0000] Authentication failed: invalid API key
```

#### `logout`

*(Currently, this command is a placeholder as the CLI doesn't manage persistent login sessions/tokens. It serves primarily to demonstrate command structure.)*

```bash
./osuploadercli logout
# Example Output:
# INFO[0000] Logout command executed (Note: CLI currently does not manage sessions).
```

#### `search`

Searches for subtitles on OpenSubtitles.

```bash
./osuploadercli search --query "My Movie Title" --type movie --lang en
./osuploadercli search --imdbid 1234567 --lang es
./osuploadercli search --type series --query "My Show" --season 1 --episode 5 --lang fr
```

**Flags:**

*   `--query` (`-q`): Search query (movie/series title).
*   `--type`: Type of media (`movie` or `series`).
*   `--imdbid`: IMDb ID (e.g., `tt1234567`). *Note: Using IMDb ID often yields better results.*
*   `--lang`: Language code(s) (comma-separated, e.g., `en`, `el,en`, `pt-br`).
*   `--season`: Season number (for series).
*   `--episode`: Episode number (for series).
*   `--parent`: Parent IMDb ID (useful for searching episodes within a specific series).

**Example Output:**

```
INFO[0000] Searching subtitles...
--- Result 1 ---
FileName:     My.Movie.Title.2023.1080p.BluRay.x264-GROUP.en.srt
Language:     en
Format:       srt
FPS:          23.976
Votes:        15
Rating:       9.5
IMDb ID:      1234567
Movie:        My Movie Title (2023)
DownloadLink: <link>
--- Result 2 ---
...
```

#### `upload`

Scans a directory (or a single file path) for video and subtitle files, matches them, extracts metadata, and adds them to the processing queue for potential upload. *(Note: Actual upload to OpenSubtitles is currently blocked by an XML-RPC issue, but this command prepares the jobs).*

```bash
# Scan a directory non-recursively
./osuploadercli upload /path/to/your/movies

# Scan a directory recursively
./osuploadercli upload -r /path/to/your/shows

# Process a single video file (will look for matching subtitle in the same dir)
./osuploadercli upload /path/to/your/movies/movie.mkv
```

**Flags:**

*   `--recursive` (`-r`): Scan directories recursively.

**Example Output:**

```
INFO[0000] Starting upload process for path: /path/to/your/movies recursive: false
INFO[0000] Scanning directory: /path/to/your/movies
INFO[0000] Found 1 video files and 1 subtitle files
INFO[0000] Processing video: /path/to/your/movies/My.Movie.2023.mkv
INFO[0000] Matched subtitle: /path/to/your/movies/My.Movie.2023.en.srt
INFO[0000] Consolidating metadata for video: My.Movie.2023.mkv sub: My.Movie.2023.en.srt
INFO[0000] Successfully added 1 job(s) to the queue. 0 job(s) were duplicates or invalid.
```

Jobs added by this command are stored in `