# Makefile для проекта GophKeeper

.PHONY: run build clean

# Цель по умолчанию
default: run

# Запуск клиента
run:
	@echo "Запуск GophKeeper клиента..."
	@go run ./cmd/gophkeeper/main.go

# Сборка клиента
build:
	@echo "Сборка GophKeeper клиента..."
	@go build -o ./bin/gophkeeper ./cmd/gophkeeper/main.go
	@echo "Клиент собран: ./bin/gophkeeper"

# Очистка
clean:
	@echo "Очистка..."
	@rm -f ./bin/gophkeeper
	@echo "Очищено."

# Запуск всех тестов
test:
	@echo "Запуск всех тестов..."
	@go test -v ./... | tee logs/test.log

# Тест с генерацией отчёта о покрытии
test-coverage:
	@echo "Запуск тестов с генерацией покрытия..."
	go test -coverprofile=logs/coverage.out ./...
	go tool cover -html=logs/coverage.out -o logs/coverage.html
	go tool cover -func=logs/coverage.out | tee logs/coverage.log
