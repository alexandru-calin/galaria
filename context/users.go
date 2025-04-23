package context

import (
	"context"

	"github.com/alexandru-calin/galaria/models"
)

type key string

const (
	userKey  key = "user"
	themeKey key = "theme"
)

func WithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

func User(ctx context.Context) *models.User {
	val := ctx.Value(userKey)

	user, ok := val.(*models.User)
	if !ok {
		return nil
	}

	return user
}

func WithTheme(ctx context.Context, theme string) context.Context {
	return context.WithValue(ctx, themeKey, theme)
}

func Theme(ctx context.Context) string {
	val := ctx.Value(themeKey)

	theme, ok := val.(string)
	if !ok {
		return ""
	}

	return theme
}
