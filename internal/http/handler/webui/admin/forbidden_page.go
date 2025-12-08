package admin

import (
	"net/http"

	"github.com/a-h/templ"
	common "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
)

func (h *Handler) getForbiddenPage(w http.ResponseWriter, r *http.Request) {
	forbiddenPage := common.ErrorPage(common.ErrorPageVModel{
		Message: "Only administrators can access this section.",
		Links: []common.LinkItem{
			{
				URL:   string(common.BaseURL(r.Context(), common.WithPath("/auth/logout"))),
				Label: "Logout",
			},
			{
				URL:   string(common.BaseURL(r.Context(), common.WithPath("/"))),
				Label: "Back to home page",
			},
		},
	})

	w.WriteHeader(http.StatusForbidden)

	templ.Handler(forbiddenPage).ServeHTTP(w, r)
}
