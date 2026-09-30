package dialog

import (
	"context"
	"log/slog"
	"strconv"
	"strings"

	"section-near-home/backend/internal/clients/maxapi"
	"section-near-home/backend/internal/clients/mockapi"
	"section-near-home/backend/internal/logging"
)

// Префиксы payload инлайн-кнопок. Формат payload: "<префикс><значение>".
const (
	payloadPrefixAction   = "action:"
	payloadPrefixDistrict = "district:"
	payloadPrefixAge      = "age:"
	payloadPrefixSport    = "sport:"
)

// Скалярные значения payload, используемые в маршрутизации.
const (
	payloadActionStart   = payloadPrefixAction + "start"
	payloadActionRestart = payloadPrefixAction + "restart"
)

// Команды бота. MAX использует двоеточие как разделитель команды и payload.
const (
	commandStart = "/start"
	commandHelp  = "/help"
	commandReset = "/reset"
)

// Machine является диалоговым автоматом бота.
// Он не содержит бизнес-логику шагов: его задача - маршрутизация событий
// к обработчикам из onboarding.go и results.go.
type Machine struct {
	maxClient   *maxapi.Client
	mockClient  *mockapi.Client
	store       *Store
	logger      *slog.Logger
	botUsername string
}

// NewMachine создаёт диалоговый автомат с явным объявлением зависимостей.
func NewMachine(
	maxClient *maxapi.Client,
	mockClient *mockapi.Client,
	store *Store,
	botUsername string,
	logger *slog.Logger,
) *Machine {
	return &Machine{
		maxClient:   maxClient,
		mockClient:  mockClient,
		store:       store,
		botUsername: botUsername,
		logger:      logging.WithComponent(logger, "dialog"),
	}
}

// Handle обрабатывает входящее событие от MAX и маршрутизирует его
// к соответствующему обработчику в зависимости от типа события.
func (machine *Machine) Handle(ctx context.Context, update maxapi.Update) error {
	switch event := update.Payload.(type) {
	case maxapi.MessageReceived:
		return machine.handleMessage(ctx, event)
	case maxapi.ButtonPressed:
		return machine.handleButton(ctx, event)
	case maxapi.BotStarted:
		return machine.handleBotStarted(ctx, event)
	case maxapi.BotStopped:
		return machine.handleBotStopped(ctx, event)
	default:
		machine.logger.Debug("unsupported update type ignored",
			"update_type", string(update.Type))
		return nil
	}
}

// handleMessage обрабатывает входящие текстовые сообщения пользователя.
// Порядок проверок: пустой текст, затем команды, затем switch по состоянию.
func (machine *Machine) handleMessage(ctx context.Context, event maxapi.MessageReceived) error {
	ctx = logging.WithUserID(ctx, event.UserID)
	ctx = logging.WithChatID(ctx, event.ChatID)

	text := strings.TrimSpace(event.Text)
	if text == "" {
		machine.logger.InfoContext(ctx, "non-text attachment received")
		return machine.maxClient.SendMessage(ctx, event.UserID, messageNonTextOnly)
	}
	machine.logger.InfoContext(ctx, "text message received", "text", text)

	if strings.HasPrefix(text, "/") {
		return machine.handleCommand(ctx, event.UserID, text)
	}

	state := machine.store.GetOrCreate(event.UserID)

	if state.IsSearching() {
		return machine.maxClient.SendMessage(ctx, event.UserID, messageSearchInProgress)
	}

	switch state.CurrentState {
	case StateIdle:
		return machine.startOnboarding(ctx, event.UserID)
		// ... остальные ветки без строки StateShowResults ...
	}

	switch state.CurrentState {
	case StateIdle:
		return machine.startOnboarding(ctx, event.UserID)
	case StateAskDistrict:
		return machine.maxClient.SendMessage(ctx, event.UserID, messageDistrictButtonsOnly)
	case StateAskChildAge:
		return machine.maxClient.SendMessage(ctx, event.UserID, messageAgeButtonsOnly)
	case StateAskSport:
		return machine.maxClient.SendMessage(ctx, event.UserID, messageSportButtonsOnly)
	case StateShowResults:
		return machine.maxClient.SendMessage(ctx, event.UserID, messageSearchInProgress)
	case StateCompleted:
		machine.store.Reset(event.UserID)
		return machine.startOnboarding(ctx, event.UserID)
	default:
		machine.store.Reset(event.UserID)
		return machine.startOnboarding(ctx, event.UserID)
	}
}

