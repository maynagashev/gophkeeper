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