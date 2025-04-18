# Структура Экранов TUI Клиента GophKeeper

Этот документ описывает основные экраны (состояния) TUI-клиента и их назначение в виде иерархической структуры.

## Краткий обзор экранов (Outline)

* `welcomeScreen`: **Приветствие при запуске.**
* `passwordInputScreen`: Ввод пароля для существующего KDBX.
* `newKdbxPasswordScreen`: Создание нового KDBX и пароля.
* `entryListScreen`: **Основной экран со списком записей.**
  * `entryDetailScreen`: Просмотр деталей записи.
  * `entryEditScreen`: Редактирование записи.
    * `attachmentListDeleteScreen`: Удаление вложений.
    * `attachmentPathInputScreen`: Добавление вложения (ввод пути).
  * `entryAddScreen`: Добавление новой записи.
* `syncServerScreen`: **Меню и статус синхронизации/сервера.**
  * `serverUrlInputScreen`: Ввод URL сервера.
  * `loginRegisterChoiceScreen`: Выбор: вход или регистрация.
    * `loginScreen`: Вход на сервер.
    * `registerScreen`: Регистрация на сервере.

## Структура и Описание Экранов

* **`welcomeScreen`**
  * **Назначение:** Приветствие пользователя при запуске.
  * **Компоненты:** Статический текст.
  * **Переходы:**
    * `Enter` -> `passwordInputScreen` (если KDBX существует)
    * `Enter` -> `newKdbxPasswordScreen` (если KDBX не существует)
    * `q`/`Ctrl+C` -> Выход.

* **`passwordInputScreen`**
  * **Назначение:** Запрос мастер-пароля для существующего файла KDBX.
  * **Компоненты:** `textinput.Model` (для пароля).
  * **Переходы:**
    * `Enter` (успешно) -> `entryListScreen`
    * `Enter` (ошибка) -> Остается на экране (отобразить ошибку)
    * `Ctrl+C` -> Выход.

* **`newKdbxPasswordScreen`**
  * **Назначение:** Запрос и подтверждение нового мастер-пароля при создании файла KDBX.
  * **Компоненты:** 2 x `textinput.Model` (для пароля и подтверждения).
  * **Переходы:**
    * `Enter` (успешно) -> `entryListScreen`
    * `Enter` (ошибка) -> Остается на экране (отобразить ошибку)
    * `Tab`/`↑`/`↓`: Переключение между полями ввода.
    * `Esc`/`Ctrl+C`: Выход.

* **`entryListScreen` (Основной экран)**
  * **Назначение:** Отображение списка записей из KDBX файла.
  * **Компоненты:** `list.Model` (список записей).
  * **Переходы/Действия:**
    * `Enter` -> `entryDetailScreen`
    * `a` -> `entryAddScreen`
    * `s` -> `syncServerScreen`
    * `l` -> `loginRegisterChoiceScreen` (или другие экраны входа/URL)
    * `Ctrl+S`: Сохранение изменений в KDBX.
    * `↑`/`↓`: Навигация по списку.
    * `/`: Активация/деактивация режима фильтрации.
    * `q`: Выход (если не в режиме фильтрации).
    * `Esc`: Выход из режима фильтрации.

  * **`entryDetailScreen`**
    * **Назначение:** Отображение полной информации о выбранной записи.
    * **Компоненты:** Статический текст (поля записи).
    * **Переходы:**
      * `e` -> `entryEditScreen`
      * `Esc`/`b` -> `entryListScreen`

  * **`entryEditScreen`**
    * **Назначение:** Редактирование полей существующей записи.
    * **Компоненты:** Несколько `textinput.Model` (для полей записи).
    * **Переходы/Действия:**
      * `Enter` -> Сохранить и вернуться к `entryListScreen`.
      * `Esc` -> Отменить и вернуться к `entryDetailScreen`.
      * `Ctrl+O` -> (TODO) `attachmentPathInputScreen` (для добавления вложения)
      * `Ctrl+D` -> `attachmentListDeleteScreen`
      * `Tab`/`Shift+Tab`/`↑`/`↓`: Навигация между полями.

    * **`attachmentListDeleteScreen`**
      * **Назначение:** Отображение списка вложений текущей записи для выбора и удаления.
      * **Компоненты:** `list.Model` (список вложений).
      * **Переходы:**
        * `Enter`/`d` -> Удалить и вернуться к `entryEditScreen`.
        * `Esc`/`b` -> Отменить и вернуться к `entryEditScreen`.
        * `↑`/`↓`: Навигация по списку.

    * **`attachmentPathInputScreen`** (Вызывается из `entryEditScreen` или `entryAddScreen`)
      * **Назначение:** Запрос пути к файлу для добавления в качестве вложения.
      * **Компоненты:** `textinput.Model`.
      * **Переходы:**
        * `Enter` -> Добавить вложение и вернуться к `entryEditScreen`/`entryAddScreen`.
        * `Esc` -> Отменить и вернуться к `entryEditScreen`/`entryAddScreen`.

  * **`entryAddScreen`**
    * **Назначение:** Создание новой записи.
    * **Компоненты:** Несколько `textinput.Model` (для полей записи).
    * **Переходы/Действия:**
      * `Enter` -> Добавить и вернуться к `entryListScreen`.
      * `Esc` -> Отменить и вернуться к `entryListScreen`.
      * `Ctrl+O` -> (TODO) `attachmentPathInputScreen` (для добавления вложения)
      * `Tab`/`Shift+Tab`/`↑`/`↓`: Навигация между полями.

