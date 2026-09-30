package maxapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	maxigo "github.com/maxigo-bot/maxigo-client"

	"section-near-home/backend/internal/logging"
)

const (
	defaultRequestTimeout = 30 * time.Second
	updatesPollTimeout    = 25
	updatesBatchLimit     = 100
)

// Client является обёрткой над maxigo-client.
// Все внешние типы maxigo инкапсулированы внутри пакета:
// транспорт и бизнес-логика работают только с типами types.go.
//
// Retry намеренно отключен: при long polling таймаут не является ошибкой,
// а включение retry может привести к залипанию цикла при сетевых проблемах.
type Client struct {
	maxigoClient *maxigo.Client
	logger       *slog.Logger
}

// NewClient создаёт клиент MAX API по токену бота.
// WithRetry не используется - см. комментарий к типу Client.
func NewClient(token string, logger *slog.Logger) (*Client, error) {
	maxigoClient, err := maxigo.New(token,
		maxigo.WithTimeout(defaultRequestTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("create maxigo client: %w", err)
	}

	return &Client{
		maxigoClient: maxigoClient,
		logger:       logging.WithComponent(logger, "maxapi"),
	}, nil
}

// UnsubscribeWebhooks снимает все активные подписки на webhook.
// Вызывается при старте приложения, чтобы long polling начал работать:
// если другой сервис (например, PuzzleBot) держит webhook,
// платформа MAX направляет события туда и не отдаёт их через GetUpdates.
func (client *Client) UnsubscribeWebhooks(ctx context.Context) error {
	subscriptions, err := client.maxigoClient.GetSubscriptions(ctx)
	if err != nil {
		logging.ErrorWithCause(client.logger, "get subscriptions failed", err)
		return fmt.Errorf("get subscriptions: %w", err)
	}

	for _, subscription := range subscriptions {
		if _, err := client.maxigoClient.Unsubscribe(ctx, subscription.URL); err != nil {
			logging.ErrorWithCause(client.logger, "unsubscribe failed", err,
				"webhook_url", subscription.URL)
			return fmt.Errorf("unsubscribe %s: %w", subscription.URL, err)
		}
		client.logger.InfoContext(ctx, "webhook removed", "webhook_url", subscription.URL)
	}

	if len(subscriptions) == 0 {
		client.logger.DebugContext(ctx, "no active webhooks found")
	}
	return nil
}

// GetBot возвращает информацию о боте и проверяет валидность токена.
func (client *Client) GetBot(ctx context.Context) (BotInfo, error) {
	bot, err := client.maxigoClient.GetBot(ctx)
	if err != nil {
		logging.ErrorWithCause(client.logger, "get bot failed", err)
		return BotInfo{}, fmt.Errorf("get bot: %w", err)
	}

	info := BotInfo{
		UserID: bot.UserID,
		Name:   bot.FirstName,
	}
	if bot.Username != nil {
		info.Username = *bot.Username
	}

	client.logger.InfoContext(ctx, "bot is available",
		"bot_name", info.Name,
		"username", info.Username)
	return info, nil
}

// SetCommands регистрирует команды бота в платформе MAX.
// Команды отображаются пользователю при вводе /.
func (client *Client) SetCommands(ctx context.Context, commands []Command) error {
	maxigoCommands := make([]maxigo.BotCommand, 0, len(commands))
	for _, command := range commands {
		description := command.Description
		maxigoCommands = append(maxigoCommands, maxigo.BotCommand{
			Name:        command.Name,
			Description: &description,
		})
	}

	if _, err := client.maxigoClient.SetCommands(ctx, maxigoCommands); err != nil {
		logging.ErrorWithCause(client.logger, "set commands failed", err)
		return fmt.Errorf("set commands: %w", err)
	}

	client.logger.InfoContext(ctx, "commands registered", "count", len(commands))
	return nil
}

// GetUpdates забирает очередную пачку событий через long polling и возвращает
// их в виде типизированных структур Update. Нераспознанные типы событий
// возвращаются с UpdateTypeUnknown для безопасной обработки.
func (client *Client) GetUpdates(ctx context.Context, marker int64) ([]Update, int64, error) {
	result, err := client.maxigoClient.GetUpdates(ctx, maxigo.GetUpdatesOpts{
		Limit:   updatesBatchLimit,
		Timeout: updatesPollTimeout,
		Marker:  marker,
	})
	if err != nil {
		return nil, marker, err
	}

	updates := make([]Update, 0, len(result.Updates))
	for _, raw := range result.Updates {
		updates = append(updates, parseUpdate(raw, client.logger))
	}

	newMarker := marker
	if result.Marker != nil {
		newMarker = *result.Marker
	}
	return updates, newMarker, nil
}

// SendMessage отправляет пользователю текстовое сообщение без клавиатуры.
func (client *Client) SendMessage(ctx context.Context, userID int64, text string) error {
	body := &maxigo.NewMessageBody{
		Text: maxigo.Some(text),
	}
	if _, err := client.maxigoClient.SendMessageToUser(ctx, userID, body); err != nil {
		logging.ErrorWithCause(client.logger, "send message failed", err, "user_id", userID)
		return fmt.Errorf("send message to user %d: %w", userID, err)
	}
	client.logger.DebugContext(ctx, "message sent", "user_id", userID)
	return nil
}

// SendMessageWithKeyboard отправляет сообщение с инлайн-клавиатурой.
// buttonRows является набором рядов кнопок; каждый ряд - срез кнопок.
func (client *Client) SendMessageWithKeyboard(ctx context.Context, userID int64, text string, buttonRows [][]Button) error {
	maxigoRows := make([][]maxigo.Button, 0, len(buttonRows))
	for _, row := range buttonRows {
		maxigoRow := make([]maxigo.Button, 0, len(row))
		for _, button := range row {
			switch {
			case button.OpenAppUsername != "":
				maxigoRow = append(maxigoRow, maxigo.NewOpenAppButton(button.Text, button.OpenAppUsername))
			case button.LinkURL != "":
				maxigoRow = append(maxigoRow, maxigo.NewLinkButton(button.Text, button.LinkURL))
			default:
				maxigoRow = append(maxigoRow, maxigo.NewCallbackButton(button.Text, button.Payload))
			}
		}
		maxigoRows = append(maxigoRows, maxigoRow)
	}

	body := &maxigo.NewMessageBody{
		Text: maxigo.Some(text),
		Attachments: []maxigo.AttachmentRequest{
			maxigo.NewInlineKeyboardAttachment(maxigoRows),
		},
	}

	if _, err := client.maxigoClient.SendMessageToUser(ctx, userID, body); err != nil {
		logging.ErrorWithCause(client.logger, "send message with keyboard failed", err, "user_id", userID)
		return fmt.Errorf("send message with keyboard to user %d: %w", userID, err)
	}
	client.logger.DebugContext(ctx, "message with keyboard sent", "user_id", userID)
	return nil
}

// SendTyping отправляет действие "печатает..." в диалог с пользователем.
// Используется для отображения статуса "Ищу секции..." согласно ТЗ.
func (client *Client) SendTyping(ctx context.Context, userID int64) error {
	if _, err := client.maxigoClient.SendAction(ctx, userID, maxigo.ActionTypingOn); err != nil {
		logging.ErrorWithCause(client.logger, "send typing action failed", err, "user_id", userID)
		return fmt.Errorf("send typing action: %w", err)
	}
	return nil
}

// AnswerCallback подтверждает нажатие кнопки уведомлением пользователю.
func (client *Client) AnswerCallback(ctx context.Context, callbackID string, notification string) error {
	answer := &maxigo.CallbackAnswer{}
	if notification != "" {
		answer.Notification = maxigo.Some(notification)
	} else {

		answer.Notification = maxigo.Some(" ")
	}

	if _, err := client.maxigoClient.AnswerCallback(ctx, callbackID, answer); err != nil {
		logging.ErrorWithCause(client.logger, "answer callback failed", err, "callback_id", callbackID)
		return fmt.Errorf("answer callback %s: %w", callbackID, err)
	}
	return nil
}

// IsTimeoutError сообщает, является ли ошибка штатным таймаутом long polling.
// Используется транспортом для продолжения цикла без логирования ошибки.
func IsTimeoutError(err error) bool {
	var maxError *maxigo.Error
	if errors.As(err, &maxError) {
		return maxError.Kind == maxigo.ErrTimeout
	}
	return false
}

// parseUpdate преобразует сырой JSON в типизированную структуру Update.
func parseUpdate(raw json.RawMessage, logger *slog.Logger) Update {
	var base maxigo.Update
	if err := json.Unmarshal(raw, &base); err != nil {
		logger.Error("failed to unmarshal base update", "error", err)
		return Update{Type: UpdateTypeUnknown}
	}

	switch base.UpdateType {
	case maxigo.UpdateMessageCreated:
		var upd maxigo.MessageCreatedUpdate
		if err := json.Unmarshal(raw, &upd); err != nil {
			logger.Error("failed to parse MessageCreated", "error", err)
			return Update{Type: UpdateTypeUnknown}
		}

		text := ""
		if upd.Message.Body.Text != nil {
			text = *upd.Message.Body.Text
		}

		chatID := upd.Message.Sender.UserID
		if upd.Message.Recipient.ChatID != nil {
			chatID = *upd.Message.Recipient.ChatID
		}

		return Update{
			Type: UpdateTypeMessageReceived,
			Payload: MessageReceived{
				UserID:    upd.Message.Sender.UserID,
				ChatID:    chatID,
				Text:      text,
				MessageID: upd.Message.Body.MID, // просто string
			},
		}
	case maxigo.UpdateMessageCallback:
		var upd maxigo.MessageCallbackUpdate
		if err := json.Unmarshal(raw, &upd); err != nil {
			logger.Error("failed to parse MessageCallback", "error", err)
			return Update{Type: UpdateTypeUnknown}
		}

		// Message — указатель, проверяем на nil
		var chatID, userID int64
		var messageID string
		userID = upd.Callback.User.UserID // ← правильный источник
		chatID = upd.Callback.User.UserID
		if upd.Message != nil {
			if upd.Message.Recipient.ChatID != nil {
				chatID = *upd.Message.Recipient.ChatID
			}
			messageID = upd.Message.Body.MID
		}
		// На всякий случай берём UserID из Callback, если Message нет
		if userID == 0 {
			userID = upd.Callback.User.UserID
			chatID = upd.Callback.User.UserID
		}

		return Update{
			Type: UpdateTypeButtonPressed,
			Payload: ButtonPressed{
				UserID:     userID,
				ChatID:     chatID,
				CallbackID: upd.Callback.CallbackID,
				Payload:    upd.Callback.Payload,
				MessageID:  messageID,
			},
		}
	case maxigo.UpdateBotStarted:
		var upd maxigo.BotStartedUpdate
		if err := json.Unmarshal(raw, &upd); err != nil {
			logger.Error("failed to parse BotStarted", "error", err)
			return Update{Type: UpdateTypeUnknown}
		}
		return Update{
			Type:    UpdateTypeBotStarted,
			Payload: BotStarted{UserID: upd.User.UserID},
		}

	case maxigo.UpdateBotStopped:
		var upd maxigo.BotStoppedUpdate
		if err := json.Unmarshal(raw, &upd); err != nil {
			logger.Error("failed to parse BotStopped", "error", err)
			return Update{Type: UpdateTypeUnknown}
		}
		return Update{
			Type:    UpdateTypeBotStopped,
			Payload: BotStopped{UserID: upd.User.UserID},
		}

	default:
		logger.Debug("unsupported update type ignored", "update_type", base.UpdateType)
		return Update{Type: UpdateTypeUnknown}
	}
}
