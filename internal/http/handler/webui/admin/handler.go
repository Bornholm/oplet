package admin

import (
	"net/http"

	"github.com/bornholm/oplet/internal/http/authz"
	"github.com/bornholm/oplet/internal/store"
)

type Handler struct {
	mux   *http.ServeMux
	store *store.Store
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler(store *store.Store) *Handler {
	h := &Handler{
		mux:   http.NewServeMux(),
		store: store,
	}

	// Admin-only middleware - only admins can access admin pages
	assertAdmin := authz.Middleware(http.HandlerFunc(h.getForbiddenPage), authz.Has(authz.RoleAdmin))

	// Admin routes
	h.mux.Handle("GET /", assertAdmin(http.HandlerFunc(h.getIndexPage)))

	return h
}

var _ http.Handler = &Handler{}
