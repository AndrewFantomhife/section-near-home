include .env
export

PROJECT_ROOT := $(CURDIR)
export PROJECT_ROOT

env-up:
	@docker compose up -d

env-down:
	@docker compose down

env-build:
	@docker compose up -d --build

env-cleanup:
	@docker compose down -v --rmi all --remove-orphans

backend-run:
	@go run ./backend/cmd/bot/main.go