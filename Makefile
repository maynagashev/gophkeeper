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
	@go build -o gophkeeper ./cmd/gophkeeper/main.go
	@echo "Клиент собран: ./gophkeeper"

# Очистка
clean:
	@echo "Очистка..."
	@rm -f gophkeeper
	@echo "Очищено." 