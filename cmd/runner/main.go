package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"

	"github.com/bornholm/oplet/internal/runner"
	"github.com/bornholm/oplet/internal/slogx"
	"github.com/pkg/errors"
)

var (
	rawLogLevel string = slog.LevelInfo.String()
	authToken   string = ""
	serverURL   string = ""
)

func init() {
	flag.StringVar(&rawLogLevel, "log-level", rawLogLevel, "logging level")
	flag.StringVar(&serverURL, "server-url", serverURL, "server url")
	flag.StringVar(&authToken, "auth-token", authToken, "auth token")
}

func main() {
	flag.Parse()

	if authToken == "" {
		authToken = os.Getenv("OPLET_RUNNER_AUTH_TOKEN")
	}

	if serverURL == "" {
		serverURL = os.Getenv("OPLET_RUNNER_SERVER_URL")
	}

	if serverURL == "" {
		serverURL = "http://localhost:3002"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(rawLogLevel)); err != nil {
		slog.ErrorContext(ctx, "could not parse log level", slogx.Error(errors.WithStack(err)))
		os.Exit(1)
	}

	logger := slog.New(slogx.ContextHandler{
		Handler: slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.Level(logLevel),
			AddSource: true,
		}),
	})

	slog.SetDefault(logger)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	go func() {
		slog.InfoContext(ctx, "use ctrl+c to interrupt")
		<-sig
		cancel()
	}()

	runner, err := runner.New(serverURL, authToken)
	if err != nil {
		slog.ErrorContext(ctx, "could not create runner", slogx.Error(errors.WithStack(err)))
		os.Exit(1)
	}

	if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.ErrorContext(ctx, "could not execute runner", slogx.Error(errors.WithStack(err)))
		os.Exit(1)
	}
}
