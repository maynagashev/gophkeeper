# API взаимодействия клиента и сервера

## Основные принципы

REST API с JWT-авторизацией:

- Базовый URL: `/api`
- Все запросы кроме `/register` и `/login` требуют заголовок авторизации `Authorization: Bearer <jwt-token>`
- Ответы возвращаются в формате JSON
- Для ошибок используются стандартные HTTP-коды состояния с подробным описанием в теле ответа

## Авторизация

### Регистрация нового пользователя

```bash
POST /api/register
```

**Запрос**:

```json
{
  "username": "string",
  "password": "string",
  "email": "string" // опционально
}
```

**Успешный ответ** (201 Created):

```json
{
  "user_id": "string",
  "token": "string" // JWT-токен для авторизации
}
```

### Аутентификация пользователя

```bash
POST /api/login
```

**Запрос**:

```json
{
  "username": "string",
  "password": "string"
}
```

**Успешный ответ** (200 OK):

```json
{
  "user_id": "string",
  "token": "string" // JWT-токен для авторизации
}
```

## Синхронизация

### Получение метаданных о файле базы

```bash
GET /api/vault
```

**Успешный ответ** (200 OK):

```json
{
  "exists": true,
  "version_id": "string",
  "content_modified_at": "timestamp",
  "size": number,
  "hash": "string"
}
```

### Загрузка файла базы с сервера

```bash
GET /api/vault/download
```

**Успешный ответ** (200 OK):

- Бинарные данные зашифрованной базы данных
- Заголовок `Content-Type: application/octet-stream`

### Загрузка файла базы на сервер

```bash
POST /api/vault/upload
```

**Запрос**:

- Бинарные данные зашифрованной базы данных в теле запроса
- Заголовок `Content-Type: application/octet-stream`
- **Обязательный заголовок `X-Kdbx-Content-Modified-At`**: Время последнего изменения контента KDBX (из `Root.LastModificationTime`) в формате RFC3339 UTC (например: `2023-10-27T10:30:00Z`). Клиент *должен* передавать это значение для корректной работы синхронизации LWW.

**Успешный ответ** (200 OK):

```json
{
  "version_id": "string",
  "saved_at": "timestamp"
}
```

### Получение списка доступных версий

```bash
GET /api/vault/versions
```

**Успешный ответ** (200 OK):

```json
{
  "versions": [
    {
      "version_id": "string",
      "created_at": "timestamp",
      "size": number,
      "metadata": {
        "device_name": "string",
        "client_version": "string",
        "comment": "string"
      }
    }
  ],
  "current_version_id": "string"
}
```

**Параметры ответа**:

- `versions` — массив доступных версий, отсортированный по дате создания (от новых к старым)
- `current_version_id` — идентификатор текущей активной версии
- Для каждой версии указываются:
  - Уникальный идентификатор
  - Дата и время создания
  - Размер файла
  - Метаданные (устройство, версия клиента и т.д.)

**Примечание: Поле `last_modified` было заменено на `content_modified_at` для более точного сравнения версий по времени фактического изменения данных.**

## Дополнительные операции

### Удаление аккаунта пользователя

```bash
DELETE /api/account
```

**Успешный ответ** (204 No Content)

### Откат к предыдущей версии базы

```bash
POST /api/vault/rollback
```

**Запрос**:

```json
{
  "version_id": "string" // Опционально. ID версии для отката. Если не указан, откат к последней версии перед текущей
}
```

**Успешный ответ** (200 OK):

```json
{
  "version_id": "string", // ID версии, к которой выполнен откат
  "previous_version_id": "string" // ID версии до отката (теперь сохранённой в истории)
}
```

### Принцип работы отката

1. Сервер хранит несколько последних версий зашифрованных баз данных для каждого пользователя
2. При запросе отката проверяется наличие указанной версии в истории
3. При успешной проверке указанная версия становится текущей
4. Клиент при следующей синхронизации получает восстановленную версию
5. Текущая версия до отката сохраняется в истории

### Применение отката

- Восстановление после случайного удаления или изменения данных
- Разрешение конфликтов, когда автоматическое слияние невозможно
- Отмена изменений, внесенных вредоносным ПО или неавторизованным доступом

## Коды ошибок

| Код  | Описание                                                  |
|------|-----------------------------------------------------------|
| 400  | Некорректные данные запроса                               |
| 401  | Ошибка аутентификации или отсутствие токена               |
| 403  | Недостаточно прав для выполнения операции                 |
| 404  | Запрашиваемый ресурс не найден                            |
| 409  | Конфликт при обновлении данных                            |
| 429  | Превышен лимит запросов                                   |
| 500  | Внутренняя ошибка сервера                                 |
