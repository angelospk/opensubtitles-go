package queue

import (
	"encoding/json"
	"fmt"

	// "log" // Standard log is imported but not used directly; logrus is used
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	log "github.com/sirupsen/logrus"
)

// Default filenames for persistence
const (
	defaultQueueFile   = "queue.json"
	defaultHistoryFile = "history.json"
)

// QueueManagerInterface defines the methods for managing the upload queue and history.
type QueueManagerInterface interface {
	AddToQueue(jobs []metadata.UploadJob) (addedCount int, skippedCount int)
	GetNextPendingJob() *metadata.UploadJob
	UpdateJobStatus(jobID string, status metadata.JobStatus, message string) error
	MoveJobToHistory(jobID string) error
	LoadQueueState() error
	SaveQueueState() error
	ClearQueue() error
	ClearHistory() error
	RemoveJobFromQueue(jobID string) error
	GetQueue() []metadata.UploadJob
	GetHistory() []metadata.UploadJob
}

// Ensure QueueManager implements QueueManagerInterface
var _ QueueManagerInterface = (*QueueManager)(nil)

// QueueManager manages the upload job queue and history.
type QueueManager struct {
	queue     []metadata.UploadJob
	history   []metadata.UploadJob // Keep history simple as a slice for now
	queueLock sync.RWMutex
	histLock  sync.RWMutex

	queueFilePath   string
	historyFilePath string
	logger          *log.Logger // Use logrus Logger type
}

// NewQueueManager creates and initializes a new QueueManager.
// It attempts to load existing queue and history from default file paths.
// configDir specifies the directory where queue.json and history.json will be stored.
func NewQueueManager(configDir string, logger *log.Logger) (*QueueManager, error) {
	if logger == nil {
		// Create a default logrus logger if none provided
		logger = log.New()
		logger.SetFormatter(&log.TextFormatter{})
		logger.SetOutput(os.Stdout)
		logger.SetLevel(log.InfoLevel)
	}

	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	qm := &QueueManager{
		queue:           []metadata.UploadJob{},
		history:         []metadata.UploadJob{},
		queueFilePath:   filepath.Join(configDir, defaultQueueFile),
		historyFilePath: filepath.Join(configDir, defaultHistoryFile),
		logger:          logger,
	}

	// Load existing state, log errors but don't fail initialization
	if err := qm.LoadQueueState(); err != nil {
		qm.logger.Warnf("Failed to load queue state from %s: %v. Starting with empty queue.", qm.queueFilePath, err)
	}
	if err := qm.LoadHistory(); err != nil {
		qm.logger.Warnf("Failed to load history from %s: %v. Starting with empty history.", qm.historyFilePath, err)
	}

	qm.logger.Infof("QueueManager initialized. Queue: %d items, History: %d items.", len(qm.queue), len(qm.history))
	return qm, nil
}

// --- Persistence --- //

// SaveQueueState saves the current queue to its JSON file.
func (qm *QueueManager) SaveQueueState() error {
	qm.queueLock.RLock() // Read lock to access queue data
	data, err := json.MarshalIndent(qm.queue, "", "  ")
	qm.queueLock.RUnlock() // Release lock before I/O

	if err != nil {
		qm.logger.Errorf("Error marshaling queue state: %v", err)
		return fmt.Errorf("failed to marshal queue state: %w", err)
	}

	if err := os.WriteFile(qm.queueFilePath, data, 0644); err != nil {
		qm.logger.Errorf("Error writing queue state to %s: %v", qm.queueFilePath, err)
		return fmt.Errorf("failed to write queue state file %s: %w", qm.queueFilePath, err)
	}
	// qm.logger.Debugf("Queue state saved to %s (%d items)", qm.queueFilePath, len(qm.queue))
	return nil
}

