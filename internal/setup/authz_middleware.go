package setup

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bornholm/oplet/internal/config"
	"github.com/bornholm/oplet/internal/http/authz"
	httpCtx "github.com/bornholm/oplet/internal/http/context"
	"github.com/bornholm/oplet/internal/http/handler/authn"
	"github.com/bornholm/oplet/internal/http/handler/webui/common"
	"github.com/bornholm/oplet/internal/store"
	"github.com/bornholm/oplet/internal/store/repository/user"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func getAuthzMiddlewareFromConfig(ctx context.Context, conf *config.Config) (func(http.Handler) http.Handler, error) {
	inactiveByDefault := conf.HTTP.Authn.InactiveByDefault
	defaultAdminEmail := conf.HTTP.Authn.DefaultAdminEmail

	st, err := getStoreFromConfig(ctx, conf)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	userRepo := user.NewRepository(st)

	return func(next http.Handler) http.Handler {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			authnUser := authn.ContextUser(ctx)

			if authnUser == nil {
				common.HandleError(w, r, common.NewError(http.StatusText(http.StatusUnauthorized), "You are not authenticated. Please login.", http.StatusUnauthorized))
				return
			}

			storedUser, err := userRepo.GetBySubject(ctx, authnUser.Provider, authnUser.Subject)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				common.HandleError(w, r, errors.WithStack(err))
				return
			}

			if storedUser == nil {
				role := authz.RoleUser
				if authnUser.Email == defaultAdminEmail {
					role = authz.RoleAdmin
				}

				storedUser = &store.User{
					DisplayName: authnUser.DisplayName,
					Subject:     authnUser.Subject,
					Provider:    authnUser.Provider,
					Email:       authnUser.Email,
					Role:        role,
					IsActive:    true,
				}

				if inactiveByDefault {
					storedUser.IsActive = false
				}

				if err := userRepo.Create(ctx, storedUser); err != nil {
					common.HandleError(w, r, errors.WithStack(err))
					return
				}

				slog.InfoContext(ctx, "Created new user with role",
					slog.String("email", storedUser.Email),
					slog.String("assigned_role", storedUser.Role),
					slog.String("user_subject", storedUser.Subject),
					slog.Bool("is_active", storedUser.IsActive))
			}

			// Update stored user if changed
			needsUpdate := false

			if authnUser.Email == defaultAdminEmail {
				storedUser.Role = authz.RoleAdmin
				storedUser.IsActive = true
				needsUpdate = true
			}

			// Check if user is active
			if !storedUser.IsActive {
				slog.WarnContext(ctx, "Inactive user attempted access",
					slog.String("email", authnUser.Email),
					slog.String("user_subject", authnUser.Subject))
				common.HandleError(w, r, common.NewError(http.StatusText(http.StatusForbidden), "Your account is not activated. Please contact an administrator.", http.StatusForbidden))
				return
			}

			if storedUser.Email != authnUser.Email {
				storedUser.Email = authnUser.Email
				needsUpdate = true
			}

			if storedUser.DisplayName != authnUser.DisplayName {
				storedUser.DisplayName = authnUser.DisplayName
				needsUpdate = true
			}

			if needsUpdate {
				if err := userRepo.Update(ctx, storedUser); err != nil {
					common.HandleError(w, r, errors.WithStack(err))
					return
				}
			}

			ctx = httpCtx.SetUser(ctx, storedUser)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})

		return handler
	}, nil
}
