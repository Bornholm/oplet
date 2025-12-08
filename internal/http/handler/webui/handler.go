package webui

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/bornholm/oplet/internal/file"
	adminModule "github.com/bornholm/oplet/internal/http/handler/webui/admin"
	taskModule "github.com/bornholm/oplet/internal/http/handler/webui/task"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/task"
)

type Handler struct {
	mux *http.ServeMux
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(store *store.Store, taskProvider task.Provider, taskExecutor task.Executor, fileStorage *file.Storage, logger *slog.Logger) *Handler {
	mux := http.NewServeMux()

	h := &Handler{
		mux: mux,
	}

	mount(mux, "/", taskModule.NewHandler(store, taskProvider, taskExecutor, fileStorage, logger))
	mount(mux, "/admin/", adminModule.NewHandler(store, taskProvider, fileStorage, logger))

	return h
}

func mount(mux *http.ServeMux, prefix string, handler http.Handler) {
	trimmed := strings.TrimSuffix(prefix, "/")

	if len(trimmed) > 0 {
		mux.Handle(prefix, http.StripPrefix(trimmed, handler))
	} else {
		mux.Handle(prefix, handler)
	}
}

var _ http.Handler = &Handler{}
