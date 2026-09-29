package logging

import (
	"context"
	"log/slog"
)

type contextKey string

const (
	contextKeyRequestID contextKey = "request_id"
	contextKeyUserID    contextKey = "user_id"
	contextKeyChatID    contextKey = "chat_id"
)

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, contextKeyRequestID, requestID)
}

func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, contextKeyUserID, userID)
}

func WithChatID(ctx context.Context, chatID int64) context.Context {
	return context.WithValue(ctx, contextKeyChatID, chatID)
}

func FromContext(ctx context.Context) []any {
	var attrs []any

	if requestID, ok := ctx.Value(contextKeyRequestID).(string); ok {
		attrs = append(attrs, "request_id", requestID)
	}

	if userID, ok := ctx.Value(contextKeyUserID).(int64); ok {
		attrs = append(attrs, "user_id", userID)
	}

	if chatID, ok := ctx.Value(contextKeyChatID).(int64); ok {
		attrs = append(attrs, "chat_id", chatID)
	}

	return attrs
}

func InfoContext(ctx context.Context, logger *slog.Logger, message string, args ...any) {
	allArgs := append(FromContext(ctx), args...)
	logger.InfoContext(ctx, message, allArgs...)
}

func ErrorContext(ctx context.Context, logger *slog.Logger, message string, args ...any) {
	allArgs := append(FromContext(ctx), args...)
	logger.ErrorContext(ctx, message, allArgs...)
}
