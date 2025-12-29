package runner

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/bornholm/oplet/internal/task"
	"github.com/bornholm/oplet/internal/task/docker"
	"github.com/pkg/errors"
)

type Options struct {
	HTTPClient        *http.Client
	Executor          task.Executor
	Logger            *slog.Logger
	ExecutionInterval time.Duration
}

type OptionFunc func(opts *Options) error

func NewOptions(funcs ...OptionFunc) (*Options, error) {
	dockerExecutor, err := docker.NewExecutor(slog.Default())
	if err != nil {
		return nil, errors.Wrap(err, "could not create default docker executor")
	}

	opts := &Options{
		HTTPClient:        http.DefaultClient,
		Executor:          dockerExecutor,
		Logger:            slog.Default(),
		ExecutionInterval: time.Second * 5,
	}

	for _, fn := range funcs {
		if err := fn(opts); err != nil {
			return nil, errors.WithStack(err)
		}
	}

	return opts, nil
}
