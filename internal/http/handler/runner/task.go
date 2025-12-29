package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/store/repository/execution"
	"github.com/bornholm/oplet/internal/task"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func (h *Handler) handleTaskRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	runner, err := contextRunner(ctx)
	if err != nil {
		handleInternalError(h, w, r, err, "could not retrieve runner from context")
		return
	}

	taskExecutionRepo := execution.NewRepository(h.store)

	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()

	pollInterval := 3 * time.Second

	for {
		select {
		case <-timer.C:
			w.WriteHeader(http.StatusNoContent)
			return

		default:
			nextExecution, err := taskExecutionRepo.NextTask(ctx)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				handleInternalError(h, w, r, err, "could not retrieve next task")
				return
			}

			if nextExecution == nil {
				time.Sleep(pollInterval)
				continue
			}

			// Parse input parameters and build environment
			environment := make(map[string]string)

			// Get task definition to understand input types
			taskDef, err := h.taskProvider.FetchTaskDefinition(ctx, nextExecution.Task.ImageRef)
			if err != nil {
				h.logger.WarnContext(ctx, "could not fetch task definition",
					"execution_id", nextExecution.ID, "error", err)
			} else {
				// Process input parameters from form submission
				if nextExecution.InputParameters != "" {
					var params map[string]interface{}
					if err := json.Unmarshal([]byte(nextExecution.InputParameters), &params); err != nil {
						h.logger.WarnContext(ctx, "could not parse input parameters",
							"execution_id", nextExecution.ID, "error", err)
					} else {
						// Add form values to environment (non-file inputs)
						for _, input := range taskDef.Inputs {
							if input.Type != task.TypeFile {
								if value, exists := params[input.Name]; exists {
									if input.Type == task.TypeBoolean {
										// Handle boolean conversion
										if boolVal, ok := value.(bool); ok {
											if boolVal {
												environment[input.Name] = "true"
											} else {
												environment[input.Name] = "false"
											}
										}
									} else {
										environment[input.Name] = fmt.Sprintf("%v", value)
									}
								}
							}
						}
					}
				}

				// Process configuration parameters
				for _, config := range nextExecution.Task.Configurations {
					var configInput *task.Input
					for _, ci := range taskDef.Configuration {
						if ci.Name == config.Name {
							configInput = ci
							break
						}
					}
					if configInput == nil {
						continue
					}

					value := config.Value

					if configInput.Type == task.TypeBoolean {
						if value == "on" {
							value = "true"
						} else {
							value = "false"
						}
					}

					environment[config.Name] = value
				}
			}

			response := TaskRequestResponse{
				ExecutionID:     nextExecution.ID,
				TaskID:          nextExecution.TaskID,
				ImageRef:        nextExecution.Task.ImageRef,
				Environment:     environment,
				InputParameters: nextExecution.InputParameters,
				RunnerToken:     nextExecution.RunnerToken,
				InputsDir:       "/oplet/inputs",
				OutputsDir:      "/oplet/outputs",
				CreatedAt:       nextExecution.CreatedAt,
			}

			writeJSONResponse(w, http.StatusOK, response)

			h.logger.InfoContext(ctx, "task assigned to runner",
				"runner_id", runner.ID,
				"execution_id", nextExecution.ID,
				"task_id", nextExecution.TaskID)
			return
		}
	}
}

