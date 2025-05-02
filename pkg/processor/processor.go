package processor

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/angelospk/osuploadergui/pkg/core/metadata"
)

// Known video and subtitle extensions
var videoExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true,
}
var subtitleExtensions = map[string]bool{
	".srt": true, ".sub": true, ".ssa": true, ".ass": true, ".vtt": true,
}

// Processor handles the scanning of directories and creation of UploadJobs.
type Processor struct {
	apiClients metadata.APIClientProvider
	logger     *log.Logger
}

// NewProcessor creates a new Processor instance.
func NewProcessor(clients metadata.APIClientProvider, logger *log.Logger) *Processor {
	if logger == nil {
		logger = log.New(os.Stdout, "PROCESSOR: ", log.LstdFlags)
	}
	return &Processor{
		apiClients: clients,
		logger:     logger,
	}
}

// ScanDirectoryResult holds the lists of video and subtitle files found.
type ScanDirectoryResult struct {
	VideoFiles    []string
	SubtitleFiles []string
}

// ScanDirectory recursively scans a directory for video and subtitle files.
func (p *Processor) ScanDirectory(ctx context.Context, rootPath string) (*ScanDirectoryResult, error) {
	result := &ScanDirectoryResult{}

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			p.logger.Printf("Error accessing path %q: %v\n", path, err)
			return err // Propagate error up
		}
		if ctx.Err() != nil {
			p.logger.Println("Context cancelled during directory scan")
			return ctx.Err() // Stop walking if context is cancelled
		}

		if d.IsDir() {
			// If it's a directory, continue walking unless it's the root itself
			// which is implicitly handled by starting the walk.
			return nil
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(path))
		if videoExtensions[ext] {
			result.VideoFiles = append(result.VideoFiles, path)
		} else if subtitleExtensions[ext] {
			result.SubtitleFiles = append(result.SubtitleFiles, path)
		}

		return nil
	})

	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		p.logger.Printf("Error walking directory %q: %v\n", rootPath, err)
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err() // Return context error if walk was cancelled
	}

	p.logger.Printf("Scan complete. Found %d video files and %d subtitle files in %s\n",
		len(result.VideoFiles), len(result.SubtitleFiles), rootPath)
	return result, nil
}

// CreateJobsFromDirectory scans a directory, matches files, and creates UploadJobs.
// It prioritizes matching subtitles to videos found in the same scan.
func (p *Processor) CreateJobsFromDirectory(ctx context.Context, rootPath string) ([]metadata.UploadJob, error) {
	scanResult, err := p.ScanDirectory(ctx, rootPath)
	if err != nil {
		return nil, err
	}

	var jobs []metadata.UploadJob
	processedSubtitles := make(map[string]bool) // Keep track of subtitles already matched

	p.logger.Printf("Attempting to match %d videos with %d subtitles...", len(scanResult.VideoFiles), len(scanResult.SubtitleFiles))

	// 1. Iterate through videos and try to find matching subtitles
	for _, videoFile := range scanResult.VideoFiles {
		if ctx.Err() != nil {
			return nil, ctx.Err() // Check context cancellation
		}

		p.logger.Printf("Processing video: %s", filepath.Base(videoFile))

		matchingSubtitle := metadata.FindMatchingSubtitle(videoFile, scanResult.SubtitleFiles)
		if matchingSubtitle != "" {
			if _, processed := processedSubtitles[matchingSubtitle]; !processed {
				p.logger.Printf("  Found matching subtitle: %s", filepath.Base(matchingSubtitle))
				// Consolidate metadata and create job
				videoInfo, subInfo, err := metadata.ConsolidateMetadata(ctx, videoFile, matchingSubtitle, p.apiClients)
				if err != nil {
					p.logger.Printf("  Error consolidating metadata for %s + %s: %v", filepath.Base(videoFile), filepath.Base(matchingSubtitle), err)
					continue // Skip this pair if consolidation fails
				}

				// Create job only if essential info is present (e.g., subtitle language)
				if subInfo.Language == "" {
					p.logger.Printf("  Skipping job creation for %s: Subtitle language could not be detected.", filepath.Base(matchingSubtitle))
				} else {
					jobs = append(jobs, metadata.UploadJob{
						VideoInfo:    videoInfo,
						SubtitleInfo: subInfo,
						Status:       metadata.StatusPending,
					})
					p.logger.Printf("  Created job for %s + %s", filepath.Base(videoFile), filepath.Base(matchingSubtitle))
				}
				processedSubtitles[matchingSubtitle] = true
			} else {
				p.logger.Printf("  Skipping already processed subtitle: %s", filepath.Base(matchingSubtitle))
			}
		} else {
			p.logger.Printf("  No matching subtitle found for video %s", filepath.Base(videoFile))
		}
	}

	p.logger.Printf("Initial matching complete. %d jobs created based on videos.", len(jobs))

	// 2. Iterate through remaining subtitles (those not matched to a video)
	p.logger.Printf("Processing %d remaining subtitles without matched videos...", len(scanResult.SubtitleFiles)-len(processedSubtitles))
	for _, subtitleFile := range scanResult.SubtitleFiles {
		if ctx.Err() != nil {
			return nil, ctx.Err() // Check context cancellation
		}
		if _, processed := processedSubtitles[subtitleFile]; !processed {
			p.logger.Printf("Processing standalone subtitle: %s", filepath.Base(subtitleFile))
			// Consolidate metadata for subtitle only (video path will be empty)
			_, subInfo, err := metadata.ConsolidateMetadata(ctx, "", subtitleFile, p.apiClients) // Pass empty video path
			if err != nil {
				p.logger.Printf("  Error consolidating metadata for subtitle %s: %v", filepath.Base(subtitleFile), err)
				continue // Skip if consolidation fails
			}

			// Create job if essential info (language) is present
			if subInfo.Language == "" {
				p.logger.Printf("  Skipping job creation for %s: Subtitle language could not be detected.", filepath.Base(subtitleFile))
			} else {
				jobs = append(jobs, metadata.UploadJob{
					// VideoInfo will be default/empty (nil pointer)
					SubtitleInfo: subInfo,
					Status:       metadata.StatusPending,
				})
				p.logger.Printf("  Created job for standalone subtitle %s", filepath.Base(subtitleFile))
			}
			// Mark as processed (though technically redundant here as we only iterate once)
			processedSubtitles[subtitleFile] = true
		}
	}

	p.logger.Printf("Processing complete. Total %d jobs created.", len(jobs))
	return jobs, nil
}
