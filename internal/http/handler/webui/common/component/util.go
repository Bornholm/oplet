package component

import (
	"context"
	"net/url"
	"strconv"

	"github.com/a-h/templ"
	"github.com/bornholm/oplet/internal/http/authz"
	httpCtx "github.com/bornholm/oplet/internal/http/context"
	httpURL "github.com/bornholm/oplet/internal/http/url"
	"github.com/pkg/errors"
)

var (
	WithPath        = httpURL.WithPath
	WithPathf       = httpURL.WithPathf
	WithoutValues   = httpURL.WithoutValues
	WithValuesReset = httpURL.WithValuesReset
	WithValues      = httpURL.WithValues
)

func WithUser(username string, password string) httpURL.MutationFunc {
	return func(u *url.URL) {
		u.User = url.UserPassword(username, password)
	}
}

func BaseURL(ctx context.Context, funcs ...httpURL.MutationFunc) templ.SafeURL {
	baseURL := httpCtx.BaseURL(ctx)
	mutated := httpURL.Mutate(baseURL, funcs...)
	return templ.SafeURL(mutated.String())
}

func CurrentURL(ctx context.Context, funcs ...httpURL.MutationFunc) templ.SafeURL {
	currentURL := clone(httpCtx.CurrentURL(ctx))
	mutated := httpURL.Mutate(currentURL, funcs...)
	return templ.SafeURL(mutated.String())
}

func MatchPath(ctx context.Context, path string) bool {
	currentURL := httpCtx.CurrentURL(ctx)
	return currentURL.Path == path
}

func clone[T any](v *T) *T {
	copy := *v
	return &copy
}

func AssertUser(ctx context.Context, funcs ...authz.AssertFunc) bool {
	user := httpCtx.User(ctx)
	if user == nil {
		return false
	}

	allowed, err := authz.Assert(ctx, user, funcs...)
	if err != nil {
		panic(errors.WithStack(err))
	}

	return allowed
}

var User = httpCtx.User

func FormatID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
