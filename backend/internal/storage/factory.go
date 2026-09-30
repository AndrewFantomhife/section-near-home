package storage

import "log/slog"

// Kind перечисляет доступные реализации хранилища.
// Тип определён в пакете storage, так как именно здесь находится
// фабрика и все реализации интерфейса Repository.
type Kind string

const (
	// KindInMemory хранит данные в памяти процесса.
	// Используется по умолчанию для MVP и локальной разработки.
	// Данные теряются при перезапуске контейнера.
	KindInMemory Kind = "memory"
)

// NewRepository создаёт реализацию Repository выбранного типа.
// Неизвестные значения трактуются как InMemory для безопасного поведения.
func NewRepository(kind Kind, logger *slog.Logger) Repository {
	switch kind {
	case KindInMemory:
		return NewInMemoryRepository(logger)
	default:
		return NewInMemoryRepository(logger)
	}
}
