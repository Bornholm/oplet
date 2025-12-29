package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/bornholm/oplet/internal/config"
	"github.com/bornholm/oplet/internal/setup"
	"github.com/bornholm/oplet/internal/slogx"
	"github.com/pkg/errors"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conf, err := config.Parse()
	if err != nil {
		slog.ErrorContext(ctx, "could not parse config", slog.Any("error", errors.WithStack(err)))
		os.Exit(1)
	}

	logger := slog.New(slogx.ContextHandler{
		Handler: slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     slog.Level(conf.Logger.Level),
			AddSource: true,
		}),
	})

	slog.SetDefault(logger)

	slog.DebugContext(ctx, "using configuration", slog.Any("config", conf))

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	go func() {
		slog.InfoContext(ctx, "use ctrl+c to interrupt")
		<-sig
		cancel()
	}()

	if err := setup.SeedFromConfig(ctx, conf); err != nil {
		slog.ErrorContext(ctx, "could not seed store", slogx.Error(errors.WithStack(err)))
		os.Exit(1)
	}

	if conf.Runner.Enabled {
		if err := setup.StartEmbeddedRunner(ctx, conf); err != nil {
			slog.ErrorContext(ctx, "could start embedded runner", slogx.Error(errors.WithStack(err)))
			os.Exit(1)
		}
	}

	server, err := setup.NewHTTPServerFromConfig(ctx, conf)
	if err != nil {
		slog.ErrorContext(ctx, "could not setup http server", slogx.Error(errors.WithStack(err)))
		os.Exit(1)
	}

	slog.InfoContext(ctx, "starting server", slog.String("address", conf.HTTP.Address))

	if err := server.Run(ctx); err != nil {
		slog.Error("could not run server", slogx.Error(errors.WithStack(err)))
		os.Exit(1)
	}
}
