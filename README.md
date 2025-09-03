# Friends QR Scanner

Проект QR сканера для регистрации пользователей на событиях.

## Структура проекта

- `main.go` - Go бекенд сервер
- `go.mod` - зависимости Go
- `static/` - фронтенд файлы
  - `index.html` - главная страница
  - `style.css` - стили
  - `app.js` - основная логика приложения
  - `qr-scanner.min.js` - библиотека QR сканера
  - `qr-scanner-worker.min.js` - worker для QR сканера
  - `crypto-js.min.js` - библиотека для MD5 хеширования

## Установка и запуск

1. Убедитесь, что у вас установлен Go (версия 1.21 или выше)

2. Установите зависимости:
```bash
go mod tidy
```

3. Запустите сервер:
```bash
go run main.go
```

4. Откройте браузер и перейдите по адресу: `http://localhost:8080`

## Использование

### Параметры URL

- `event_id` - ID события (по умолчанию "default")
- `manager_name` - имя менеджера, добавляющего пользователя
- `mode` - режим сканирования ("default" или "secure")
- `salt` - соль для secure режима

Пример: `http://localhost:8080?event_id=conference2024&manager_name=John&mode=secure&salt=mysalt`

### API эндпоинты

#### POST /scannerPostData
Добавляет пользователя в базу данных.

Тело запроса:
```json
{
    "event_id": "conference2024",
    "user_id": "user123",
    "manager_name": "John"
}
```

Ответ:
```json
{
    "message": "ok"
}
```

#### GET /scannerResults?event_id=conference2024
Возвращает HTML таблицу с результатами сканирования для указанного события.

### База данных

При первом запуске автоматически создается файл `db.sqlite` с таблицей `friends_scanner`:

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

## Функциональность

1. **QR сканирование** - использует камеру устройства для сканирования QR кодов
2. **Извлечение user_id** - поддерживает два режима:
   - `default`: извлекает ID из URL вида `/user/user123`
   - `secure`: проверяет MD5 хеш в URL вида `/ab/user123` (где `ab` - первые 2 символа MD5(user123+salt))
3. **Регистрация пользователей** - сохраняет данные в SQLite базу
4. **Предотвращение дублирования** - уникальное ограничение на пару (event_id, user_id)
5. **Просмотр результатов** - HTML таблица с зарегистрированными пользователями

## Технологии

- **Бекенд**: Go, SQLite
- **Фронтенд**: HTML, CSS, JavaScript (без Node.js)
- **QR сканирование**: qr-scanner библиотека
- **Хеширование**: собственная реализация MD5
