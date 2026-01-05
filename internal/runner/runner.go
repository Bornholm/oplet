package runner

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/bornholm/oplet/internal/slogx"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/task"
	"github.com/pkg/errors"
)

type Runner struct {
	serverURL         *url.URL
	authToken         string
	http              *http.Client
	executor          task.Executor
	logger            *slog.Logger
	executionInterval time.Duration
	client            *Client
}

func (r *Runner) Run(ctx context.Context) error {
	// Send initial heartbeat
	if _, err := r.client.SendHeartbeat(ctx); err != nil {
		r.logger.WarnContext(ctx, "failed to send initial heartbeat", slogx.Error(err))
	}

	// Start heartbeat ticker
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	// Start main execution loop
	executionTicker := time.NewTicker(r.executionInterval)
	defer executionTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.WithStack(ctx.Err())
		case <-heartbeatTicker.C:
			if _, err := r.client.SendHeartbeat(ctx); err != nil {
				r.logger.WarnContext(ctx, "failed to send heartbeat", slogx.Error(err))
			}
		case <-executionTicker.C:
			if err := r.executeNextTask(ctx); err != nil {
				r.logger.ErrorContext(ctx, "task execution error", slogx.Error(err))
				// Continue running even if a task fails
			}
		}
	}
}

func (r *Runner) executeNextTask(ctx context.Context) error {
	// Request next task
	taskResp, err := r.client.RequestTask(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to request task")
	}

	if taskResp == nil {
		// No tasks available
		return nil
	}

	r.logger.InfoContext(ctx, "received task assignment",
		"execution_id", taskResp.ExecutionID,
		"task_id", taskResp.TaskID,
		"image_ref", taskResp.ImageRef)

	// Execute the task
	return r.executeTask(ctx, taskResp)
}

func (r *Runner) executeTask(ctx context.Context, taskResp *TaskRequestResponse) error {
	// Update status to indicate we're starting
	if err := r.client.UpdateTaskStatus(ctx, taskResp.TaskID, TaskStatusRequest{
		Status:    store.StatusPullingImage,
		StartedAt: timePtr(time.Now()),
	}); err != nil {
		r.logger.WarnContext(ctx, "failed to update task status", slogx.Error(err))
	}

	// Download input files
	inputs, err := r.downloadInputFiles(ctx, taskResp)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to download input files",
			"execution_id", taskResp.ExecutionID,
			"error", err)

		// Update status to failed
		if statusErr := r.client.UpdateTaskStatus(ctx, taskResp.TaskID, TaskStatusRequest{
			Status:     store.StatusFailed,
			Error:      err.Error(),
			FinishedAt: timePtr(time.Now()),
		}); statusErr != nil {
			r.logger.WarnContext(ctx, "failed to update failed task status", slogx.Error(statusErr))
		}

		return errors.Wrap(err, "failed to download input files")
	}

	// Create execution request
	execReq := task.ExecutionRequest{
		ImageRef:    taskResp.ImageRef,
		Environment: taskResp.Environment,
		Inputs:      inputs,
		OnChange:    r.createExecutionCallback(ctx, taskResp),
	}

	// Execute the task
	if err := r.executor.Execute(ctx, execReq); err != nil {
		r.logger.ErrorContext(ctx, "task execution failed",
			"execution_id", taskResp.ExecutionID,
			"error", err)

		// Update status to failed
		if statusErr := r.client.UpdateTaskStatus(ctx, taskResp.TaskID, TaskStatusRequest{
			Status:     store.StatusFailed,
			Error:      err.Error(),
			FinishedAt: timePtr(time.Now()),
		}); statusErr != nil {
			r.logger.WarnContext(ctx, "failed to update failed task status", slogx.Error(statusErr))
		}

		return errors.Wrap(err, "task execution failed")
	}

	return nil
}

