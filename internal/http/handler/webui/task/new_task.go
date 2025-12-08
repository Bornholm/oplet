package task

import (
	"archive/tar"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	httpCtx "github.com/bornholm/oplet/internal/http/context"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	commonComp "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
	"github.com/bornholm/oplet/internal/http/handler/webui/task/component"
	"github.com/bornholm/oplet/internal/http/url"
	"github.com/bornholm/oplet/internal/slogx"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/store/repository/execution"
	taskRepository "github.com/bornholm/oplet/internal/store/repository/task"
	"github.com/bornholm/oplet/internal/task"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/errors"
)

func (h *Handler) getNewTaskPage(w http.ResponseWriter, r *http.Request) {
	// Get task ID from URL path
	rawTaskID := r.PathValue("taskID")
	if rawTaskID == "" {
		common.HandleError(w, r, errors.New("task ID is required"))
		return
	}

	taskID, err := strconv.ParseUint(rawTaskID, 10, 64)
	if err != nil {
		common.HandleError(w, r, err)
		return
	}

	taskRepository := taskRepository.NewRepository(h.store)

	ctx := r.Context()

	task, err := taskRepository.GetByID(ctx, uint(taskID))
	if err != nil {
		common.HandleError(w, r, err)
		return
	}

	// Get task definition from provider
	taskDef, err := h.taskProvider.FetchTaskDefinition(ctx, task.ImageRef)
	if err != nil {
		common.HandleError(w, r, errors.Wrapf(err, "failed to fetch task %d definition", taskID))
		return
	}

	// Handle form submission
	if r.Method == "POST" {
		h.handleNewTaskSubmission(w, r, task, taskDef)
		return
	}

	// Fill view model with task and form
	vmodel, err := h.fillNewTaskPageViewModel(r, uint(taskID), taskDef)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	page := component.NewTaskPage(*vmodel)
	templ.Handler(page).ServeHTTP(w, r)
}

