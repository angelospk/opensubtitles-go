package queue_test

import (
	"fmt"
	"io/ioutil"
	// "log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	"github.com/angelospk/osuploadergui/pkg/core/queue"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Helper to create a QueueManager for testing
func setupTestQueueManager(t *testing.T) (*queue.QueueManager, func()) {
	tempDir := t.TempDir()
	// Use logrus logger, discard output for most tests
	logger := log.New()
	logger.SetOutput(ioutil.Discard) // Discard logs unless debugging
	// logger.SetOutput(os.Stdout) // Uncomment to see logs
	logger.SetLevel(log.InfoLevel)

	qm, err := queue.NewQueueManager(tempDir, logger)
	assert.NoError(t, err, "Failed to create QueueManager for testing")
	return qm, func() { os.RemoveAll(tempDir) } // Cleanup function
}

func TestQueueManager_Initialization(t *testing.T) {
	qm, cleanup := setupTestQueueManager(t)
	defer cleanup()

	assert.NotNil(t, qm)
	assert.Empty(t, qm.GetQueue(), "Initial queue should be empty")
	assert.Empty(t, qm.GetHistory(), "Initial history should be empty")
}

func TestQueueManager_Persistence(t *testing.T) {
	tempDir := t.TempDir() // Need consistent dir across instances
	defer os.RemoveAll(tempDir)

	// Use logrus logger
	logger := log.New()
	logger.SetOutput(ioutil.Discard)
	logger.SetLevel(log.InfoLevel)

	// Instance 1: Add items and save
	qm1, err := queue.NewQueueManager(tempDir, logger)
	assert.NoError(t, err)

	job1 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub1.srt"}}
	job2 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub2.srt"}}
	added, skipped := qm1.AddToQueue([]metadata.UploadJob{job1, job2})
	assert.Equal(t, 2, added)
	assert.Equal(t, 0, skipped)
	// AddToQueue already saves, but call explicitly for clarity if needed
	err = qm1.SaveQueueState()
	assert.NoError(t, err)
	err = qm1.SaveHistory() // Save empty history
	assert.NoError(t, err)

	// Instance 2: Load state
	qm2, err := queue.NewQueueManager(tempDir, logger)
	assert.NoError(t, err)
	assert.Len(t, qm2.GetQueue(), 2, "Queue should have 2 items after loading")
	assert.Len(t, qm2.GetHistory(), 0, "History should be empty after loading")
	assert.Equal(t, "sub1.srt", qm2.GetQueue()[0].SubtitleInfo.FilePath)
}

func TestQueueManager_AddToQueue(t *testing.T) {
	qm, cleanup := setupTestQueueManager(t)
	defer cleanup()

	job1 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub1.srt"}}
	job2 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub2.srt"}}
	job3Invalid := metadata.UploadJob{} // Missing SubtitleInfo
	job1Duplicate := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub1.srt"}}

	// Add valid jobs
	added, skipped := qm.AddToQueue([]metadata.UploadJob{job1})
	assert.Equal(t, 1, added)
	assert.Equal(t, 0, skipped)
	assert.Len(t, qm.GetQueue(), 1)
	assert.Equal(t, metadata.StatusPending, qm.GetQueue()[0].Status)
	assert.NotZero(t, qm.GetQueue()[0].SubmittedAt)

	added, skipped = qm.AddToQueue([]metadata.UploadJob{job2})
	assert.Equal(t, 1, added)
	assert.Equal(t, 0, skipped)
	assert.Len(t, qm.GetQueue(), 2)

	// Add invalid job
	added, skipped = qm.AddToQueue([]metadata.UploadJob{job3Invalid})
	assert.Equal(t, 0, added)
	assert.Equal(t, 1, skipped)
	assert.Len(t, qm.GetQueue(), 2)

	// Add duplicate job
	added, skipped = qm.AddToQueue([]metadata.UploadJob{job1Duplicate})
	assert.Equal(t, 0, added)
	assert.Equal(t, 1, skipped)
	assert.Len(t, qm.GetQueue(), 2)

	// Add multiple at once (valid + duplicate)
	job4 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub4.srt"}}
	added, skipped = qm.AddToQueue([]metadata.UploadJob{job1Duplicate, job4})
	assert.Equal(t, 1, added)
	assert.Equal(t, 1, skipped)
	assert.Len(t, qm.GetQueue(), 3)
	assert.Equal(t, "sub4.srt", qm.GetQueue()[2].SubtitleInfo.FilePath)
}

func TestQueueManager_GetNextPendingJob(t *testing.T) {
	qm, cleanup := setupTestQueueManager(t)
	defer cleanup()

	job1 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub1.srt"}, Status: metadata.StatusPending}
	job2 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub2.srt"}, Status: metadata.StatusUploading}
	job3 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub3.srt"}, Status: metadata.StatusPending}

	// No jobs
	nextJob := qm.GetNextPendingJob()
	assert.Nil(t, nextJob)

	// Add jobs
	_, _ = qm.AddToQueue([]metadata.UploadJob{job1, job2, job3})

	// Get first pending
	nextJob = qm.GetNextPendingJob()
	assert.NotNil(t, nextJob)
	assert.Equal(t, "sub1.srt", nextJob.SubtitleInfo.FilePath)

	// Mark first as non-pending and get next
	// Use jobID (FilePath) to update
	err := qm.UpdateJobStatus(job1.SubtitleInfo.FilePath, metadata.StatusComplete, "Done")
	assert.NoError(t, err)

	nextJob = qm.GetNextPendingJob()
	assert.NotNil(t, nextJob)
	assert.Equal(t, "sub3.srt", nextJob.SubtitleInfo.FilePath)

	// Mark last pending as non-pending
	err = qm.UpdateJobStatus(job3.SubtitleInfo.FilePath, metadata.StatusFailed, "Error")
	assert.NoError(t, err)

	nextJob = qm.GetNextPendingJob()
	assert.Nil(t, nextJob, "Should be no pending jobs left")
}

func TestQueueManager_UpdateJobStatus(t *testing.T) {
	qm, cleanup := setupTestQueueManager(t)
	defer cleanup()

	job1 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub1.srt"}}
	jobID1 := job1.SubtitleInfo.FilePath
	_, _ = qm.AddToQueue([]metadata.UploadJob{job1})

	// Test valid update
	err := qm.UpdateJobStatus(jobID1, metadata.StatusUploading, "Processing...")
	assert.NoError(t, err)
	queue := qm.GetQueue()
	assert.Len(t, queue, 1)
	assert.Equal(t, metadata.StatusUploading, queue[0].Status)
	assert.Equal(t, "Processing...", queue[0].Message)
	assert.True(t, queue[0].CompletedAt.IsZero(), "CompletedAt should not be set yet")

	// Test setting completed status
	err = qm.UpdateJobStatus(jobID1, metadata.StatusComplete, "Success")
	assert.NoError(t, err)
	queue = qm.GetQueue()
	assert.Equal(t, metadata.StatusComplete, queue[0].Status)
	assert.False(t, queue[0].CompletedAt.IsZero(), "CompletedAt should be set")

	// Test invalid job ID
	err = qm.UpdateJobStatus("nonexistent.srt", metadata.StatusFailed, "Not found")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in queue")
}

func TestQueueManager_MoveJobToHistory(t *testing.T) {
	qm, cleanup := setupTestQueueManager(t)
	defer cleanup()

	job1 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub1.srt"}}
	job2 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub2.srt"}}
	jobID1 := job1.SubtitleInfo.FilePath
	jobID2 := job2.SubtitleInfo.FilePath

	_, _ = qm.AddToQueue([]metadata.UploadJob{job1, job2})
	assert.Len(t, qm.GetQueue(), 2)
	assert.Len(t, qm.GetHistory(), 0)

	// Move first job
	err := qm.MoveJobToHistory(jobID1)
	assert.NoError(t, err)
	assert.Len(t, qm.GetQueue(), 1, "Queue should have 1 item left")
	assert.Len(t, qm.GetHistory(), 1, "History should have 1 item")
	assert.Equal(t, jobID2, qm.GetQueue()[0].SubtitleInfo.FilePath, "Remaining job should be job2")
	assert.Equal(t, jobID1, qm.GetHistory()[0].SubtitleInfo.FilePath, "Moved job should be job1")

	// Move second job
	err = qm.MoveJobToHistory(jobID2)
	assert.NoError(t, err)
	assert.Len(t, qm.GetQueue(), 0)
	assert.Len(t, qm.GetHistory(), 2)
	assert.Equal(t, jobID2, qm.GetHistory()[0].SubtitleInfo.FilePath) // Prepended
	assert.Equal(t, jobID1, qm.GetHistory()[1].SubtitleInfo.FilePath)

	// Move non-existent job
	err = qm.MoveJobToHistory("nonexistent.srt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in queue")
}

func TestQueueManager_Clear(t *testing.T) {
	qm, cleanup := setupTestQueueManager(t)
	defer cleanup()

	job1 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub1.srt"}}
	job2 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub2.srt"}}
	_, _ = qm.AddToQueue([]metadata.UploadJob{job1, job2})
	err := qm.MoveJobToHistory(job1.SubtitleInfo.FilePath)
	assert.NoError(t, err)

	assert.Len(t, qm.GetQueue(), 1)
	assert.Len(t, qm.GetHistory(), 1)

	err = qm.ClearQueue()
	assert.NoError(t, err)
	assert.Len(t, qm.GetQueue(), 0)
	assert.Len(t, qm.GetHistory(), 1) // History should remain

	err = qm.ClearHistory()
	assert.NoError(t, err)
	assert.Len(t, qm.GetHistory(), 0)
}

func TestQueueManager_RemoveJobFromQueue(t *testing.T) {
	qm, cleanup := setupTestQueueManager(t)
	defer cleanup()

	job1 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub1.srt"}}
	job2 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub2.srt"}}
	job3 := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: "sub3.srt"}}
	jobID1 := job1.SubtitleInfo.FilePath
	jobID2 := job2.SubtitleInfo.FilePath
	jobID3 := job3.SubtitleInfo.FilePath

	_, _ = qm.AddToQueue([]metadata.UploadJob{job1, job2, job3})
	assert.Len(t, qm.GetQueue(), 3)

	// Remove middle job
	err := qm.RemoveJobFromQueue(jobID2)
	assert.NoError(t, err)
	queue := qm.GetQueue()
	assert.Len(t, queue, 2)
	assert.Equal(t, jobID1, queue[0].SubtitleInfo.FilePath)
	assert.Equal(t, jobID3, queue[1].SubtitleInfo.FilePath)

	// Remove first job
	err = qm.RemoveJobFromQueue(jobID1)
	assert.NoError(t, err)
	queue = qm.GetQueue()
	assert.Len(t, queue, 1)
	assert.Equal(t, jobID3, queue[0].SubtitleInfo.FilePath)

	// Remove last job
	err = qm.RemoveJobFromQueue(jobID3)
	assert.NoError(t, err)
	assert.Empty(t, qm.GetQueue())

	// Remove non-existent job
	err = qm.RemoveJobFromQueue("nonexistent.srt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found in queue")
}

