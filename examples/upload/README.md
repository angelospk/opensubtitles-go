# Example: Subtitle Upload via XML-RPC

This example demonstrates how to upload a subtitle file to OpenSubtitles using the `opensubtitles-go` library's XML-RPC uploader interface.

## What This Example Does
- Prompts the user for their OpenSubtitles username and password (password is MD5-hashed for transmission).
- Logs in to the OpenSubtitles XML-RPC API using the provided credentials and a user agent string.
- Asks for the path to a subtitle file, the IMDb ID of the associated video, and the language code.
- Uploads the subtitle file to OpenSubtitles, handling duplicate detection and error reporting.
- Prints the resulting subtitle URL if the upload is successful.
- Logs out and closes the uploader session.

## Prerequisites
- **Go 1.18+** installed ([download Go](https://golang.org/dl/)).
- The `opensubtitles-go` library and its dependencies installed (see [main README](../../README.md)).
- An [OpenSubtitles.org](https://www.opensubtitles.org/) account (username and password).
- A valid User-Agent string (the example uses `opensubtitles-api v5.1.2` by default).
- A subtitle file (e.g., `.srt`, `.sub`) and the IMDb ID for the associated video.

## How to Run
1. Open a terminal and navigate to this directory:
   ```sh
   cd examples/upload
   ```
2. Run the example:
   ```sh
   go run main.go
   ```
3. Follow the prompts:
   - Enter your OpenSubtitles username.
   - Enter your password (it will be MD5-hashed before sending).
   - Enter the full path to the subtitle file you wish to upload.
   - Enter the IMDb ID (numbers only, e.g., `1371111`).
   - Enter the language code (e.g., `en`, `fre`, `pob`).

If the upload is successful, the program will print the URL of the uploaded subtitle. If the subtitle already exists, you will be notified.

## How the XML-RPC Upload Works
- The uploader logs in using your credentials and a user agent.
- It prepares the subtitle file and metadata (IMDb ID, language, etc.).
- The file is uploaded via the legacy XML-RPC API (required for uploads; the REST API does not support this yet).
- The uploader logs out and closes the session when done.

## Notes & Caveats
- **Password Security:** Your password is MD5-hashed before being sent, but always use caution with credentials.
- **Duplicate Detection:** If the subtitle already exists in the OpenSubtitles database, the upload will not proceed and you will be notified.
- **Video File (Optional):** The example does not require a video file, but the uploader supports providing one for more accurate matching.
- **Known Issues:**
  - The XML-RPC upload endpoint is legacy and may be less reliable than the REST API.
  - Ensure your subtitle file is valid and matches the video for best results.
  - If you encounter `xmlrpc UploadSubtitles call failed: reading body EOF`, it may be a server or protocol issue (see project progress for troubleshooting).

## Further Reading
- [opensubtitles-go main README](../../README.md)
- [OpenSubtitles API Documentation](https://opensubtitles.stoplight.io/)
- [OpenSubtitles.org](https://www.opensubtitles.org/) 