package setup

import (
	"context"
	"log/slog"
	"path/filepath"

	"github.com/bornholm/oplet/internal/config"
	"github.com/bornholm/oplet/internal/store"
	"github.com/pkg/errors"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var getStoreFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (*store.Store, error) {
	if err := ensureBaseDirectory(conf.Storage.Database.DSN); err != nil {
		return nil, errors.WithStack(err)
	}

	dialector := sqlite.Open(conf.Storage.Database.DSN)

	var logLevel logger.LogLevel
	switch slog.Level(conf.Logger.Level) {
	case slog.LevelError:
		logLevel = logger.Error
	case slog.LevelWarn:
		logLevel = logger.Warn
	case slog.LevelInfo:
		logLevel = logger.Info
	default:
		logLevel = logger.Error
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if slog.Level(conf.Logger.Level) == slog.LevelDebug {
		db = db.Debug()
	}

	internalDB, err := db.DB()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	internalDB.SetMaxOpenConns(1)

	if err := db.Exec("PRAGMA journal_mode=wal; PRAGMA foreign_keys=on; PRAGMA busy_timeout=30000").Error; err != nil {
		return nil, errors.WithStack(err)
	}

	return store.New(db), nil
})

func ensureBaseDirectory(filePath string) error {
	baseDir := filepath.Dir(filePath)
	if err := ensureDirectory(baseDir); err != nil {
		return errors.WithStack(err)
	}

	return nil
}
