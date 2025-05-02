package cmd_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/angelospk/osuploadergui/cmd/cli/cmd"
	"github.com/angelospk/osuploadergui/pkg/core/metadata"
	"github.com/angelospk/osuploadergui/pkg/core/queue"
	"github.com/angelospk/osuploadergui/pkg/processor"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks for Upload Command Dependencies ---

type MockProcessor struct {
	mock.Mock
}

func (m *MockProcessor) CreateJobsFromDirectory(ctx context.Context, dirPath string, recursive bool) ([]metadata.UploadJob, error) {
	args := m.Called(ctx, dirPath, recursive)
	// Return nil for jobs slice if first arg is nil
	var jobs []metadata.UploadJob
	if args.Get(0) != nil {
		jobs = args.Get(0).([]metadata.UploadJob)
	}
	return jobs, args.Error(1)
}

type MockQueueManager struct {
	mock.Mock
}

func (m *MockQueueManager) AddToQueue(jobs []metadata.UploadJob) (addedCount int, skippedCount int) {
	args := m.Called(jobs)
	return args.Int(0), args.Int(1)
}

// Implement other QueueManager methods if needed by other tests or command logic
func (m *MockQueueManager) LoadQueueState() error                  { return nil }
func (m *MockQueueManager) SaveQueueState() error                  { return nil }
func (m *MockQueueManager) GetNextPendingJob() *metadata.UploadJob { return nil }
func (m *MockQueueManager) UpdateJobStatus(jobID string, status metadata.JobStatus, message string) error {
	return nil
}
func (m *MockQueueManager) MoveJobToHistory(jobID string) error   { return nil }
func (m *MockQueueManager) ClearQueue() error                     { return nil }
func (m *MockQueueManager) ClearHistory() error                   { return nil }
func (m *MockQueueManager) RemoveJobFromQueue(jobID string) error { return nil }
func (m *MockQueueManager) GetQueue() []metadata.UploadJob        { return nil }
func (m *MockQueueManager) GetHistory() []metadata.UploadJob      { return nil }

// Helper function to execute upload command with mocks
func executeUploadCommand(t *testing.T, mockProc *MockProcessor, mockQueue *MockQueueManager, args []string) (string, string, error) {
	// Store and replace dependency creation functions (assuming refactor in upload.go)
	originalNewProcessor := cmd.NewProcessorFunc
	originalNewQueueManager := cmd.NewQueueManagerFunc
	defer func() {
		cmd.NewProcessorFunc = originalNewProcessor
		cmd.NewQueueManagerFunc = originalNewQueueManager
	}()

	cmd.NewProcessorFunc = func(apiProvider metadata.APIClientProvider, logger *logrus.Logger) processor.ProcessorInterface {
		return mockProc
	}
	cmd.NewQueueManagerFunc = func(queueFile, historyFile string, logger *logrus.Logger) (queue.QueueManagerInterface, error) {
		return mockQueue, nil
	}

	outBuf := bytes.NewBufferString("")
	errBuf := bytes.NewBufferString("")
	cmd.RootCmd.SetOut(outBuf)
	cmd.RootCmd.SetErr(errBuf)
	cmd.RootCmd.SetArgs(append([]string{"upload"}, args...))

	// Disable actual API client creation for upload test
	originalNewAPIProvider := cmd.NewAPIProviderFunc
	defer func() { cmd.NewAPIProviderFunc = originalNewAPIProvider }()
	cmd.NewAPIProviderFunc = func(apiKey string) (metadata.APIClientProvider, error) {
		// Return empty provider as processor is mocked
		return metadata.APIClientProvider{}, nil
	}

	err := cmd.RootCmd.Execute()

	// Reset args
	cmd.RootCmd.SetArgs([]string{})
	return outBuf.String(), errBuf.String(), err
}

