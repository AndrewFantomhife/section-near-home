package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	maxigo "github.com/maxigo-bot/maxigo-client"
)

func main() {

	loadDotEnv()
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== ACT backend: проверка связи с MAX ===")

	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Println("=== ACT backend: проверка связи с MAX ===")

	// --- 1. Читаем токен ---
	token := os.Getenv("MAX_BOT_TOKEN")
	if token == "" {
		log.Fatal("MAX_BOT_TOKEN не задан. Проверьте .env или переменные окружения.")
	}
	log.Printf("Токен получен: %d символов, начинается на %q", len(token), safePrefix(token))

	// --- 2. Создаём клиент ---
	client, err := maxigo.New(token, maxigo.WithTimeout(30*time.Second))
	if err != nil {
		log.Fatalf("Не удалось создать клиент maxigo: %v", err)
	}
	log.Println("Клиент maxigo создан")

	// --- 3. Контекст с отменой по Ctrl+C ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Получен сигнал остановки, завершаем…")
		cancel()
	}()

	// --- 4. Проверяем, что бот доступен ---
	log.Println("Запрашиваем информацию о боте (GetBot)…")
	bot, err := client.GetBot(ctx)
	if err != nil {
		log.Fatalf("GetBot не удался: %v", describeMaxError(err))
	}
	username := "(не задан)"
	if bot.Username != nil {
		username = *bot.Username
	}
	log.Printf("✓ Бот доступен: имя=%q, user_id=%d, username=%q",
		bot.FirstName, bot.UserID, username)
	// --- 5. Ставим команды боту (для удобства) ---
	startDesc := "Запустить бота"
	helpDesc := "Что умеет бот"
	if _, err := client.SetCommands(ctx, []maxigo.BotCommand{
		{Name: "start", Description: &startDesc},
		{Name: "help", Description: &helpDesc},
	}); err != nil {
		log.Printf("SetCommands не удался (не критично): %v", describeMaxError(err))
	} else {
		log.Println("✓ Команды /start и /help установлены")
	}

	// --- 6. Запускаем long polling ---
	log.Println("Запускаем long polling. Напишите боту /start в MAX…")
	log.Println("Для остановки нажмите Ctrl+C")
	log.Println("----------------------------------------")

	var marker int64
	for {
		select {
		case <-ctx.Done():
			log.Println("Контекст отменён, выходим из polling")
			return
		default:
		}

		result, err := client.GetUpdates(ctx, maxigo.GetUpdatesOpts{
			Limit:   100,
			Timeout: 30,
			Marker:  marker,
		})
		if err != nil {
			var e *maxigo.Error
			if ok := asMaxError(err, &e); ok && e.Kind == maxigo.ErrTimeout {
				// штатный таймаут long polling — просто продолжаем
				continue
			}
			log.Printf("GetUpdates ошибка: %v", describeMaxError(err))
			time.Sleep(2 * time.Second)
			continue
		}

		if len(result.Updates) == 0 {
			log.Println("(polling tick, обновлений нет)")
		}

		for _, raw := range result.Updates {
			handleRawUpdate(ctx, client, raw)
		}

		if result.Marker != nil {
			marker = *result.Marker
		}
	}
}

// handleRawUpdate разбирает одно обновление и логирует его.
func handleRawUpdate(ctx context.Context, client *maxigo.Client, raw json.RawMessage) {
	var base maxigo.Update
	if err := json.Unmarshal(raw, &base); err != nil {
		log.Printf("Не удалось разобрать base update: %v", err)
		return
	}

	log.Printf(">>> Обновление типа %q", base.UpdateType)

	switch base.UpdateType {
	case maxigo.UpdateMessageCreated:
		var upd maxigo.MessageCreatedUpdate
		if err := json.Unmarshal(raw, &upd); err != nil {
			log.Printf("  ошибка парсинга MessageCreated: %v", err)
			return
		}
		userID := upd.Message.Sender.UserID
		chatID := upd.Message.Recipient.ChatID
		text := ""
		if upd.Message.Body.Text != nil {
			text = *upd.Message.Body.Text
		}
		log.Printf("  сообщение от user_id=%d chat_id=%d: %q", userID, chatID, text)

		// Эхо-ответ, чтобы убедиться, что отправка работает
		reply := fmt.Sprintf("Привет! Я бот ACT. Ты написал: %q", text)
		if _, err := client.SendMessageToUser(ctx, userID, &maxigo.NewMessageBody{
			Text: maxigo.Some(reply),
		}); err != nil {
			log.Printf("  ответ не отправлен: %v", describeMaxError(err))
		} else {
			log.Printf("  ответ отправлен user_id=%d", userID)
		}

	case maxigo.UpdateMessageCallback:
		var upd maxigo.MessageCallbackUpdate
		if err := json.Unmarshal(raw, &upd); err != nil {
			log.Printf("  ошибка парсинга MessageCallback: %v", err)
			return
		}
		payload := upd.Callback.Payload
		log.Printf("  callback payload=%q", payload)

		if _, err := client.AnswerCallback(ctx, upd.Callback.CallbackID, &maxigo.CallbackAnswer{
			Notification: maxigo.Some("Принято"),
		}); err != nil {
			log.Printf("  AnswerCallback не удался: %v", describeMaxError(err))
		}

	case maxigo.UpdateBotStarted:
		var upd maxigo.BotStartedUpdate
		if err := json.Unmarshal(raw, &upd); err != nil {
			log.Printf("  ошибка парсинга BotStarted: %v", err)
			return
		}
		log.Printf("  пользователь user_id=%d нажал Start", upd.User.UserID)
		if _, err := client.SendMessageToUser(ctx, upd.User.UserID, &maxigo.NewMessageBody{
			Text: maxigo.Some("Привет! Я ACT — помогу найти секцию рядом с домом."),
		}); err != nil {
			log.Printf("  приветствие не отправлено: %v", describeMaxError(err))
		}

	default:
		log.Printf("  (тип %q пока не обрабатываем)", base.UpdateType)
	}
}

// describeMaxError раскрывает *maxigo.Error в читаемый вид.
func describeMaxError(err error) string {
	var e *maxigo.Error
	if asMaxError(err, &e) {
		return fmt.Sprintf("[%s] op=%s status=%d msg=%s",
			kindName(e.Kind), e.Op, e.StatusCode, e.Message)
	}
	return err.Error()
}

func asMaxError(err error, target **maxigo.Error) bool {
	for err != nil {
		if e, ok := err.(*maxigo.Error); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func kindName(k maxigo.ErrorKind) string {
	switch k {
	case maxigo.ErrAPI:
		return "API"
	case maxigo.ErrNetwork:
		return "NETWORK"
	case maxigo.ErrTimeout:
		return "TIMEOUT"
	case maxigo.ErrDecode:
		return "DECODE"
	}
	return "UNKNOWN"
}

func safePrefix(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:8] + "…"
}
