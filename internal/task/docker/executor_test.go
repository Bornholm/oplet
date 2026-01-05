package docker

import (
	"testing"

	"github.com/bornholm/oplet/internal/slogx"
	"github.com/bornholm/oplet/internal/task/testsuite"
	"github.com/pkg/errors"
)

func TestExecutor(t *testing.T) {
	logger := slogx.NewTestLogger(t)
	executor, err := NewExecutor(logger)
	if err != nil {
		t.Fatalf("%+v", errors.WithStack(err))
	}

	testsuite.RunExecutorTestSuite(t, executor)
}
