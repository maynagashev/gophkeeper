# Корневой Makefile для управления модулями GophKeeper

.PHONY: all server client build build-client build-server clean clean-client clean-server lint lint-client lint-server test test-client test-server test-coverage

# Цель по умолчанию: запустить линтеры и тесты
all: lint test

# --- Запуск --- #
server:
	@echo "Запуск GophKeeper сервера..."
	@make -C server run

client:
	@echo "Запуск GophKeeper клиента..."
	@make -C client run

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

# --- Покрытие тестами (только для клиента пока) --- #
test-coverage:
	@echo "Генерация отчета о покрытии для клиента..."
	@make -C client test-coverage