func (h *Handler) handleTaskTrace(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	runner, err := contextRunner(ctx)
	if err != nil {
		handleInternalError(h, w, r, err, "could not retrieve runner from context")
		return
	}

	taskID, err := getTaskIDFromPath(r)
	if err != nil {
		handleValidationError(w, err)
		return
	}

	var req TaskTraceRequest
	if err := parseJSONRequest(r, &req); err != nil {
		handleValidationError(w, err)
		return
	}

	if err := req.Validate(); err != nil {
		handleValidationError(w, err)
		return
	}

	// Get execution by task ID and runner token
	executionRepo := execution.NewRepository(h.store)
	taskIDUint, err := strconv.ParseUint(taskID, 10, 32)
	if err != nil {
		handleValidationError(w, ErrInvalidRequest("invalid task ID"))
		return
	}

	// Find execution by task ID (simplified for now)
	executions, err := executionRepo.GetByTaskID(ctx, uint(taskIDUint), 1, 0)
	if err != nil {
		handleInternalError(h, w, r, err, "could not retrieve execution")
		return
	}

	if len(executions) == 0 {
		handleNotFoundError(w, "execution")
		return
	}

	exec := executions[0]

	// Add logs to execution
	logsAdded := 0
	for _, logEntry := range req.Logs {
		dbLog := &store.TaskExecutionLog{
			ExecutionID: exec.ID,
			Timestamp:   logEntry.Timestamp,
			Source:      logEntry.Source,
			Message:     logEntry.Message,
		}

		if err := executionRepo.AddLog(ctx, exec.ID, dbLog); err != nil {
			h.logger.WarnContext(ctx, "could not add log entry",
				"execution_id", exec.ID, "error", err)
			continue
		}
		logsAdded++
	}

	response := TaskTraceResponse{
		ExecutionID: exec.ID,
		LogsAdded:   logsAdded,
	}

	writeJSONResponse(w, http.StatusOK, response)

	h.logger.DebugContext(ctx, "logs added to execution",
		"runner_id", runner.ID,
		"execution_id", exec.ID,
		"logs_added", logsAdded)
}

func (h *Handler) handleTaskInputs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	runner, err := contextRunner(ctx)
	if err != nil {
		handleInternalError(h, w, r, err, "could not retrieve runner from context")
		return
	}

	taskID, err := getTaskIDFromPath(r)
	if err != nil {
		handleValidationError(w, err)
		return
	}

	taskIDUint, err := strconv.ParseUint(taskID, 10, 32)
	if err != nil {
		handleValidationError(w, ErrInvalidRequest("invalid task ID"))
		return
	}

	// Get execution
	executionRepo := execution.NewRepository(h.store)
	executions, err := executionRepo.GetByTaskID(ctx, uint(taskIDUint), 1, 0)
	if err != nil {
		handleInternalError(h, w, r, err, "could not retrieve execution")
		return
	}

	if len(executions) == 0 {
		handleNotFoundError(w, "execution")
		return
	}

	exec := executions[0]

	// Get input files for this execution
	inputFiles, err := executionRepo.GetFiles(ctx, exec.ID, false) // false = input files
	if err != nil {
		handleInternalError(h, w, r, err, "could not retrieve input files")
		return
	}

	// Check if specific file is requested
	filename := r.URL.Query().Get("file")
	if filename != "" {
		// Download specific file
		h.downloadInputFile(w, r, exec.ID, filename, inputFiles)
		return
	}

	// Return list of available input files
	fileList := make([]map[string]interface{}, len(inputFiles))
	for i, file := range inputFiles {
		fileList[i] = map[string]interface{}{
			"filename":  file.Filename,
			"file_size": file.FileSize,
			"mime_type": file.MimeType,
		}
	}

	response := map[string]interface{}{
		"execution_id": exec.ID,
		"files":        fileList,
	}

	writeJSONResponse(w, http.StatusOK, response)

	h.logger.DebugContext(ctx, "input files listed",
		"runner_id", runner.ID,
		"execution_id", exec.ID,
		"file_count", len(inputFiles))
}

func (h *Handler) downloadInputFile(w http.ResponseWriter, r *http.Request, executionID uint, filename string, inputFiles []*store.TaskExecutionFile) {
	ctx := r.Context()

	// Find the requested file
	var targetFile *store.TaskExecutionFile
	for _, file := range inputFiles {
		if file.Filename == filename {
			targetFile = file
			break
		}
	}

	if targetFile == nil {
		handleNotFoundError(w, "input file")
		return
	}

	// Open the file from storage
	fileReader, err := h.fileStorage.GetFile(targetFile.FilePath)
	if err != nil {
		handleInternalError(h, w, r, err, "could not open input file")
		return
	}
	defer fileReader.Close()

	// Set appropriate headers
	w.Header().Set("Content-Type", targetFile.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", targetFile.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", targetFile.FileSize))

	// Stream the file content
	if _, err := io.Copy(w, fileReader); err != nil {
		h.logger.ErrorContext(ctx, "failed to stream input file",
			"execution_id", executionID,
			"filename", filename,
			"error", err)
		return
	}

	h.logger.InfoContext(ctx, "input file downloaded",
		"execution_id", executionID,
		"filename", filename,
		"size", targetFile.FileSize)
}

