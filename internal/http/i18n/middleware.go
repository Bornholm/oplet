package i18n

import (
	"log/slog"
	"net/http"

	"github.com/bornholm/oplet/internal/http/context"
	"github.com/bornholm/oplet/internal/slogx"
	"github.com/invopop/ctxi18n"
)

func Middleware(defaultLang string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lang := defaultLang

			ctx := r.Context()

			if user := context.User(ctx); user != nil && user.PreferredLanguage != "" {
				lang = user.PreferredLanguage
			} else if acceptLanguage := r.Header.Get("Accept-Language"); acceptLanguage != "" {
				lang = acceptLanguage
			}

			ctx, err := ctxi18n.WithLocale(ctx, lang)
			if err != nil {
				slog.WarnContext(ctx, "could not set locale", slogx.Error(err))
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})

		return http.HandlerFunc(fn)
	}
}