func (r *Runner) downloadInputFiles(ctx context.Context, taskResp *TaskRequestResponse) (map[string]io.ReadCloser, error) {
	inputs := make(map[string]io.ReadCloser)

	// List available input files
	fileList, err := r.client.ListInputFiles(ctx, taskResp.TaskID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list input files")
	}

	// Download each input file using parameter name as key
	for _, fileInfo := range fileList {
		parameterName, ok := fileInfo["filename"].(string)
		if !ok {
			r.logger.WarnContext(ctx, "invalid filename in file list", "file_info", fileInfo)
			continue
		}

		fileReader, err := r.client.DownloadInputFile(ctx, taskResp.TaskID, parameterName)
		if err != nil {
			r.logger.WarnContext(ctx, "failed to download input file",
				"parameter_name", parameterName, "error", err)
			continue
		}

		// Use parameter name as key so file is positioned correctly in /oplet/inputs
		inputs[parameterName] = fileReader

		r.logger.InfoContext(ctx, "downloaded input file",
			"execution_id", taskResp.ExecutionID,
			"parameter_name", parameterName)
	}

	return inputs, nil
}

func (r *Runner) uploadOutputFiles(ctx context.Context, taskResp *TaskRequestResponse, outputs *tar.Reader) {
	if outputs == nil {
		return
	}

	// Create a map to collect output files
	outputFiles := make(map[string]io.Reader)

	// Read all files from the tar archive
	for {
		header, err := outputs.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			r.logger.ErrorContext(ctx, "failed to read output tar",
				"execution_id", taskResp.ExecutionID,
				"error", err)
			return
		}

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		filename := filepath.Base(header.Name)

		// Read file content into memory
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, outputs); err != nil {
			r.logger.WarnContext(ctx, "failed to read output file content",
				"execution_id", taskResp.ExecutionID,
				"filename", filename,
				"error", err)
			continue
		}

		outputFiles[filename] = bytes.NewReader(buf.Bytes())

		r.logger.InfoContext(ctx, "prepared output file for upload",
			"execution_id", taskResp.ExecutionID,
			"filename", filename,
			"size", buf.Len())
	}

	// Upload all output files
	if len(outputFiles) > 0 {
		if err := r.client.UploadOutputFiles(ctx, taskResp.TaskID, outputFiles); err != nil {
			r.logger.ErrorContext(ctx, "failed to upload output files",
				"execution_id", taskResp.ExecutionID,
				"error", err)
		} else {
			r.logger.InfoContext(ctx, "successfully uploaded output files",
				"execution_id", taskResp.ExecutionID,
				"file_count", len(outputFiles))
		}
	}
}

func (r *Runner) createExecutionCallback(ctx context.Context, taskResp *TaskRequestResponse) func(task.Execution) {
	return func(e task.Execution) {
		// Map execution state to task status
		status := r.mapExecutionStateToStatus(e.State)

		statusReq := TaskStatusRequest{
			Status:      status,
			ContainerID: e.ContainerID,
			Timestamp:   time.Now().UnixMicro(),
		}

		if !e.StartedAt.IsZero() {
			statusReq.StartedAt = &e.StartedAt
		}
		if !e.FinishedAt.IsZero() {
			statusReq.FinishedAt = &e.FinishedAt
		}
		if e.ExitCode != 0 {
			statusReq.ExitCode = &e.ExitCode
		}
		if e.Error != nil {
			statusReq.Error = e.Error.Error()
		}

		// Update task status
		if err := r.client.UpdateTaskStatus(ctx, taskResp.TaskID, statusReq); err != nil {
			r.logger.WarnContext(ctx, "failed to update task status",
				"execution_id", taskResp.ExecutionID,
				"state", e.State,
				"error", err)
		}

		// Handle specific states
		switch e.State {
		case task.ExecutionStateContainerStarted:
			r.startLogStreaming(ctx, taskResp, e.ContainerID)
		case task.ExecutionStateFilesDownloaded:
			// Upload output files when they are downloaded from container
			if e.Outputs != nil {
				r.uploadOutputFiles(ctx, taskResp, e.Outputs)
			}
		case task.ExecutionStateSucceeded:
			r.logger.InfoContext(ctx, "task execution succeeded",
				"execution_id", taskResp.ExecutionID)
		case task.ExecutionStateFailed:
			r.logger.ErrorContext(ctx, "task execution failed",
				"execution_id", taskResp.ExecutionID,
				"error", e.Error)
		}
	}
}

