package setup

import (
	"context"
	"log/slog"

	"github.com/bornholm/oplet/internal/config"
	"github.com/bornholm/oplet/internal/http"
	"github.com/bornholm/oplet/internal/http/handler/metrics"
	"github.com/bornholm/oplet/internal/http/handler/runner"
	"github.com/bornholm/oplet/internal/http/handler/webui"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	"github.com/bornholm/oplet/internal/http/i18n"
	"github.com/bornholm/oplet/internal/http/pprof"
	"github.com/pkg/errors"
)

func NewHTTPServerFromConfig(ctx context.Context, conf *config.Config) (*http.Server, error) {
	authn, err := getAuthnHandlerFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not configure authn handler from config")
	}

	authnMiddleware := authn.Middleware()
	i18nMiddleware := i18n.Middleware(conf.I18n.DefaultLanguage)
	authzMiddleware, err := getAuthzMiddlewareFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not configure authz handler from config")
	}

	assets := common.NewHandler()

	options := []http.OptionFunc{
		http.WithAddress(conf.HTTP.Address),
		http.WithBaseURL(conf.HTTP.BaseURL),
		http.WithMount("/assets/", assets),
		http.WithMount("/auth/", i18nMiddleware(authn)),
		http.WithMount("/metrics/", authnMiddleware(authzMiddleware(metrics.NewHandler()))),
	}

	store, err := getStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not configure store from config")
	}

	taskProvider, err := getTaskProviderFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not configure task provider")
	}

	taskExecutor, err := getTaskExecutorFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not configure task executor")
	}

	fileStorage, err := getFileStorageFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.Wrap(err, "could not configure task executor")
	}

	runner := runner.NewHandler(store, taskProvider, fileStorage, slog.Default())
	options = append(options, http.WithMount("/runner/", runner))

	webui := webui.NewHandler(store, taskProvider, taskExecutor, fileStorage, slog.Default())
	options = append(options, http.WithMount("/", i18nMiddleware(authnMiddleware(authzMiddleware(i18nMiddleware(webui))))))

	options = append(options, http.WithMount("/pprof/", authnMiddleware(pprof.NewHandler())))

	// Create HTTP server

	server := http.NewServer(options...)

	return server, nil
}