func (h *Handler) fillNewTaskPageViewModel(r *http.Request, taskID uint, taskDef *task.Definition) (*component.NewTaskPageVModel, error) {
	vmodel := &component.NewTaskPageVModel{
		Task:   taskDef,
		TaskID: taskID,
	}

	ctx := r.Context()

	err := common.FillViewModel(
		ctx,
		vmodel, r,
		h.fillNewTaskPageNavbarVModel,
		h.fillNewTaskPageFormVModel,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillNewTaskPageNavbarVModel(ctx context.Context, vmodel *component.NewTaskPageVModel, r *http.Request) error {
	if err := commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (h *Handler) fillNewTaskPageFormVModel(ctx context.Context, vmodel *component.NewTaskPageVModel, r *http.Request) error {
	if vmodel.Task == nil {
		return errors.New("task definition is required")
	}

	// Create form for validation
	taskForm := NewTaskForm(vmodel.Task)

	vmodel.Form = taskForm

	return nil
}

func (h *Handler) handleNewTaskSubmission(w http.ResponseWriter, r *http.Request, storeTask *store.Task, taskDef *task.Definition) {
	ctx := r.Context()
	user := httpCtx.User(ctx)
	if user == nil {
		h.getForbiddenPage(w, r)
		return
	}

	// Create form for validation
	taskForm := NewTaskForm(taskDef)

	if err := taskForm.Handle(r); err != nil {
		common.HandleError(w, r, err)
		return
	}

	// Validate form
	if !taskForm.IsValid() {
		// Re-render form with errors
		vmodel, err := h.fillNewTaskPageViewModel(r, storeTask.ID, taskDef)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		// Update form with validation errors
		vmodel.Form = taskForm

		page := component.NewTaskPage(*vmodel)
		templ.Handler(page).ServeHTTP(w, r)
		return
	}

	// Create execution record
	executionRepo := execution.NewRepository(h.store)
	taskExecution := &store.TaskExecution{
		TaskID:          storeTask.ID,
		UserID:          user.ID,
		Status:          store.StatusPending,
		InputParameters: h.marshalInputParameters(taskForm.Values, taskForm.Files),
	}

	if err := executionRepo.Create(ctx, taskExecution); err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	h.logger.InfoContext(ctx, "created task execution",
		"execution_id", taskExecution.ID,
		"task_id", storeTask.ID,
		"user_id", user.ID)

	// Store input files and create execution request
	req, err := h.createExecutionRequest(ctx, storeTask.ImageRef, taskForm.Values, taskForm.Files, taskDef, taskExecution.ID)
	if err != nil {
		// Mark execution as failed
		executionRepo.SetCompleted(ctx, taskExecution.ID, -1, err.Error())
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	// Set up execution callback
	req.OnChange = h.createExecutionCallback(taskExecution.ID)

	// Start execution asynchronously
	go func() {
		execCtx := context.Background()
		h.logger.InfoContext(execCtx, "starting task execution", "execution_id", taskExecution.ID)

		if err := h.taskExecutor.Execute(execCtx, req); err != nil {
			h.logExecutionError(execCtx, taskExecution.ID, err)
		}
	}()

	progressURL := commonComp.BaseURL(ctx, url.WithPath("/tasks", commonComp.FormatID(taskExecution.TaskID), "executions", commonComp.FormatID(taskExecution.ID)))

	// Redirect to progress page
	http.Redirect(w, r, string(progressURL), http.StatusSeeOther)
}

func (h *Handler) marshalInputParameters(values map[string]string, files map[string][]*multipart.FileHeader) string {
	params := make(map[string]interface{})

	// Add form values
	for key, value := range values {
		params[key] = value
	}

	// Add file information
	fileInfo := make(map[string][]string)
	for key, fileHeaders := range files {
		filenames := make([]string, len(fileHeaders))
		for i, header := range fileHeaders {
			filenames[i] = header.Filename
		}
		fileInfo[key] = filenames
	}
	if len(fileInfo) > 0 {
		params["_files"] = fileInfo
	}

	// Marshal to JSON
	data, err := json.Marshal(params)
	if err != nil {
		h.logger.Warn("failed to marshal input parameters", "error", err)
		return "{}"
	}

	return string(data)
}

func (h *Handler) createExecutionRequest(ctx context.Context, imageRef string, values map[string]string, files map[string][]*multipart.FileHeader, taskDef *task.Definition, executionID uint) (task.ExecutionRequest, error) {
	req := task.ExecutionRequest{
		ImageRef:    imageRef,
		Environment: make(map[string]string),
		Inputs:      make(map[string]io.ReadCloser),
		InputsDir:   "/oplet/inputs",
		OutputsDir:  "/oplet/outputs",
	}

	executionRepo := execution.NewRepository(h.store)

	// Process environment variables
	for _, input := range taskDef.Inputs {
		if input.InputType == task.InputTypeEnv {
			if value, exists := values[input.Name]; exists {
				req.Environment[input.Name] = value
			} else if input.Required {
				return req, fmt.Errorf("required environment variable %s not provided", input.Name)
			}
		}
	}

	// Process configuration inputs
	for _, config := range taskDef.Configuration {
		if config.InputType == task.InputTypeEnv {
			if value, exists := values[config.Name]; exists {
				req.Environment[config.Name] = value
			}
		}
	}

	// Process file inputs
	for _, input := range taskDef.Inputs {
		if input.InputType == task.InputTypeFile {
			if fileHeaders, exists := files[input.Name]; exists && len(fileHeaders) > 0 {
				fileHeader := fileHeaders[0]

				// Open uploaded file
				uploadedFile, err := fileHeader.Open()
				if err != nil {
					return req, errors.Wrapf(err, "failed to open uploaded file %s", input.Name)
				}
				defer uploadedFile.Close()

				filename := filepath.Base(fileHeader.Filename)

				// Store input file
				storedFile, err := h.fileStorage.StoreInputFile(executionID, filename, uploadedFile)
				if err != nil {
					return req, errors.Wrapf(err, "failed to store input file %s", input.Name)
				}

				// Record in database
				dbFile := &store.TaskExecutionFile{
					ExecutionID: executionID,
					Filename:    storedFile.OriginalName,
					FilePath:    storedFile.StoredPath,
					FileSize:    storedFile.Size,
					MimeType:    storedFile.MimeType,
					IsOutput:    false,
				}

				if err := executionRepo.AddFile(ctx, executionID, dbFile); err != nil {
					h.logger.WarnContext(ctx, "failed to record input file in database",
						"execution_id", executionID, "filename", fileHeader.Filename, "error", err)
				}

				// Open stored file for container input
				storedReader, err := h.fileStorage.GetFile(storedFile.StoredPath)
				if err != nil {
					return req, errors.Wrapf(err, "failed to read stored file %s", storedFile.StoredPath)
				}

				req.Inputs[input.Name] = storedReader

				h.logger.InfoContext(ctx, "stored input file",
					"execution_id", executionID,
					"filename", fileHeader.Filename,
					"size", storedFile.Size)

			} else if input.Required {
				return req, fmt.Errorf("required file %s not provided", input.Name)
			}
		}
	}

	return req, nil
}

func (h *Handler) createExecutionCallback(executionID uint) func(task.Execution) {
	return func(e task.Execution) {
		ctx := context.Background()
		executionRepo := execution.NewRepository(h.store)

		// Update execution status and container ID
		status := mapExecutionStateToStatus(e.State)

		// Update basic execution info
		if err := h.updateExecutionState(ctx, executionID, status, e.ContainerID, e.StartedAt, e.FinishedAt); err != nil {
			h.logger.ErrorContext(ctx, "failed to update execution state",
				"execution_id", executionID, "error", err)
			return
		}

		// Log state change
		logEntry := &store.TaskExecutionLog{
			ExecutionID: executionID,
			Timestamp:   time.Now().UnixMicro(),
			Source:      "system",
			Message:     fmt.Sprintf("Status changed to: %s", status),
		}

		if err := executionRepo.AddLog(ctx, executionID, logEntry); err != nil {
			h.logger.ErrorContext(ctx, "failed to add execution log",
				"execution_id", executionID, "error", err)
		}

		// Handle specific states
		switch e.State {
		case task.ExecutionStateContainerStarted:
			h.startLogStreaming(ctx, executionID, e.ContainerID)

		case task.ExecutionStateContainerFinished:
			h.handleExecutionFinished(ctx, executionID, e)

		case task.ExecutionStateFilesDownloaded:
			if e.Outputs != nil {
				h.persistOutputFiles(ctx, executionID, e.Outputs)
			}

		case task.ExecutionStateFailed:
			h.handleExecutionFailed(ctx, executionID, e.Error)

		case task.ExecutionStateSucceeded:
			h.handleExecutionSucceeded(ctx, executionID)
		}

		h.logger.DebugContext(ctx, "execution state updated",
			"execution_id", executionID,
			"state", e.State,
			"status", status)
	}
}

func (h *Handler) logExecutionError(ctx context.Context, executionID uint, err error) {
	h.logger.ErrorContext(ctx, "task execution error",
		"execution_id", executionID, "error", err)

	executionRepo := execution.NewRepository(h.store)
	logEntry := &store.TaskExecutionLog{
		ExecutionID: executionID,
		Timestamp:   time.Now().UnixMicro(),
		Source:      "system",
		Message:     fmt.Sprintf("Execution error: %s", err.Error()),
	}

	if logErr := executionRepo.AddLog(ctx, executionID, logEntry); logErr != nil {
		h.logger.ErrorContext(ctx, "failed to log execution error",
			"execution_id", executionID, "log_error", logErr)
	}
}

func (h *Handler) updateExecutionState(ctx context.Context, executionID uint, status store.TaskExecutionStatus, containerID string, startedAt, finishedAt time.Time) error {
	executionRepo := execution.NewRepository(h.store)

	// Get current execution
	execution, err := executionRepo.GetByID(ctx, executionID)
	if err != nil {
		return errors.WithStack(err)
	}

	// Update fields
	execution.Status = status
	if containerID != "" {
		execution.ContainerID = containerID
	}
	if !startedAt.IsZero() && execution.StartedAt == nil {
		execution.StartedAt = &startedAt
	}
	if !finishedAt.IsZero() && execution.FinishedAt == nil {
		execution.FinishedAt = &finishedAt
	}

	return executionRepo.Update(ctx, execution)
}

func (h *Handler) startLogStreaming(ctx context.Context, executionID uint, containerID string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.ErrorContext(ctx, "panic in log streaming", "panic", r, "execution_id", executionID)
			}
		}()

		// Get container logs
		containerLogs, err := h.taskExecutor.GetLogs(ctx, containerID)
		if err != nil {
			h.logger.ErrorContext(ctx, "could not get task logs",
				"error", err, "execution_id", executionID)
			return
		}
		defer containerLogs.Close()

		// Stream and persist logs
		if err := h.streamAndPersistLogs(ctx, executionID, containerLogs); err != nil {
			h.logger.ErrorContext(ctx, "could not stream task logs",
				"error", err, "execution_id", executionID)
		}
	}()
}

func (h *Handler) streamAndPersistLogs(ctx context.Context, executionID uint, logReader io.ReadCloser) error {
	executionRepo := execution.NewRepository(h.store)

	in, out := io.Pipe()
	defer in.Close()
	defer out.Close()

	go func() {
		if _, err := stdcopy.StdCopy(out, out, logReader); err != nil {
			h.logger.ErrorContext(ctx, "could not parse docker logs", slogx.Error(errors.WithStack(err)))
		}
	}()

	scanner := bufio.NewScanner(in)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "T") && strings.Contains(line, "Z") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				line = parts[1]
			}
		}

		logEntry := &store.TaskExecutionLog{
			ExecutionID: executionID,
			Timestamp:   time.Now().UnixMicro(),
			Source:      "container",
			Message:     line,
		}

		if err := executionRepo.AddLog(ctx, executionID, logEntry); err != nil {
			h.logger.ErrorContext(ctx, "failed to persist log entry",
				"error", err, "execution_id", executionID)
			// Continue processing other logs
		}
	}

	return scanner.Err()
}

