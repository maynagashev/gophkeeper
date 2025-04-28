# Makefile для сервера GophKeeper

# --- Переменные --- #
# Используем значения по умолчанию, если переменные окружения не установлены
POSTGRES_USER ?= gophkeeper
POSTGRES_PASSWORD ?= secret
POSTGRES_DB ?= gophkeeper
POSTGRES_HOST ?= localhost
POSTGRES_PORT ?= 5433
DATABASE_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable
MIGRATIONS_PATH=./migrations
MIGRATE_IMAGE=migrate/migrate:v4.17.1 

.PHONY: run build clean test lint db-up migrate-up migrate-down migrate-force server

# Цель по умолчанию (пока просто сборка)
default: server

# Запуск сервера
server:
	@echo "Запуск GophKeeper сервера..."
	@go run ./cmd/server/main.go | tee ./logs/server.log

# Сборка сервера
build:
	@echo "Сборка GophKeeper сервера..."
	@go build -o ./bin/gophkeeper-server ./cmd/server/main.go
	@echo "Сервер собран: ./bin/gophkeeper-server"

# Очистка
clean:
	@echo "Очистка сервера..."
	@rm -f ./bin/gophkeeper-server
	@echo "Очищено."

# Запуск всех тестов (пока не реализовано)
test:
	@echo "Запуск тестов сервера... (пока нет тестов)"
	# @go test -v ./...

# Тест с генерацией отчёта о покрытии
test-coverage:
	@echo "Запуск тестов с генерацией покрытия..."
	go test -coverprofile=logs/coverage.out ./...
	go tool cover -html=logs/coverage.out -o logs/coverage.html
	go tool cover -func=logs/coverage.out | tee logs/coverage.log


# Линтер
lint:
	@echo "Запуск линтера для сервера..."
	golangci-lint run ./... --fix

# --- Миграции БД --- #
# Запуск контейнера с БД (для локальной работы)
db-up:
	@echo "Запуск контейнера PostgreSQL..."
	@docker-compose up -d postgres

# Применение миграций с использованием Docker
migrate:
	@echo "Применение миграций БД из ${MIGRATIONS_PATH} с помощью Docker ${MIGRATE_IMAGE}..."
	@docker run --rm -v $$(pwd)/${MIGRATIONS_PATH}:/migrations --network host ${MIGRATE_IMAGE} -path /migrations -database "${DATABASE_URL}" up

# Откат последней миграции с использованием Docker
migrate-down:
	@echo "Откат последней миграции БД из ${MIGRATIONS_PATH} с помощью Docker ${MIGRATE_IMAGE}..."
	@docker run --rm -v $$(pwd)/${MIGRATIONS_PATH}:/migrations --network host ${MIGRATE_IMAGE} -path /migrations -database "${DATABASE_URL}" down 1

# Установка определенной версии миграции с использованием Docker
# Использование: make migrate-force version=<номер_версии>
migrate-force:
	@echo "Принудительная установка версии миграции ${version} из ${MIGRATIONS_PATH} с помощью Docker ${MIGRATE_IMAGE}..."
	@docker run --rm -v $$(pwd)/${MIGRATIONS_PATH}:/migrations --network host ${MIGRATE_IMAGE} -path /migrations -database "${DATABASE_URL}" force ${version}
