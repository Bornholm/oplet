package task

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/bornholm/oplet/internal/http/authz"
	httpCtx "github.com/bornholm/oplet/internal/http/context"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	commonComp "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
	"github.com/bornholm/oplet/internal/http/handler/webui/task/component"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/store/repository/execution"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func (h *Handler) getExecutionPage(w http.ResponseWriter, r *http.Request) {
	executionID := getExecutionIDFromPath(r)
	if executionID == 0 {
		common.HandleError(w, r, errors.New("invalid execution ID"))
		return
	}

	vmodel, err := h.fillExecutionPageViewModel(r, executionID)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	page := component.ExecutionPage(*vmodel)
	templ.Handler(page).ServeHTTP(w, r)
}

func (h *Handler) getExecutionLogs(w http.ResponseWriter, r *http.Request) {
	executionID := getExecutionIDFromPath(r)
	if executionID == 0 {
		common.HandleError(w, r, errors.New("invalid execution ID"))
		return
	}

	// Check permissions first
	if !h.canAccessExecution(r.Context(), executionID) {
		h.getForbiddenPage(w, r)
		return
	}

	// Get logs since last request (for incremental updates)
	since := getTimestampFromQuery(r, "since")

	executionRepo := execution.NewRepository(h.store)
	var logs []*store.TaskExecutionLog
	var err error

	if since.IsZero() {
		logs, err = executionRepo.GetLogs(r.Context(), executionID, 100, 0)
	} else {
		logs, err = executionRepo.GetLogsSince(r.Context(), executionID, since)
	}

	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	execution, err := executionRepo.GetByID(r.Context(), executionID)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	shouldRefresh := false
	if execution.Status == store.StatusSucceeded || execution.Status == store.StatusFailed || execution.Status == store.StatusKilled {
		shouldRefresh = true
	}

	logsComponent := component.LogEntries(logs, shouldRefresh)
	templ.Handler(logsComponent).ServeHTTP(w, r)
}

