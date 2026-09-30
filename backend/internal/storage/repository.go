package storage

import (
	"context"

	"section-near-home/backend/internal/domain"
)

// Repository является единым контрактом хранилища заявок и листа ожидания.
//
// Секции в контракт не входят: их источником является справочник mock-api,
// доступ к которому выполняется через пакет clients/mockapi.
//
// Сервисы и диалоговый движок зависят только от этого интерфейса,
// согласно принципу инверсии зависимостей. Реализация на JSON-файле
// добавляется следующим шагом и подменяется без изменения вызывающего кода.
//
// Все методы возвращают ошибки, обёрнутые через fmt.Errorf с %w,
// чтобы вызывающий код мог использовать errors.Is для проверки sentinel-ошибок.
type Repository interface {
	// SaveApplication сохраняет новую заявку на запись в секцию и возвращает
	// идентификатор, присвоенный хранилищем. Идентификатор в переданной структуре
	// игнорируется: назначение выполняет только хранилище.
	// Возвращаемый идентификатор нужен вызывающему коду, чтобы сообщить родителю
	// номер заявки и позже обновить её статус.
	SaveApplication(ctx context.Context, application domain.Application) (int, error)

	// ListApplicationsByUserID возвращает все заявки одного родителя
	// для сценария «Мои заявки».
	ListApplicationsByUserID(ctx context.Context, userID int64) ([]domain.Application, error)

	// UpdateApplicationStatus изменяет статус существующей заявки по идентификатору.
	// Используется для перехода pending в confirmed, для отклонения секцией
	// и для отмены родителем через механику тихого отказа.
	// Если заявка с указанным идентификатором не найдена, возвращает ошибку,
	// обёрнутую вокруг ErrApplicationNotFound.
	UpdateApplicationStatus(ctx context.Context, applicationID int, status domain.ApplicationStatus) error

	// SaveWaitlistEntry добавляет родителя в очередь на освободившееся место
	// и возвращает идентификатор, присвоенный хранилищем.
	SaveWaitlistEntry(ctx context.Context, entry domain.WaitlistEntry) (int, error)

	// ListWaitlistEntriesBySectionID возвращает очередь секции в порядке постановки.
	// Первый элемент списка является следующим кандидатом на освободившееся место.
	ListWaitlistEntriesBySectionID(ctx context.Context, sectionID int) ([]domain.WaitlistEntry, error)

	// RemoveWaitlistEntry удаляет запись из очереди после того, как место
	// предложено и принято, либо после отказа родителя от предложения.
	// Метод идемпотентен: если запись уже удалена (повторное уведомление),
	// возвращается nil без ошибки. Это позволяет безопасно обрабатывать
	// дублирующиеся события освобождения места.
	RemoveWaitlistEntry(ctx context.Context, entryID int) error
}
