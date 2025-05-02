package queue_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	"github.com/angelospk/osuploadergui/pkg/core/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a QueueManager in a temporary directory
func setupTestQueueManager(t *testing.T) (*queue.QueueManager, string) {
	tempDir := t.TempDir()
	logger := log.New(ioutil.Discard, "", 0) // Discard logs during tests
	qm, err := queue.NewQueueManager(tempDir, logger)
	require.NoError(t, err, "NewQueueManager should not return an error")
	require.NotNil(t, qm, "NewQueueManager should return a valid manager")
	return qm, tempDir
}

// Helper to create a sample UploadJob
func createSampleJob(subPath string, status metadata.JobStatus) metadata.UploadJob {
	return metadata.UploadJob{
		SubtitleInfo: &metadata.SubtitleInfo{
			FilePath: subPath,
			FileName: filepath.Base(subPath),
			Language: "en", // Assume some basic info
		},
		VideoInfo: &metadata.VideoInfo{
			FilePath: "/path/to/video.mkv",
			FileName: "video.mkv",
		},
		Status: status,
	}
}

func TestNewQueueManager(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(ioutil.Discard, "", 0)

	// Test 1: Successful creation
	qm, err := queue.NewQueueManager(tempDir, logger)
	assert.NoError(t, err)
	assert.NotNil(t, qm)
	assert.DirExists(t, tempDir) // Ensure directory was created if needed

	// Test 2: Error creating directory (e.g., permissions)
	// Make the tempDir read-only to simulate creation failure
	// Note: This might not work reliably on all OS/filesystems
	// _ = os.Chmod(tempDir, 0400)
	// _, err = queue.NewQueueManager(filepath.Join(tempDir, "subdir", "subsubdir"), logger)
	// assert.Error(t, err)
	// _ = os.Chmod(tempDir, 0755) // Restore permissions
}

func TestSaveLoadQueueState(t *testing.T) {
	qm, tempDir := setupTestQueueManager(t)
	queueFile := filepath.Join(tempDir, "queue.json")

	// 1. Initial state should be empty
	assert.Empty(t, qm.GetQueue(), "Initial queue should be empty")

	// 2. Add jobs and save
	job1 := createSampleJob("/subs/sub1.srt", metadata.StatusPending)
	job2 := createSampleJob("/subs/sub2.srt", metadata.StatusPending)
	err := qm.AddToQueue(job1, job2)
	require.NoError(t, err)
	assert.Len(t, qm.GetQueue(), 2, "Queue should have 2 items after adding")

	// Check file exists after AddToQueue (which calls Save)
	assert.FileExists(t, queueFile)

	// 3. Create a new manager in the same directory to load the state
	logger := log.New(ioutil.Discard, "", 0)
	qm2, err := queue.NewQueueManager(tempDir, logger)
	require.NoError(t, err)
	loadedQueue := qm2.GetQueue()
	assert.Len(t, loadedQueue, 2, "Loaded queue should have 2 items")
	// Basic check - more thorough checks might compare fields
	assert.Equal(t, job1.SubtitleInfo.FilePath, loadedQueue[0].SubtitleInfo.FilePath)
	assert.Equal(t, job2.SubtitleInfo.FilePath, loadedQueue[1].SubtitleInfo.FilePath)
	assert.NotZero(t, loadedQueue[0].SubmittedAt, "SubmittedAt should be set")

	// 4. Test loading non-existent file (should be handled gracefully)
	qm3, _ := setupTestQueueManager(t) // Uses a *new* temp dir
	assert.Empty(t, qm3.GetQueue())

	// 5. Test loading empty file
	require.NoError(t, os.WriteFile(queueFile, []byte{}, 0644)) // Overwrite with empty file
	qm4, err := queue.NewQueueManager(tempDir, logger)
	require.NoError(t, err)
	assert.Empty(t, qm4.GetQueue(), "Queue loaded from empty file should be empty")

	// 6. Test loading invalid JSON
	require.NoError(t, os.WriteFile(queueFile, []byte("invalid json"), 0644))
	_, err = queue.NewQueueManager(tempDir, logger) // Should log error, but return qm
	assert.NoError(t, err)                          // NewQueueManager itself doesn't error on load failure, it logs
	// We can't easily assert logger output without capturing it, but ensure it doesn't panic
}