func (h *Handler) handleExecutionSucceeded(ctx context.Context, executionID uint) {
	executionRepo := execution.NewRepository(h.store)

	logEntry := &store.TaskExecutionLog{
		ExecutionID: executionID,
		Timestamp:   time.Now().UnixMicro(),
		Source:      "system",
		Message:     "Task execution completed successfully",
	}

	if err := executionRepo.AddLog(ctx, executionID, logEntry); err != nil {
		h.logger.ErrorContext(ctx, "failed to log execution success",
			"execution_id", executionID, "error", err)
	}

	h.logger.InfoContext(ctx, "task execution succeeded", "execution_id", executionID)
}

func (h *Handler) handleExecutionFinished(ctx context.Context, executionID uint, e task.Execution) {
	executionRepo := execution.NewRepository(h.store)

	// Update execution with completion details
	if err := executionRepo.SetCompleted(ctx, executionID, e.ExitCode, ""); err != nil {
		h.logger.ErrorContext(ctx, "failed to mark execution as completed",
			"execution_id", executionID, "error", err)
	}

	// Log completion
	logEntry := &store.TaskExecutionLog{
		ExecutionID: executionID,
		Timestamp:   time.Now().UnixMicro(),
		Source:      "system",
		Message:     fmt.Sprintf("Container finished with exit code: %d", e.ExitCode),
	}

	if err := executionRepo.AddLog(ctx, executionID, logEntry); err != nil {
		h.logger.ErrorContext(ctx, "failed to log execution completion",
			"execution_id", executionID, "error", err)
	}
}

