package seed

import (
	"context"
	"time"

	"github.com/bornholm/oplet/internal/store"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExecFunc func(ctx context.Context, db *gorm.DB) error

type Seeder struct {
	id   string
	exec ExecFunc
}

func New(id string, exec ExecFunc) *Seeder {
	return &Seeder{
		id:   id,
		exec: exec,
	}
}

func (r *Repository) Seed(ctx context.Context, force bool, seeders ...*Seeder) error {
	for _, s := range seeders {
		err := r.store.WithRetry(ctx, func(ctx context.Context, db *gorm.DB) error {
			var count int64
			if err := db.Model(&store.Seed{}).Where("id = ?", s.id).Count(&count).Error; err != nil {
				return errors.WithStack(err)
			}

			if !force && count > 0 {
				return nil
			}

			if err := s.exec(ctx, db); err != nil {
				return errors.WithStack(err)
			}

			seed := &store.Seed{
				ID:         s.id,
				ExecutedAt: time.Now(),
			}

			if err := db.Clauses(clause.OnConflict{
				DoUpdates: clause.AssignmentColumns([]string{"executed_at"}),
			}).Create(seed).Error; err != nil {
				return errors.WithStack(err)
			}

			return nil
		})
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
