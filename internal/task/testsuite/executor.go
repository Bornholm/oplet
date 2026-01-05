package testsuite

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bornholm/oplet/internal/task"
	"github.com/pkg/errors"
)

type executorTestCase struct {
	Name string
	Run  func(t *testing.T, executor task.Executor) error
}

var executorTestCases = []executorTestCase{
	{
		Name: "Capture logs",
		Run:  testCaptureLogs,
	},
}

func RunExecutorTestSuite(t *testing.T, executor task.Executor) {
	for _, tc := range executorTestCases {
		t.Run(tc.Name, func(t *testing.T) {
			if err := tc.Run(t, executor); err != nil {
				t.Fatalf("%+v", errors.WithStack(err))
			}
		})
	}
}

func testCaptureLogs(t *testing.T, executor task.Executor) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var (
		wg           sync.WaitGroup
		executionErr error
	)

	wg.Add(1)

	err := executor.Execute(ctx, task.ExecutionRequest{
		ImageRef: "docker.io/bornholm/oplet-hello-world-task:latest",
		Environment: map[string]string{
			"text_env": "foo",
		},
		OnChange: func(e task.Execution) {
			t.Logf("container state: %d", e.State)
			switch e.State {
			case task.ExecutionStateContainerStarted:
				logs, err := executor.GetLogs(ctx, e.ContainerID)
				if err != nil {
					executionErr = errors.WithStack(err)
					return
				}

				go func() {
					for e := range logs {
						t.Logf("[container] #%d %s", e.Clock, e.Message)
					}
				}()
			case task.ExecutionStateKilled:
				fallthrough
			case task.ExecutionStateSucceeded:
				fallthrough
			case task.ExecutionStateFailed:
				wg.Done()
				return
			}
		},
	})
	if err != nil {
		return errors.WithStack(err)
	}

	wg.Wait()

	if executionErr != nil {
		return errors.WithStack(executionErr)
	}

	return nil
}
