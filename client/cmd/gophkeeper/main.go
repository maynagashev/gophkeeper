package main

import (
	"flag"
	"log"
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

// Переменные для версии и даты сборки, устанавливаются через ldflags.
var (
	version = "dev" // Значение по умолчанию, если не установлено при сборке
	//nolint:gochecknoglobals // Устанавливается через ldflags при сборке
	buildDate = "unknown" // Значение по умолчанию
	//nolint:gochecknoglobals // Устанавливается через ldflags при сборке
	commitHash = "N/A" // Значение по умолчанию
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
	// Добавляем флаг для версии
	versionFlag := flag.Bool("version", false, "Показать версию и дату сборки")

	// Настройка логирования
	setupLogging()

	// Определение флагов
	kdbxPathFlag := flag.String("db", defaultDBPath, "Путь к файлу базы данных KDBX (переопределяет "+dbPathEnvVar+")")
	debugModeFlag := flag.Bool("debug", false, "Включить режим отладки TUI")
	serverURLFlag := flag.String("server-url", "", "URL сервера GophKeeper (например, https://localhost:8443)")

	// Парсинг флагов командной строки
	flag.Parse()

	// Если указан флаг --version, выводим информацию и выходим
	if *versionFlag {
		// Используем стандартный log для вывода в консоль, так как slog настроен на файл
		log.SetOutput(os.Stdout) // Направляем вывод log в stdout
		log.SetFlags(0)          // Убираем префиксы даты/времени
		log.Println("GophKeeper Client")
		log.Printf("Version: %s", version)
		log.Printf("Build Date: %s", buildDate)
		log.Printf("Commit Hash: %s", commitHash)
		os.Exit(0)
	}

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

	slog.Info("Запуск GophKeeper",
		"db_path", finalPath,
		"source", source,
		"debug_mode", *debugModeFlag,
		"server_url", *serverURLFlag,
	)

	// Запускаем TUI, передавая финальный путь и флаг отладки
	tui.Start(finalPath, *debugModeFlag, *serverURLFlag)
}