func TestSaveLoadHistory(t *testing.T) {
	qm, tempDir := setupTestQueueManager(t)
	historyFile := filepath.Join(tempDir, "history.json")

	// 1. Initial state
	assert.Empty(t, qm.GetHistory(), "Initial history should be empty")

	// 2. Add jobs to queue and move to history
	job1 := createSampleJob("/history/sub1.srt", metadata.StatusComplete)
	job2 := createSampleJob("/history/sub2.srt", metadata.StatusFailed)

	err := qm.AddToQueue(job1, job2) // Add first
	require.NoError(t, err)

	// Simulate processing and moving
	require.NoError(t, qm.UpdateJobStatus(0, metadata.StatusComplete, "OK"))
	require.NoError(t, qm.MoveJobToHistory(0))                                // Move job1
	require.NoError(t, qm.UpdateJobStatus(0, metadata.StatusFailed, "Error")) // Update job2 (now at index 0)
	require.NoError(t, qm.MoveJobToHistory(0))                                // Move job2

	assert.Len(t, qm.GetQueue(), 0, "Queue should be empty after moving")
	currentHistory := qm.GetHistory()
	assert.Len(t, currentHistory, 2, "History should have 2 items")
	assert.FileExists(t, historyFile)

	// Check order (prepended) and status
	assert.Equal(t, job2.SubtitleInfo.FilePath, currentHistory[0].SubtitleInfo.FilePath)
	assert.Equal(t, metadata.StatusFailed, currentHistory[0].Status)
	assert.Equal(t, job1.SubtitleInfo.FilePath, currentHistory[1].SubtitleInfo.FilePath)
	assert.Equal(t, metadata.StatusComplete, currentHistory[1].Status)

	// 3. Load history in a new manager
	logger := log.New(ioutil.Discard, "", 0)
	qm2, err := queue.NewQueueManager(tempDir, logger)
	require.NoError(t, err)
	loadedHistory := qm2.GetHistory()
	assert.Len(t, loadedHistory, 2, "Loaded history should have 2 items")
	assert.Equal(t, job2.SubtitleInfo.FilePath, loadedHistory[0].SubtitleInfo.FilePath)
	assert.Equal(t, job1.SubtitleInfo.FilePath, loadedHistory[1].SubtitleInfo.FilePath)
}

func TestAddToQueue(t *testing.T) {
	qm, _ := setupTestQueueManager(t)

	job1 := createSampleJob("/subs/add1.srt", metadata.StatusPending)
	job2 := createSampleJob("/subs/add2.srt", metadata.StatusPending)
	job3Invalid := metadata.UploadJob{Status: metadata.StatusPending} // Missing subtitle info
	job1Duplicate := createSampleJob("/subs/add1.srt", metadata.StatusPending)

	// Add valid job
	err := qm.AddToQueue(job1)
	assert.NoError(t, err)
	assert.Len(t, qm.GetQueue(), 1)
	assert.Equal(t, metadata.StatusPending, qm.GetQueue()[0].Status)
	assert.NotZero(t, qm.GetQueue()[0].SubmittedAt)

	// Add multiple valid jobs
	err = qm.AddToQueue(job2)
	assert.NoError(t, err)
	assert.Len(t, qm.GetQueue(), 2)

	// Add invalid job (should be skipped)
	err = qm.AddToQueue(job3Invalid)
	assert.NoError(t, err) // No error, just logs a skip
	assert.Len(t, qm.GetQueue(), 2)

	// Add duplicate job (should be skipped)
	err = qm.AddToQueue(job1Duplicate)
	assert.NoError(t, err) // No error, just logs a skip
	assert.Len(t, qm.GetQueue(), 2)

	// Add mix (valid, duplicate, invalid)
	job4 := createSampleJob("/subs/add4.srt", metadata.StatusPending)
	err = qm.AddToQueue(job4, job1Duplicate, job3Invalid)
	assert.NoError(t, err)
	assert.Len(t, qm.GetQueue(), 3) // Only job4 should be added
	assert.Equal(t, job4.SubtitleInfo.FilePath, qm.GetQueue()[2].SubtitleInfo.FilePath)
}