func (h *Handler) handleTaskOutputs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	runner, err := contextRunner(ctx)
	if err != nil {
		handleInternalError(h, w, r, err, "could not retrieve runner from context")
		return
	}

	taskID, err := getTaskIDFromPath(r)
	if err != nil {
		handleValidationError(w, err)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		handleValidationError(w, ErrInvalidRequest("could not parse multipart form: %v", err))
		return
	}

	taskIDUint, err := strconv.ParseUint(taskID, 10, 32)
	if err != nil {
		handleValidationError(w, ErrInvalidRequest("invalid task ID"))
		return
	}

	// Get execution
	executionRepo := execution.NewRepository(h.store)
	executions, err := executionRepo.GetByTaskID(ctx, uint(taskIDUint), 1, 0)
	if err != nil {
		handleInternalError(h, w, r, err, "could not retrieve execution")
		return
	}

	if len(executions) == 0 {
		handleNotFoundError(w, "execution")
		return
	}

	exec := executions[0]
	filesStored := 0

	// Process uploaded output files
	for fieldName, fileHeaders := range r.MultipartForm.File {
		for _, fileHeader := range fileHeaders {
			if err := h.storeOutputFile(ctx, exec.ID, fieldName, fileHeader); err != nil {
				h.logger.WarnContext(ctx, "could not store output file",
					"execution_id", exec.ID, "filename", fileHeader.Filename, "error", err)
				continue
			}
			filesStored++
		}
	}

	response := TaskOutputsResponse{
		ExecutionID: exec.ID,
		FilesStored: filesStored,
		Message:     fmt.Sprintf("Stored %d output files", filesStored),
	}

	writeJSONResponse(w, http.StatusOK, response)

	h.logger.InfoContext(ctx, "output files stored",
		"runner_id", runner.ID,
		"execution_id", exec.ID,
		"files_stored", filesStored)
}

// Helper methods for file storage
func (h *Handler) storeInputFile(ctx context.Context, executionID uint, fieldName string, fileHeader *multipart.FileHeader) error {
	file, err := fileHeader.Open()
	if err != nil {
		return errors.Wrapf(err, "could not open uploaded file %s", fileHeader.Filename)
	}
	defer file.Close()

	filename := filepath.Base(fileHeader.Filename)
	storedFile, err := h.fileStorage.StoreInputFile(executionID, filename, file)
	if err != nil {
		return errors.Wrapf(err, "could not store input file %s", filename)
	}

	// Record in database
	executionRepo := execution.NewRepository(h.store)
	dbFile := &store.TaskExecutionFile{
		ExecutionID: executionID,
		Filename:    storedFile.OriginalName,
		FilePath:    storedFile.StoredPath,
		FileSize:    storedFile.Size,
		MimeType:    storedFile.MimeType,
		IsOutput:    false,
	}

	return executionRepo.AddFile(ctx, executionID, dbFile)
}

func (h *Handler) storeOutputFile(ctx context.Context, executionID uint, fieldName string, fileHeader *multipart.FileHeader) error {
	file, err := fileHeader.Open()
	if err != nil {
		return errors.Wrapf(err, "could not open uploaded file %s", fileHeader.Filename)
	}
	defer file.Close()

	filename := filepath.Base(fileHeader.Filename)
	storedFile, err := h.fileStorage.StoreOutputFile(executionID, filename, file)
	if err != nil {
		return errors.Wrapf(err, "could not store output file %s", filename)
	}

	// Record in database
	executionRepo := execution.NewRepository(h.store)
	dbFile := &store.TaskExecutionFile{
		ExecutionID: executionID,
		Filename:    storedFile.OriginalName,
		FilePath:    storedFile.StoredPath,
		FileSize:    storedFile.Size,
		MimeType:    storedFile.MimeType,
		IsOutput:    true,
	}

	return executionRepo.AddFile(ctx, executionID, dbFile)
}
