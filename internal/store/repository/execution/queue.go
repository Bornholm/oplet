package execution

import (
	"context"
	"time"

	"github.com/bornholm/oplet/internal/crypto"
	"github.com/bornholm/oplet/internal/store"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const tokenSize int = 32

func (r *Repository) NextTask(ctx context.Context) (*store.TaskExecution, error) {
	var execution store.TaskExecution
	err := r.store.WithTx(ctx, func(ctx context.Context, db *gorm.DB) error {
		err := db.Model(&execution).
			Preload(clause.Associations).
			Preload("Task.Configurations").
			Where("started_at is null").
			Order("created_at ASC").
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			First(&execution).
			Error
		if err != nil {
			return errors.WithStack(err)
		}

		now := time.Now()

		execution.StartedAt = &now

		token, err := crypto.RandomToken(tokenSize)
		if err != nil {
			return errors.WithStack(err)
		}

		execution.RunnerToken = token

		if err := db.Save(&execution).Error; err != nil {
			return errors.WithStack(err)
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &execution, nil
}
