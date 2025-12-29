package task

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	httpCtx "github.com/bornholm/oplet/internal/http/context"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	commonComp "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
	taskForm "github.com/bornholm/oplet/internal/http/handler/webui/common/task"
	"github.com/bornholm/oplet/internal/http/handler/webui/task/component"
	"github.com/bornholm/oplet/internal/http/url"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/store/repository/execution"
	taskRepository "github.com/bornholm/oplet/internal/store/repository/task"
	"github.com/bornholm/oplet/internal/task"
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
	taskForm := taskForm.NewInputForm(vmodel.Task)

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
	taskForm := taskForm.NewInputForm(taskDef)

	if err := taskForm.Handle(r); err != nil {
		common.HandleError(w, r, err)
		return
	}

	// Validate form
	if !taskForm.IsValid(ctx) {
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
		InputParameters: h.marshalInputParameters(taskDef, taskForm.Values, taskForm.Files),
	}

	if err := executionRepo.Create(ctx, taskExecution); err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	h.logger.InfoContext(ctx, "created task execution",
		"execution_id", taskExecution.ID,
		"task_id", storeTask.ID,
		"user_id", user.ID)

	// Store input files for runner to download later
	if err := h.storeInputFiles(ctx, taskForm.Files, taskDef, taskExecution.ID); err != nil {
		// Mark execution as failed
		executionRepo.SetCompleted(ctx, taskExecution.ID, -1, err.Error())
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	progressURL := commonComp.BaseURL(ctx, url.WithPath("/tasks", commonComp.FormatID(taskExecution.TaskID), "executions", commonComp.FormatID(taskExecution.ID)))

	// Redirect to progress page
	http.Redirect(w, r, string(progressURL), http.StatusSeeOther)
}

func (h *Handler) marshalInputParameters(taskDef *task.Definition, values map[string]string, files map[string][]*multipart.FileHeader) string {
	params := make(map[string]interface{})

	// Create a map of input types for quick lookup
	inputTypes := make(map[string]task.Type)
	for _, input := range taskDef.Inputs {
		inputTypes[input.Name] = input.Type
	}

	// Add form values with proper type conversion
	for key, value := range values {
		inputType, exists := inputTypes[key]
		if exists && inputType == task.TypeBoolean {
			// Handle boolean conversion like in createExecutionRequest
			if value == "on" {
				params[key] = true
			} else {
				params[key] = false
			}
		} else {
			params[key] = value
		}
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

// storeInputFiles stores uploaded input files for later download by runners
func (h *Handler) storeInputFiles(ctx context.Context, files map[string][]*multipart.FileHeader, taskDef *task.Definition, executionID uint) error {
	executionRepo := execution.NewRepository(h.store)

	// Process file inputs
	for _, input := range taskDef.Inputs {
		if input.Type == task.TypeFile {
			if fileHeaders, exists := files[input.Name]; exists && len(fileHeaders) > 0 {
				fileHeader := fileHeaders[0]

				// Open uploaded file
				uploadedFile, err := fileHeader.Open()
				if err != nil {
					return errors.Wrapf(err, "failed to open uploaded file %s", input.Name)
				}
				defer uploadedFile.Close()

				// Use parameter name as filename (not original filename)
				// This ensures the file will be positioned correctly in /oplet/inputs
				parameterName := input.Name

				// Store input file using parameter name
				storedFile, err := h.fileStorage.StoreInputFile(executionID, parameterName, uploadedFile)
				if err != nil {
					return errors.Wrapf(err, "failed to store input file %s", input.Name)
				}

				// Record in database with parameter name as filename
				dbFile := &store.TaskExecutionFile{
					ExecutionID: executionID,
					Filename:    parameterName, // Use parameter name, not original filename
					FilePath:    storedFile.StoredPath,
					FileSize:    storedFile.Size,
					MimeType:    storedFile.MimeType,
					IsOutput:    false,
				}

				if err := executionRepo.AddFile(ctx, executionID, dbFile); err != nil {
					h.logger.WarnContext(ctx, "failed to record input file in database",
						"execution_id", executionID, "parameter_name", parameterName, "original_filename", fileHeader.Filename, "error", err)
				}

				h.logger.InfoContext(ctx, "stored input file for runner download",
					"execution_id", executionID,
					"parameter_name", parameterName,
					"original_filename", fileHeader.Filename,
					"size", storedFile.Size)

			} else if input.Required {
				return fmt.Errorf("required file %s not provided", input.Name)
			}
		}
	}

	return nil
}
