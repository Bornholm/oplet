package setup

import (
	"context"
	"net/http"

	"github.com/bornholm/oplet/internal/config"
	httpCtx "github.com/bornholm/oplet/internal/http/context"
	"github.com/bornholm/oplet/internal/http/handler/authn"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/store/repository/user"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	anonymousUser = "anonymous"
	wildcard      = "*"
)

func getAuthzMiddlewareFromConfig(ctx context.Context, conf *config.Config) (func(http.Handler) http.Handler, error) {
	whitelist := conf.HTTP.Authn.Whitelist
	defaultRole := conf.HTTP.Authn.DefaultRole
	roleMappings := conf.HTTP.Authn.RoleMappings

	st, err := getStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	hasRoleMappings := len(roleMappings) > 0
	findUserRole := func(email string) string {
		if !hasRoleMappings {
			return defaultRole
		}

		role, exists := roleMappings[email]
		if exists {
			return role
		}

		return defaultRole
	}

	indexedWhitelist := make(map[string]struct{}, len(whitelist))
	for _, e := range whitelist {
		indexedWhitelist[e] = struct{}{}
	}

	hasWhitelist := len(whitelist) > 0

	inWhitelist := func(email string) bool {
		if !hasWhitelist {
			return true
		}

		_, exists := indexedWhitelist[email]
		if exists {
			return true
		}

		return false
	}

	userRepo := user.NewRepository(st)

	return func(next http.Handler) http.Handler {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authnUser := authn.ContextUser(ctx)

			if authnUser == nil || !inWhitelist(authnUser.Email) {
				user := store.NewUser(authnUser.Provider, authnUser.Subject, authnUser.DisplayName, anonymousUser)
				ctx = httpCtx.SetUser(ctx, user)
				r = r.WithContext(ctx)
				next.ServeHTTP(w, r)
				return
			}

			storedUser, err := userRepo.GetBySubject(ctx, authnUser.Provider, authnUser.Subject)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				common.HandleError(w, r, errors.WithStack(err))
				return
			}

			if storedUser == nil {
				storedUser = &store.User{
					DisplayName: authnUser.DisplayName,
					Subject:     authnUser.Subject,
					Provider:    authnUser.Provider,
				}

				if err := userRepo.Create(ctx, storedUser); err != nil {
					common.HandleError(w, r, errors.WithStack(err))
					return
				}
			}

			if storedUser.DisplayName != authnUser.DisplayName {
				storedUser.DisplayName = authnUser.DisplayName
				if err := userRepo.Update(ctx, storedUser); err != nil {
					common.HandleError(w, r, errors.WithStack(err))
					return
				}
			}

			role := findUserRole(authnUser.Email)
			storedUser.Roles = []string{role}

			ctx = httpCtx.SetUser(ctx, storedUser)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})

		return handler
	}, nil
}
