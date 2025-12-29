package admin

import (
	"log/slog"
	"net/http"

	"github.com/bornholm/oplet/internal/file"
	"github.com/bornholm/oplet/internal/http/authz"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/task"

	commonComp "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
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
		logger:       logger.With("component", "admin-handler"),
	}

	// Admin-only middleware - only admins can access admin pages
	assertAdmin := authz.Middleware(http.HandlerFunc(h.getForbiddenPage), authz.Has(authz.RoleAdmin))

	h.mux.Handle("GET /", assertAdmin(http.HandlerFunc(redirect("/admin/users"))))

	// Task management routes
	h.mux.Handle("GET /tasks", assertAdmin(http.HandlerFunc(h.getTaskListPage)))
	h.mux.Handle("GET /tasks/new", assertAdmin(http.HandlerFunc(h.getTaskFormPage)))
	h.mux.Handle("POST /tasks/new", assertAdmin(http.HandlerFunc(h.handleTaskFormSubmission)))
	h.mux.Handle("GET /tasks/{taskID}/edit", assertAdmin(http.HandlerFunc(h.getTaskFormPage)))
	h.mux.Handle("POST /tasks/{taskID}/edit", assertAdmin(http.HandlerFunc(h.handleTaskFormSubmission)))
	h.mux.Handle("DELETE /tasks/{taskID}", assertAdmin(http.HandlerFunc(h.handleTaskDeletion)))

	// User management routes
	h.mux.Handle("GET /users", assertAdmin(http.HandlerFunc(h.getUserListPage)))
	h.mux.Handle("GET /users/{userID}/edit", assertAdmin(http.HandlerFunc(h.getUserFormPage)))
	h.mux.Handle("POST /users/{userID}/role", assertAdmin(http.HandlerFunc(h.handleUserRoleUpdate)))
	h.mux.Handle("POST /users/{userID}/status", assertAdmin(http.HandlerFunc(h.handleUserStatusUpdate)))

	return h
}

var _ http.Handler = &Handler{}

func redirect(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		redirectURL := commonComp.BaseURL(r.Context(), commonComp.WithPath(path))
		http.Redirect(w, r, string(redirectURL), http.StatusTemporaryRedirect)
	}
}
