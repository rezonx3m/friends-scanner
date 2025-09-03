# Многоэтапная сборка для оптимального размера образа
FROM golang:1.21-bullseye AS builder

# Устанавливаем необходимые пакеты для сборки
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Устанавливаем рабочую директорию
WORKDIR /app

# Копируем go mod файлы
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем исходный код
COPY main.go ./

# Собираем приложение
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o friends-scanner .

# Финальный образ
FROM debian:bullseye-slim

# Устанавливаем необходимые пакеты для runtime
RUN apt-get update && apt-get install -y \
    ca-certificates \
    sqlite3 \
    && rm -rf /var/lib/apt/lists/*

# Создаем пользователя для безопасности
RUN useradd -r -s /bin/false appuser

# Создаем рабочую директорию
WORKDIR /app

# Копируем бинарный файл из builder stage
COPY --from=builder /app/friends-scanner .

# Копируем статические файлы
COPY static ./static

# Устанавливаем права
RUN chown -R appuser:appuser /app

# Переключаемся на непривилегированного пользователя
USER appuser

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./friends-scanner", "-port=8080"]
