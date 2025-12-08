package user

import (
	"context"

	"github.com/bornholm/oplet/internal/store"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// Create creates a new user
func (r *Repository) Create(ctx context.Context, user *store.User) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Create(user).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

// GetByID retrieves a user by its ID
func (r *Repository) GetByID(ctx context.Context, id uint) (*store.User, error) {
	var user store.User
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.First(&user, id).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetBySubject retrieves a user by its provider/subject tuple
func (r *Repository) GetBySubject(ctx context.Context, provider, subject string) (*store.User, error) {
	var user store.User
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("provider = ? and subject = ?", provider, subject).First(&user).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// List retrieves all users with optional pagination
func (r *Repository) List(ctx context.Context, limit, offset int) ([]*store.User, error) {
	var users []*store.User
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Order("created_at DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}
		if err := query.Find(&users).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}

// Update updates an existing user
func (r *Repository) Update(ctx context.Context, user *store.User) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Save(user).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

// Delete deletes a user by ID
func (r *Repository) Delete(ctx context.Context, id uint) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Delete(&store.User{}, id).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

// Count returns the total number of users
func (r *Repository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Model(&store.User{}).Count(&count).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}
