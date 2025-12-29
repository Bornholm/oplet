package runner

import (
	"context"

	"github.com/bornholm/oplet/internal/crypto"
	"github.com/bornholm/oplet/internal/store"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const tokenSize int = 32
const embeddedRunnerName string = "Oplet Embedded Runner"

func (r *Repository) GetEmbeddedRunner(ctx context.Context) (*store.Runner, error) {
	var runner store.Runner
	err := r.store.WithTx(ctx, func(ctx context.Context, db *gorm.DB) error {
		if err := db.Preload(clause.Associations).Where("name = ?", embeddedRunnerName).First(&runner).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.WithStack(err)
		}

		if !runner.CreatedAt.IsZero() {
			return nil
		}

		token, err := crypto.RandomToken(tokenSize)
		if err != nil {
			return errors.Wrap(err, "could not generate embedded runner token")
		}

		runner = store.Runner{
			Name:  embeddedRunnerName,
			Token: token,
		}

		if err := db.Create(&runner).Error; err != nil {
			return errors.WithStack(err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &runner, nil
}
