package storage

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"section-near-home/backend/internal/domain"
	"section-near-home/backend/internal/logging"
)

// Compile-time проверка: тип InMemoryRepository удовлетворяет интерфейсу Repository.
// Если контракт изменится, сборка упадёт именно здесь, а не в точке вызова.
var _ Repository = (*InMemoryRepository)(nil)

// InMemoryRepository реализует Repository на map-ах с защитой sync.RWMutex.
// Хранилище живёт только в памяти процесса: при перезапуске контейнера
// все заявки и записи очереди теряются. Это осознанное ограничение MVP,
// зафиксированное в документации.
type InMemoryRepository struct {
	mu sync.RWMutex

	applications       map[int]domain.Application
	applicationsByUser map[int64][]int
	nextApplicationID  int

	waitlistEntries          map[int]domain.WaitlistEntry
	waitlistEntriesBySection map[int][]int
	nextWaitlistEntryID      int

	logger *slog.Logger
}

// NewInMemoryRepository создаёт пустое in-memory хранилище.
func NewInMemoryRepository(logger *slog.Logger) *InMemoryRepository {
	return &InMemoryRepository{
		applications:             make(map[int]domain.Application),
		applicationsByUser:       make(map[int64][]int),
		nextApplicationID:        1,
		waitlistEntries:          make(map[int]domain.WaitlistEntry),
		waitlistEntriesBySection: make(map[int][]int),
		nextWaitlistEntryID:      1,
		logger:                   logging.WithComponent(logger, "inmemory_repository"),
	}
}

// SaveApplication сохраняет заявку и возвращает присвоенный идентификатор.
// ID в переданной структуре игнорируется: назначение выполняет только хранилище.
func (repository *InMemoryRepository) SaveApplication(
	ctx context.Context,
	application domain.Application,
) (int, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	assignedID := repository.nextApplicationID
	repository.nextApplicationID++

	application.ID = assignedID
	if application.CreatedAt.IsZero() {
		application.CreatedAt = time.Now()
	}
	if application.Status == "" {
		application.Status = domain.ApplicationStatusPending
	}

	repository.applications[assignedID] = application
	repository.applicationsByUser[application.UserID] = append(
		repository.applicationsByUser[application.UserID],
		assignedID,
	)

	repository.logger.InfoContext(ctx, "application saved",
		"application_id", assignedID,
		"user_id", application.UserID,
		"section_id", application.SectionID,
		"parent_phone", application.MaskedParentPhone())

	return assignedID, nil
}

// ListApplicationsByUserID возвращает заявки одного родителя в порядке создания
// (от новых к старым). Возвращается копия среза, чтобы вызывающий код не мог
// случайно изменить внутреннее состояние хранилища.
func (repository *InMemoryRepository) ListApplicationsByUserID(
	ctx context.Context,
	userID int64,
) ([]domain.Application, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	ids := repository.applicationsByUser[userID]
	result := make([]domain.Application, 0, len(ids))
	for _, id := range ids {
		if application, exists := repository.applications[id]; exists {
			result = append(result, application)
		}
	}

	// Сортировка: новые заявки в начале списка.
	sort.Slice(result, func(leftIndex, rightIndex int) bool {
		return result[leftIndex].CreatedAt.After(result[rightIndex].CreatedAt)
	})

	repository.logger.DebugContext(ctx, "applications listed by user",
		"user_id", userID,
		"count", len(result))

	return result, nil
}

// UpdateApplicationStatus изменяет статус существующей заявки.
// Возвращает ошибку, если заявка с указанным идентификатором не найдена.
func (repository *InMemoryRepository) UpdateApplicationStatus(
	ctx context.Context,
	applicationID int,
	status domain.ApplicationStatus,
) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	application, exists := repository.applications[applicationID]
	if !exists {
		return fmt.Errorf("%w: application_id=%d", ErrApplicationNotFound, applicationID)
	}

	previousStatus := application.Status
	application.Status = status
	repository.applications[applicationID] = application

	repository.logger.InfoContext(ctx, "application status updated",
		"application_id", applicationID,
		"previous_status", previousStatus,
		"new_status", status)

	return nil
}

// SaveWaitlistEntry добавляет запись в очередь на освободившееся место
// и возвращает присвоенный идентификатор.
func (repository *InMemoryRepository) SaveWaitlistEntry(
	ctx context.Context,
	entry domain.WaitlistEntry,
) (int, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	assignedID := repository.nextWaitlistEntryID
	repository.nextWaitlistEntryID++

	entry.ID = assignedID
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}

	repository.waitlistEntries[assignedID] = entry
	repository.waitlistEntriesBySection[entry.SectionID] = append(
		repository.waitlistEntriesBySection[entry.SectionID],
		assignedID,
	)

	repository.logger.InfoContext(ctx, "waitlist entry saved",
		"entry_id", assignedID,
		"user_id", entry.UserID,
		"section_id", entry.SectionID)

	return assignedID, nil
}

// ListWaitlistEntriesBySectionID возвращает очередь секции в порядке постановки.
// Первый элемент списка является следующим кандидатом на освободившееся место.
func (repository *InMemoryRepository) ListWaitlistEntriesBySectionID(
	ctx context.Context,
	sectionID int,
) ([]domain.WaitlistEntry, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	ids := repository.waitlistEntriesBySection[sectionID]
	result := make([]domain.WaitlistEntry, 0, len(ids))
	for _, id := range ids {
		if entry, exists := repository.waitlistEntries[id]; exists {
			result = append(result, entry)
		}
	}

	// Сортировка: порядок постановки в очередь (по CreatedAt, от старых к новым).
	sort.Slice(result, func(leftIndex, rightIndex int) bool {
		return result[leftIndex].CreatedAt.Before(result[rightIndex].CreatedAt)
	})

	repository.logger.DebugContext(ctx, "waitlist entries listed",
		"section_id", sectionID,
		"count", len(result))

	return result, nil
}

// RemoveWaitlistEntry удаляет запись из очереди. Если запись не найдена,
// метод возвращает nil: это позволяет идемпотентно обрабатывать повторные
// уведомления об освобождении места.
func (repository *InMemoryRepository) RemoveWaitlistEntry(
	ctx context.Context,
	entryID int,
) error {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	entry, exists := repository.waitlistEntries[entryID]
	if !exists {
		repository.logger.DebugContext(ctx, "waitlist entry already removed",
			"entry_id", entryID)
		return nil
	}

	delete(repository.waitlistEntries, entryID)

	sectionIDs := repository.waitlistEntriesBySection[entry.SectionID]
	filteredIDs := make([]int, 0, len(sectionIDs))
	for _, id := range sectionIDs {
		if id != entryID {
			filteredIDs = append(filteredIDs, id)
		}
	}
	repository.waitlistEntriesBySection[entry.SectionID] = filteredIDs

	repository.logger.InfoContext(ctx, "waitlist entry removed",
		"entry_id", entryID,
		"section_id", entry.SectionID)

	return nil
}