func (r *Runner) startLogStreaming(ctx context.Context, taskResp *TaskRequestResponse, containerID string) {
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				r.logger.ErrorContext(ctx, "panic in log streaming",
					"execution_id", taskResp.ExecutionID,
					"panic", rec)
			}
		}()

		var localClock uint = 0

		logs := make([]LogEntry, 0)
		submitLogs := func() {
			if len(logs) == 0 {
				return
			}

			if submitErr := r.client.SubmitLogs(ctx, taskResp.TaskID, logs); submitErr != nil {
				r.logger.WarnContext(ctx, "failed to submit logs",
					"execution_id", taskResp.ExecutionID,
					"error", submitErr)
			}

			logs = make([]LogEntry, 0)
		}

		defer submitLogs()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		defer func() {
			r.logger.InfoContext(ctx, "stopped streaming logs")
		}()

		for {
			logs = make([]LogEntry, 0)

			// Get container logs
			logEntries, err := r.executor.GetLogs(ctx, containerID)
			if err != nil {
				r.logger.ErrorContext(ctx, "failed to get container logs",
					"execution_id", taskResp.ExecutionID,
					"container_id", containerID,
					"error", errors.Cause(err))

				if errors.Is(err, task.ErrContainerNotFound) {
					return
				}

				continue
			}

			for {
				select {
				case <-ctx.Done():
					return

				case e, ok := <-logEntries:
					if !ok {
						submitLogs()
						break
					}

					if e.Clock >= localClock {
						logs = append(logs, LogEntry{
							Timestamp: e.Timestamp.UnixMicro(),
							Source:    "container",
							Message:   e.Message,
							Clock:     e.Clock,
						})
						localClock = e.Clock
					}

				case <-ticker.C:
					submitLogs()
				}
			}
		}

	}()
}

func (r *Runner) mapExecutionStateToStatus(state task.ExecutionState) store.TaskExecutionStatus {
	switch state {
	case task.ExecutionStateProcessingRequest:
		return store.StatusPending
	case task.ExecutionStatePullingImage:
		return store.StatusPullingImage
	case task.ExecutionStateImagePulled:
		return store.StatusImagePulled
	case task.ExecutionStateCreatingContainer:
		return store.StatusCreatingContainer
	case task.ExecutionStateContainerCreated:
		return store.StatusContainerCreated
	case task.ExecutionStateUploadingFiles:
		return store.StatusUploadingFiles
	case task.ExecutionStateFilesUploaded:
		return store.StatusFilesUploaded
	case task.ExecutionStateStartingContainer:
		return store.StatusStartingContainer
	case task.ExecutionStateContainerStarted:
		return store.StatusContainerStarted
	case task.ExecutionStateContainerRunning:
		return store.StatusRunning
	case task.ExecutionStateContainerFinished:
		return store.StatusFinished
	case task.ExecutionStateDownloadingFiles:
		return store.StatusDownloadingFiles
	case task.ExecutionStateFilesDownloaded:
		return store.StatusFilesDownloaded
	case task.ExecutionStateSucceeded:
		return store.StatusSucceeded
	case task.ExecutionStateFailed:
		return store.StatusFailed
	case task.ExecutionStateKilled:
		return store.StatusKilled
	default:
		return store.StatusPending
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}

func New(rawServerURL string, authToken string, funcs ...OptionFunc) (*Runner, error) {
	opts, err := NewOptions(funcs...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	serverURL, err := url.Parse(rawServerURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	client, err := NewClient(rawServerURL, authToken, opts.HTTPClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create API client")
	}

	return &Runner{
		serverURL:         serverURL,
		authToken:         authToken,
		http:              opts.HTTPClient,
		executor:          opts.Executor,
		logger:            opts.Logger.With("component", "runner"),
		executionInterval: opts.ExecutionInterval,
		client:            client,
	}, nil
}
