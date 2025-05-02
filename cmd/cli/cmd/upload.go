package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/angelospk/osuploadergui/pkg/core/imdb"
	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	"github.com/angelospk/osuploadergui/pkg/core/opensubtitles"
	"github.com/angelospk/osuploadergui/pkg/core/queue"
	"github.com/angelospk/osuploadergui/pkg/core/trakt"
	"github.com/angelospk/osuploadergui/pkg/processor"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// --- Dependency Injection Functions for Testing ---

var NewAPIProviderFunc = func(apiKey string) (metadata.APIClientProvider, error) {
	// TODO: Allow http client customization?
	osClient := opensubtitles.NewClient(apiKey, http.DefaultClient)
	traktClient, err := trakt.NewClient()
	if err != nil {
		return metadata.APIClientProvider{}, fmt.Errorf("failed to initialize Trakt client: %w", err)
	}
	imdbClient := imdb.NewClient()

	// Return concrete types satisfying interfaces
	return metadata.APIClientProvider{
		OSClient:    osClient,
		TraktClient: traktClient,
		IMDbClient:  imdbClient,
	}, nil
}

var NewProcessorFunc = func(apiProvider metadata.APIClientProvider, logger *logrus.Logger) processor.ProcessorInterface {
	return processor.NewProcessor(apiProvider, logger)
}

var NewQueueManagerFunc = func(queueFile, historyFile string, logger *logrus.Logger) (queue.QueueManagerInterface, error) {
	// Use default files if paths are empty (though they should be set)
	// Assuming config dir comes from viper or a flag later?
	// For now, let NewQueueManager handle default paths based on empty configDir.
	configDir := "." // Or get from viper/flag
	return queue.NewQueueManager(configDir, logger)
}

// --- End Dependency Injection ---

var (
	uploadRecursive bool
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload [path...]",
	Short: "Scan paths for video/subtitle pairs and add them to the upload queue",
	Long: `Scans one or more directories or files for videos and matching subtitles.
It consolidates metadata using filename parsing and API lookups (if configured),
and then adds the identified pairs as jobs to the upload queue.

If a file path is provided, it will scan the directory containing that file.
Use the --recursive flag to scan subdirectories.`, // Added usage info
	Args: cobra.MinimumNArgs(1), // Require at least one path
	RunE: runUploadCmd,          // Changed to runUploadCmd which initializes deps
}

func init() {
	RootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().BoolVarP(&uploadRecursive, "recursive", "r", false, "Scan directories recursively")
	// Add flag for config dir?
	// Add flag for specific queue/history file paths?
}

// runUploadCmd initializes dependencies and calls runUpload
func runUploadCmd(cmd *cobra.Command, args []string) error {
	logger := logrus.New() // Create logger instance
	// Configure logger (level from viper/flag?)
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	apiKey := viper.GetString(CfgKeyOSAPIKey)
	if apiKey == "" {
		return fmt.Errorf("OpenSubtitles API key not configured. Use --api-key flag, OSUPLOADER_OS_API_KEY env var, or config file")
	}

	// Initialize API Provider
	apiProvider, err := NewAPIProviderFunc(apiKey)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize API provider")
		return fmt.Errorf("failed to initialize API provider: %w", err)
	}

	// Initialize Processor
	proc := NewProcessorFunc(apiProvider, logger)

	// Initialize Queue Manager
	// TODO: Get configDir properly
	qm, err := NewQueueManagerFunc("", "", logger) // Let func handle default
	if err != nil {
		logger.WithError(err).Error("Failed to initialize Queue Manager")
		return fmt.Errorf("failed to initialize queue manager: %w", err)
	}

	// Call the core logic function
	return runUpload(cmd.Context(), args, uploadRecursive, proc, qm, logger)
}

// runUpload contains the core logic for scanning and adding jobs
func runUpload(ctx context.Context, paths []string, recursive bool, proc processor.ProcessorInterface, qm queue.QueueManagerInterface, logger *logrus.Logger) error {
	totalAdded := 0
	totalSkipped := 0
	var firstError error // Keep track of the first error encountered

	for _, pathArg := range paths {
		if ctx.Err() != nil {
			logger.Warn("Context cancelled, stopping upload scan.")
			break // Stop processing paths if context is cancelled
		}

		// Check if path exists
		fileInfo, err := os.Stat(pathArg)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Errorf("Error: Path does not exist: %s", pathArg)
				if firstError == nil { // Store only the first error
					firstError = fmt.Errorf("path does not exist: %s", pathArg)
				}
				continue // Skip to the next path
			}
			logger.Errorf("Error stating path %s: %v", pathArg, err)
			if firstError == nil {
				firstError = fmt.Errorf("error stating path %s: %w", pathArg, err)
			}
			continue
		}

		// If a file is given, use its directory
		scanPath := pathArg
		if !fileInfo.IsDir() {
			scanPath = filepath.Dir(pathArg)
			logger.Infof("File provided, scanning directory: %s", scanPath)
		} else {
			logger.Infof("Scanning path: %s (Recursive: %t)", scanPath, recursive)
		}

		// Create jobs from the directory
		jobs, err := proc.CreateJobsFromDirectory(ctx, scanPath, recursive)
		if err != nil {
			logger.WithError(err).Errorf("Error creating jobs from path: %s", scanPath)
			if firstError == nil {
				firstError = fmt.Errorf("failed to create jobs from path %s: %w", scanPath, err)
			}
			continue // Skip to next path on processor error
		}

		if len(jobs) > 0 {
			logger.Infof("Found %d potential jobs.", len(jobs))
			added, skipped := qm.AddToQueue(jobs)
			logger.Infof("Added %d jobs to the queue (%d skipped).", added, skipped)
			totalAdded += added
			totalSkipped += skipped
		} else {
			logger.Infof("No processable video/subtitle pairs found in: %s", scanPath)
		}
	}

	logger.Infof("Scan complete. Total jobs added: %d, Total skipped: %d", totalAdded, totalSkipped)

	// Return the first error encountered during processing
	return firstError
}
