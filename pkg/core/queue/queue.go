package queue

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/angelospk/osuploadergui/pkg/core/metadata"
)

// Default filenames for persistence
const (
	defaultQueueFile   = "queue.json"
	defaultHistoryFile = "history.json"
)

// QueueManager manages the upload job queue and history.
type QueueManager struct {
	queue     []metadata.UploadJob
	history   []metadata.UploadJob // Keep history simple as a slice for now
	queueLock sync.RWMutex
	histLock  sync.RWMutex

	queueFilePath   string
	historyFilePath string
	logger          *log.Logger
}

// NewQueueManager creates and initializes a new QueueManager.
// It attempts to load existing queue and history from default file paths.
// configDir specifies the directory where queue.json and history.json will be stored.
func NewQueueManager(configDir string, logger *log.Logger) (*QueueManager, error) {
	if logger == nil {
		logger = log.New(os.Stdout, "QUEUE: ", log.LstdFlags)
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
		qm.logger.Printf("Warning: Failed to load queue state from %s: %v. Starting with empty queue.", qm.queueFilePath, err)
	}
	if err := qm.LoadHistory(); err != nil {
		qm.logger.Printf("Warning: Failed to load history from %s: %v. Starting with empty history.", qm.historyFilePath, err)
	}

	qm.logger.Printf("QueueManager initialized. Queue: %d items, History: %d items.", len(qm.queue), len(qm.history))
	return qm, nil
}

// --- Persistence --- //

// SaveQueueState saves the current queue to its JSON file.
func (qm *QueueManager) SaveQueueState() error {
	qm.queueLock.RLock() // Read lock to access queue data
	defer qm.queueLock.RUnlock()

	data, err := json.MarshalIndent(qm.queue, "", "  ")
	if err != nil {
		qm.logger.Printf("Error marshaling queue state: %v", err)
		return fmt.Errorf("failed to marshal queue state: %w", err)
	}

	if err := os.WriteFile(qm.queueFilePath, data, 0644); err != nil {
		qm.logger.Printf("Error writing queue state to %s: %v", qm.queueFilePath, err)
		return fmt.Errorf("failed to write queue state file %s: %w", qm.queueFilePath, err)
	}
	// qm.logger.Printf("Queue state saved to %s (%d items)", qm.queueFilePath, len(qm.queue))
	return nil
}

// LoadQueueState loads the queue from its JSON file.
func (qm *QueueManager) LoadQueueState() error {
	qm.queueLock.Lock() // Full lock to modify queue
	defer qm.queueLock.Unlock()

	data, err := os.ReadFile(qm.queueFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			qm.logger.Printf("Queue file %s does not exist, starting fresh.", qm.queueFilePath)
			qm.queue = []metadata.UploadJob{} // Ensure it's empty
			return nil                        // Not an error if file doesn't exist yet
		}
		qm.logger.Printf("Error reading queue state from %s: %v", qm.queueFilePath, err)
		return fmt.Errorf("failed to read queue state file %s: %w", qm.queueFilePath, err)
	}

	if len(data) == 0 { // Handle empty file case
		qm.logger.Printf("Queue file %s is empty, starting fresh.", qm.queueFilePath)
		qm.queue = []metadata.UploadJob{}
		return nil
	}

	var loadedQueue []metadata.UploadJob
	if err := json.Unmarshal(data, &loadedQueue); err != nil {
		qm.logger.Printf("Error unmarshaling queue state from %s: %v", qm.queueFilePath, err)
		return fmt.Errorf("failed to unmarshal queue state from %s: %w", qm.queueFilePath, err)
	}

	qm.queue = loadedQueue
	qm.logger.Printf("Queue state loaded from %s (%d items)", qm.queueFilePath, len(qm.queue))
	return nil
}

// SaveHistory saves the current history to its JSON file.
func (qm *QueueManager) SaveHistory() error {
	qm.histLock.RLock()
	defer qm.histLock.RUnlock()

	data, err := json.MarshalIndent(qm.history, "", "  ")
	if err != nil {
		qm.logger.Printf("Error marshaling history: %v", err)
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(qm.historyFilePath, data, 0644); err != nil {
		qm.logger.Printf("Error writing history to %s: %v", qm.historyFilePath, err)
		return fmt.Errorf("failed to write history file %s: %w", qm.historyFilePath, err)
	}
	// qm.logger.Printf("History saved to %s (%d items)", qm.historyFilePath, len(qm.history))
	return nil
}

// LoadHistory loads the history from its JSON file.
func (qm *QueueManager) LoadHistory() error {
	qm.histLock.Lock()
	defer qm.histLock.Unlock()

	data, err := os.ReadFile(qm.historyFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			qm.logger.Printf("History file %s does not exist, starting fresh.", qm.historyFilePath)
			qm.history = []metadata.UploadJob{}
			return nil
		}
		qm.logger.Printf("Error reading history from %s: %v", qm.historyFilePath, err)
		return fmt.Errorf("failed to read history file %s: %w", qm.historyFilePath, err)
	}

	if len(data) == 0 { // Handle empty file case
		qm.logger.Printf("History file %s is empty, starting fresh.", qm.historyFilePath)
		qm.history = []metadata.UploadJob{}
		return nil
	}

	var loadedHistory []metadata.UploadJob
	if err := json.Unmarshal(data, &loadedHistory); err != nil {
		qm.logger.Printf("Error unmarshaling history from %s: %v", qm.historyFilePath, err)
		return fmt.Errorf("failed to unmarshal history from %s: %w", qm.historyFilePath, err)
	}

	qm.history = loadedHistory
	qm.logger.Printf("History loaded from %s (%d items)", qm.historyFilePath, len(qm.history))
	return nil
}

