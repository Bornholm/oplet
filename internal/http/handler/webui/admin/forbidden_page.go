package admin

import (
	"net/http"

	"github.com/a-h/templ"
	common "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
)

func (h *Handler) getForbiddenPage(w http.ResponseWriter, r *http.Request) {
	forbiddenPage := common.ErrorPage(common.ErrorPageVModel{
		Message: "L'accès à cette page d'administration ne vous est pas autorisé. Seuls les administrateurs peuvent accéder à cette section.",
		Links: []common.LinkItem{
			common.LinkItem{
				URL:   string(common.BaseURL(r.Context(), common.WithPath("/auth/logout"))),
				Label: "Se déconnecter",
			},
			common.LinkItem{
				URL:   string(common.BaseURL(r.Context(), common.WithPath("/"))),
				Label: "Retour à l'accueil",
			},
		},
	})

	w.WriteHeader(http.StatusForbidden)

	templ.Handler(forbiddenPage).ServeHTTP(w, r)
}