func TestGetNextPendingJob(t *testing.T) {
	qm, _ := setupTestQueueManager(t)

	job1 := createSampleJob("/subs/pending1.srt", metadata.StatusPending)
	job2 := createSampleJob("/subs/processing.srt", metadata.StatusProcessing)
	job3 := createSampleJob("/subs/pending2.srt", metadata.StatusPending)

	// 1. Empty queue
	nextJob, index := qm.GetNextPendingJob()
	assert.Nil(t, nextJob)
	assert.Equal(t, -1, index)

	// 2. Add jobs
	require.NoError(t, qm.AddToQueue(job1, job2, job3))
	// Queue: [job1(Pending, 0), job2(Processing, 1), job3(Pending, 2)]

	// 3. Get first pending job
	nextJob, index = qm.GetNextPendingJob()
	require.NotNil(t, nextJob)
	assert.Equal(t, 0, index, "Index of first pending job should be 0")
	assert.Equal(t, job1.SubtitleInfo.FilePath, nextJob.SubtitleInfo.FilePath)
	assert.Equal(t, metadata.StatusPending, nextJob.Status)

	// 4. Update first job's status and get next pending job
	firstPendingIndex := index // Store the index we just found (0)
	require.NoError(t, qm.UpdateJobStatus(firstPendingIndex, metadata.StatusReady, ""), "Update status of job at index 0")
	// Queue: [job1(Ready, 0), job2(Processing, 1), job3(Pending, 2)]

	nextJob, index = qm.GetNextPendingJob()
	require.NotNil(t, nextJob, "Should find the next pending job")
	assert.Equal(t, 2, index, "Index of the next pending job should be 2") // job3 is the next pending
	assert.Equal(t, job3.SubtitleInfo.FilePath, nextJob.SubtitleInfo.FilePath)
	assert.Equal(t, metadata.StatusPending, nextJob.Status)

	// 5. Update the last pending job (which is at index 2)
	secondPendingIndex := index // Store the index we just found (2)
	require.NoError(t, qm.UpdateJobStatus(secondPendingIndex, metadata.StatusComplete, ""), "Update status of job at index 2")
	// Queue: [job1(Ready, 0), job2(Processing, 1), job3(Complete, 2)]

	nextJob, index = qm.GetNextPendingJob()
	assert.Nil(t, nextJob, "Should be no more pending jobs")
	assert.Equal(t, -1, index, "Index should be -1 when no pending jobs")
}

func TestUpdateJobStatus(t *testing.T) {
	qm, _ := setupTestQueueManager(t)

	job1 := createSampleJob("/subs/update.srt", metadata.StatusPending)
	require.NoError(t, qm.AddToQueue(job1))

	// 1. Update status and message
	newStatus := metadata.StatusProcessing
	newMessage := "Looking up metadata"
	err := qm.UpdateJobStatus(0, newStatus, newMessage)
	assert.NoError(t, err)
	updatedQueue := qm.GetQueue()
	assert.Len(t, updatedQueue, 1)
	assert.Equal(t, newStatus, updatedQueue[0].Status)
	assert.Equal(t, newMessage, updatedQueue[0].Message)
	assert.Zero(t, updatedQueue[0].CompletedAt) // Should not be set yet

	// 2. Update to a final status (sets CompletedAt)
	finalStatus := metadata.StatusFailed
	finalMessage := "API error"
	err = qm.UpdateJobStatus(0, finalStatus, finalMessage)
	assert.NoError(t, err)
	updatedQueue = qm.GetQueue()
	assert.Equal(t, finalStatus, updatedQueue[0].Status)
	assert.Equal(t, finalMessage, updatedQueue[0].Message)
	assert.NotZero(t, updatedQueue[0].CompletedAt)

	// 3. Invalid index
	err = qm.UpdateJobStatus(1, metadata.StatusPending, "")
	assert.Error(t, err)
	err = qm.UpdateJobStatus(-1, metadata.StatusPending, "")
	assert.Error(t, err)
}

