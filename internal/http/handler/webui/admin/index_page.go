package admin

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/oplet/internal/http/handler/webui/admin/component"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	commonComp "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
	"github.com/pkg/errors"
)

func (h *Handler) getIndexPage(w http.ResponseWriter, r *http.Request) {
	vmodel, err := h.fillIndexPageViewModel(r)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	adminPage := component.IndexPage(*vmodel)

	templ.Handler(adminPage).ServeHTTP(w, r)
}

func (h *Handler) fillIndexPageViewModel(r *http.Request) (*component.IndexPageVModel, error) {
	vmodel := &component.IndexPageVModel{}

	ctx := r.Context()

	err := common.FillViewModel(
		ctx,
		vmodel, r,
		h.fillIndexPageNavbarVModel,
		h.fillIndexPageAdminDataVModel,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillIndexPageNavbarVModel(ctx context.Context, vmodel *component.IndexPageVModel, r *http.Request) error {
	if err := commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (h *Handler) fillIndexPageAdminDataVModel(ctx context.Context, vmodel *component.IndexPageVModel, r *http.Request) error {
	// TODO: Add admin-specific data filling logic here
	// For example: user statistics, system status, etc.

	return nil
}
