package setup

import (
	"context"
	"log/slog"
	"os"

	"github.com/bornholm/oplet/internal/config"
	"github.com/bornholm/oplet/internal/file"
	"github.com/pkg/errors"
)

var getFileStorageFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (*file.Storage, error) {
	if err := ensureDirectory(conf.Storage.File.Dir); err != nil {
		return nil, errors.WithStack(err)
	}

	storage := file.NewStorage(conf.Storage.File.Dir, slog.Default())

	return storage, nil
})

func ensureDirectory(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0750); err != nil {
		return errors.Wrapf(err, "could not ensure directory '%s'", dirPath)
	}

	return nil
}