func TestMoveJobToHistory(t *testing.T) {
	qm, _ := setupTestQueueManager(t)

	job1 := createSampleJob("/subs/move1.srt", metadata.StatusComplete)
	job2 := createSampleJob("/subs/move2.srt", metadata.StatusPending)
	require.NoError(t, qm.AddToQueue(job1, job2))

	// 1. Move first job
	job1.Status = metadata.StatusComplete // Simulate completion before move
	require.NoError(t, qm.UpdateJobStatus(0, job1.Status, "Done"))
	err := qm.MoveJobToHistory(0)
	assert.NoError(t, err)
	assert.Len(t, qm.GetQueue(), 1, "Queue should have 1 item left")
	assert.Len(t, qm.GetHistory(), 1, "History should have 1 item")
	assert.Equal(t, job2.SubtitleInfo.FilePath, qm.GetQueue()[0].SubtitleInfo.FilePath) // job2 remains
	assert.Equal(t, job1.SubtitleInfo.FilePath, qm.GetHistory()[0].SubtitleInfo.FilePath)
	assert.Equal(t, job1.Status, qm.GetHistory()[0].Status)

	// 2. Move the remaining job
	job2.Status = metadata.StatusSkipped
	require.NoError(t, qm.UpdateJobStatus(0, job2.Status, "Skipped by user"))
	err = qm.MoveJobToHistory(0)
	assert.NoError(t, err)
	assert.Empty(t, qm.GetQueue(), "Queue should be empty")
	assert.Len(t, qm.GetHistory(), 2, "History should have 2 items")
	assert.Equal(t, job2.SubtitleInfo.FilePath, qm.GetHistory()[0].SubtitleInfo.FilePath) // job2 is now first in history
	assert.Equal(t, job1.SubtitleInfo.FilePath, qm.GetHistory()[1].SubtitleInfo.FilePath)

	// 3. Invalid index
	err = qm.MoveJobToHistory(0) // Queue is empty
	assert.Error(t, err)
}

func TestClearQueue(t *testing.T) {
	qm, tempDir := setupTestQueueManager(t)
	job1 := createSampleJob("/subs/clearQ1.srt", metadata.StatusPending)
	job2 := createSampleJob("/subs/clearQ2.srt", metadata.StatusPending)

	// 1. Clear empty queue
	err := qm.ClearQueue()
	assert.NoError(t, err)
	assert.Empty(t, qm.GetQueue())

	// 2. Add jobs and clear
	require.NoError(t, qm.AddToQueue(job1, job2))
	assert.Len(t, qm.GetQueue(), 2)
	err = qm.ClearQueue()
	assert.NoError(t, err)
	assert.Empty(t, qm.GetQueue())

	// 3. Check persistence (load again)
	qm2, err := queue.NewQueueManager(tempDir, log.New(ioutil.Discard, "", 0))
	require.NoError(t, err)
	assert.Empty(t, qm2.GetQueue(), "Loaded queue should be empty after clearing")
}

func TestClearHistory(t *testing.T) {
	qm, tempDir := setupTestQueueManager(t)
	job1 := createSampleJob("/subs/clearH1.srt", metadata.StatusComplete)

	// 1. Clear empty history
	err := qm.ClearHistory()
	assert.NoError(t, err)
	assert.Empty(t, qm.GetHistory())

	// 2. Add job to history and clear
	require.NoError(t, qm.AddToQueue(job1))
	require.NoError(t, qm.UpdateJobStatus(0, metadata.StatusComplete, ""))
	require.NoError(t, qm.MoveJobToHistory(0))
	assert.Len(t, qm.GetHistory(), 1)
	err = qm.ClearHistory()
	assert.NoError(t, err)
	assert.Empty(t, qm.GetHistory())

	// 3. Check persistence
	qm2, err := queue.NewQueueManager(tempDir, log.New(ioutil.Discard, "", 0))
	require.NoError(t, err)
	assert.Empty(t, qm2.GetHistory(), "Loaded history should be empty after clearing")
}

