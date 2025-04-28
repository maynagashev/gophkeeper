# Makefile для проекта GophKeeper

.PHONY: client build clean client

# Цель по умолчанию
default: client

# Запуск клиента
client:
	@echo "Запуск GophKeeper клиента..."
	@go run ./cmd/gophkeeper/main.go -debug --server-url=https://localhost:8443 | tee logs/run.log

# Сборка клиента
build:
	@echo "Сборка GophKeeper клиента..."
	# Получаем версию из Git (тег, или хэш коммита если тега нет, + '-dirty' если есть не закомиченные изменения)
	@{ \
		version=$$(git describe --tags --always --dirty --abbrev=7); \
		buildDate=$$(date -u '+%Y-%m-%dT%H:%M:%SZ'); \
		ldflags="-X 'main.version=$$version' -X 'main.buildDate=$$buildDate'"; \
		echo "Версия: $$version, Дата сборки: $$buildDate"; \
		go build -ldflags="$$ldflags" -o ./bin/gophkeeper ./cmd/gophkeeper/main.go; \
	}
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
