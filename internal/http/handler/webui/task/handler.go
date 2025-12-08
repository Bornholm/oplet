package task

import (
	"log/slog"
	"net/http"

	"github.com/bornholm/oplet/internal/file"
	"github.com/bornholm/oplet/internal/http/authz"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/task"
)

type Handler struct {
	mux          *http.ServeMux
	store        *store.Store
	taskProvider task.Provider
	taskExecutor task.Executor
	fileStorage  *file.Storage
	logger       *slog.Logger
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(store *store.Store, taskProvider task.Provider, taskExecutor task.Executor, fileStorage *file.Storage, logger *slog.Logger) *Handler {
	h := &Handler{
		mux:          http.NewServeMux(),
		store:        store,
		taskProvider: taskProvider,
		taskExecutor: taskExecutor,
		fileStorage:  fileStorage,
		logger:       logger.With("component", "task-handler"),
	}

	assertUser := authz.Middleware(http.HandlerFunc(h.getForbiddenPage), authz.OneOf(authz.Has(authz.RoleUser), authz.Has(authz.RoleAdmin)))
	// assertAdmin := authz.Middleware(http.HandlerFunc(h.getForbiddenPage), authz.Has(authz.RoleUser))

	h.mux.Handle("GET /", assertUser(http.HandlerFunc(h.getIndexPage)))
	h.mux.Handle("GET /tasks", assertUser(http.HandlerFunc(h.getIndexPage)))
	h.mux.Handle("GET /tasks/{taskID}/new", assertUser(http.HandlerFunc(h.getNewTaskPage)))
	h.mux.Handle("POST /tasks/{taskID}/new", assertUser(http.HandlerFunc(h.getNewTaskPage)))

	// Add new routes for execution tracking
	h.mux.Handle("GET /tasks/{taskID}/executions/{executionID}", assertUser(http.HandlerFunc(h.getExecutionProgressPage)))
	h.mux.Handle("GET /tasks/{taskID}/executions/{executionID}/logs", assertUser(http.HandlerFunc(h.getExecutionLogs)))
	h.mux.Handle("GET /tasks/{taskID}/executions/{executionID}/files/{filename}", assertUser(http.HandlerFunc(h.downloadExecutionFile)))
	h.mux.Handle("GET /tasks/{taskID}/executions", assertUser(http.HandlerFunc(h.getTaskExecutionHistory)))
	h.mux.Handle("GET /tasks/executions", assertUser(http.HandlerFunc(h.getGlobalExecutionHistory)))
	h.mux.Handle("GET /health", http.HandlerFunc(h.getHealthCheck))

	return h
}

var _ http.Handler = &Handler{}