func TestRemoveJobFromQueue(t *testing.T) {
	qm, _ := setupTestQueueManager(t)
	job1 := createSampleJob("/subs/remove1.srt", metadata.StatusPending)
	job2 := createSampleJob("/subs/remove2.srt", metadata.StatusReady)
	job3 := createSampleJob("/subs/remove3.srt", metadata.StatusPending)
	require.NoError(t, qm.AddToQueue(job1, job2, job3))
	assert.Len(t, qm.GetQueue(), 3)

	// 1. Remove middle job
	err := qm.RemoveJobFromQueue(1)
	assert.NoError(t, err)
	currentQueue := qm.GetQueue()
	assert.Len(t, currentQueue, 2)
	assert.Equal(t, job1.SubtitleInfo.FilePath, currentQueue[0].SubtitleInfo.FilePath)
	assert.Equal(t, job3.SubtitleInfo.FilePath, currentQueue[1].SubtitleInfo.FilePath)

	// 2. Remove first job
	err = qm.RemoveJobFromQueue(0)
	assert.NoError(t, err)
	currentQueue = qm.GetQueue()
	assert.Len(t, currentQueue, 1)
	assert.Equal(t, job3.SubtitleInfo.FilePath, currentQueue[0].SubtitleInfo.FilePath)

	// 3. Remove last job
	err = qm.RemoveJobFromQueue(0)
	assert.NoError(t, err)
	assert.Empty(t, qm.GetQueue())

	// 4. Invalid index
	err = qm.RemoveJobFromQueue(0) // Empty queue
	assert.Error(t, err)
	require.NoError(t, qm.AddToQueue(job1))
	err = qm.RemoveJobFromQueue(1)
	assert.Error(t, err)
	err = qm.RemoveJobFromQueue(-1)
	assert.Error(t, err)
}

// TestConcurrency aims to ensure basic thread safety with locks.
// This is not exhaustive but checks for obvious race conditions.
func TestConcurrency(t *testing.T) {
	qm, _ := setupTestQueueManager(t)
	numGoroutines := 50
	var wg sync.WaitGroup

	wg.Add(numGoroutines * 4) // Add, Update, Move, Remove

	// Add jobs concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(n int) {
			defer wg.Done()
			job := createSampleJob(fmt.Sprintf("/subs/concurrent_%d.srt", n), metadata.StatusPending)
			_ = qm.AddToQueue(job)
		}(i)
	}

	// Give AddToQueue some time to populate before proceeding
	time.Sleep(100 * time.Millisecond)

	// Concurrently update and move jobs
	for i := 0; i < numGoroutines; i++ {
		// Update
		go func(n int) {
			defer wg.Done()
			_, index := qm.GetNextPendingJob() // Find *any* pending job
			if index != -1 {
				_ = qm.UpdateJobStatus(index, metadata.StatusReady, "Processed")
			}
		}(i)

		// Move (find a non-pending job, maybe Ready)
		go func(n int) {
			defer wg.Done()
			// Find a job that is NOT pending to move
			currentQueue := qm.GetQueue() // Get a snapshot
			for idx, job := range currentQueue {
				if job.Status != metadata.StatusPending {
					_ = qm.MoveJobToHistory(idx)
					break // Move one and stop
				}
			}
		}(i)

		// Remove (find any job)
		go func(n int) {
			defer wg.Done()
			currentQueue := qm.GetQueue()
			if len(currentQueue) > 0 {
				_ = qm.RemoveJobFromQueue(0) // Just remove the first available
			}
		}(i)
	}

	wg.Wait()

	// No specific assertion on final counts due to race nature,
	// but the primary goal is to ensure no deadlocks or panics occur.
	t.Logf("Concurrency test finished. Final Queue: %d, History: %d", len(qm.GetQueue()), len(qm.GetHistory()))
}
