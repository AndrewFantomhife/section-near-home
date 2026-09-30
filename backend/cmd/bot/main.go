package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"section-near-home/backend/internal/clients/maxapi"
	"section-near-home/backend/internal/clients/mockapi"
	"section-near-home/backend/internal/config"
	"section-near-home/backend/internal/dialog"
	"section-near-home/backend/internal/logging"
	"section-near-home/backend/internal/transport/maxpoll"
)

func main() {
	// 1. Загрузка переменных окружения из .env
	loadDotEnv()

	// 2. Инициализация структурированного логгера
	logger := logging.NewLoggerFromEnvironment()
	rootLogger := logging.WithComponent(logger, "main")

	rootLogger.Info("ACT backend starting")

	// 3. Загрузка и валидация конфигурации
	cfg, err := config.Load()
	if err != nil {
		logging.Fatal(rootLogger, "configuration validation failed",
			"error", err,
			"hint", "Check .env file and required environment variables")
	}

	rootLogger.Info("configuration loaded",
		"server_port", cfg.ServerPort,
		"mock_api_url", cfg.MockAPIURL,
		"max_api_base_url", cfg.MaxAPIBaseURL,
		"log_level", cfg.LogLevel,
		"storage_kind", cfg.StorageKind,
		"token_prefix", logging.MaskToken(cfg.MaxBotToken))

	// 4. Создание контекста с отменой по сигналу завершения
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		rootLogger.Info("shutdown signal received", "signal", sig.String())
		cancel()
	}()

	// 5. Создание клиента MAX API
	maxClient, err := maxapi.NewClient(cfg.MaxBotToken, logger)
	if err != nil {
		logging.Fatal(rootLogger, "failed to create MAX client", "error", err)
	}

	// 6. Снятие активных вебхуков для работы long polling
	if err := maxClient.UnsubscribeWebhooks(ctx); err != nil {
		logging.ErrorWithCause(rootLogger, "failed to unsubscribe webhooks (non-critical)", err)
		// Не критично: если вебхуков нет, метод просто вернёт пустой список
	}

	// 7. Проверка доступности бота и получение username
	botInfo, err := maxClient.GetBot(ctx)
	if err != nil {
		logging.Fatal(rootLogger, "failed to connect to MAX Bot API", "error", err)
	}

	rootLogger.Info("bot is available",
		"bot_name", botInfo.Name,
		"bot_username", botInfo.Username,
		"user_id", botInfo.UserID)

	// 8. Регистрация команд бота
	if err := maxClient.SetCommands(ctx, []maxapi.Command{
		{Name: "start", Description: "Начать подбор секции"},
		{Name: "help", Description: "Показать справку"},
		{Name: "reset", Description: "Сбросить диалог"},
	}); err != nil {
		rootLogger.Warn("failed to set commands (non-critical)", "error", err)
	}

	// 9. Создание клиента mock-api
	mockClient := mockapi.NewClient(cfg.MockAPIURL, logger)

	// 10. Создание хранилища состояний диалога
	store := dialog.NewStore()

	// Репозиторий заявок будет подключен при реализации ApplicationService.
	// В MVP заявки не создаются из бота (запись через мини-приложение),
	// поэтому repository пока не используется.
	// repository := storage.NewRepository(storage.Kind(cfg.StorageKind), logger)

	// 11. Создание диалогового движка
	// Порядок аргументов: клиенты, store, конфигурация (username), инфраструктура (logger)
	dialogMachine := dialog.NewMachine(maxClient, mockClient, store, botInfo.Username, logger)
	rootLogger.Info("dialog machine initialized")

	// 12. Создание и запуск poller в отдельной горутине
	poller := maxpoll.NewPoller(maxClient, dialogMachine, logger)

	pollerErrCh := make(chan error, 1)
	go func() {
		pollerErrCh <- poller.Run(ctx)
	}()

	rootLogger.Info("poller started in background goroutine")

	// 13. Ожидание завершения poller или сигнала отмены
	select {
	case err := <-pollerErrCh:
		if err != nil {
			if errors.Is(err, context.Canceled) {
				rootLogger.Info("poller stopped as expected", "reason", err)
			} else {
				logging.Fatal(rootLogger, "poller exited unexpectedly", "error", err)
			}
		}
	case <-ctx.Done():
		rootLogger.Info("context cancelled, waiting for poller to stop")
		// Poller завершится при следующей итерации цикла, когда проверит ctx.Done()
		if err := <-pollerErrCh; err != nil && !errors.Is(err, context.Canceled) {
			rootLogger.Warn("poller exited with error during shutdown", "error", err)
		}
	}

	rootLogger.Info("ACT backend stopped")
}
