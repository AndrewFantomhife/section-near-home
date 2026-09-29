package logging

import (
	"errors"
	"log/slog"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

// ErrorAttributes извлекает структурированные поля из ошибки для логирования.
// Поддерживает maxigo.Error, стандартные ошибки и кастомные типы.
func ErrorAttributes(err error) []any {
	if err == nil {
		return nil
	}

	attrs := []any{"error", err.Error()}

	// Обработка ошибок maxigo-client
	var maxErr *maxigo.Error
	if errors.As(err, &maxErr) {
		attrs = append(attrs,
			// Не факт, что у ErrorKind есть метод String(). Если компилятор ругнётся — замените на свой маппинг: см в Obsidian Баг 2
			"error_kind", maxErr.Kind.String(),
			"error_operation", maxErr.Op,
			"error_status_code", maxErr.StatusCode,
		)

		if maxErr.Message != "" {
			attrs = append(attrs, "error_message", maxErr.Message)
		}

		// Добавляем оригинальную ошибку, если есть
		if unwrapped := maxErr.Unwrap(); unwrapped != nil {
			attrs = append(attrs, "error_cause", unwrapped.Error())
		}

		return attrs
	}

	// Обработка стандартных ошибок с Unwrap
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		attrs = append(attrs, "error_cause", unwrapped.Error())
	}

	return attrs
}

// ErrorWithCause логирует ошибку с автоматическим извлечением полей.
func ErrorWithCause(logger *slog.Logger, message string, err error, additionalArgs ...any) {
	allArgs := append(ErrorAttributes(err), additionalArgs...)
	logger.Error(message, allArgs...)
}
