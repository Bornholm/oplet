package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/bornholm/oplet/internal/slogx"
	"github.com/bornholm/oplet/internal/task"
)

// DockerExecutor implements task.Executor using Docker client
type DockerExecutor struct {
	client *client.Client
	logger *slog.Logger
}

// NewExecutor creates a new Docker executor
func NewExecutor(logger *slog.Logger) (*DockerExecutor, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Docker client")
	}

	return &DockerExecutor{
		client: cli,
		logger: logger.With("component", "docker-executor"),
	}, nil
}

// Execute implements task.Executor.Execute
func (e *DockerExecutor) Execute(ctx context.Context, req task.ExecutionRequest) error {
	e.logger.Info("starting container execution", "image", req.ImageRef)

	onChange := req.OnChange
	if onChange == nil {
		onChange = func(e task.Execution) {}
	}

	go func() {
		execution := task.Execution{
			State: task.ExecutionStateProcessingRequest,
		}

		onChange(execution)

		// Set timeout if specified
		if req.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, req.Timeout)
			defer cancel()
		}

		execution.State = task.ExecutionStatePullingImage
		onChange(execution)

		// Pull image if needed
		if err := e.pullImage(ctx, req.ImageRef); err != nil {
			execution.State = task.ExecutionStateFailed
			execution.Error = &task.ExecutionError{
				Type:    task.ErrorTypeImagePullFailed,
				Message: fmt.Sprintf("failed to pull image %s", req.ImageRef),
				Cause:   errors.WithStack(err),
			}
			onChange(execution)
			return
		}

		execution.State = task.ExecutionStateImagePulled
		onChange(execution)

		execution.State = task.ExecutionStateCreatingContainer
		onChange(execution)

		// Create container
		containerID, err := e.createContainer(ctx, req)
		if err != nil {
			execution.State = task.ExecutionStateFailed
			execution.Error = &task.ExecutionError{
				Type:    task.ErrorTypeDockerDaemonError,
				Message: "failed to create container",
				Cause:   errors.WithStack(err),
			}
			onChange(execution)
			return
		}

		execution.ContainerID = containerID
		execution.State = task.ExecutionStateContainerCreated
		onChange(execution)

		// Ensure cleanup
		defer func() {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if err := e.remove(cleanupCtx, containerID); err != nil {
				e.logger.Warn("failed to cleanup container", "container_id", containerID, "error", err)
			}
		}()

		// Upload files
		if len(req.Inputs) > 0 {
			execution.State = task.ExecutionStateUploadingFiles
			onChange(execution)

			if err := e.uploadFiles(ctx, containerID, req.InputsDir, req.Inputs); err != nil {
				execution.State = task.ExecutionStateFailed
				execution.Error = &task.ExecutionError{
					Type:        task.ErrorTypeFileUploadFailed,
					Message:     "failed to upload files to container",
					ContainerID: containerID,
					Cause:       errors.WithStack(err),
				}
				onChange(execution)
				return
			}

			execution.State = task.ExecutionStateFilesUploaded
			onChange(execution)
		}

		execution.StartedAt = time.Now()
		execution.State = task.ExecutionStateStartingContainer
		onChange(execution)

		// Start container
		if err := e.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
			execution.State = task.ExecutionStateFailed
			execution.Error = &task.ExecutionError{
				Type:        task.ErrorTypeDockerDaemonError,
				Message:     "failed to start container",
				ContainerID: containerID,
				Cause:       errors.WithStack(err),
			}
			onChange(execution)
			return
		}

		execution.State = task.ExecutionStateContainerStarted
		onChange(execution)

		// Wait for container to finish
		statusCh, errCh := e.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				execution.State = task.ExecutionStateFailed
				execution.Error = &task.ExecutionError{
					Type:        task.ErrorTypeDockerDaemonError,
					Message:     "error waiting for container",
					ContainerID: containerID,
					Cause:       errors.WithStack(err),
				}
				onChange(execution)
				return
			}
		case status := <-statusCh:
			execution.ExitCode = int(status.StatusCode)
			execution.FinishedAt = time.Now()
			execution.State = task.ExecutionStateContainerFinished
			onChange(execution)
		}

		execution.State = task.ExecutionStateDownloadingFiles
		onChange(execution)

		// Download output files
		outputFiles, close, err := e.downloadFiles(ctx, containerID, req.OutputsDir)
		if err != nil {
			execution.State = task.ExecutionStateFailed
			execution.Error = &task.ExecutionError{
				Type:        task.ErrorTypeFileDownloadFailed,
				Message:     "error downloading files",
				ContainerID: containerID,
				Cause:       errors.WithStack(err),
			}
			onChange(execution)
			return
		} else {
			execution.Outputs = outputFiles
			defer close()
		}

		execution.State = task.ExecutionStateFilesDownloaded
		onChange(execution)

		e.logger.Info("container execution completed",
			"container_id", containerID,
			"exit_code", execution.ExitCode,
			"duration", execution.FinishedAt.Sub(execution.StartedAt))

		execution.State = task.ExecutionStateSucceeded
		onChange(execution)
	}()

	return nil
}