// TestQueueManager_Concurrency tests basic concurrent access.
func TestQueueManager_Concurrency(t *testing.T) {
	qm, cleanup := setupTestQueueManager(t)
	defer cleanup()

	var wg sync.WaitGroup
	numGoroutines := 50
	numJobsPerRoutine := 10

	// Concurrent Adds
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			jobsToAdd := []metadata.UploadJob{}
			for j := 0; j < numJobsPerRoutine; j++ {
				job := metadata.UploadJob{SubtitleInfo: &metadata.SubtitleInfo{FilePath: fmt.Sprintf("sub_%d_%d.srt", routineID, j)}}
				jobsToAdd = append(jobsToAdd, job)
			}
			_, _ = qm.AddToQueue(jobsToAdd)
		}(i)
	}
	wg.Wait()

	assert.Len(t, qm.GetQueue(), numGoroutines*numJobsPerRoutine, "All unique jobs should be added")

	// Concurrent GetNext, Update, Move
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numJobsPerRoutine; j++ {
				job := qm.GetNextPendingJob()
				if job != nil {
					jobID := job.SubtitleInfo.FilePath // Use ID for update/move
					_ = qm.UpdateJobStatus(jobID, metadata.StatusUploading, "Started")
					// Simulate work
					time.Sleep(time.Millisecond * time.Duration(j%5+1))
					_ = qm.UpdateJobStatus(jobID, metadata.StatusComplete, "Done")
					_ = qm.MoveJobToHistory(jobID)
				}
			}
		}()
	}
	wg.Wait()

	assert.Empty(t, qm.GetQueue(), "Queue should be empty after processing")
	assert.Len(t, qm.GetHistory(), numGoroutines*numJobsPerRoutine, "History should contain all processed jobs")
}
