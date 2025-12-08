package task

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	commonComp "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
	"github.com/bornholm/oplet/internal/http/handler/webui/task/component"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/store/repository/task"
	"github.com/pkg/errors"
)

func (h *Handler) getIndexPage(w http.ResponseWriter, r *http.Request) {
	vmodel, err := h.fillIndexPageViewModel(r)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	taskPage := component.IndexPage(*vmodel)

	templ.Handler(taskPage).ServeHTTP(w, r)
}

func (h *Handler) fillIndexPageViewModel(r *http.Request) (*component.IndexPageVModel, error) {
	vmodel := &component.IndexPageVModel{}

	ctx := r.Context()

	err := common.FillViewModel(
		ctx,
		vmodel, r,
		h.fillIndexPageNavbarVModel,
		h.fillIndexPageTasksVModel,
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

func (h *Handler) fillIndexPageTasksVModel(ctx context.Context, vmodel *component.IndexPageVModel, r *http.Request) error {
	// Get search query from URL parameters
	searchQuery := r.URL.Query().Get("q")
	vmodel.SearchQuery = searchQuery

	// Create task repository
	taskRepo := task.NewRepository(h.store)

	var tasks []*store.Task
	var err error

	// If there's a search query, use Search function, otherwise list all tasks
	if searchQuery != "" {
		tasks, err = taskRepo.Search(ctx, searchQuery)
	} else {
		// List all tasks with reasonable pagination (0 means no limit)
		tasks, err = taskRepo.List(ctx, 0, 0)
	}

	if err != nil {
		return errors.WithStack(err)
	}

	vmodel.Tasks = tasks

	return nil
}
