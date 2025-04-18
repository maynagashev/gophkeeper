# Makefile для проекта GophKeeper

.PHONY: run build clean

# Цель по умолчанию
default: run

# Запуск клиента
run:
	@echo "Запуск GophKeeper клиента..."
	@go run ./cmd/gophkeeper/main.go -debug | tee logs/run.log

# Сборка клиента
build:
	@echo "Сборка GophKeeper клиента..."
	@go build -o ./bin/gophkeeper ./cmd/gophkeeper/main.go
	@echo "Клиент собран: ./bin/gophkeeper"

# Запуск с указанием пути через флаг
run-flag:
	@echo "Запуск GophKeeper клиента с флагом -db example/test.kdbx..."
	@go run ./cmd/gophkeeper/main.go -db example/test.kdbx | tee logs/run_flag.log

# Запуск с указанием пути через переменную окружения
run-env:
	@echo "Запуск GophKeeper клиента с GOPHKEEPER_DB_PATH=example/test.kdbx..."
	@GOPHKEEPER_DB_PATH=example/test.kdbx go run ./cmd/gophkeeper/main.go | tee logs/run_env.log

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

# Линтер
lint:
	@echo "Запуск линтера..."
	golangci-lint run ./... --fix
