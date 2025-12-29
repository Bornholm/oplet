package admin

import (
	"context"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/bornholm/oplet/internal/http/authz"
	"github.com/bornholm/oplet/internal/http/handler/webui/admin/component"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	commonComp "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/store/repository/user"
	"github.com/pkg/errors"
)

func (h *Handler) getUserListPage(w http.ResponseWriter, r *http.Request) {
	vmodel, err := h.fillUserListPageViewModel(r)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	userListPage := component.UserListPage(*vmodel)
	templ.Handler(userListPage).ServeHTTP(w, r)
}

func (h *Handler) getUserFormPage(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("userID")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	userRepo := user.NewRepository(h.store)
	storeUser, err := userRepo.GetByID(r.Context(), uint(userID))
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	vmodel, err := h.fillUserFormPageViewModel(r, storeUser)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	userFormPage := component.UserFormPage(*vmodel)
	templ.Handler(userFormPage).ServeHTTP(w, r)
}

func (h *Handler) fillUserFormPageViewModel(r *http.Request, user *store.User) (*component.UserFormPageVModel, error) {
	vmodel := &component.UserFormPageVModel{
		User: user,
	}

	ctx := r.Context()

	err := common.FillViewModel(
		ctx,
		vmodel, r,
		h.fillUserFormNavbarVModel,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillUserFormNavbarVModel(ctx context.Context, vmodel *component.UserFormPageVModel, r *http.Request) error {
	if err := commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (h *Handler) handleUserRoleUpdate(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("userID")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	newRole := r.FormValue("role")
	if newRole != authz.RoleUser && newRole != authz.RoleAdmin {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	userRepo := user.NewRepository(h.store)
	if err := userRepo.UpdateRole(r.Context(), uint(userID), newRole); err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	h.logger.InfoContext(r.Context(), "User role updated",
		"user_id", userID,
		"new_role", newRole)

	// Redirect back to user edit page
	redirectURL := commonComp.BaseURL(r.Context(), commonComp.WithPath("/admin/users", strconv.FormatUint(userID, 10), "edit"))
	http.Redirect(w, r, string(redirectURL), http.StatusSeeOther)
}

func (h *Handler) handleUserStatusUpdate(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("userID")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	isActiveStr := r.FormValue("is_active")
	isActive := isActiveStr == "true"

	userRepo := user.NewRepository(h.store)
	if err := userRepo.UpdateActiveStatus(r.Context(), uint(userID), isActive); err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	h.logger.InfoContext(r.Context(), "User status updated",
		"user_id", userID,
		"is_active", isActive)

	// Redirect back to user edit page
	redirectURL := commonComp.BaseURL(r.Context(), commonComp.WithPath("/admin/users", strconv.FormatUint(userID, 10), "edit"))
	http.Redirect(w, r, string(redirectURL), http.StatusSeeOther)
}

func (h *Handler) fillUserListPageViewModel(r *http.Request) (*component.UserListPageVModel, error) {
	vmodel := &component.UserListPageVModel{}

	ctx := r.Context()

	err := common.FillViewModel(
		ctx,
		vmodel, r,
		h.fillUserListPageNavbarVModel,
		h.fillUserListPageDataVModel,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillUserListPageNavbarVModel(ctx context.Context, vmodel *component.UserListPageVModel, r *http.Request) error {
	if err := commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (h *Handler) fillUserListPageDataVModel(ctx context.Context, vmodel *component.UserListPageVModel, r *http.Request) error {
	userRepo := user.NewRepository(h.store)

	// Get all users (for now, without pagination)
	users, total, err := userRepo.ListWithPagination(ctx, 0, 0)
	if err != nil {
		return errors.WithStack(err)
	}

	vmodel.Users = users
	vmodel.TotalUsers = total

	return nil
}
