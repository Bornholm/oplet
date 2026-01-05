package setup

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/bornholm/oplet/internal/config"
	"github.com/bornholm/oplet/internal/runner"
	"github.com/bornholm/oplet/internal/slogx"
	runnerRepository "github.com/bornholm/oplet/internal/store/repository/runner"
	"github.com/pkg/errors"
)

func StartEmbeddedRunner(ctx context.Context, conf *config.Config) error {
	st, err := getStoreFromConfig(ctx, conf)
	if err != nil {
		return errors.WithStack(err)
	}

	repo := runnerRepository.NewRepository(st)

	embeddedRunner, err := repo.GetEmbeddedRunner(ctx)
	if err != nil {
		return errors.Wrap(err, "could not retrieve embedded runner")
	}

	runner, err := runner.New(conf.Runner.ServerURL, embeddedRunner.Token)
	if err != nil {
		slog.ErrorContext(ctx, "could not create runner", slogx.Error(errors.WithStack(err)))
		os.Exit(1)
	}

	go func() {
		logger := slog.Default().With("component", "embedded-runner")

		for {
			logger.InfoContext(ctx, "starting embedded runner")

			if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				logger.ErrorContext(ctx, "embedded runner failed", slogx.Error(errors.WithStack(err)))
			}

			time.Sleep(time.Second)
		}
	}()

	return nil
}
