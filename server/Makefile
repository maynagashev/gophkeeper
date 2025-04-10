# Makefile для сервера GophKeeper

.PHONY: run build clean test lint

# Цель по умолчанию (пока просто сборка)
default: build

# Запуск сервера (пока не реализовано)
run:
	@echo "Запуск GophKeeper сервера... (пока не реализовано)"
	# @go run ./cmd/server/main.go

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

# Линтер
lint:
	@echo "Запуск линтера для сервера..."
	golangci-lint run ./... --fix
