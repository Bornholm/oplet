package setup

import (
	"context"

	"github.com/bornholm/oplet/internal/config"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/store/repository/seed"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func SeedFromConfig(ctx context.Context, conf *config.Config) error {
	if !conf.Seed.Enabled {
		return nil
	}

	st, err := getStoreFromConfig(ctx, conf)
	if err != nil {
		return errors.WithStack(err)
	}

	repo := seed.NewRepository(st)

	seeders := make([]*seed.Seeder, 0)

	if len(conf.Seed.DefaultTasks) > 0 {
		taskProvider, err := getTaskProviderFromConfig(ctx, conf)
		if err != nil {
			return errors.Wrap(err, "could not configure task provider")
		}

		seeders = append(seeders, seed.New(
			"default-tasks",
			func(ctx context.Context, db *gorm.DB) error {
				for _, t := range conf.Seed.DefaultTasks {
					taskDefinition, err := taskProvider.FetchTaskDefinition(ctx, t)
					if err != nil {
						return errors.Wrapf(err, "could not retrieve task definition for image ref '%s'", t)
					}

					task := store.Task{
						Name:        taskDefinition.Name,
						ImageRef:    t,
						Author:      taskDefinition.Author,
						Description: taskDefinition.Description,
					}

					if err := db.Create(&task).Error; err != nil {
						return errors.WithStack(err)
					}
				}

				return nil
			},
		))
	}

	if err := repo.Seed(ctx, false, seeders...); err != nil {
		return errors.Wrap(err, "could not execute store seeding")
	}

	return nil
}
