# ============================================================
# Секция у дома - Makefile
# Управление инфраструктурой разработки проекта
# ============================================================

# Импорт локальных переменных окружения
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# ------------------------------------------------------------
# Константы и производные переменные
# ------------------------------------------------------------
COMPOSE_FILE    := docker-compose.yml
PROJECT_NAME    ?= section-near-home
BACKEND_PORT    ?= 8080
MOCK_API_PORT   ?= 3001
MINI_APP_PORT   ?= 8081

# Имена контейнеров формируются из PROJECT_NAME для соблюдения DRY
BACKEND_CONTAINER   := $(PROJECT_NAME)-backend
MOCK_API_CONTAINER  := $(PROJECT_NAME)-mock-api
MINI_APP_CONTAINER  := $(PROJECT_NAME)-mini-app

# Имена сервисов из docker-compose.yml
BACKEND_SERVICE     := backend-service
MOCK_API_SERVICE    := mosck-api-service
MINI_APP_SERVICE    := mini-app-service

# Базовая команда Docker Compose с явным указанием файла конфигурации
DOCKER_COMPOSE      := docker compose -f $(COMPOSE_FILE)

# Цвета для вывода в терминал (ANSI escape codes)
COLOR_RESET   := \033[0m
COLOR_INFO    := \033[36m
COLOR_SUCCESS := \033[32m
COLOR_WARN    := \033[33m
COLOR_ERROR   := \033[31m

# Цель по умолчанию - вывод справки
.DEFAULT_GOAL := help

# Все цели являются фантомными (не создают файлов с такими именами)
.PHONY: help init env-up env-down env-build env-rebuild env-cleanup \
        backend-run mock-rebuild env-diag env-logs \
        env-logs-backend env-logs-mock env-logs-miniapp

# ------------------------------------------------------------
# Справка (описания на русском для документации команды)
# ------------------------------------------------------------
help: ## Вывести список доступных команд
	@echo ""
	@echo "$(COLOR_INFO)Section Near Home - Development Environment$(COLOR_RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_INFO)%-18s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo ""

# ------------------------------------------------------------
# Инициализация
# ------------------------------------------------------------
init: ## Создать файл .env из шаблона .env.example
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "$(COLOR_SUCCESS)[OK] File .env created from template.$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_WARN)[SKIP] File .env already exists.$(COLOR_RESET)"; \
	fi

# ------------------------------------------------------------
# Управление жизненным циклом среды
# ------------------------------------------------------------
env-up: ## Запустить все сервисы в фоновом режиме
	@echo "$(COLOR_INFO)[INFO] Starting services...$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) up -d
	@echo "$(COLOR_SUCCESS)[OK] Services started.$(COLOR_RESET)"

env-down: ## Остановить все сервисы и удалить контейнеры
	@echo "$(COLOR_INFO)[INFO] Stopping services...$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) down
	@echo "$(COLOR_SUCCESS)[OK] Services stopped.$(COLOR_RESET)"

env-build: ## Пересобрать образы с использованием кэша и запустить
	@echo "$(COLOR_INFO)[INFO] Rebuilding images (with cache)...$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) up -d --build
	@echo "$(COLOR_SUCCESS)[OK] Rebuild completed.$(COLOR_RESET)"

env-rebuild: ## Полная пересборка без кэша (только при изменении базовых образов)
	@echo "$(COLOR_WARN)[WARN] Full rebuild without Docker cache. This may take some time.$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) down
	@$(DOCKER_COMPOSE) build --no-cache
	@$(DOCKER_COMPOSE) up -d
	@echo ""
	@echo "$(COLOR_SUCCESS)[OK] Full rebuild completed. Current status:$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) ps

env-cleanup: ## Полная очистка: контейнеры, сети, тома, образы
	@echo "$(COLOR_WARN)[WARN] Removing all containers, volumes and images of the project.$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) down -v --rmi all --remove-orphans
	@echo "$(COLOR_SUCCESS)[OK] Environment fully cleaned.$(COLOR_RESET)"

# ------------------------------------------------------------
# Локальный запуск (вне Docker)
# ------------------------------------------------------------
backend-run: ## Запустить Go-бэкенд локально через go run
	@echo "$(COLOR_INFO)[INFO] Running backend locally on port $(BACKEND_PORT)...$(COLOR_RESET)"
	@cd backend && SERVER_PORT=$(BACKEND_PORT) MOCK_API_URL=http://localhost:$(MOCK_API_PORT) go run ./cmd/bot/main.go

# ------------------------------------------------------------
# Частичная пересборка
# ------------------------------------------------------------
mock-rebuild: ## Пересобрать только mock-api (при правке db.json)
	@echo "$(COLOR_INFO)[INFO] Rebuilding service $(MOCK_API_SERVICE)...$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) up --build -d $(MOCK_API_SERVICE)
	@echo ""
	@echo "$(COLOR_INFO)[INFO] Content of db.json inside container:$(COLOR_RESET)"
	@docker exec $(MOCK_API_CONTAINER) cat /app/db.json

# ------------------------------------------------------------
# Диагностика и логи
# ------------------------------------------------------------
env-diag: ## Полная диагностика состояния среды
	@echo "$(COLOR_INFO)========== 1. Container Status ==========$(COLOR_RESET)"
	@$(DOCKER_COMPOSE) ps
	@echo ""
	@echo "$(COLOR_INFO)========== 2. Mock-API Healthcheck ==========$(COLOR_RESET)"
	@docker inspect --format='{{json .State.Health}}' $(MOCK_API_CONTAINER) 2>/dev/null || echo "$(COLOR_WARN)Container $(MOCK_API_CONTAINER) not found.$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_INFO)========== 3. db.json Inside Container ==========$(COLOR_RESET)"
	@docker exec $(MOCK_API_CONTAINER) cat /app/db.json 2>/dev/null || echo "$(COLOR_WARN)Failed to read db.json.$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_INFO)========== 4. Endpoint Check /sections/2 ==========$(COLOR_RESET)"
	@docker exec $(MOCK_API_CONTAINER) node -e "require('http').get('http://localhost:$(MOCK_API_PORT)/sections/2', r => {let d='';r.on('data',c=>d+=c);r.on('end',()=>console.log(d))})" 2>/dev/null || echo "$(COLOR_WARN)Endpoint unavailable.$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_SUCCESS)========== Diagnostics Completed ==========$(COLOR_RESET)"

env-logs: ## Показать логи всех сервисов в реальном времени
	@$(DOCKER_COMPOSE) logs -f

env-logs-backend: ## Показать логи бэкенда
	@$(DOCKER_COMPOSE) logs -f $(BACKEND_SERVICE)

env-logs-mock: ## Показать логи mock-api
	@$(DOCKER_COMPOSE) logs -f $(MOCK_API_SERVICE)

env-logs-miniapp: ## Показать логи мини-приложения
	@$(DOCKER_COMPOSE) logs -f $(MINI_APP_SERVICE)