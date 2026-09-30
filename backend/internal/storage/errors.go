package storage

import "errors"

// ErrApplicationNotFound возвращается, когда заявка с указанным идентификатором
// не найдена в хранилище. Используется в UpdateApplicationStatus через errors.Is.
var ErrApplicationNotFound = errors.New("application not found")

// ErrWaitlistEntryNotFound зарезервирована для будущих методов работы с очередью.
// В текущей реализации не используется: RemoveWaitlistEntry идемпотентен
// (возвращает nil при отсутствии записи), а ListWaitlistEntriesBySectionID
// возвращает пустой список вместо ошибки, если секции нет в очереди.
var ErrWaitlistEntryNotFound = errors.New("waitlist entry not found")