// --- Queue Operations --- //

// AddToQueue adds one or more jobs to the end of the queue and saves the state.
func (qm *QueueManager) AddToQueue(jobs ...metadata.UploadJob) error {
	qm.queueLock.Lock()
	defer qm.queueLock.Unlock()

	addedCount := 0
	for _, job := range jobs {
		// Basic validation: ensure subtitle info exists
		if job.SubtitleInfo == nil || job.SubtitleInfo.FilePath == "" {
			qm.logger.Printf("Skipping job add: Subtitle information is missing or incomplete.")
			continue
		}

		// Prevent duplicates based on subtitle file path (simple check)
		isDuplicate := false
		for _, existingJob := range qm.queue {
			if existingJob.SubtitleInfo != nil && existingJob.SubtitleInfo.FilePath == job.SubtitleInfo.FilePath {
				qm.logger.Printf("Skipping duplicate job add for subtitle: %s", job.SubtitleInfo.FilePath)
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			newJob := job                          // Make a copy
			newJob.Status = metadata.StatusPending // Ensure status is pending
			newJob.SubmittedAt = time.Now()
			qm.queue = append(qm.queue, newJob)
			addedCount++
		}
	}

	if addedCount > 0 {
		qm.logger.Printf("Added %d new job(s) to the queue.", addedCount)
		// Save the state after adding
		// Run save in RLock block after Unlock to prevent deadlock
		qm.queueLock.Unlock()
		err := qm.SaveQueueState()
		qm.queueLock.Lock() // Re-acquire lock before returning
		return err
	}

	return nil // No error if no jobs were added (all duplicates/invalid)
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
// Returns the job and its index, or nil and -1 if no pending jobs.
func (qm *QueueManager) GetNextPendingJob() (*metadata.UploadJob, int) {
	qm.queueLock.RLock()
	defer qm.queueLock.RUnlock()

	for i, job := range qm.queue {
		if job.Status == metadata.StatusPending {
			jobCopy := job // Return a copy
			return &jobCopy, i
		}
	}
	return nil, -1
}

// UpdateJobStatus updates the status and message of a job at a specific index in the queue.
// It also saves the queue state.
func (qm *QueueManager) UpdateJobStatus(index int, status metadata.JobStatus, message string) error {
	qm.queueLock.Lock()
	defer qm.queueLock.Unlock()

	if index < 0 || index >= len(qm.queue) {
		return fmt.Errorf("invalid job index: %d", index)
	}

	qm.queue[index].Status = status
	qm.queue[index].Message = message
	if status == metadata.StatusComplete || status == metadata.StatusFailed || status == metadata.StatusSkipped {
		qm.queue[index].CompletedAt = time.Now()
	}
	qm.logger.Printf("Updated job at index %d to status %s (Msg: %s)", index, status, message)

	// Save state after update - SaveQueueState takes RLock, which is compatible
	// with the Lock already held here.
	return qm.SaveQueueState()
}

// MoveJobToHistory removes a job from the queue at the given index and adds it to history.
// Saves both queue and history states.
func (qm *QueueManager) MoveJobToHistory(index int) error {
	qm.queueLock.Lock() // Need write lock for queue

	if index < 0 || index >= len(qm.queue) {
		qm.queueLock.Unlock()
		return fmt.Errorf("invalid job index for moving to history: %d", index)
	}

	jobToMove := qm.queue[index]

	// Remove job from queue
	qm.queue = append(qm.queue[:index], qm.queue[index+1:]...)
	qm.logger.Printf("Removed job from queue index %d", index)
	qm.queueLock.Unlock() // Release queue lock *before* saving and acquiring history lock

	// Save queue state first (now that queueLock is released)
	saveErr := qm.SaveQueueState()
	if saveErr != nil {
		qm.logger.Printf("Failed to save queue after removing job for history: %v", saveErr)
		// Continue to try saving history, but report this error
	}

	// Add to history
	qm.histLock.Lock()
	qm.history = append([]metadata.UploadJob{jobToMove}, qm.history...) // Prepend to history
	qm.logger.Printf("Added job to history (Total: %d)", len(qm.history))
	qm.histLock.Unlock() // Release history lock before saving history

	// Save history state
	histErr := qm.SaveHistory()
	if histErr != nil {
		qm.logger.Printf("Failed to save history after moving job: %v", histErr)
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

// RemoveJobFromQueue removes a job from the queue by its index and saves the state.
func (qm *QueueManager) RemoveJobFromQueue(index int) error {
	qm.queueLock.Lock()
	defer qm.queueLock.Unlock()

	if index < 0 || index >= len(qm.queue) {
		return fmt.Errorf("invalid job index for removal: %d", index)
	}

	removedJobPath := "(unknown)"
	if qm.queue[index].SubtitleInfo != nil {
		removedJobPath = qm.queue[index].SubtitleInfo.FilePath
	}

	qm.queue = append(qm.queue[:index], qm.queue[index+1:]...)
	qm.logger.Printf("Removed job at index %d (Subtitle: %s)", index, removedJobPath)

	// Save state after removal - SaveQueueState takes RLock
	return qm.SaveQueueState()
}
