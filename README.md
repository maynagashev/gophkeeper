# gophkeeper

Менеджер паролей GophKeeper (выпускной проект Яндекс Практикум).

[![GophKeeper Tests](https://github.com/maynagashev/gophkeeper/actions/workflows/test.yml/badge.svg)](https://github.com/maynagashev/gophkeeper/actions/workflows/test.yml)

![gophkeeper_2x](https://pictures.s3.yandex.net/resources/gophkeeper_2x_1650456239.png)

Ссылки:

- [Техническое задание](docs/specification.md)
- [Архитектура](docs/architecture.md)
- [План реализации](docs/plan.md)
- [API для взаимодействия клиента и сервера](docs/api.md)
- [Логика Синхронизации Клиента](docs/sync.md)
- [Структура Экранов TUI Клиента](docs/tui_structure.md)
- [Пользовательский сценарий GophKeeper (Клиент)](docs/user_flow.md)
- [Настройка HTTPS для GophKeeper](docs/https_setup.md)

## Тестовая база данных

Файл базы данных паролей для проверки работы клиента в репозитории (с предзаполненными данными, хотя можно создать свой):

- путь: `client/gophkeeper.kdbx`
- пароль: `123`

Тестовый файл для автоматических тестов (открытие kdbx, чтение записей):

- путь: `client/example/test.kdbx`
- пароль: `test`

## Параметры командной строки

### Сервер (`gophkeeper/server`)

Сервер настраивается с помощью флагов командной строки или соответствующих переменных окружения. Флаги имеют приоритет над переменными окружения.

- `-port <порт>` или `SERVER_PORT=<порт>`:
    Порт для запуска HTTPS-сервера. По умолчанию: `8443`.
- `-cert-file <путь>` или `TLS_CERT_FILE=<путь>`:
    **Обязательно.** Путь к файлу TLS-сертификата (`.pem`).
- `-key-file <путь>` или `TLS_KEY_FILE=<путь>`:
    **Обязательно.** Путь к файлу приватного TLS-ключа (`.pem`).
- `-database-dsn <строка_подключения>` или `DATABASE_DSN=<строка_подключения>`:
    **Обязательно.** Строка подключения к базе данных PostgreSQL (например, `postgres://user:password@host:port/dbname?sslmode=disable`).

### Клиент (`gophkeeper/client`)

- `-db <путь>` или `GOPHKEEPER_DB_PATH=<путь>`:
    Путь к файлу базы данных KDBX. Если флаг `-db` указан, он имеет приоритет над переменной окружения. Если ни флаг, ни переменная не заданы, используется `gophkeeper.kdbx` в текущей директории.
- `-server-url <url>`:
    URL сервера GophKeeper для подключения (например, `https://localhost:8443`). Если указан, переопределяет URL, сохраненный в файле KDBX.
- `-debug`:
    Включает режим отладки TUI, отображая дополнительную информацию в нижней части экрана. По умолчанию выключен.
- `-version`:
    Показывает версию клиента, дату сборки и хеш коммита, после чего завершает работу.

## Возможности

### Сервер

- Регистрация и аутентификация пользователей.
- Безопасное хранение зашифрованных данных (файлов KDBX).
- Синхронизация данных между клиентами одного пользователя.
- Хранение истории версий файлов KDBX.
- Возможность отката к предыдущей версии данных на сервере.
- Взаимодействие с клиентами по защищенному протоколу HTTPS.

### Клиент (CLI/TUI)

- Работа на Linux, Windows, macOS.
- Локальное хранение данных в зашифрованном файле KDBX.
- Создание, открытие и редактирование файлов KDBX.
- Использование мастер-пароля для локального шифрования/дешифрования (пароль никогда не передается на сервер).
- Поддержка различных типов данных:
  - Логины и пароли.
  - Данные банковских карт.
  - Произвольные текстовые заметки.
  - Бинарные данные (файлы) в виде вложений.
- Полнофункциональный терминальный интерфейс (TUI) для удобной работы с записями.
- Быстрый поиск и фильтрация записей.
- Взаимодействие с сервером GophKeeper для:
  - Регистрации и входа.
  - Синхронизации данных (загрузка/скачивание).
  - Просмотра истории версий и отката к предыдущей версии.
- Отображение версии и даты сборки клиента (команда `gophkeeper --version`).
- Автоматическая блокировка KDBX-файла для предотвращения конфликтов при одновременном доступе с одного компьютера.

## Основные Экраны Клиента (TUI)

- **Экран приветствия/ввода пароля:** Запрос мастер-пароля для существующего файла KDBX или создание нового.
- **Основной экран (Список записей):** Отображение всех записей с возможностью навигации, поиска и фильтрации.
- **Экран просмотра деталей:** Показ полной информации о выбранной записи (включая вложения).
- **Экран редактирования/добавления:** Форма для изменения существующей или создания новой записи.
- **Экран Синхронизации и Сервера:** Управление подключением к серверу (URL, вход/регистрация), запуск синхронизации, просмотр версий.

## Важные моменты

- **Разрешение конфликтов:** При синхронизации используется стратегия "Последняя запись побеждает" (Last Write Wins - LWW) на уровне всего файла. Версия файла (локальная или серверная) с более поздним временем последнего изменения содержимого (хранится в метаданных KDBX) считается актуальной и перезаписывает другую. Сервер также выполняет проверки при загрузке, чтобы отклонить устаревшие версии.
- **Версия KDBX:** Клиент может открывать файлы KDBX версий 3.1 и 4.0, но при создании нового файла или сохранении изменений всегда используется формат **KDBX 4.0**.
- **Локальная блокировка:** При запуске клиент пытается установить эксклюзивную блокировку на файл KDBX (через `.lock` файл). Если файл уже открыт другим экземпляром клиента на том же компьютере, он будет открыт в режиме "только для чтения", чтобы предотвратить повреждение данных.
- **HTTPS:** Взаимодействие клиента с сервером происходит только по защищенному протоколу HTTPS.
- **Хранение данных для синхронизации:** URL сервера и токен аутентификации (JWT) сохраняются непосредственно в метаданных KDBX-файла (`CustomData`). Это позволяет клиенту "помнить" настройки подключения между запусками. **Важно:** Сохранение или изменение этих данных также обновляет время последней модификации файла, что влияет на логику синхронизации (LWW).
- **Время жизни токена:** Токен аутентификации (JWT), получаемый от сервера при входе, действителен в течение **24 часов**. По истечении этого времени потребуется повторный вход.

## Как пользоваться

1. **Запуск Клиента:**
    - В терминале выполните команду `gophkeeper --kdbx path/to/your.kdbx`.
    - Если файл `your.kdbx` существует, вас попросят ввести мастер-пароль.
    - Если файл не существует, будет предложено создать его и установить мастер-пароль.
    - Если путь не указан, клиент попытается открыть/создать `gophkeeper.kdbx` в текущей директории.
2. **Работа с Записями (Основной экран):**
    - Используйте клавиши `↑` и `↓` для навигации по списку записей.
    - Нажмите `Enter`, чтобы просмотреть детали выбранной записи.
    - Нажмите `/`, чтобы активировать режим поиска/фильтрации, введите текст и нажмите `Enter`. Нажмите `Esc`, чтобы выйти из режима фильтрации.
    - Нажмите `a`, чтобы добавить новую запись.
    - Находясь на экране просмотра деталей, нажмите `e`, чтобы перейти к редактированию записи.
    - Нажмите `Ctrl+S`, чтобы сохранить изменения в локальный файл KDBX.
3. **Синхронизация с Сервером:**
    - Нажмите `s` на основном экране, чтобы перейти в меню "Синхронизация и Сервер".
    - **Настройка:** Если вы подключаетесь впервые, выберите "Настроить сервер / Изменить URL" и введите URL вашего сервера GophKeeper (например, `https://your-server.com`).
    - **Вход/Регистрация:** Выберите "Войти / Зарегистрироваться", затем выберите `(Р)егистрация` или `(В)ход` и введите имя пользователя и пароль для сервера.
    - **Синхронизация:** После успешного входа выберите "Синхронизировать сейчас". Клиент сравнит локальную и серверную версии и выполнит загрузку или скачивание данных при необходимости.
    - **Просмотр версий:** Выберите "Просмотреть версии" для отображения истории изменений на сервере и возможности отката.
4. **Выход:** Нажмите `q` или `Ctrl+C` для выхода из приложения.

## Примеры использования TUI

[![asciicast](https://asciinema.org/a/nsdB7LNGqGuJPeFKxImvfLc5k.svg)](https://asciinema.org/a/nsdB7LNGqGuJPeFKxImvfLc5k)