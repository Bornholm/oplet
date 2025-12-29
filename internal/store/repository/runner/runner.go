package runner

import (
	"context"
	"time"

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
