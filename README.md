# Friends QR Scanner

Проект QR сканера для регистрации пользователей на событиях с интеграцией 2GIS API и системой розыгрыша призов.

## Структура проекта

- `main.go` - Go бекенд сервер
- `go.mod` - зависимости Go
- `static/` - фронтенд файлы
  - `index.html` - главная страница сканера
  - `doc.html` - документация и создание событий
  - `results.html` - шаблон страницы результатов
  - `style.css` - стили
  - `app.js` - основная логика приложения
  - `qr-scanner.min.js` - библиотека QR сканера
  - `qr-scanner-worker.min.js` - worker для QR сканера
- `Dockerfile` - конфигурация Docker образа
- `docker-compose.yml` - конфигурация Docker Compose
- `data/` - директория для хранения базы данных (при использовании Docker)

## Установка и запуск

### Локальный запуск

1. Убедитесь, что у вас установлен Go (версия 1.23 или выше)

2. Установите зависимости:
```bash
go mod tidy
```

3. Запустите сервер:
```bash
go run main.go
```

4. Откройте браузер и перейдите по адресу: `http://localhost:8080`

### Запуск с Docker

1. Соберите и запустите контейнер:
```bash
docker-compose up -d
```

2. Откройте браузер и перейдите по адресу: `http://localhost:8080`

3. Остановка контейнера:
```bash
docker-compose down
```

### Параметры запуска

Сервер поддерживает флаг `-port` для указания порта:
```bash
go run main.go -port=3000
```

## Использование

### 1. Создание события

Перед началом сканирования необходимо создать событие:

1. Откройте `/doc` в браузере
2. Заполните форму создания события:
   - `event_id` - уникальный идентификатор события
   - `password` - пароль для доступа к результатам
3. Нажмите "Создать ивент"

### 2. Сканирование QR-кодов

Откройте главную страницу с параметрами:

```
http://localhost:8080?event_id=conference2024&manager_name=John
```

#### Параметры URL:

- `event_id` - ID события (обязательный)
- `manager_name` - имя менеджера, добавляющего пользователя (опционально)

### 3. Просмотр результатов

Откройте страницу результатов:

```
http://localhost:8080/results?event_id=conference2024&password=yourpassword
```

На странице результатов доступны:
- Список всех зарегистрированных участников
- Статистика по менеджерам
- Информация о пользователях из 2GIS (имя, аватар, short_user_id)
- Система выбора победителей

### 4. Розыгрыш призов

На странице результатов доступны функции:
- **Определить победителя** - случайный выбор N победителей из участников
- **Обнулить результаты** - сброс списка победителей для повторного розыгрыша

## API эндпоинты

### POST /create-event
Создает новое событие с защитой паролем.

**Тело запроса:**
```json
{
    "event_id": "conference2024",
    "password": "secretpassword"
}
```

**Ответ:**
```json
{
    "message": "Ивент успешно создан"
}
```

### POST /scan
Добавляет пользователя в базу данных события.

**Тело запроса:**
```json
{
    "event_id": "conference2024",
    "user_id": "user123",
    "manager_name": "John"
}
```

**Ответ:**
```json
{
    "message": "ok"
}
```

**Возможные ошибки:**
- `"event not found"` - событие не существует
- `"duplicate key value violates unique constraint"` - пользователь уже зарегистрирован
- `"incorrect params"` - неверные параметры

### GET /results?event_id=conference2024&password=yourpassword
Возвращает HTML страницу с результатами сканирования для указанного события.

**Требует авторизации по паролю события.**

### POST /select-winners
Случайным образом выбирает победителей из участников события.

**Тело запроса:**
```json
{
    "event_id": "conference2024",
    "count": 3
}
```

**Ответ:**
```json
{
    "message": "ok"
}
```

### POST /reset-winners
Обнуляет список победителей для события.

**Тело запроса:**
```json
{
    "event_id": "conference2024"
}
```

**Ответ:**
```json
{
    "message": "ok"
}
```

## База данных

При первом запуске автоматически создается файл `db.sqlite` с четырьмя таблицами:

### friends_scanner
Основная таблица регистраций:
```sql
CREATE TABLE friends_scanner (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    manager_name TEXT,
    add_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, user_id)
);
```

### users
Кеш информации о пользователях из 2GIS API:
```sql
CREATE TABLE users (
    user_id TEXT PRIMARY KEY,
    short_user_id TEXT,
    name TEXT,
    avatar TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### winners
Таблица победителей:
```sql
CREATE TABLE winners (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    won_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(event_id, user_id)
);
```

### events
Таблица событий с хешированными паролями:
```sql
CREATE TABLE events (
    event_id TEXT PRIMARY KEY,
    password TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## Функциональность

1. **QR сканирование** - использует камеру устройства для сканирования QR кодов
2. **Извлечение user_id** - извлекает ID пользователя из URL вида `/user/user123`
3. **Интеграция с 2GIS API** - автоматическое получение информации о пользователе (имя, аватар)
4. **Управление событиями** - создание событий с защитой паролем (bcrypt)
5. **Регистрация пользователей** - сохранение данных в SQLite базу
6. **Предотвращение дублирования** - уникальное ограничение на пару (event_id, user_id)
7. **Статистика** - подсчет регистраций по менеджерам
8. **Розыгрыш призов** - случайный выбор победителей из участников
9. **Просмотр результатов** - красивая HTML страница с результатами и аватарами

## Технологии

- **Бекенд**: Go 1.23, SQLite3
- **Фронтенд**: HTML, CSS, JavaScript (без Node.js)
- **QR сканирование**: qr-scanner библиотека
- **Хеширование**: bcrypt (пароли на бекенде)
- **Внешние API**: 2GIS Public Profile API
- **Контейнеризация**: Docker, Docker Compose

## Безопасность

- Пароли событий хешируются с использованием bcrypt
- CORS настроен для безопасной работы API
- Docker контейнер работает от непривилегированного пользователя
