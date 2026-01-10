package pprof

import (
	"expvar"
	"net/http"
	"net/http/pprof"
)

type Handler struct {
	mux *http.ServeMux
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func NewHandler() *Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", pprof.Index)
	mux.HandleFunc("/cmdline", pprof.Cmdline)
	mux.HandleFunc("/profile", pprof.Profile)
	mux.HandleFunc("/symbol", pprof.Symbol)
	mux.HandleFunc("/trace", pprof.Trace)
	mux.Handle("/vars", expvar.Handler())
	mux.HandleFunc("/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		pprof.Handler(name).ServeHTTP(w, r)
	})

	return &Handler{mux}
}

var _ http.Handler = &Handler{}
