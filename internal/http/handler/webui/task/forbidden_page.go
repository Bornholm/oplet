package task

import (
	"net/http"

	"github.com/a-h/templ"
	common "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
)

func (h *Handler) getForbiddenPage(w http.ResponseWriter, r *http.Request) {
	forbiddenPage := common.ErrorPage(common.ErrorPageVModel{
		Message: "You are not authorized to access this page. Please contact an administrator.",
	})

	w.WriteHeader(http.StatusForbidden)

	templ.Handler(forbiddenPage).ServeHTTP(w, r)
}
