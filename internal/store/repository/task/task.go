package task

import (
	"context"
	"fmt"

	"github.com/bornholm/oplet/internal/store"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// Create creates a new task
func (r *Repository) Create(ctx context.Context, task *store.Task) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Create(task).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

// GetByID retrieves a task by its ID
func (r *Repository) GetByID(ctx context.Context, id uint) (*store.Task, error) {
	var task store.Task
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.First(&task, id).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByImageRef retrieves a task by its image reference (unique field)
func (r *Repository) GetByImageRef(ctx context.Context, imageRef string) (*store.Task, error) {
	var task store.Task
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("image_ref = ?", imageRef).First(&task).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// List retrieves all tasks with optional pagination
func (r *Repository) List(ctx context.Context, limit, offset int) ([]*store.Task, error) {
	var tasks []*store.Task
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Order("created_at DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}
		if err := query.Find(&tasks).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// Update updates an existing task
func (r *Repository) Update(ctx context.Context, task *store.Task) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Save(task).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

// Delete deletes a task by ID
func (r *Repository) Delete(ctx context.Context, id uint) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Delete(&store.Task{}, id).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

// Search performs a full-text search on tasks
func (r *Repository) Search(ctx context.Context, query string) ([]*store.Task, error) {
	var tasks []*store.Task
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		// Use LIKE for full-text search across name, description, author, and image_ref
		searchPattern := fmt.Sprintf("%%%s%%", query)
		if err := db.Where(
			"name LIKE ? OR description LIKE ? OR author LIKE ? OR image_ref LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		).Order("created_at DESC").Find(&tasks).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

// Count returns the total number of tasks
func (r *Repository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Model(&store.Task{}).Count(&count).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