func (h *Handler) handleExecutionFailed(ctx context.Context, executionID uint, execError error) {
	executionRepo := execution.NewRepository(h.store)

	errorMsg := ""
	if execError != nil {
		errorMsg = execError.Error()
	}

	// Update execution with error
	if err := executionRepo.SetCompleted(ctx, executionID, -1, errorMsg); err != nil {
		h.logger.ErrorContext(ctx, "failed to mark execution as failed",
			"execution_id", executionID, "error", err)
	}

	// Log error
	logEntry := &store.TaskExecutionLog{
		ExecutionID: executionID,
		Timestamp:   time.Now().UnixMicro(),
		Source:      "system",
		Message:     fmt.Sprintf("Execution failed: %s", errorMsg),
	}

	if err := executionRepo.AddLog(ctx, executionID, logEntry); err != nil {
		h.logger.ErrorContext(ctx, "failed to log execution failure",
			"execution_id", executionID, "error", err)
	}

	h.logger.ErrorContext(ctx, "task execution failed",
		"execution_id", executionID, "error", execError)
}

func (h *Handler) persistOutputFiles(ctx context.Context, executionID uint, outputs *tar.Reader) {
	if outputs == nil {
		return
	}

	executionRepo := execution.NewRepository(h.store)

	for {
		header, err := outputs.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			h.logExecutionError(ctx, executionID, errors.Wrap(err, "failed to read output tar"))
			return
		}

		// Skip directories
		if header.Typeflag == tar.TypeDir {
			continue
		}

		filename := filepath.Base(header.Name)

		// Store file
		storedFile, err := h.fileStorage.StoreOutputFile(executionID, filename, outputs)
		if err != nil {
			h.logExecutionError(ctx, executionID, errors.Wrapf(err, "failed to store output file %s", header.Name))
			continue
		}

		// Record in database
		dbFile := &store.TaskExecutionFile{
			ExecutionID: executionID,
			Filename:    storedFile.OriginalName,
			FilePath:    storedFile.StoredPath,
			FileSize:    storedFile.Size,
			MimeType:    storedFile.MimeType,
			IsOutput:    true,
		}

		if err := executionRepo.AddFile(ctx, executionID, dbFile); err != nil {
			h.logExecutionError(ctx, executionID, errors.Wrapf(err, "failed to record output file %s", header.Name))
			// Don't delete the file, just log the error
		}

		h.logger.InfoContext(ctx, "stored output file",
			"execution_id", executionID,
			"filename", header.Name,
			"size", storedFile.Size)
	}
}

func mapExecutionStateToStatus(state task.ExecutionState) store.TaskExecutionStatus {
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
	default:
		return store.StatusPending
	}
}