// TestUploadCommand_NonExistentPath tests if the command handles non-existent paths.
func TestUploadCommand_NonExistentPath(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	errorBuffer := bytes.NewBufferString("")
	cmd.RootCmd.SetOut(outputBuffer)
	cmd.RootCmd.SetErr(errorBuffer)

	nonExistentPath := filepath.Join(t.TempDir(), "does_not_exist")
	cmd.RootCmd.SetArgs([]string{"upload", nonExistentPath})

	// ExecuteC captures the error instead of os.Exit(1)
	_, err := cmd.RootCmd.ExecuteC()

	// Expect error from RunE (or potentially ExecuteC itself if arg parsing failed)
	assert.Error(t, err, "Expected an error for non-existent path")

	// Check the returned error message OR the stderr buffer for the logged message
	if err != nil {
		// Prefer checking the returned error message if available
		assert.Contains(t, err.Error(), "path does not exist:", "Expected error message content")
	} else {
		// Fallback: check stderr buffer if no error was returned (less ideal)
		errorOutput := errorBuffer.String()
		assert.Contains(t, errorOutput, "Error: Path does not exist:", "Expected specific error message in logs")
	}

	// Reset args
	cmd.RootCmd.SetArgs([]string{})
}

func TestUploadCommand_Success_SinglePath(t *testing.T) {
	mockProc := new(MockProcessor)
	mockQueue := new(MockQueueManager)

	tempDir := t.TempDir()
	dummyFilePath := filepath.Join(tempDir, "dummy.mkv")
	file, _ := os.Create(dummyFilePath)
	file.Close()

	expectedJobs := []metadata.UploadJob{
		{VideoInfo: &metadata.VideoInfo{FileName: "dummy.mkv"}, Status: metadata.StatusReady},
	}

	mockProc.On("CreateJobsFromDirectory", mock.Anything, tempDir, false).Return(expectedJobs, nil).Once()
	mockQueue.On("AddToQueue", expectedJobs).Return(1, 0).Once()

	args := []string{tempDir} // Pass the directory containing the dummy file
	output, errOutput, err := executeUploadCommand(t, mockProc, mockQueue, args)

	assert.NoError(t, err)
	assert.Empty(t, errOutput)
	assert.Contains(t, output, "Scanning path:")
	assert.Contains(t, output, tempDir)
	assert.Contains(t, output, "Found 1 potential jobs.")
	assert.Contains(t, output, "Added 1 jobs to the queue (0 skipped).")

	mockProc.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
}

func TestUploadCommand_Success_Recursive(t *testing.T) {
	mockProc := new(MockProcessor)
	mockQueue := new(MockQueueManager)

	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	os.Mkdir(subDir, 0755)
	dummyFilePath := filepath.Join(subDir, "dummy.avi") // Put file in subdir
	file, _ := os.Create(dummyFilePath)
	file.Close()

	expectedJobs := []metadata.UploadJob{
		{VideoInfo: &metadata.VideoInfo{FileName: "dummy.avi"}, Status: metadata.StatusReady},
	}

	// Expect recursive flag to be true
	mockProc.On("CreateJobsFromDirectory", mock.Anything, tempDir, true).Return(expectedJobs, nil).Once()
	mockQueue.On("AddToQueue", expectedJobs).Return(1, 0).Once()

	args := []string{"--recursive", tempDir} // Pass the parent directory
	output, errOutput, err := executeUploadCommand(t, mockProc, mockQueue, args)

	assert.NoError(t, err)
	assert.Empty(t, errOutput)
	assert.Contains(t, output, "Scanning path:")
	assert.Contains(t, output, tempDir)
	assert.Contains(t, output, "Found 1 potential jobs.")
	assert.Contains(t, output, "Added 1 jobs to the queue (0 skipped).")

	mockProc.AssertExpectations(t)
	mockQueue.AssertExpectations(t)
}

func TestUploadCommand_ProcessorError(t *testing.T) {
	mockProc := new(MockProcessor)
	mockQueue := new(MockQueueManager)

	tempDir := t.TempDir()

	mockProc.On("CreateJobsFromDirectory", mock.Anything, tempDir, false).Return(nil, assert.AnError).Once()

	args := []string{tempDir}
	_, errOutput, err := executeUploadCommand(t, mockProc, mockQueue, args)

	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	assert.Contains(t, err.Error(), "failed to create jobs from path")
	assert.Contains(t, errOutput, "Error creating jobs") // Check log message

	mockProc.AssertExpectations(t)
	mockQueue.AssertNotCalled(t, "AddToQueue", mock.Anything)
}

// TODO: Add tests for flag parsing (e.g., --recursive)