// LoadQueueState loads the queue from its JSON file.
func (qm *QueueManager) LoadQueueState() error {
	data, err := os.ReadFile(qm.queueFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			qm.logger.Infof("Queue file %s does not exist, starting fresh.", qm.queueFilePath)
			qm.queueLock.Lock()
			qm.queue = []metadata.UploadJob{} // Ensure it's empty
			qm.queueLock.Unlock()
			return nil // Not an error if file doesn't exist yet
		}
		qm.logger.Errorf("Error reading queue state from %s: %v", qm.queueFilePath, err)
		return fmt.Errorf("failed to read queue state file %s: %w", qm.queueFilePath, err)
	}

	if len(data) == 0 { // Handle empty file case
		qm.logger.Infof("Queue file %s is empty, starting fresh.", qm.queueFilePath)
		qm.queueLock.Lock()
		qm.queue = []metadata.UploadJob{}
		qm.queueLock.Unlock()
		return nil
	}

	var loadedQueue []metadata.UploadJob
	if err := json.Unmarshal(data, &loadedQueue); err != nil {
		qm.logger.Errorf("Error unmarshaling queue state from %s: %v", qm.queueFilePath, err)
		return fmt.Errorf("failed to unmarshal queue state from %s: %w", qm.queueFilePath, err)
	}

	qm.queueLock.Lock()
	qm.queue = loadedQueue
	qm.queueLock.Unlock()
	qm.logger.Infof("Queue state loaded from %s (%d items)", qm.queueFilePath, len(qm.queue))
	return nil
}

// SaveHistory saves the current history to its JSON file.
func (qm *QueueManager) SaveHistory() error {
	qm.histLock.RLock()
	data, err := json.MarshalIndent(qm.history, "", "  ")
	qm.histLock.RUnlock()

	if err != nil {
		qm.logger.Errorf("Error marshaling history: %v", err)
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(qm.historyFilePath, data, 0644); err != nil {
		qm.logger.Errorf("Error writing history to %s: %v", qm.historyFilePath, err)
		return fmt.Errorf("failed to write history file %s: %w", qm.historyFilePath, err)
	}
	// qm.logger.Debugf("History saved to %s (%d items)", qm.historyFilePath, len(qm.history))
	return nil
}

// LoadHistory loads the history from its JSON file.
func (qm *QueueManager) LoadHistory() error {
	data, err := os.ReadFile(qm.historyFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			qm.logger.Infof("History file %s does not exist, starting fresh.", qm.historyFilePath)
			qm.histLock.Lock()
			qm.history = []metadata.UploadJob{}
			qm.histLock.Unlock()
			return nil
		}
		qm.logger.Errorf("Error reading history from %s: %v", qm.historyFilePath, err)
		return fmt.Errorf("failed to read history file %s: %w", qm.historyFilePath, err)
	}

	if len(data) == 0 { // Handle empty file case
		qm.logger.Infof("History file %s is empty, starting fresh.", qm.historyFilePath)
		qm.histLock.Lock()
		qm.history = []metadata.UploadJob{}
		qm.histLock.Unlock()
		return nil
	}

	var loadedHistory []metadata.UploadJob
	if err := json.Unmarshal(data, &loadedHistory); err != nil {
		qm.logger.Errorf("Error unmarshaling history from %s: %v", qm.historyFilePath, err)
		return fmt.Errorf("failed to unmarshal history from %s: %w", qm.historyFilePath, err)
	}

	qm.histLock.Lock()
	qm.history = loadedHistory
	qm.histLock.Unlock()
	qm.logger.Infof("History loaded from %s (%d items)", qm.historyFilePath, len(qm.history))
	return nil
}

// --- Queue Operations --- //

