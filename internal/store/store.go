package store

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/glebarez/go-sqlite"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var models = []any{
	&Task{},
	&Seed{},
	&User{},
	&TaskExecution{},
	&TaskExecutionLog{},
	&TaskExecutionFile{},
	&TaskConfiguration{},
	&Runner{},
}

type Store struct {
	getDatabase func(ctx context.Context) (*gorm.DB, error)
}

func (s *Store) WithTx(ctx context.Context, fn func(ctx context.Context, db *gorm.DB) error) error {
	db, err := s.getDatabase(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := fn(ctx, tx); err != nil {
			return errors.WithStack(err)
		}

		return nil
	})
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *Store) WithDatabase(ctx context.Context, fn func(ctx context.Context, db *gorm.DB) error) error {
	db, err := s.getDatabase(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	if err := fn(ctx, db); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *Store) WithRetry(ctx context.Context, fn func(ctx context.Context, db *gorm.DB) error, codes ...int) error {
	db, err := s.getDatabase(ctx)
	if err != nil {
		return errors.WithStack(err)
	}

	backoff := 500 * time.Millisecond
	maxRetries := 10
	retries := 0

	for {
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := fn(ctx, tx); err != nil {
				return errors.WithStack(err)
			}

			return nil
		})
		if err != nil {
			if retries >= maxRetries {
				return errors.WithStack(err)
			}

			var sqliteErr *sqlite.Error
			if errors.As(err, &sqliteErr) {
				if !slices.Contains(codes, sqliteErr.Code()) {
					return errors.WithStack(err)
				}

				slog.DebugContext(ctx, "transaction failed, will retry", slog.Int("retries", retries), slog.Duration("backoff", backoff), slog.Any("error", errors.WithStack(err)))

				retries++
				time.Sleep(backoff)
				backoff *= 2
				continue
			}

			return errors.WithStack(err)
		}

		return nil
	}
}

func (s *Store) Ping(ctx context.Context) error {
	return s.WithDatabase(ctx, func(ctx context.Context, db *gorm.DB) error {
		sqlDB, err := db.DB()
		if err != nil {
			return errors.WithStack(err)
		}
		return sqlDB.Ping()
	})
}

func New(db *gorm.DB) *Store {
	return &Store{
		getDatabase: createGetDatabase(db),
	}
}

func createGetDatabase(db *gorm.DB) func(ctx context.Context) (*gorm.DB, error) {
	var (
		migrateOnce sync.Once
		migrateErr  error
	)

	return func(ctx context.Context) (*gorm.DB, error) {
		migrateOnce.Do(func() {
			if err := db.AutoMigrate(models...); err != nil {
				migrateErr = errors.WithStack(err)
				return
			}
		})
		if migrateErr != nil {
			return nil, errors.WithStack(migrateErr)
		}

		return db, nil
	}
}