// handleButton обрабатывает нажатия инлайн-кнопок по значению payload.
// Подтверждение нажатия выполняется внутри ветвей, чтобы иметь возможность
// передать текстовую подсказку при невалидных данных.
func (machine *Machine) handleButton(ctx context.Context, event maxapi.ButtonPressed) error {
	ctx = logging.WithUserID(ctx, event.UserID)
	ctx = logging.WithChatID(ctx, event.ChatID)
	machine.logger.InfoContext(ctx, "button pressed", "payload", event.Payload)

	state := machine.store.GetOrCreate(event.UserID)
	if state.IsSearching() {
		machine.answerCallbackSilently(ctx, event.CallbackID)
		machine.logger.DebugContext(ctx, "button ignored during search",
			"payload", event.Payload)
		return nil
	}

	switch {
	case event.Payload == payloadActionStart || event.Payload == payloadActionRestart:
		machine.answerCallbackSilently(ctx, event.CallbackID)
		return machine.startOnboarding(ctx, event.UserID)

	case strings.HasPrefix(event.Payload, payloadPrefixDistrict):
		machine.answerCallbackSilently(ctx, event.CallbackID)
		district := strings.TrimPrefix(event.Payload, payloadPrefixDistrict)
		return machine.processDistrict(ctx, event.UserID, district)

	case strings.HasPrefix(event.Payload, payloadPrefixAge):
		rawAge := strings.TrimPrefix(event.Payload, payloadPrefixAge)
		age, err := strconv.Atoi(rawAge)
		if err != nil || !isValidChildAge(age) {
			machine.logger.WarnContext(ctx, "invalid age payload", "payload", event.Payload)
			machine.answerCallbackWithNotification(ctx, event.CallbackID, messageInvalidAgeCallback)
			return nil
		}
		machine.answerCallbackSilently(ctx, event.CallbackID)
		return machine.processChildAge(ctx, event.UserID, age)

	case strings.HasPrefix(event.Payload, payloadPrefixSport):
		machine.answerCallbackSilently(ctx, event.CallbackID)
		sport := strings.TrimPrefix(event.Payload, payloadPrefixSport)
		return machine.processSport(ctx, event.UserID, sport)

	default:
		machine.logger.WarnContext(ctx, "unknown button payload", "payload", event.Payload)
		machine.answerCallbackWithNotification(ctx, event.CallbackID, messageUnknownAction)
		return nil
	}
}

// handleCommand обрабатывает slash-команды с учётом разделителя MAX.
func (machine *Machine) handleCommand(ctx context.Context, userID int64, command string) error {
	switch normalizeCommand(command) {
	case commandStart:
		machine.store.Reset(userID)
		return machine.startOnboarding(ctx, userID)
	case commandHelp:
		return machine.maxClient.SendMessage(ctx, userID, messageHelp)
	case commandReset:
		machine.store.Reset(userID)
		return machine.maxClient.SendMessage(ctx, userID, messageResetDone)
	default:
		return machine.maxClient.SendMessage(ctx, userID, messageUnknownCommand)
	}
}

// handleBotStarted обрабатывает событие запуска бота пользователем.
func (machine *Machine) handleBotStarted(ctx context.Context, event maxapi.BotStarted) error {
	ctx = logging.WithUserID(ctx, event.UserID)
	machine.logger.InfoContext(ctx, "bot started by user")
	return machine.startOnboarding(ctx, event.UserID)
}

// handleBotStopped обрабатывает событие остановки бота пользователем.
func (machine *Machine) handleBotStopped(ctx context.Context, event maxapi.BotStopped) error {
	ctx = logging.WithUserID(ctx, event.UserID)
	machine.logger.InfoContext(ctx, "bot stopped by user")
	machine.store.Delete(event.UserID)
	return nil
}

// answerCallbackSilently подтверждает нажатие кнопки без текстового уведомления.
func (machine *Machine) answerCallbackSilently(ctx context.Context, callbackID string) {
	if err := machine.maxClient.AnswerCallback(ctx, callbackID, ""); err != nil {
		machine.logger.WarnContext(ctx, "answer callback failed", "error", err)
	}
}

// answerCallbackWithNotification подтверждает нажатие кнопки текстовой подсказкой.
func (machine *Machine) answerCallbackWithNotification(ctx context.Context, callbackID string, notification string) {
	if err := machine.maxClient.AnswerCallback(ctx, callbackID, notification); err != nil {
		machine.logger.WarnContext(ctx, "answer callback failed", "error", err)
	}
}

// normalizeCommand приводит команду к каноническому виду.
// MAX передаёт команды в формате /start:payload, поэтому всё после
// двоеточия отбрасывается, а регистр приводится к нижнему.
func normalizeCommand(command string) string {
	normalized := strings.ToLower(strings.TrimSpace(command))
	if separatorIndex := strings.Index(normalized, ":"); separatorIndex != -1 {
		normalized = normalized[:separatorIndex]
	}
	return normalized
}