// AddToQueue adds a slice of jobs to the end of the queue, skipping duplicates.
// It returns the number of jobs actually added and the number skipped.
func (qm *QueueManager) AddToQueue(jobs []metadata.UploadJob) (addedCount int, skippedCount int) {
	qm.queueLock.Lock()
	defer qm.queueLock.Unlock()

	needsSave := false
	for _, job := range jobs {
		// Basic validation: ensure subtitle info exists
		if job.SubtitleInfo == nil || job.SubtitleInfo.FilePath == "" {
			qm.logger.Warnf("Skipping job add: Subtitle information is missing or incomplete.")
			skippedCount++
			continue
		}

		// Prevent duplicates based on subtitle file path (simple check)
		isDuplicate := false
		for _, existingJob := range qm.queue {
			// Also check history to prevent re-adding completed/failed jobs?
			// For now, just check current queue.
			if existingJob.SubtitleInfo != nil && existingJob.SubtitleInfo.FilePath == job.SubtitleInfo.FilePath {
				qm.logger.Infof("Skipping duplicate job add for subtitle: %s", job.SubtitleInfo.FilePath)
				isDuplicate = true
				skippedCount++
				break
			}
		}
		if !isDuplicate {
			newJob := job // Make a copy
			// Assign a unique ID if not already present (e.g., based on subtitle hash or path?)
			// For now, assume jobs might have IDs from processor or rely on index?
			// Let's use subtitle path as a temporary unique key for status updates
			// if newJob.ID == "" { ... }

			newJob.Status = metadata.StatusPending // Ensure status is pending
			newJob.SubmittedAt = time.Now()
			qm.queue = append(qm.queue, newJob)
			addedCount++
			needsSave = true
		}
	}

	if needsSave {
		qm.logger.Infof("Added %d new job(s) to the queue (%d skipped).", addedCount, skippedCount)
		// Save the state after adding
		qm.queueLock.Unlock()
		err := qm.SaveQueueState()
		qm.queueLock.Lock() // Re-acquire lock
		if err != nil {
			qm.logger.Errorf("Failed to save queue state after adding jobs: %v", err)
			// Continue, but log the error. State might be inconsistent until next save.
		}
	}

	return addedCount, skippedCount
}

// GetQueue returns a copy of the current queue.
func (qm *QueueManager) GetQueue() []metadata.UploadJob {
	qm.queueLock.RLock()
	defer qm.queueLock.RUnlock()

	// Return a copy to prevent external modification
	queueCopy := make([]metadata.UploadJob, len(qm.queue))
	copy(queueCopy, qm.queue)
	return queueCopy
}

// GetHistory returns a copy of the current history.
func (qm *QueueManager) GetHistory() []metadata.UploadJob {
	qm.histLock.RLock()
	defer qm.histLock.RUnlock()

	historyCopy := make([]metadata.UploadJob, len(qm.history))
	copy(historyCopy, qm.history)
	return historyCopy
}

// GetNextPendingJob finds the first job in the queue with StatusPending.
// Returns a pointer to a copy of the job, or nil if no pending jobs.
func (qm *QueueManager) GetNextPendingJob() *metadata.UploadJob {
	qm.queueLock.RLock()
	defer qm.queueLock.RUnlock()

	for _, job := range qm.queue {
		if job.Status == metadata.StatusPending {
			jobCopy := job // Return a copy
			// Find the index and return only the job pointer, matching the interface.
			return &jobCopy // Return pointer to the copy
		}
	}
	return nil // No pending job found
}

// UpdateJobStatus updates the status and message of a job identified by its SubtitleInfo.FilePath.
// TODO: Refactor methods to use a reliable unique Job ID instead of index or subtitle path.
func (qm *QueueManager) UpdateJobStatus(jobID string, status metadata.JobStatus, message string) error {
	qm.queueLock.Lock()
	// defer qm.queueLock.Unlock() // Unlock needs to happen before SaveQueueState

	foundIndex := -1
	// Need the index here
	for i := range qm.queue {
		// Use SubtitleInfo.FilePath as the temporary Job ID
		if qm.queue[i].SubtitleInfo != nil && qm.queue[i].SubtitleInfo.FilePath == jobID {
			foundIndex = i // Assign index 'i'
			break
		}
	}

	if foundIndex == -1 {
		qm.queueLock.Unlock() // Unlock before returning error
		return fmt.Errorf("job with ID '%s' not found in queue", jobID)
	}

	qm.queue[foundIndex].Status = status
	qm.queue[foundIndex].Message = message
	if status == metadata.StatusComplete || status == metadata.StatusFailed || status == metadata.StatusSkipped {
		qm.queue[foundIndex].CompletedAt = time.Now()
	}
	qm.logger.Infof("Updated job '%s' to status %s (Msg: %s)", jobID, status, message)

	// Save state after update
	qm.queueLock.Unlock()
	err := qm.SaveQueueState()
	// qm.queueLock.Lock() // No need to re-lock here
	if err != nil {
		qm.logger.Errorf("Failed to save queue state after updating job status: %v", err)
	}
	return err
}

