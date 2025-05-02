package processor

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	log "github.com/sirupsen/logrus"
)

// ProcessorInterface defines the methods for processing files and creating jobs.
type ProcessorInterface interface {
	CreateJobsFromDirectory(ctx context.Context, dirPath string, recursive bool) ([]metadata.UploadJob, error)
}

// Ensure Processor implements ProcessorInterface
var _ ProcessorInterface = (*Processor)(nil)

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
		logger = log.New()
		logger.SetFormatter(&log.TextFormatter{})
		logger.SetOutput(os.Stdout)
		logger.SetLevel(log.InfoLevel)
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
func (p *Processor) ScanDirectory(ctx context.Context, rootPath string, recursive bool) (*ScanDirectoryResult, error) {
	result := &ScanDirectoryResult{}

	err := filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			p.logger.Warnf("Error accessing path %q: %v", path, err)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if ctx.Err() != nil {
			p.logger.Info("Context cancelled during directory scan")
			return ctx.Err()
		}

		if d.IsDir() {
			if path != rootPath && !recursive {
				p.logger.Debugf("Skipping directory (not recursive): %s", path)
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if videoExtensions[ext] {
			result.VideoFiles = append(result.VideoFiles, path)
		} else if subtitleExtensions[ext] {
			result.SubtitleFiles = append(result.SubtitleFiles, path)
		}

		return nil
	})

	if err != nil && err != context.Canceled && err != context.DeadlineExceeded && err != filepath.SkipDir {
		p.logger.Errorf("Error walking directory %q: %v", rootPath, err)
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.logger.Infof("Scan complete. Found %d video files and %d subtitle files in %s (Recursive: %t)",
		len(result.VideoFiles), len(result.SubtitleFiles), rootPath, recursive)
	return result, nil
}

// CreateJobsFromDirectory scans a directory, matches files, and creates UploadJobs.
// It respects the recursive flag.
func (p *Processor) CreateJobsFromDirectory(ctx context.Context, rootPath string, recursive bool) ([]metadata.UploadJob, error) {
	scanResult, err := p.ScanDirectory(ctx, rootPath, recursive)
	if err != nil {
		return nil, err
	}

	var jobs []metadata.UploadJob
	processedSubtitles := make(map[string]bool)

	p.logger.Infof("Attempting to match %d videos with %d subtitles...", len(scanResult.VideoFiles), len(scanResult.SubtitleFiles))

	for _, videoFile := range scanResult.VideoFiles {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		p.logger.Infof("Processing video: %s", filepath.Base(videoFile))

		matchingSubtitle := metadata.FindMatchingSubtitle(videoFile, scanResult.SubtitleFiles)
		if matchingSubtitle != "" {
			if _, processed := processedSubtitles[matchingSubtitle]; !processed {
				p.logger.Infof("  Found matching subtitle: %s", filepath.Base(matchingSubtitle))
				videoInfo, subInfo, err := metadata.ConsolidateMetadata(ctx, videoFile, matchingSubtitle, p.apiClients)
				if err != nil {
					p.logger.Warnf("  Error consolidating metadata for %s + %s: %v", filepath.Base(videoFile), filepath.Base(matchingSubtitle), err)
					continue
				}

				if subInfo.Language == "" {
					p.logger.Warnf("  Skipping job creation for %s: Subtitle language could not be detected.", filepath.Base(matchingSubtitle))
				} else {
					jobs = append(jobs, metadata.UploadJob{
						VideoInfo:    videoInfo,
						SubtitleInfo: subInfo,
						Status:       metadata.StatusPending,
					})
					p.logger.Infof("  Created job for %s + %s", filepath.Base(videoFile), filepath.Base(matchingSubtitle))
				}
				processedSubtitles[matchingSubtitle] = true
			} else {
				p.logger.Infof("  Skipping already processed subtitle: %s", filepath.Base(matchingSubtitle))
			}
		} else {
			p.logger.Infof("  No matching subtitle found for video %s", filepath.Base(videoFile))
		}
	}

	p.logger.Infof("Initial matching complete. %d jobs created based on videos.", len(jobs))

	p.logger.Infof("Processing %d remaining subtitles without matched videos...", len(scanResult.SubtitleFiles)-len(processedSubtitles))
	for _, subtitleFile := range scanResult.SubtitleFiles {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if _, processed := processedSubtitles[subtitleFile]; !processed {
			p.logger.Infof("Processing standalone subtitle: %s", filepath.Base(subtitleFile))
			_, subInfo, err := metadata.ConsolidateMetadata(ctx, "", subtitleFile, p.apiClients)
			if err != nil {
				p.logger.Warnf("  Error consolidating metadata for subtitle %s: %v", filepath.Base(subtitleFile), err)
				continue
			}

			if subInfo.Language == "" {
				p.logger.Warnf("  Skipping job creation for %s: Subtitle language could not be detected.", filepath.Base(subtitleFile))
			} else {
				jobs = append(jobs, metadata.UploadJob{
					SubtitleInfo: subInfo,
					Status:       metadata.StatusPending,
				})
				p.logger.Infof("  Created job for standalone subtitle %s", filepath.Base(subtitleFile))
			}
			processedSubtitles[subtitleFile] = true
		}
	}

	p.logger.Infof("Processing complete. Total %d jobs created.", len(jobs))
	return jobs, nil
}
