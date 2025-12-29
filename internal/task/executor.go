package task

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
)

const (
	InputsDir  string = "/oplet/inputs"
	OutputsDir string = "/oplet/outputs"
)

// ExecutionRequest represents a container execution request
type ExecutionRequest struct {
	ImageRef    string                   // Docker image reference
	Environment map[string]string        // Environment variables
	Inputs      map[string]io.ReadCloser // Files to upload to container
	Timeout     time.Duration            // Execution timeout (optional)

	OnChange func(Execution)
}

type ExecutionState int

const (
	ExecutionStateProcessingRequest ExecutionState = iota
	ExecutionStatePullingImage
	ExecutionStateImagePulled
	ExecutionStateCreatingContainer
	ExecutionStateContainerCreated
	ExecutionStateUploadingFiles
	ExecutionStateFilesUploaded
	ExecutionStateStartingContainer
	ExecutionStateContainerStarted
	ExecutionStateContainerRunning
	ExecutionStateContainerFinished
	ExecutionStateDownloadingFiles
	ExecutionStateFilesDownloaded
	ExecutionStateSucceeded
	ExecutionStateFailed
	ExecutionStateKilled
)

// ExecutionResult represents the result of container execution
type Execution struct {
	ContainerID string // Docker container ID
	State       ExecutionState
	ExitCode    int       // Container exit code
	StartedAt   time.Time // Execution start time
	FinishedAt  time.Time // Execution finish time
	Error       error     // Execution error (if any)
	Inputs      map[string]io.ReadCloser
	Outputs     *tar.Reader
}

// Executor defines the interface for container execution
type Executor interface {
	// Execute runs a container and returns the execution result
	Execute(ctx context.Context, req ExecutionRequest) error

	// GetLogs streams logs from a running container
	GetLogs(ctx context.Context, containerID string) (io.ReadCloser, error)
}

// ContainerInfo provides information about a container
type ContainerInfo struct {
	ID       string            // Container ID
	ImageRef string            // Image reference
	Status   ContainerStatus   // Current status
	Labels   map[string]string // Container labels
	Created  time.Time         // Creation time
	Started  time.Time         // Start time (if started)
	Finished time.Time         // Finish time (if finished)
}

// ContainerStatus represents the status of a container
type ContainerStatus string

const (
	ContainerStatusCreated ContainerStatus = "created"
	ContainerStatusRunning ContainerStatus = "running"
	ContainerStatusExited  ContainerStatus = "exited"
	ContainerStatusError   ContainerStatus = "error"
)

// ExecutionError provides detailed error information
type ExecutionError struct {
	Type        ExecutionErrorType // Error type
	Message     string             // Error message
	ContainerID string             // Container ID (if available)
	ExitCode    int                // Container exit code (if available)
	Cause       error              // Underlying error
}

func (e *ExecutionError) Error() string {
	if e.ContainerID != "" {
		return fmt.Sprintf("%s (container: %s): %s", e.Type, e.ContainerID, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *ExecutionError) Unwrap() error {
	return e.Cause
}

// ExecutionErrorType represents different types of execution errors
type ExecutionErrorType string

const (
	ErrorTypeImagePullFailed       ExecutionErrorType = "image_pull_failed"
	ErrorTypeContainerFailed       ExecutionErrorType = "container_failed"
	ErrorTypeTimeout               ExecutionErrorType = "timeout"
	ErrorTypeInsufficientResources ExecutionErrorType = "insufficient_resources"
	ErrorTypeFileUploadFailed      ExecutionErrorType = "file_upload_failed"
	ErrorTypeFileDownloadFailed    ExecutionErrorType = "file_download_failed"
	ErrorTypeNetworkError          ExecutionErrorType = "network_error"
	ErrorTypeDockerDaemonError     ExecutionErrorType = "docker_daemon_error"
)

// Predefined errors
var (
	ErrContainerFailed         = errors.New("container execution failed")
	ErrTimeout                 = errors.New("execution timeout")
	ErrImageNotFound           = errors.New("image not found")
	ErrInsufficientResources   = errors.New("insufficient resources")
	ErrFileUploadFailed        = errors.New("file upload failed")
	ErrFileDownloadFailed      = errors.New("file download failed")
	ErrDockerDaemonUnavailable = errors.New("docker daemon unavailable")
)
