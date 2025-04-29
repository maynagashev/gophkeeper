# Makefile для сервера GophKeeper

# --- Переменные --- #
# Строка подключения к БД. Используется для миграций.
# Значение по умолчанию соответствует настройкам в docker-compose.yml
DATABASE_DSN ?= postgres://gophkeeper:secret@localhost:5433/gophkeeper?sslmode=disable

# Пути к файлам TLS (используются при запуске сервера)
TLS_CERT_FILE ?= ./certs/server.crt
TLS_KEY_FILE ?= ./certs/server.key

MIGRATIONS_PATH=./migrations
MIGRATE_IMAGE=migrate/migrate:v4.17.1 

.PHONY: run build clean test lint db-up migrate-up migrate-down migrate-force server

# Цель по умолчанию (пока просто сборка)
default: server

# Запуск сервера
server:
	@echo "Запуск GophKeeper сервера..."
	@go run ./cmd/server --database-dsn="${DATABASE_DSN}" --cert-file="${TLS_CERT_FILE}" --key-file="${TLS_KEY_FILE}" | tee ./logs/server.log

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
	# 1. Генерируем сырой профиль покрытия
	go test -coverprofile=logs/coverage.raw.out ./...
	# 2. Фильтруем сырой профиль, исключая моки
	@echo "Фильтрация профиля покрытия (исключая моки)..."
	@cat logs/coverage.raw.out | grep -v '^github.com/maynagashev/gophkeeper/server/internal/mocks' > logs/coverage.out
	# 3. Генерируем HTML отчет из очищенного профиля
	go tool cover -html=logs/coverage.out -o logs/coverage.html
	# 4. Выводим покрытие по функциям и общий итог из очищенного профиля
	@echo "\nПокрытие по функциям (без моков):"
	go tool cover -func=logs/coverage.out | tee logs/coverage.log
	# Общий итог уже будет в конце вывода tee logs/coverage.log


# Линтер
lint:
	@echo "Запуск линтера для сервера..."
	golangci-lint run ./... --fix

# --- Генерация моков --- #
# Установка mockery (если не установлен)
install-mockery:
	@echo "Установка mockery..."
	@go install github.com/vektra/mockery/v2@latest

# Генерация моков для всех интерфейсов в указанных директориях
mocks:
	@echo "Генерация моков..."
	# Генерируем моки в директорию internal/mocks
	@rm -rf internal/mocks # Удаляем старые моки перед генерацией
	# Убрали --keeptree, моки будут плоским списком в internal/mocks
	@mockery --all --output internal/mocks --case=underscore --with-expecter

# --- Миграции БД --- #
# Запуск контейнера с БД (для локальной работы)
db-up:
	@echo "Запуск контейнера PostgreSQL..."
	@docker-compose up -d postgres

# Применение миграций с использованием Docker
migrate:
	@echo "Применение миграций БД из ${MIGRATIONS_PATH} с помощью Docker ${MIGRATE_IMAGE}..."
	@docker run --rm -v $$(pwd)/${MIGRATIONS_PATH}:/migrations --network host ${MIGRATE_IMAGE} -path /migrations -database "${DATABASE_DSN}" up

# Откат последней миграции с использованием Docker
migrate-down:
	@echo "Откат последней миграции БД из ${MIGRATIONS_PATH} с помощью Docker ${MIGRATE_IMAGE}..."
	@docker run --rm -v $$(pwd)/${MIGRATIONS_PATH}:/migrations --network host ${MIGRATE_IMAGE} -path /migrations -database "${DATABASE_DSN}" down 1

# Установка определенной версии миграции с использованием Docker
# Использование: make migrate-force version=<номер_версии>
migrate-force:
	@echo "Принудительная установка версии миграции ${version} из ${MIGRATIONS_PATH} с помощью Docker ${MIGRATE_IMAGE}..."
	@docker run --rm -v $$(pwd)/${MIGRATIONS_PATH}:/migrations --network host ${MIGRATE_IMAGE} -path /migrations -database "${DATABASE_DSN}" force ${version}
