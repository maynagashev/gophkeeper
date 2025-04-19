package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/maynagashev/gophkeeper/client/internal/tui"
)

const (
	logDir             = "logs"
	logFileName        = "client.log"
	logFilePermissions = 0666
	// Имя переменной окружения для пути к файлу KDBX.
	dbPathEnvVar = "GOPHKEEPER_DB_PATH"
	// Путь к файлу KDBX по умолчанию.
	defaultDBPath = "gophkeeper.kdbx"
)

// setupLogging настраивает логирование в файл logs/gophkeeper.log.
func setupLogging() {
	// Создаем директорию logs, если ее нет
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		// Используем panic, так как без логов продолжать нет смысла
		panic("Не удалось создать директорию для логов: " + err.Error())
	}
	logPath := filepath.Join(logDir, logFileName)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, logFilePermissions)
	if err != nil {
		panic("Не удалось открыть лог-файл: " + err.Error())
	}
	// Важно: Не закрываем logFile здесь через defer, иначе он закроется
	// сразу после выхода из setupLogging. Файл должен оставаться открытым
	// на время работы приложения. Его закроет ОС при завершении процесса.
	// Либо можно вернуть *os.File и закрывать его в main через defer.
	// Пока оставим так, для простоты.

	// Используем NewTextHandler для читаемости логов
	logHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(logHandler))
	slog.Info("Логгер инициализирован", "path", logPath)
}

func main() {
	// Настройка логирования
	setupLogging()

	// Определение флагов
	kdbxPathFlag := flag.String("db", defaultDBPath, "Путь к файлу базы данных KDBX (переопределяет "+dbPathEnvVar+")")
	debugModeFlag := flag.Bool("debug", false, "Включить режим отладки TUI")

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

	slog.Info("Запуск GophKeeper", "db_path", finalPath, "source", source, "debug_mode", *debugModeFlag)

	// Запускаем TUI, передавая финальный путь и флаг отладки
	tui.Start(finalPath, *debugModeFlag)
}
