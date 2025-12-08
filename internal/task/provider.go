package task

import (
	"context"

	"github.com/pkg/errors"
)

var (
	ErrInvalidTask = errors.New("invalid task")
)

type Provider interface {
	FetchTaskDefinition(ctx context.Context, imageRef string) (*Definition, error)
}