func (h *Handler) downloadExecutionFile(w http.ResponseWriter, r *http.Request) {
	executionID := getExecutionIDFromPath(r)
	filename := r.PathValue("filename")

	if executionID == 0 || filename == "" {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	// Check permissions
	if !h.canAccessExecution(r.Context(), executionID) {
		h.getForbiddenPage(w, r)
		return
	}

	executionRepo := execution.NewRepository(h.store)
	file, err := executionRepo.GetFileByPath(r.Context(), executionID, filename)
	if err != nil {
		if errors.Is(err, errors.New("record not found")) {
			http.NotFound(w, r)
			return
		}
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	// Security check: ensure file path is within execution directory
	if !h.isValidFilePath(executionID, file.FilePath) {
		http.Error(w, "Invalid file path", http.StatusForbidden)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", file.FileSize))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Filename))

	// Serve file
	http.ServeFile(w, r, file.FilePath)
}

func (h *Handler) getTaskExecutionHistory(w http.ResponseWriter, r *http.Request) {
	taskID := getTaskIDFromPath(r)
	if taskID == 0 {
		common.HandleError(w, r, errors.New("invalid task ID"))
		return
	}

	vmodel, err := h.fillTaskExecutionHistoryPageViewModel(r, taskID)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	page := component.TaskExecutionHistoryPage(*vmodel)
	templ.Handler(page).ServeHTTP(w, r)
}

func (h *Handler) getGlobalExecutionHistory(w http.ResponseWriter, r *http.Request) {
	vmodel, err := h.fillGlobalExecutionHistoryPageViewModel(r)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	page := component.GlobalExecutionHistoryPage(*vmodel)
	templ.Handler(page).ServeHTTP(w, r)
}

func (h *Handler) getHealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check database connectivity
	if err := h.store.Ping(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  "database unavailable",
		})
		return
	}

	// Check file system
	if err := h.checkFileSystemHealth(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "unhealthy",
			"error":  "file system unavailable",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// View model filling functions

func (h *Handler) fillExecutionPageViewModel(r *http.Request, executionID uint) (*component.ExecutionPageVModel, error) {
	vmodel := &component.ExecutionPageVModel{}
	ctx := r.Context()
	user := httpCtx.User(ctx)

	// Get execution with user authorization
	executionRepo := execution.NewRepository(h.store)
	var exec *store.TaskExecution
	var err error

	if user != nil && slices.Contains(user.Roles, authz.RoleAdmin) {
		// Admin can access any execution
		exec, err = executionRepo.GetByID(ctx, executionID)
	} else if user != nil {
		// Regular user can only access their own executions
		exec, err = executionRepo.GetByIDForUser(ctx, executionID, user.ID)
	} else {
		return nil, errors.New("unauthorized access")
	}

	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Get task
	taskRepo := NewTaskRepository(h.store)
	task, err := taskRepo.GetByID(ctx, exec.TaskID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Get recent logs
	logs, err := executionRepo.GetLogs(ctx, executionID, 100, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Get output files
	outputFiles, err := executionRepo.GetFiles(ctx, executionID, true)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	vmodel.Task = task
	vmodel.Execution = exec
	vmodel.Logs = logs
	vmodel.OutputFiles = outputFiles
	vmodel.IsRunning = isRunning(exec.Status)

	// Fill common view model parts
	err = common.FillViewModel(ctx, vmodel, r, h.fillExecutionProgressNavbarVModel)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillTaskExecutionHistoryPageViewModel(r *http.Request, taskID uint) (*component.TaskExecutionHistoryPageVModel, error) {
	vmodel := &component.TaskExecutionHistoryPageVModel{}
	ctx := r.Context()
	user := httpCtx.User(ctx)

	// Parse filters from query parameters
	filters := execution.ExecutionFilters{
		Status:   r.URL.Query().Get("status"),
		DateFrom: r.URL.Query().Get("date_from"),
		DateTo:   r.URL.Query().Get("date_to"),
	}

	// Get task
	taskRepo := NewTaskRepository(h.store)
	task, err := taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Get executions with user filtering
	executionRepo := execution.NewRepository(h.store)
	var executions []*store.TaskExecution
	var stats *execution.ExecutionStats

	if user != nil {
		// Use filtered search if any filters are provided
		if filters.Status != "" || filters.DateFrom != "" || filters.DateTo != "" {
			executions, err = executionRepo.SearchExecutionsForUserByTask(ctx, user.ID, taskID, filters, 50, 0)
		} else {
			// Regular user can only see their own executions for the task
			executions, err = executionRepo.GetByTaskIDForUser(ctx, taskID, user.ID, 50, 0)
		}
		if err != nil {
			return nil, errors.WithStack(err)
		}
		stats, err = executionRepo.GetExecutionStatsForUser(ctx, taskID, user.ID)
	} else {
		return nil, errors.New("unauthorized access")
	}

	if err != nil {
		return nil, errors.WithStack(err)
	}

	vmodel.Task = task
	vmodel.Executions = executions
	vmodel.Stats = stats
	vmodel.Filters = filters

	// Fill common view model parts
	err = common.FillViewModel(ctx, vmodel, r, h.fillTaskHistoryNavbarVModel)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillGlobalExecutionHistoryPageViewModel(r *http.Request) (*component.GlobalExecutionHistoryPageVModel, error) {
	vmodel := &component.GlobalExecutionHistoryPageVModel{}
	ctx := r.Context()
	user := httpCtx.User(ctx)

	// Parse filters from query parameters
	filters := execution.ExecutionFilters{
		Status:   r.URL.Query().Get("status"),
		DateFrom: r.URL.Query().Get("date_from"),
		DateTo:   r.URL.Query().Get("date_to"),
	}

	// Get executions with user filtering
	executionRepo := execution.NewRepository(h.store)
	var executions []*store.TaskExecution
	var err error

	if user != nil {
		executions, err = executionRepo.SearchExecutionsForUser(ctx, user.ID, filters, 50, 0)
	} else {
		return nil, errors.New("unauthorized access")
	}

	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Convert to ExecutionWithTask
	execWithTasks := make([]*component.ExecutionWithTask, len(executions))
	for i, exec := range executions {
		execWithTasks[i] = &component.ExecutionWithTask{
			Execution: exec,
			Task:      exec.Task, // Should be preloaded
		}
	}

	vmodel.Executions = execWithTasks
	vmodel.Filters = filters

	// Fill common view model parts
	err = common.FillViewModel(ctx, vmodel, r, h.fillGlobalHistoryNavbarVModel)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillExecutionProgressNavbarVModel(ctx context.Context, vmodel *component.ExecutionPageVModel, r *http.Request) error {
	return commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r)
}

func (h *Handler) fillTaskHistoryNavbarVModel(ctx context.Context, vmodel *component.TaskExecutionHistoryPageVModel, r *http.Request) error {
	return commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r)
}

func (h *Handler) fillGlobalHistoryNavbarVModel(ctx context.Context, vmodel *component.GlobalExecutionHistoryPageVModel, r *http.Request) error {
	return commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r)
}

// Helper functions

func getExecutionIDFromPath(r *http.Request) uint {
	rawExecutionID := r.PathValue("executionID")
	if rawExecutionID == "" {
		return 0
	}

	executionID, err := strconv.ParseUint(rawExecutionID, 10, 32)
	if err != nil {
		return 0
	}

	return uint(executionID)
}

func getTaskIDFromPath(r *http.Request) uint {
	rawTaskID := r.PathValue("taskID")
	if rawTaskID == "" {
		return 0
	}

	taskID, err := strconv.ParseUint(rawTaskID, 10, 32)
	if err != nil {
		return 0
	}

	return uint(taskID)
}

func getTimestampFromQuery(r *http.Request, param string) time.Time {
	timestampStr := r.URL.Query().Get(param)
	if timestampStr == "" {
		return time.Time{}
	}

	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return time.Time{}
	}

	return timestamp
}

func isRunning(status store.TaskExecutionStatus) bool {
	return status != store.StatusSucceeded &&
		status != store.StatusFailed &&
		status != store.StatusFinished
}

func (h *Handler) canAccessExecution(ctx context.Context, executionID uint) bool {
	user := httpCtx.User(ctx)
	if user == nil {
		return false
	}

	// Check if user is admin - admins can access all executions
	if slices.Contains(user.Roles, authz.RoleAdmin) {
		return true
	}

	// For regular users, check if they own the execution
	executionRepo := execution.NewRepository(h.store)
	exec, err := executionRepo.GetByIDForUser(ctx, executionID, user.ID)
	if err != nil || exec == nil {
		return false
	}

	return true
}

func (h *Handler) isValidFilePath(executionID uint, filePath string) bool {
	if h.fileStorage == nil {
		return false
	}

	expectedPrefix := h.fileStorage.GetExecutionPath(executionID)
	return strings.HasPrefix(filePath, expectedPrefix)
}

func (h *Handler) checkFileSystemHealth() error {
	if h.fileStorage == nil {
		return errors.New("file storage not configured")
	}

	// Try to create a test file
	testPath := h.fileStorage.GetBasePath() + "/.health_check"
	if err := h.fileStorage.EnsureDirectoryExists(); err != nil {
		errors.WithStack(err)
	}

	// Simple write test
	file, err := h.fileStorage.GetFile(testPath)
	if err == nil {
		file.Close()
	}

	return nil
}

// NewTaskRepository creates a task repository - this should be moved to a proper location
func NewTaskRepository(store *store.Store) *TaskRepository {
	return &TaskRepository{store: store}
}

type TaskRepository struct {
	store *store.Store
}

func (r *TaskRepository) GetByID(ctx context.Context, id uint) (*store.Task, error) {
	var task store.Task
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.First(&task, id).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}
