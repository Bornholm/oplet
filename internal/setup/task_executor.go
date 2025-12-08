package setup

import (
	"context"
	"log/slog"

	"github.com/bornholm/oplet/internal/config"
	"github.com/bornholm/oplet/internal/task"
	"github.com/bornholm/oplet/internal/task/docker"
	"github.com/pkg/errors"
)

var getTaskExecutorFromConfig = createFromConfigOnce(func(ctx context.Context, conf *config.Config) (task.Executor, error) {
	executor, err := docker.NewExecutor(slog.Default())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return executor, nil
})
