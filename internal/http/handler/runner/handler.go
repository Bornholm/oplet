package runner

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bornholm/oplet/internal/file"
	"github.com/bornholm/oplet/internal/slogx"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/store/repository/execution"
	"github.com/bornholm/oplet/internal/store/repository/runner"
	"github.com/bornholm/oplet/internal/task"
	"github.com/pkg/errors"
)

type Handler struct {
	mux          *http.ServeMux
	store        *store.Store
	taskProvider task.Provider
	fileStorage  *file.Storage
	logger       *slog.Logger
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(store *store.Store, taskProvider task.Provider, fileStorage *file.Storage, logger *slog.Logger) *Handler {
	h := &Handler{
		mux:          http.NewServeMux(),
		store:        store,
		taskProvider: taskProvider,
		fileStorage:  fileStorage,
		logger:       logger.With("component", "runner-handler"),
	}

	h.mux.HandleFunc("POST /heartbeat", h.assertRunner(h.handleHeartbeat))
	h.mux.HandleFunc("GET /request-task", h.assertRunner(h.handleTaskRequest))
	h.mux.HandleFunc("GET /tasks/{taskID}/inputs", h.assertRunner(h.handleTaskInputs))
	h.mux.HandleFunc("POST /tasks/{taskID}/trace", h.assertRunner(h.handleTaskTrace))
	h.mux.HandleFunc("POST /tasks/{taskID}/status", h.assertRunner(h.handleTaskStatus))
	h.mux.HandleFunc("POST /tasks/{taskID}/outputs", h.assertRunner(h.handleTaskOutputs))

	return h
}

func (h *Handler) assertRunner(next http.HandlerFunc) http.HandlerFunc {

	repo := runner.NewRepository(h.store)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")
		runnerToken := strings.TrimPrefix(authorization, "Bearer ")

		ctx := r.Context()

		if runnerToken == "" {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		runner, err := repo.GetRunnerByToken(ctx, runnerToken)
		if err != nil {
			h.logger.WarnContext(ctx, "could not retrieve runner from token", slogx.Error(errors.WithStack(err)), slog.String("token", runnerToken))

			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		if err := repo.UpdateContactAt(ctx, runner.ID, time.Now()); err != nil {
			h.logger.WarnContext(ctx, "could not update runner", slogx.Error(errors.WithStack(err)))

			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		ctx = withContextRunner(ctx, runner)

		r = r.WithContext(ctx)

		next(w, r)
	})
}

// handleHeartbeat handles POST /runner/heartbeat
func (h *Handler) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	runner, err := contextRunner(ctx)
	if err != nil {
		handleInternalError(h, w, r, err, "could not retrieve runner from context")
		return
	}

	response := HeartbeatResponse{
		ID:          runner.ID,
		Name:        runner.Name,
		ContactedAt: time.Now(),
	}

	writeJSONResponse(w, http.StatusOK, response)

	h.logger.DebugContext(ctx, "heartbeat received",
		"runner_id", runner.ID,
		"runner_name", runner.Name)
}

// handleTaskStatus handles POST /runner/tasks/{taskID}/status
func (h *Handler) handleTaskStatus(w http.ResponseWriter, r *http.Request) {
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

	var req TaskStatusRequest
	if err := parseJSONRequest(r, &req); err != nil {
		handleValidationError(w, err)
		return
	}

	if err := req.Validate(); err != nil {
		handleValidationError(w, err)
		return
	}

	// Get execution by task ID
	taskIDUint, err := strconv.ParseUint(taskID, 10, 32)
	if err != nil {
		handleValidationError(w, ErrInvalidRequest("invalid task ID"))
		return
	}

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

	// Update execution status
	exec.Status = req.Status
	if req.ContainerID != "" {
		exec.ContainerID = req.ContainerID
	}
	if req.ExitCode != nil {
		exec.ExitCode = req.ExitCode
	}
	if req.Error != "" {
		exec.ErrorMessage = req.Error
	}
	if req.StartedAt != nil {
		exec.StartedAt = req.StartedAt
	}
	if req.FinishedAt != nil {
		exec.FinishedAt = req.FinishedAt
	}

	if err := executionRepo.Update(ctx, exec); err != nil {
		handleInternalError(h, w, r, err, "could not update execution status")
		return
	}

	// Add system log entry for status change
	logEntry := &store.TaskExecutionLog{
		ExecutionID: exec.ID,
		Timestamp:   req.Timestamp,
		Source:      "system",
		Message:     fmt.Sprintf("Status changed to: %s", req.Status),
		Clock:       uint(req.Timestamp),
	}

	if err := executionRepo.AddLog(ctx, exec.ID, logEntry); err != nil {
		h.logger.WarnContext(ctx, "could not add status change log",
			"execution_id", exec.ID, "error", err)
	}

	response := TaskStatusResponse{
		ExecutionID: exec.ID,
		Status:      req.Status,
		UpdatedAt:   time.Now(),
	}

	writeJSONResponse(w, http.StatusOK, response)

	h.logger.InfoContext(ctx, "task status updated",
		"runner_id", runner.ID,
		"execution_id", exec.ID,
		"task_id", taskID,
		"status", req.Status)
}

var _ http.Handler = &Handler{}
