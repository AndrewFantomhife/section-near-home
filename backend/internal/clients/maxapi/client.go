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

// Button описывает callback-кнопку инлайн-клавиатуры.
// Payload возвращается боту при нажатии кнопки.
type Button struct {
	Text    string
	Payload string
}

// BotInfo содержит данные бота, необходимые приложению.
type BotInfo struct {
	UserID   int64
	Name     string
	Username string
}

// Updates содержит пачку событий MAX и новый маркер очереди.
type Updates struct {
	Raw    []json.RawMessage
	Marker int64
}

// Client является обёрткой над maxigo-client.
// Он изолирует бизнес-логику от сторонней библиотеки MAX Bot API:
// при замене библиотеки изменится только этот пакет.
type Client struct {
	maxigoClient *maxigo.Client
	logger       *slog.Logger
}

// NewClient создаёт клиент MAX API по токену бота.
func NewClient(token string, logger *slog.Logger) (*Client, error) {
	maxigoClient, err := maxigo.New(token,
		maxigo.WithTimeout(defaultRequestTimeout),
		maxigo.WithRetry(),
	)
	if err != nil {
		return nil, fmt.Errorf("create maxigo client: %w", err)
	}

	return &Client{
		maxigoClient: maxigoClient,
		logger:       logging.WithComponent(logger, "maxapi"),
	}, nil
}

// GetBot возвращает информацию о боте и тем самым проверяет валидность токена.
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

// GetUpdates забирает очередную пачку событий через long polling.
// Штатный таймаут возвращается ошибкой, которую распознаёт IsTimeoutError.
func (client *Client) GetUpdates(ctx context.Context, marker int64) (Updates, error) {
	result, err := client.maxigoClient.GetUpdates(ctx, maxigo.GetUpdatesOpts{
		Limit:   updatesBatchLimit,
		Timeout: updatesPollTimeout,
		Marker:  marker,
	})
	if err != nil {
		return Updates{}, err
	}

	updates := Updates{
		Raw:    result.Updates,
		Marker: marker,
	}
	if result.Marker != nil {
		updates.Marker = *result.Marker
	}
	return updates, nil
}

// SendMessage отправляет пользователю текстовое сообщение с необязательной
// инлайн-клавиатурой. buttonRows является набором рядов кнопок;
// передайте nil, чтобы отправить сообщение без клавиатуры.
func (client *Client) SendMessage(ctx context.Context, userID int64, text string, buttonRows [][]Button) error {
	body := &maxigo.NewMessageBody{
		Text: maxigo.Some(text),
	}

	if len(buttonRows) > 0 {
		body.Attachments = []maxigo.AttachmentRequest{
			maxigo.NewInlineKeyboardAttachment(convertButtonRows(buttonRows)),
		}
	}

	if _, err := client.maxigoClient.SendMessageToUser(ctx, userID, body); err != nil {
		logging.ErrorWithCause(client.logger, "send message failed", err, "user_id", userID)
		return fmt.Errorf("send message to user %d: %w", userID, err)
	}

	client.logger.DebugContext(ctx, "message sent", "user_id", userID)
	return nil
}

// AnswerCallback подтверждает нажатие кнопки уведомлением пользователю.
func (client *Client) AnswerCallback(ctx context.Context, callbackID string, notification string) error {
	answer := &maxigo.CallbackAnswer{}
	if notification != "" {
		answer.Notification = maxigo.Some(notification)
	}

	if _, err := client.maxigoClient.AnswerCallback(ctx, callbackID, answer); err != nil {
		logging.ErrorWithCause(client.logger, "answer callback failed", err, "callback_id", callbackID)
		return fmt.Errorf("answer callback %s: %w", callbackID, err)
	}
	return nil
}

// IsTimeoutError сообщает, является ли ошибка штатным таймаутом long polling.
// Транспорт использует предикат, чтобы продолжить цикл без логирования ошибки.
func IsTimeoutError(err error) bool {
	var maxError *maxigo.Error
	if errors.As(err, &maxError) {
		return maxError.Kind == maxigo.ErrTimeout
	}
	return false
}

// convertButtonRows преобразует прикладные кнопки в ряды клавиатуры maxigo.
func convertButtonRows(buttonRows [][]Button) [][]maxigo.Button {
	converted := make([][]maxigo.Button, 0, len(buttonRows))
	for _, row := range buttonRows {
		convertedRow := make([]maxigo.Button, 0, len(row))
		for _, button := range row {
			convertedRow = append(convertedRow, maxigo.NewCallbackButton(button.Text, button.Payload))
		}
		converted = append(converted, convertedRow)
	}
	return converted
}