// pullImage pulls the image
func (e *DockerExecutor) pullImage(ctx context.Context, imageRef string) error {
	e.logger.Info("pulling image", "image", imageRef)

	reader, err := e.client.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		spew.Dump(err)
		return errors.Wrapf(err, "failed to pull image %s", imageRef)
	}
	defer reader.Close()

	output, err := io.ReadAll(reader)
	if err != nil {
		return errors.Wrap(err, "failed to read image pull output")
	}

	spew.Dump(string(output))

	e.logger.Info("image pulled successfully", "image", imageRef)
	return nil
}

// createContainer creates a new container with the specified configuration
func (e *DockerExecutor) createContainer(ctx context.Context, req task.ExecutionRequest) (string, error) {
	// Build environment variables
	env := make([]string, 0, len(req.Environment))
	for key, value := range req.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Build container configuration
	config := &container.Config{
		Image:        req.ImageRef,
		Env:          env,
		AttachStdout: true,
		AttachStderr: true,
	}

	containerMounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: "oplet-inputs",
			Target: req.InputsDir,
		},
		{
			Type:   mount.TypeVolume,
			Source: "oplet-outputs",
			Target: req.OutputsDir,
		},
	}

	// Build host configuration
	hostConfig := &container.HostConfig{
		NetworkMode:    container.NetworkMode("bridge"),
		ReadonlyRootfs: false,
		AutoRemove:     false,
		Mounts:         containerMounts,
	}

	// Create container
	resp, err := e.client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", errors.Wrap(err, "failed to create container")
	}

	e.logger.Debug("container created", "container_id", resp.ID, "image", req.ImageRef)

	return resp.ID, nil
}

// uploadFiles uploads files to the container using TAR streams
func (e *DockerExecutor) uploadFiles(ctx context.Context, containerID string, inputsDir string, files map[string]io.ReadCloser) error {
	if len(files) == 0 {
		return nil
	}

	e.logger.Debug("uploading files to container", "container_id", containerID, "file_count", len(files))

	// Create TAR archive
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for filename, file := range files {
		// Read file content
		content, err := io.ReadAll(file)
		if err != nil {
			return errors.Wrapf(err, "failed to read file %s", filename)
		}

		// Create TAR header
		header := &tar.Header{
			Name: filename,
			Mode: 0644,
			Size: int64(len(content)),
		}

		// Write header and content
		if err := tw.WriteHeader(header); err != nil {
			return errors.Wrapf(err, "failed to write TAR header for %s", filename)
		}
		if _, err := tw.Write(content); err != nil {
			return errors.Wrapf(err, "failed to write TAR content for %s", filename)
		}
	}

	if err := tw.Close(); err != nil {
		return errors.Wrap(err, "failed to close TAR writer")
	}

	// Upload TAR archive to container
	err := e.client.CopyToContainer(ctx, containerID, inputsDir, &buf, container.CopyToContainerOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to copy files to container %s", containerID)
	}

	e.logger.Debug("files uploaded successfully", "container_id", containerID)
	return nil
}

// downloadFiles downloads files from the container's output directory
func (e *DockerExecutor) downloadFiles(ctx context.Context, containerID string, outputsDir string) (*tar.Reader, func(), error) {

	// Check if output directory exists
	reader, _, err := e.client.CopyFromContainer(ctx, containerID, outputsDir)
	if err != nil {
		// Output directory doesn't exist or is empty - this is not an error
		e.logger.Debug("no output files found", "container_id", containerID, "output_path", outputsDir)
		return nil, func() {}, nil
	}

	close := func() {
		if err := reader.Close(); err != nil {
			e.logger.ErrorContext(ctx, "could not close downloaded files", slogx.Error(errors.WithStack(err)))
		}
	}

	return tar.NewReader(reader), close, nil
}

// GetLogs implements task.Executor.GetLogs
func (e *DockerExecutor) GetLogs(ctx context.Context, containerID string) (io.ReadCloser, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     true,
	}

	reader, err := e.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, &task.ExecutionError{
			Type:        task.ErrorTypeDockerDaemonError,
			Message:     "failed to get container logs",
			ContainerID: containerID,
			Cause:       errors.WithStack(err),
		}
	}

	return reader, nil
}

// remove implements task.Executor.Remove
func (e *DockerExecutor) remove(ctx context.Context, containerID string) error {
	options := container.RemoveOptions{
		Force:         true, // Force removal even if running
		RemoveVolumes: true,
	}

	err := e.client.ContainerRemove(ctx, containerID, options)
	if err != nil {
		// Don't return an error if container doesn't exist
		if strings.Contains(err.Error(), "No such container") {
			return nil
		}
		spew.Dump(err)
		return &task.ExecutionError{
			Type:        task.ErrorTypeDockerDaemonError,
			Message:     "failed to remove container",
			ContainerID: containerID,
			Cause:       errors.WithStack(err),
		}
	}

	e.logger.Debug("container removed", "container_id", containerID)
	return nil
}

// Ensure DockerExecutor implements task.Executor interface
var _ task.Executor = &DockerExecutor{}
