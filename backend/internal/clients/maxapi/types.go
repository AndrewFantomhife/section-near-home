package maxapi

// Update является базовым типом события от MAX Bot API.
// Транспорт использует поле Type для маршрутизации через type switch.
type Update struct {
	Type    UpdateType
	Payload any
}

// UpdateType перечисляет поддерживаемые типы событий.
type UpdateType string

const (
	UpdateTypeMessageReceived UpdateType = "message_received"
	UpdateTypeButtonPressed   UpdateType = "button_pressed"
	UpdateTypeBotStarted      UpdateType = "bot_started"
	UpdateTypeBotStopped      UpdateType = "bot_stopped"
	UpdateTypeUnknown         UpdateType = "unknown"
)

// MessageReceived описывает входящее текстовое сообщение от пользователя.
type MessageReceived struct {
	UserID    int64
	ChatID    int64
	Text      string
	MessageID string
}

// ButtonPressed описывает нажатие инлайн-кнопки.
type ButtonPressed struct {
	UserID     int64
	ChatID     int64
	CallbackID string
	Payload    string
	MessageID  string
}

// BotStarted описывает событие запуска бота пользователем (команда /start).
type BotStarted struct {
	UserID int64
}

// BotStopped описывает событие остановки бота пользователем.
type BotStopped struct {
	UserID int64
}

// Command описывает команду бота, отображаемую в подсказках при вводе /.
type Command struct {
	Name        string
	Description string
}

// Button описывает инлайн-кнопку клавиатуры.
type Button struct {
	Text    string
	Payload string
	// OpenAppUsername — если задан, кнопка открывает мини-приложение бота
	// с этим username. Если пусто — обычная callback-кнопка.
	OpenAppUsername string
	LinkURL         string
}

// BotInfo содержит идентификационные данные бота.
type BotInfo struct {
	UserID   int64
	Name     string
	Username string
}

// IsTimeoutError сообщает, является ли ошибка штатным таймаутом long polling.
// Предикат экспортируется в пакете maxapi, чтобы транспорт не импортировал maxigo.
// Определена в client.go, приведена здесь для полноты контракта.
