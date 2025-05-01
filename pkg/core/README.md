# Core Logic Package (`pkg/core`)

This package contains the central business logic for the Go OpenSubtitles Uploader application, designed to be shared between the GUI (`cmd/gui`) and CLI (`cmd/cli`) interfaces.

It encapsulates interactions with external services, file operations, and internal state management.

## Sub-packages

*   **`opensubtitles`**: Provides a client for interacting with the official OpenSubtitles.com REST API (v1). Handles authentication, searching for features (movies/shows) and subtitles, requesting download links, and uploading new subtitles.
*   **`trakt`**: (Planned) Provides a client or wrapper for interacting with the Trakt.tv API, primarily for metadata lookup (e.g., finding IMDB/TMDB IDs).
*   **`imdb`**: (Planned/Optional) Provides a client for interacting with an unofficial IMDB suggestions endpoint as a potential fallback for feature searching.
*   **`fileops`**: (Planned) Implements file-specific operations like calculating the OpenSubtitles hash for videos, MD5 hashing for subtitles, extracting technical metadata using the `mediainfo` tool, and parsing `.nfo` files for IMDB IDs.
*   **`metadata`**: (Planned) Defines core data structures representing video and subtitle information (`VideoInfo`, `SubtitleInfo`, `UploadJob`). Contains logic for processing and consolidating metadata, such as detecting subtitle language and analyzing subtitle flags (e.g., hearing-impaired).
*   **`queue`**: (Planned) Manages the queue of subtitle upload jobs, including state persistence (saving/loading the queue to/from disk) and history tracking.
*   **`errors`**: Defines common error types and variables (e.g., `ErrUnauthorized`, `ErrNotFound`) used throughout the core package for consistent error handling. 