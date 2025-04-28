# Корневой Makefile для управления модулями GophKeeper

.PHONY: all server client build build-client build-server clean clean-client clean-server lint lint-client lint-server test test-client test-server test-coverage test-coverage-client test-coverage-server

# Цель по умолчанию: запустить линтеры и тесты
all: lint test

# --- Запуск --- #
server:
	@echo "Запуск GophKeeper сервера..."
	@make -C server server

client:
	@echo "Запуск GophKeeper клиента..."
	@make -C client client

# --- Сборка --- #
build: build-client build-server
	@echo "Сборка завершена для всех модулей."

build-client:
	@echo "Сборка клиента..."
	@make -C client build

build-server:
	@echo "Сборка сервера..."
	@make -C server build

# --- Очистка --- #
clean: clean-client clean-server
	@echo "Очистка завершена для всех модулей."

clean-client:
	@echo "Очистка клиента..."
	@make -C client clean

clean-server:
	@echo "Очистка сервера..."
	@make -C server clean

# --- Линтинг --- #
lint: lint-client lint-server
	@echo "Линтинг завершен для всех модулей."

lint-client:
	@echo "Линтинг клиента..."
	@make -C client lint

lint-server:
	@echo "Линтинг сервера..."
	@make -C server lint

# --- Тестирование --- #
test: test-client test-server
	@echo "Тестирование завершено для всех модулей."

test-client:
	@echo "Тестирование клиента..."
	@make -C client test

test-server:
	@echo "Тестирование сервера..."
	@make -C server test

# --- Покрытие тестами --- #
test-coverage: test-coverage-client test-coverage-server
	@echo "Проверка покрытия тестами завершена для всех модулей."

test-coverage-client:
	@echo "Генерация отчета о покрытии для клиента..."
	@make -C client test-coverage

test-coverage-server:
	@echo "Генерация отчета о покрытии для сервера..."
	@make -C server test-coverage

migrate:
	@echo "Применение миграций..."
	@make -C server migrate

# --- Релизная сборка клиента --- #
# Переменные для ldflags
VERSION := $(shell git describe --tags --always --dirty || echo "dev")
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
COMMIT_HASH := $(shell git rev-parse --short HEAD || echo "N/A")
LDFLAGS := -ldflags "-X 'main.version=$(VERSION)' -X 'main.buildDate=$(BUILD_DATE)' -X 'main.commitHash=$(COMMIT_HASH)'"

# Директория для бинарников
BIN_DIR := ./bin
CLIENT_CMD := ./client/cmd/gophkeeper
CLIENT_TARGET_BASE := gophkeeper

# Цель для сборки всех релизных версий клиента
.PHONY: build-client-release
build-client-release: build-client-linux build-client-windows build-client-darwin
	@echo "Релизная сборка клиента завершена. Бинарники в $(BIN_DIR)"

# Цели для конкретных платформ
.PHONY: build-client-linux build-client-windows build-client-darwin
build-client-linux:
	@echo "Сборка клиента для Linux (amd64)..."
	@mkdir -p $(BIN_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(CLIENT_TARGET_BASE)-linux-amd64 $(CLIENT_CMD)

build-client-windows:
	@echo "Сборка клиента для Windows (amd64)..."
	@mkdir -p $(BIN_DIR)
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(CLIENT_TARGET_BASE)-windows-amd64.exe $(CLIENT_CMD)

build-client-darwin:
	@echo "Сборка клиента для macOS (amd64)..."
	@mkdir -p $(BIN_DIR)
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BIN_DIR)/$(CLIENT_TARGET_BASE)-darwin-amd64 $(CLIENT_CMD)

# --- Демонстрация --- #
.PHONY: record-tui-demo
DEMO_KDBX_PATH=/tmp/gophkeeper_demo.kdbx
DEMO_CAST_PATH=docs/tui_demo.cast
# Путь к бинарнику клиента относительно текущего Makefile
CLIENT_BINARY_PATH=./client/bin/gophkeeper

# Запись демонстрации работы TUI клиента с помощью asciinema
record-tui-demo: build-client
	@echo "Подготовка к записи TUI демо..."
	@rm -f $(DEMO_KDBX_PATH)
	@echo "Запускается запись в $(DEMO_CAST_PATH)..."
	@echo "Запись начнется автоматически. Выполните базовый сценарий:"
	@echo "  создание KDBX -> добавление записи -> сохранение -> выход."
	@echo "Команда для запуска клиента: $(CLIENT_BINARY_PATH) --db $(DEMO_KDBX_PATH)"
	@echo "---"
	@asciinema rec $(DEMO_CAST_PATH) --title "GophKeeper TUI Demo" --command "$(CLIENT_BINARY_PATH) --db $(DEMO_KDBX_PATH) --debug"
	@echo "---"
	@echo "Запись завершена и сохранена в $(DEMO_CAST_PATH)"
	@echo "Очистка временного файла: $(DEMO_KDBX_PATH)..."
	@rm -f $(DEMO_KDBX_PATH)
	@echo "Готово."