// MoveJobToHistory removes a job identified by ID (SubtitleInfo.FilePath) from the queue
// and adds it to history. Saves both queue and history states.
func (qm *QueueManager) MoveJobToHistory(jobID string) error {
	qm.queueLock.Lock() // Need write lock for queue

	foundIndex := -1
	var jobToMove metadata.UploadJob
	// Need the index here
	for i := range qm.queue {
		// Use SubtitleInfo.FilePath as the temporary Job ID
		if qm.queue[i].SubtitleInfo != nil && qm.queue[i].SubtitleInfo.FilePath == jobID {
			foundIndex = i          // Assign index 'i'
			jobToMove = qm.queue[i] // Store the job to move
			break
		}
	}

	if foundIndex == -1 {
		qm.queueLock.Unlock()
		return fmt.Errorf("job with ID '%s' not found in queue for moving to history", jobID)
	}

	// Remove job from queue
	qm.queue = append(qm.queue[:foundIndex], qm.queue[foundIndex+1:]...)
	qm.logger.Infof("Removed job '%s' from queue index %d", jobID, foundIndex)
	qm.queueLock.Unlock() // Release queue lock *before* saving and acquiring history lock

	// Save queue state first (now that queueLock is released)
	saveErr := qm.SaveQueueState()
	if saveErr != nil {
		qm.logger.Errorf("Failed to save queue after removing job for history: %v", saveErr)
		// Continue to try saving history, but report this error
	}

	// Add to history
	qm.histLock.Lock()
	qm.history = append([]metadata.UploadJob{jobToMove}, qm.history...) // Prepend to history
	qm.logger.Infof("Added job '%s' to history (Total: %d)", jobID, len(qm.history))
	qm.histLock.Unlock() // Release history lock before saving history

	// Save history state
	histErr := qm.SaveHistory()
	if histErr != nil {
		qm.logger.Errorf("Failed to save history after moving job: %v", histErr)
		// Combine errors if both failed
		if saveErr != nil {
			saveErr = fmt.Errorf("queue save failed: %w; history save failed: %w", saveErr, histErr)
		} else {
			saveErr = fmt.Errorf("history save failed: %w", histErr)
		}
	}

	return saveErr
}

// ClearQueue removes all jobs from the queue and saves the state.
func (qm *QueueManager) ClearQueue() error {
	qm.queueLock.Lock()
	defer qm.queueLock.Unlock()

	if len(qm.queue) == 0 {
		qm.logger.Println("Queue is already empty.")
		return nil
	}

	qm.queue = []metadata.UploadJob{}
	qm.logger.Printf("Cleared all jobs from the queue.")

	// Save the empty state - SaveQueueState takes RLock
	return qm.SaveQueueState()
}

// ClearHistory removes all jobs from the history and saves the state.
func (qm *QueueManager) ClearHistory() error {
	qm.histLock.Lock()
	defer qm.histLock.Unlock()

	if len(qm.history) == 0 {
		qm.logger.Println("History is already empty.")
		return nil
	}

	qm.history = []metadata.UploadJob{}
	qm.logger.Printf("Cleared all jobs from the history.")

	// Save the empty state - SaveHistory takes RLock
	return qm.SaveHistory()
}

// RemoveJobFromQueue removes a job identified by ID (SubtitleInfo.FilePath) from the queue
// and saves the state.
func (qm *QueueManager) RemoveJobFromQueue(jobID string) error {
	qm.queueLock.Lock()
	// defer qm.queueLock.Unlock() // Unlock needs to happen before SaveQueueState

	foundIndex := -1
	// Need the index here
	for i := range qm.queue {
		// Use SubtitleInfo.FilePath as the temporary Job ID
		if qm.queue[i].SubtitleInfo != nil && qm.queue[i].SubtitleInfo.FilePath == jobID {
			foundIndex = i // Assign index 'i'
			break
		}
	}

	if foundIndex == -1 {
		qm.queueLock.Unlock() // Unlock before returning error
		return fmt.Errorf("job with ID '%s' not found in queue for removal", jobID)
	}

	removedJobPath := jobID // We already have the path
	qm.queue = append(qm.queue[:foundIndex], qm.queue[foundIndex+1:]...)
	qm.logger.Infof("Removed job at index %d (ID: %s)", foundIndex, removedJobPath)

	// Save state after removal
	qm.queueLock.Unlock()
	err := qm.SaveQueueState()
	// qm.queueLock.Lock() // No need to re-lock
	if err != nil {
		qm.logger.Errorf("Failed to save queue state after removing job: %v", err)
	}
	return err
}
