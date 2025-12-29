package runner

import (
	"context"

	"github.com/bornholm/oplet/internal/store"
	"github.com/pkg/errors"
)

type contextKey string

const (
	contextKeyRunner contextKey = "runner"
)

func withContextRunner(ctx context.Context, runner *store.Runner) context.Context {
	return context.WithValue(ctx, contextKeyRunner, runner)
}

func contextRunner(ctx context.Context) (*store.Runner, error) {
	rawRunner := ctx.Value(contextKeyRunner)
	if rawRunner == nil {
		return nil, errors.New("could not find runner in context")
	}

	runner, ok := rawRunner.(*store.Runner)
	if !ok {
		return nil, errors.Errorf("unexpected runner type '%T'", rawRunner)
	}

	return runner, nil
}
