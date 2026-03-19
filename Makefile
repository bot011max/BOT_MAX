.PHONY: help init up down logs clean backup restore test security

# Цвета для вывода
GREEN := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
RESET := $(shell tput -Txterm sgr0)

help:
	@echo "$(YELLOW)Доступные команды:$(RESET)"
	@echo "  $(GREEN)make init$(RESET)     - инициализация безопасности (создание ключей)"
	@echo "  $(GREEN)make up$(RESET)       - запуск всех сервисов"
	@echo "  $(GREEN)make down$(RESET)     - остановка всех сервисов"
	@echo "  $(GREEN)make logs$(RESET)     - просмотр логов"
	@echo "  $(GREEN)make clean$(RESET)    - очистка (удаление контейнеров и томов)"
	@echo "  $(GREEN)make backup$(RESET)   - создание резервной копии"
	@echo "  $(GREEN)make restore$(RESET)  - восстановление из бэкапа"
	@echo "  $(GREEN)make test$(RESET)     - запуск тестов"
	@echo "  $(GREEN)make security$(RESET) - запуск тестов безопасности"

init:
	@chmod +x scripts/init-security.sh
	@./scripts/init-security.sh

up:
	@docker-compose -f deployments/docker-compose.yml --env-file .env.production up -d
	@echo "$(GREEN)✅ Сервисы запущены$(RESET)"

down:
	@docker-compose -f deployments/docker-compose.yml down
	@echo "$(GREEN)✅ Сервисы остановлены$(RESET)"

logs:
	@docker-compose -f deployments/docker-compose.yml logs -f

clean:
	@docker-compose -f deployments/docker-compose.yml down -v
	@docker system prune -f
	@echo "$(GREEN)✅ Очистка завершена$(RESET)"

backup:
	@./scripts/backup.sh

restore:
	@./scripts/restore.sh

test:
	@go test ./tests/unit/... -v
	@go test ./tests/integration/... -v

security:
	@python3 tests/security/penetration_test.py

deps:
	@go mod download
	@go mod verify

build:
	@go build -o bin/api ./cmd/api
	@go build -o bin/telegram ./cmd/telegram

run-api:
	@go run ./cmd/api/main.go

run-telegram:
	@go run ./cmd/telegram/main.go

.PHONY: init up down logs clean backup restore test security deps build run-api run-telegram
