package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/maynagashev/gophkeeper/internal/tui"
)

const (
	// Имя переменной окружения для пути к файлу KDBX.
	dbPathEnvVar = "GOPHKEEPER_DB_PATH"
	// Путь к файлу KDBX по умолчанию.
	defaultDBPath = "gophkeeper.kdbx"
)

func main() {
	// Настройка логирования
	logHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))

	// Определение флага для пути к KDBX файлу
	kdbxPathFlag := flag.String("db", defaultDBPath, "Путь к файлу базы данных KDBX (переопределяет "+dbPathEnvVar+")")

	// Парсинг флагов командной строки
	flag.Parse()

	// Определение финального пути к файлу KDBX
	finalPath := defaultDBPath
	source := "по умолчанию"

	// 1. Проверяем переменную окружения
	if envPath := os.Getenv(dbPathEnvVar); envPath != "" {
		finalPath = envPath
		source = "переменная окружения (" + dbPathEnvVar + ")"
	}

	// 2. Проверяем, был ли флаг установлен явно
	dbFlagPresent := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "db" {
			dbFlagPresent = true
		}
	})

	if dbFlagPresent {
		finalPath = *kdbxPathFlag
		source = "флаг -db"
	}

	// Проверка, что итоговый путь не пустой
	if finalPath == "" {
		slog.Error(
			"Путь к файлу базы данных не может быть пустым",
			"проверьте", "флаг -db и переменную окружения "+dbPathEnvVar,
		)
		os.Exit(1)
	}

	slog.Info("Запуск GophKeeper", "db_path", finalPath, "source", source)

	// Запускаем TUI, передавая финальный путь
	tui.Start(finalPath)
}
