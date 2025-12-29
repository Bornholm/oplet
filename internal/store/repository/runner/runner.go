package runner

import (
	"context"
	"time"

	"github.com/bornholm/oplet/internal/crypto"
	"github.com/bornholm/oplet/internal/store"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) GetRunnerByToken(ctx context.Context, token string) (*store.Runner, error) {
	var runner store.Runner
	err := r.store.WithTx(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Preload(clause.Associations).Where("token = ?", token).First(&runner).Error; err != nil {
			return errors.WithStack(err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &runner, nil
}

func (r *Repository) UpdateContactAt(ctx context.Context, runnerID uint, contactedAt time.Time) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Model(&store.Runner{}).Where("id = ?", runnerID).UpdateColumn("contacted_at", contactedAt).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

func (r *Repository) Update(ctx context.Context, runner *store.Runner) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Save(runner).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

func (r *Repository) List(ctx context.Context, offset, limit int) ([]*store.Runner, error) {
	var runners []*store.Runner
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		query := db.Order("created_at DESC")

		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}

		if err := query.Find(&runners).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return runners, nil
}

func (r *Repository) ListWithPagination(ctx context.Context, offset, limit int) ([]*store.Runner, int64, error) {
	var runners []*store.Runner
	var total int64

	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		// Count total runners
		if err := db.Model(&store.Runner{}).Count(&total).Error; err != nil {
			return errors.WithStack(err)
		}

		// Get runners with pagination
		query := db.Order("created_at DESC")
		if limit > 0 {
			query = query.Limit(limit)
		}
		if offset > 0 {
			query = query.Offset(offset)
		}

		if err := query.Find(&runners).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	return runners, total, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint) (*store.Runner, error) {
	var runner store.Runner
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.First(&runner, id).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &runner, nil
}

func (r *Repository) GetByName(ctx context.Context, name string) (*store.Runner, error) {
	var runner store.Runner
	err := r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Where("name = ?", name).First(&runner).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &runner, nil
}

func (r *Repository) Create(ctx context.Context, runner *store.Runner) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Create(runner).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

func (r *Repository) Delete(ctx context.Context, id uint) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Delete(&store.Runner{}, id).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}

func (r *Repository) RegenerateToken(ctx context.Context, id uint) (string, error) {
	var newToken string
	err := r.store.WithTx(ctx, func(ctx context.Context, db *gorm.DB) error {
		// Generate new token
		token, err := crypto.RandomToken(tokenSize)
		if err != nil {
			return errors.Wrap(err, "could not generate new runner token")
		}
		newToken = token

		// Update runner with new token
		if err := db.Model(&store.Runner{}).Where("id = ?", id).UpdateColumn("token", newToken).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return newToken, nil
}

func (r *Repository) UpdateName(ctx context.Context, id uint, name string) error {
	return r.store.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Model(&store.Runner{}).Where("id = ?", id).UpdateColumn("name", name).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
}
