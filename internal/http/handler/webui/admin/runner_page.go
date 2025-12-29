package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/bornholm/oplet/internal/crypto"
	"github.com/bornholm/oplet/internal/http/handler/webui/admin/component"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	commonComp "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
	"github.com/bornholm/oplet/internal/store"
	runnerRepo "github.com/bornholm/oplet/internal/store/repository/runner"
	"github.com/pkg/errors"
)

func (h *Handler) getRunnerListPage(w http.ResponseWriter, r *http.Request) {
	vmodel, err := h.fillRunnerListPageViewModel(r)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	runnerListPage := component.RunnerListPage(*vmodel)
	templ.Handler(runnerListPage).ServeHTTP(w, r)
}

func (h *Handler) getRunnerFormPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if this is an edit (has runnerID in path)
	rawRunnerID := r.PathValue("runnerID")
	isEdit := rawRunnerID != ""

	var runnerID uint
	var storeRunner *store.Runner

	if isEdit {
		id, err := strconv.ParseUint(rawRunnerID, 10, 32)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}
		runnerID = uint(id)

		// Get existing runner
		runnerRepository := runnerRepo.NewRepository(h.store)
		storeRunner, err = runnerRepository.GetByID(ctx, runnerID)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}
	}

	vmodel, err := h.fillRunnerFormPageViewModel(r, storeRunner, isEdit)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	runnerFormPage := component.RunnerFormPage(*vmodel)
	templ.Handler(runnerFormPage).ServeHTTP(w, r)
}

func (h *Handler) handleRunnerFormSubmission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	runnerRepository := runnerRepo.NewRepository(h.store)

	// Check if this is an edit
	rawRunnerID := r.PathValue("runnerID")
	isEdit := rawRunnerID != ""

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	runnerName := r.FormValue("name")
	if runnerName == "" {
		http.Error(w, "Runner name is required", http.StatusBadRequest)
		return
	}

	var redirectURL templ.SafeURL

	if isEdit {
		runnerID, err := strconv.ParseUint(rawRunnerID, 10, 32)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		// Update runner name
		if err := runnerRepository.UpdateName(ctx, uint(runnerID), runnerName); err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		h.logger.InfoContext(ctx, "Runner name updated",
			"runner_id", runnerID,
			"new_name", runnerName)

		redirectURL = commonComp.BaseURL(ctx, commonComp.WithPathf("/admin/runners/%d/edit", runnerID))

	} else {
		// Create new runner
		token, err := crypto.RandomToken(32)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		storeRunner := &store.Runner{
			Name:  runnerName,
			Token: token,
		}

		if err := runnerRepository.Create(ctx, storeRunner); err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		h.logger.InfoContext(ctx, "Runner created",
			"runner_id", storeRunner.ID,
			"name", runnerName)

		redirectURL = commonComp.BaseURL(ctx, commonComp.WithPathf("/admin/runners/%d/edit", storeRunner.ID))
	}

	http.Redirect(w, r, string(redirectURL), http.StatusSeeOther)
}

func (h *Handler) handleRunnerDeletion(w http.ResponseWriter, r *http.Request) {
	rawRunnerID := r.PathValue("runnerID")
	if rawRunnerID == "" {
		http.Error(w, "Runner ID is required", http.StatusBadRequest)
		return
	}

	runnerID, err := strconv.ParseUint(rawRunnerID, 10, 32)
	if err != nil {
		http.Error(w, "Invalid runner ID", http.StatusBadRequest)
		return
	}

	runnerRepository := runnerRepo.NewRepository(h.store)
	if err := runnerRepository.Delete(r.Context(), uint(runnerID)); err != nil {
		http.Error(w, "Failed to delete runner", http.StatusInternalServerError)
		return
	}

	h.logger.InfoContext(r.Context(), "Runner deleted",
		"runner_id", runnerID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *Handler) handleRunnerTokenRegeneration(w http.ResponseWriter, r *http.Request) {
	rawRunnerID := r.PathValue("runnerID")
	if rawRunnerID == "" {
		http.Error(w, "Runner ID is required", http.StatusBadRequest)
		return
	}

	runnerID, err := strconv.ParseUint(rawRunnerID, 10, 32)
	if err != nil {
		http.Error(w, "Invalid runner ID", http.StatusBadRequest)
		return
	}

	runnerRepository := runnerRepo.NewRepository(h.store)
	newToken, err := runnerRepository.RegenerateToken(r.Context(), uint(runnerID))
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	h.logger.InfoContext(r.Context(), "Runner token regenerated",
		"runner_id", runnerID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"token":  newToken,
	})
}

func (h *Handler) handleRunnerNameValidation(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Name parameter is required", http.StatusBadRequest)
		return
	}

	// Check if editing existing runner
	rawRunnerID := r.URL.Query().Get("runner_id")
	var currentRunnerID uint
	if rawRunnerID != "" {
		id, err := strconv.ParseUint(rawRunnerID, 10, 32)
		if err == nil {
			currentRunnerID = uint(id)
		}
	}

	runnerRepository := runnerRepo.NewRepository(h.store)
	existingRunner, err := runnerRepository.GetByName(r.Context(), name)

	// Name is available if:
	// 1. No runner exists with this name, OR
	// 2. The existing runner is the one being edited
	isAvailable := err != nil || (existingRunner != nil && existingRunner.ID == currentRunnerID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"available": isAvailable,
	})
}

// View model filling functions

func (h *Handler) fillRunnerListPageViewModel(r *http.Request) (*component.RunnerListPageVModel, error) {
	vmodel := &component.RunnerListPageVModel{}
	ctx := r.Context()

	err := common.FillViewModel(
		ctx,
		vmodel, r,
		h.fillRunnerListNavbarVModel,
		h.fillRunnerListDataVModel,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillRunnerFormPageViewModel(r *http.Request, storeRunner *store.Runner, isEdit bool) (*component.RunnerFormPageVModel, error) {
	vmodel := &component.RunnerFormPageVModel{
		Runner: storeRunner,
		IsEdit: isEdit,
	}
	ctx := r.Context()

	err := common.FillViewModel(
		ctx,
		vmodel, r,
		h.fillRunnerFormNavbarVModel,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillRunnerListNavbarVModel(ctx context.Context, vmodel *component.RunnerListPageVModel, r *http.Request) error {
	if err := commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (h *Handler) fillRunnerFormNavbarVModel(ctx context.Context, vmodel *component.RunnerFormPageVModel, r *http.Request) error {
	if err := commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (h *Handler) fillRunnerListDataVModel(ctx context.Context, vmodel *component.RunnerListPageVModel, r *http.Request) error {
	runnerRepository := runnerRepo.NewRepository(h.store)
	runners, total, err := runnerRepository.ListWithPagination(ctx, 0, 0) // Get all runners
	if err != nil {
		return errors.WithStack(err)
	}

	vmodel.Runners = runners
	vmodel.TotalRunners = total
	return nil
}
