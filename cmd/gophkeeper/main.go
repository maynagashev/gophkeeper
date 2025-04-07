package main

import (
	"log/slog"
	"os"

	"github.com/maynagashev/gophkeeper/internal/tui"
)

func main() {
	// Настройка логирования в файл
	logFile, err := os.OpenFile("logs/gophkeeper.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic("Не удалось открыть лог-файл: " + err.Error())
	}
	defer logFile.Close()

	// Создаем JSON обработчик, пишущий в файл
	// Уровень Debug, чтобы видеть все наши отладочные сообщения
	logger := slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	// Устанавливаем созданный логгер как стандартный
	slog.SetDefault(logger)

	slog.Info("Логгер инициализирован, запись в gophkeeper.log")

	// Запуск TUI
	tui.Start()
}
