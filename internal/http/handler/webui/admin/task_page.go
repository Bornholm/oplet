package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/a-h/templ"
	"github.com/bornholm/oplet/internal/http/handler/webui/admin/component"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	commonComp "github.com/bornholm/oplet/internal/http/handler/webui/common/component"
	taskForm "github.com/bornholm/oplet/internal/http/handler/webui/common/task"
	"github.com/bornholm/oplet/internal/slogx"
	"github.com/bornholm/oplet/internal/store"
	taskRepo "github.com/bornholm/oplet/internal/store/repository/task"
	"github.com/bornholm/oplet/internal/task"
	taskDef "github.com/bornholm/oplet/internal/task"
	"github.com/pkg/errors"
)

func (h *Handler) getTaskListPage(w http.ResponseWriter, r *http.Request) {
	vmodel, err := h.fillTaskListPageViewModel(r)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	taskListPage := component.TaskListPage(*vmodel)
	templ.Handler(taskListPage).ServeHTTP(w, r)
}

func (h *Handler) getTaskFormPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if this is an edit (has taskID in path)
	rawTaskID := r.PathValue("taskID")
	isEdit := rawTaskID != ""

	var taskID uint
	var storeTask *store.Task
	var taskDefinition *taskDef.Definition

	if isEdit {
		id, err := strconv.ParseUint(rawTaskID, 10, 32)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}
		taskID = uint(id)

		// Get existing task
		taskRepository := taskRepo.NewRepository(h.store)
		storeTask, err = taskRepository.GetByID(ctx, taskID)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		// Get task definition if we have an image ref
		if storeTask.ImageRef != "" {
			taskDefinition, err = h.taskProvider.FetchTaskDefinition(ctx, storeTask.ImageRef)
			if err != nil {
				h.logger.ErrorContext(ctx, "could not retrieve task definition", slogx.Error(errors.WithStack(err)))
				common.HandleError(w, r, common.NewError(err.Error(), "Could not retrieve the specified image", http.StatusInternalServerError))
				return
			}

			if err := h.updateTaskFromDefinition(ctx, storeTask, taskDefinition); err != nil {
				h.logger.ErrorContext(ctx, "could not update task from definition", slogx.Error(errors.WithStack(err)))
			}
		}
	}

	vmodel, err := h.fillTaskFormPageViewModel(r, storeTask, taskDefinition, isEdit)
	if err != nil {
		common.HandleError(w, r, errors.WithStack(err))
		return
	}

	taskFormPage := component.TaskFormPage(*vmodel)
	templ.Handler(taskFormPage).ServeHTTP(w, r)
}

func (h *Handler) handleTaskFormSubmission(w http.ResponseWriter, r *http.Request) {
	// Check if this is an edit
	rawTaskID := r.PathValue("taskID")
	isEdit := rawTaskID != ""

	var taskID uint
	var storeTask *store.Task

	taskRepository := taskRepo.NewRepository(h.store)
	ctx := r.Context()

	var redirectURL templ.SafeURL

	if isEdit {
		id, err := strconv.ParseUint(rawTaskID, 10, 32)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}
		taskID = uint(id)

		// Get existing task
		storeTask, err = taskRepository.GetByID(ctx, taskID)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		taskDefinition, err := h.taskProvider.FetchTaskDefinition(ctx, storeTask.ImageRef)
		if err != nil {
			h.logger.ErrorContext(ctx, "could not retrieve task definition", slogx.Error(errors.WithStack(err)))
			common.HandleError(w, r, common.NewError(err.Error(), "Could not retrieve the specified image", http.StatusInternalServerError))
			return
		}

		taskForm := taskForm.NewConfigurationForm(taskDefinition, storeTask)

		if err := taskForm.Handle(r); err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		// Validate form
		if !taskForm.IsValid(ctx) {
			// Re-render form with errors
			vmodel, err := h.fillTaskFormPageViewModel(r, storeTask, taskDefinition, true)
			if err != nil {
				common.HandleError(w, r, errors.WithStack(err))
				return
			}

			// Update form with validation errors
			vmodel.Form = taskForm

			page := component.TaskFormPage(*vmodel)
			templ.Handler(page).ServeHTTP(w, r)
			return
		}

		err = taskRepository.UpdateConfiguration(ctx, storeTask.ID, taskForm.Values)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		redirectURL = commonComp.BaseURL(ctx, commonComp.WithPathf("/admin/tasks/%d/edit", taskID))

	} else {
		// For new tasks, we need the image reference first
		imageRef := r.FormValue("image_ref")
		if imageRef == "" {
			http.Error(w, "Référence d'image requise", http.StatusBadRequest)
			return
		}

		// Fetch task definition from image reference
		taskDefinition, err := h.taskProvider.FetchTaskDefinition(ctx, imageRef)
		if err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		// Create new task with information from TaskDefinition
		storeTask = &store.Task{
			Name:        taskDefinition.Name,
			ImageRef:    imageRef,
			Author:      taskDefinition.Author,
			Description: taskDefinition.Description,
		}

		if err = taskRepository.Create(ctx, storeTask); err != nil {
			common.HandleError(w, r, errors.WithStack(err))
			return
		}

		redirectURL = commonComp.BaseURL(ctx, commonComp.WithPathf("/admin/tasks/%d/edit", storeTask.ID))
	}

	http.Redirect(w, r, string(redirectURL), http.StatusSeeOther)
}

