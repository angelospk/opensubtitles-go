package fileops

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dhowden/tag"
)

// MediaInfo holds extracted information from media files.
// Fields aligned with common needs and what dhowden/tag might provide.
type MediaInfo struct {
	Format      tag.FileType // e.g., tag.MP4, tag.MP3, tag.FLAC
	FileType    string       // User-friendly file type string
	Title       string
	Album       string
	Artist      string
	AlbumArtist string
	Composer    string
	Genre       string
	Year        int
	TrackNumber int
	TotalTracks int
	DiscNumber  int
	TotalDiscs  int
	Lyrics      string
	Comment     string
	// Note: dhowden/tag doesn't provide duration, dimensions, bitrate etc.
	// These fields are removed for now.
	// We might need another library or fallback to CLI for those.
}

// GetMediaInfo reads metadata from supported files using dhowden/tag.
// Currently supports MP4, MP3, FLAC, Ogg Vorbis/Opus.
// Does NOT provide technical details like duration, dimensions, bitrate.
func GetMediaInfo(ctx context.Context, filePath string) (*MediaInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file '%s': %w", filePath, err)
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		// If there's an error other than EOF (which ReadFrom handles), return it
		return nil, fmt.Errorf("failed to read metadata from '%s': %w", filePath, err)
	}

	// Check if metadata is nil (unsupported type or no tags found)
	if metadata == nil {
		return nil, fmt.Errorf("no supported metadata found or unsupported file type: %s", filepath.Ext(filePath))
	}

	// Populate the MediaInfo struct
	info := &MediaInfo{
		Format:      metadata.FileType(),
		FileType:    string(metadata.FileType()),
		Title:       metadata.Title(),
		Album:       metadata.Album(),
		Artist:      metadata.Artist(),
		AlbumArtist: metadata.AlbumArtist(),
		Composer:    metadata.Composer(),
		Genre:       metadata.Genre(),
		Year:        metadata.Year(),
		Lyrics:      metadata.Lyrics(),
		Comment:     metadata.Comment(),
	}

	trackNum, totalTracks := metadata.Track()
	info.TrackNumber = trackNum
	info.TotalTracks = totalTracks

	discNum, totalDiscs := metadata.Disc()
	info.DiscNumber = discNum
	info.TotalDiscs = totalDiscs

	return info, nil
}

// Placeholder for ReadNFO
func ReadNFO(ctx context.Context, filePath string) (string, error) {
	return "", fmt.Errorf("ReadNFO not implemented")
}
