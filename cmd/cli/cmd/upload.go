package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles"
	"github.com/angelospk/osuploadergui/pkg/core/queue"
	"github.com/angelospk/osuploadergui/pkg/processor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	recursive bool // Flag for recursive directory scan
)

var uploadCmd = &cobra.Command{
	Use:   "upload [path...]",
	Short: "Scan paths for subtitles/videos, match them, and add to the upload queue.",
	Long: `Scans the specified directory or file paths for video and subtitle files.
It attempts to match video and subtitle pairs based on filenames, 
consolidates metadata (using filename parsing and potentially APIs),
and adds valid pairs or standalone subtitles to the upload queue for processing.

Currently, this command only adds jobs to the queue. 
A separate command or process will be needed to actually process the queue and perform uploads.
Use the --recursive flag (-r) to scan directories recursively.`,
	Args: cobra.MinimumNArgs(1), // Require at least one path
	RunE: func(cmd *cobra.Command, args []string) error {
		paths := args

		// --- Dependency Initialization ---
		logger := log.New(os.Stderr, "CLI: ", log.LstdFlags)

		// Get config dir (needed for queue persistence)
		configDir, err := getConfigDir()
		if err != nil {
			return fmt.Errorf("could not determine config directory: %w", err)
		}

		// Initialize Queue Manager
		queueManager, err := queue.NewQueueManager(configDir, logger)
		if err != nil {
			return fmt.Errorf("failed to initialize queue manager: %w", err)
		}

		// Initialize API Clients Provider (using Viper for config)
		// We need the API key here to potentially make calls during metadata consolidation
		apiKey := viper.GetString(CfgKeyOSAPIKey)
		if apiKey == "" {
			logger.Println("Warning: OpenSubtitles API key not configured. Metadata lookup might be limited.")
			// Continue without API key for now, maybe make it mandatory later?
		}
		// TODO: Add Trakt Client ID handling
		// TODO: Add IMDb client handling (if used)
		apiProvider := metadata.APIClientProvider{
			OSClient:    opensubtitles.NewClient(apiKey, nil), // Pass API key
			TraktClient: nil,                                  // Initialize properly later
			IMDbClient:  nil,                                  // Initialize properly later
		}

		// Initialize Processor
		proc := processor.NewProcessor(apiProvider, logger)

		// --- Execute Core Logic ---
		return runUpload(paths, proc, queueManager, logger)
	},
}

// runUpload contains the core logic for scanning and queueing jobs.
// It's separated for easier testing.
func runUpload(paths []string, proc *processor.Processor, qm *queue.QueueManager, logger *log.Logger) error {
	var allJobs []metadata.UploadJob
	var finalErr error

	logger.Printf("Starting scan for paths: %v (Recursive: %v)", paths, recursive)

	for _, path := range paths {
		logger.Printf("Processing path: %s", path)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Printf("Error: Path does not exist: %s", path)
				finalErr = errors.Join(finalErr, fmt.Errorf("path does not exist: %s", path))
			} else {
				logger.Printf("Error stating path %s: %v", path, err)
				finalErr = errors.Join(finalErr, fmt.Errorf("error stating path %s: %w", path, err))
			}
			continue // Skip to next path on error
		}

		if !info.IsDir() {
			// If it's a single file, we might want different handling later?
			// For now, treat it as a directory containing only itself for scanning?
			// Or perhaps error out if not a directory?
			// Let's error for now, user should specify directories.
			logger.Printf("Error: Path is a file, not a directory: %s", path)
			finalErr = errors.Join(finalErr, fmt.Errorf("path is a file, not a directory: %s", path))
			continue
			// Alternative: Handle single file scan (e.g., find matching video/sub in same dir)
		}

		// TODO: Implement recursive scanning logic if the 'recursive' flag is true.
		// For now, only scans the top level of the specified directory.
		if recursive {
			logger.Println("Recursive flag set, scanning recursively...")
			// filepath.WalkDir would be used here, similar to processor.ScanDirectory
			// but processor.CreateJobsFromDirectory already walks recursively.
			// So, we just need to pass the directory to it.
		} else {
			logger.Println("Recursive flag not set. Scanning only top-level directory.")
			// Need to modify CreateJobsFromDirectory or add a non-recursive scan option?
			// Let's assume CreateJobsFromDirectory is always recursive for now based on its WalkDir.
			// The flag currently does nothing extra here.
		}

		// Use the processor to find files and create jobs for the current directory
		// Assuming CreateJobsFromDirectory handles recursive scan internally
		logger.Printf("Calling CreateJobsFromDirectory for: %s", path)
		jobs, err := proc.CreateJobsFromDirectory(context.Background(), path)
		if err != nil {
			logger.Printf("Error creating jobs from directory %s: %v", path, err)
			finalErr = errors.Join(finalErr, fmt.Errorf("error processing directory %s: %w", path, err))
			continue // Skip this path
		}

		logger.Printf("Found %d potential jobs in %s", len(jobs), path)
		allJobs = append(allJobs, jobs...)
	}

	if len(allJobs) > 0 {
		logger.Printf("Adding %d jobs to the queue...", len(allJobs))
		err := qm.AddToQueue(allJobs...)
		if err != nil {
			logger.Printf("Error adding jobs to queue: %v", err)
			finalErr = errors.Join(finalErr, fmt.Errorf("error adding jobs to queue: %w", err))
		} else {
			logger.Printf("Successfully added %d jobs to the queue.", len(allJobs))
		}
	} else {
		logger.Println("No valid jobs found to add to the queue.")
	}

	// TODO: Implement queue processing logic (separate command? flag?)
	logger.Println("Upload command finished adding jobs to queue.")

	return finalErr // Return combined errors, if any
}

// getConfigDir determines the directory for storing configuration files.
// Placeholder implementation - enhance as needed (e.g., use XDG spec).
func getConfigDir() (string, error) {
	// Option 1: Use explicit config dir from viper/flag if set
	// configDir := viper.GetString("configdir") // Example key
	// if configDir != "" { return configDir, nil }

	// Option 2: Use OS specific config directory
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appConfigDir := filepath.Join(dir, "osuploadercli")
	// Ensure it exists (QueueManager also does this, but good practice here too)
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", err
	}
	return appConfigDir, nil

	// Option 3: Use directory of the viper config file (if used)
	// configFile := viper.ConfigFileUsed()
	// if configFile != "" { return filepath.Dir(configFile), nil }

	// Option 4: Default to current directory or home/.appname as fallback
	// return ".", nil // Or home/.osuploadercli
}

func init() {
	RootCmd.AddCommand(uploadCmd)

	// Add the recursive flag
	uploadCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Scan directories recursively")

	// Add other flags if needed: e.g., --process-queue, --language, etc.
}