func (h *Handler) handleTaskDeletion(w http.ResponseWriter, r *http.Request) {
	rawTaskID := r.PathValue("taskID")
	if rawTaskID == "" {
		http.Error(w, "Task ID is required", http.StatusBadRequest)
		return
	}

	taskID, err := strconv.ParseUint(rawTaskID, 10, 32)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	taskRepository := taskRepo.NewRepository(h.store)
	if err := taskRepository.Delete(r.Context(), uint(taskID)); err != nil {
		http.Error(w, "Failed to delete task", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// View model filling functions

func (h *Handler) fillTaskListPageViewModel(r *http.Request) (*component.TaskListPageVModel, error) {
	vmodel := &component.TaskListPageVModel{}
	ctx := r.Context()

	err := common.FillViewModel(
		ctx,
		vmodel, r,
		h.fillTaskListNavbarVModel,
		h.fillTaskListDataVModel,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillTaskFormPageViewModel(r *http.Request, storeTask *store.Task, taskDefinition *taskDef.Definition, isEdit bool) (*component.TaskFormPageVModel, error) {
	vmodel := &component.TaskFormPageVModel{
		Task:    storeTask,
		TaskDef: taskDefinition,
		IsEdit:  isEdit,
	}
	ctx := r.Context()

	if isEdit {
		taskForm := taskForm.NewConfigurationForm(taskDefinition, storeTask)
		vmodel.Form = taskForm
	} else {
		vmodel.Form = taskForm.NewImageRefForm()
	}

	err := common.FillViewModel(
		ctx,
		vmodel, r,
		h.fillTaskFormNavbarVModel,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return vmodel, nil
}

func (h *Handler) fillTaskListNavbarVModel(ctx context.Context, vmodel *component.TaskListPageVModel, r *http.Request) error {
	if err := commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (h *Handler) fillTaskFormNavbarVModel(ctx context.Context, vmodel *component.TaskFormPageVModel, r *http.Request) error {
	if err := commonComp.FillNavbarVModel(ctx, &vmodel.Navbar, r); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (h *Handler) fillTaskListDataVModel(ctx context.Context, vmodel *component.TaskListPageVModel, r *http.Request) error {
	taskRepository := taskRepo.NewRepository(h.store)
	tasks, err := taskRepository.List(ctx, 0, 0) // Get all tasks
	if err != nil {
		return errors.WithStack(err)
	}

	vmodel.Tasks = tasks
	return nil
}

func (h *Handler) updateTaskFromDefinition(ctx context.Context, task *store.Task, definition *task.Definition) error {
	changed := false

	if task.Author != definition.Author {
		task.Author = definition.Author
		changed = true
	}

	if task.Description != definition.Description {
		task.Description = definition.Description
		changed = true
	}

	if task.Name != definition.Name {
		task.Name = definition.Name
		changed = true
	}

	if changed {
		repo := taskRepo.NewRepository(h.store)
		if err := repo.Update(ctx, task); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