* **`syncServerScreen`** (Вызывается из `entryListScreen`)
  * **Назначение:** Отображение статуса синхронизации и предоставление действий для работы с сервером.
  * **Компоненты:** Статический текст (статусы), `list.Model` (меню действий).
  * **Переходы/Действия:**
    * `Enter` на "Настроить URL" -> `serverUrlInputScreen`
    * `Enter` на "Войти/Зарегистрироваться" -> `loginRegisterChoiceScreen` (или `serverUrlInputScreen`, если URL нет)
    * `Enter` на "Синхронизировать" -> (TODO: Выполнить синхронизацию)
    * `Enter` на "Выйти" -> (TODO: Выполнить выход)
    * `Esc`/`b` -> `entryListScreen`
    * `↑`/`↓`: Навигация по меню.

  * **`serverUrlInputScreen`**
    * **Назначение:** Запрос URL сервера у пользователя.
    * **Компоненты:** `textinput.Model`.
    * **Переходы:**
      * `Enter` -> Сохранить URL и перейти к `loginRegisterChoiceScreen`.
      * `Esc` -> Отменить и вернуться к `syncServerScreen`.

  * **`loginRegisterChoiceScreen`** (Вызывается из `entryListScreen` или `syncServerScreen` или `serverUrlInputScreen`)
    * **Назначение:** Предложение пользователю выбрать между входом и регистрацией.
    * **Компоненты:** Статический текст.
    * **Переходы:**
      * `r`/`R` -> `registerScreen`
      * `l`/`L` -> `loginScreen`
      * `Esc`/`b` -> Возврат к предыдущему экрану (`entryListScreen` или `syncServerScreen`).

    * **`loginScreen`**
      * **Назначение:** Запрос имени пользователя и пароля для входа на сервер.
      * **Компоненты:** 2 x `textinput.Model` (username, password).
      * **Переходы:**
        * `Enter` -> Попытка входа через API (при успехе -> предыдущий экран, при ошибке -> остается).
        * `Esc` -> `loginRegisterChoiceScreen`.
        * `Tab`/`Shift+Tab`: Переключение между полями.

    * **`registerScreen`**
      * **Назначение:** Запрос имени пользователя и пароля для регистрации на сервере.
      * **Компоненты:** 2 x `textinput.Model` (username, password).
      * **Переходы:**
        * `Enter` -> Попытка регистрации через API (при успехе -> предыдущий экран, при ошибке -> остается).
        * `Esc` -> `loginRegisterChoiceScreen`.
        * `Tab`/`Shift+Tab`: Переключение между полями.
