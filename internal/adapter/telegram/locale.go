package telegram

import (
	"context"
	"log/slog"

	"finbot/internal/domain"
	"finbot/internal/text"
)

type localeCtxKey struct{}
type userIDCtxKey struct{}
type commandCtxKey struct{}

func withUser(ctx context.Context, user domain.User) context.Context {
	ctx = context.WithValue(ctx, userIDCtxKey{}, int64(user.TelegramID))
	ctx = context.WithValue(ctx, localeCtxKey{}, user.Locale)
	return ctx
}

func withCommand(ctx context.Context, command string) context.Context {
	return context.WithValue(ctx, commandCtxKey{}, command)
}

func localeFrom(ctx context.Context) string {
	loc, ok := ctx.Value(localeCtxKey{}).(string)
	if !ok {
		return ""
	}
	return loc
}

func userIDFrom(ctx context.Context) int64 {
	id, ok := ctx.Value(userIDCtxKey{}).(int64)
	if !ok {
		return 0
	}
	return id
}

func commandFrom(ctx context.Context) string {
	cmd, ok := ctx.Value(commandCtxKey{}).(string)
	if !ok {
		return ""
	}
	return cmd
}

func copyFrom(ctx context.Context) text.Catalog {
	return text.For(localeFrom(ctx))
}

func logAttrs(ctx context.Context, extra ...any) []any {
	attrs := []any{slog.Int64("user_id", userIDFrom(ctx))}
	if cmd := commandFrom(ctx); cmd != "" {
		attrs = append(attrs, slog.String("command", cmd))
	}
	return append(attrs, extra...)
}

func logHandlerErr(userID domain.UserID, command, msg string, err error) {
	slog.Error(
		msg,
		slog.Int64("user_id", int64(userID)),
		slog.String("command", command),
		slog.Any("err", err),
	)
}
