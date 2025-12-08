package context

import (
	"context"

	"github.com/bornholm/oplet/internal/store"
)

const keyUser = "user"

func User(ctx context.Context) *store.User {
	user, ok := ctx.Value(keyUser).(*store.User)
	if !ok {
		return nil
	}

	return user
}

func SetUser(ctx context.Context, user *store.User) context.Context {
	return context.WithValue(ctx, keyUser, user)
}
