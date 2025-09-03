#!/bin/bash

# Простая сборка для Linux без Docker
# Этот скрипт можно запустить на Ubuntu машине

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Проверяем, что мы на Linux
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    log_error "Этот скрипт предназначен для запуска на Linux/Ubuntu"
    exit 1
fi

# Проверяем Go
if ! command -v go &> /dev/null; then
    log_error "Go не установлен. Установите Go:"
    echo "  sudo apt update"
    echo "  sudo apt install golang-go"
    exit 1
fi

# Проверяем gcc для CGO
if ! command -v gcc &> /dev/null; then
    log_error "GCC не установлен. Установите build-essential:"
    echo "  sudo apt install build-essential"
    exit 1
fi

log_info "Сборка для Linux..."

# Создаем папку для сборки
BUILD_DIR="build"
mkdir -p "$BUILD_DIR"

# Загружаем зависимости
log_info "Загружаем зависимости..."
go mod download

# Проверяем код
log_info "Проверяем код..."
go vet ./...

# Собираем
log_info "Собираем бинарный файл..."
CGO_ENABLED=1 go build -ldflags="-s -w" -o "$BUILD_DIR/friends-scanner" .

# Копируем статические файлы
if [ -d "static" ]; then
    log_info "Копируем статические файлы..."
    cp -r static "$BUILD_DIR/"
fi

# Проверяем результат
if [ -f "$BUILD_DIR/friends-scanner" ]; then
    FILE_SIZE=$(du -h "$BUILD_DIR/friends-scanner" | cut -f1)
    log_success "Сборка завершена успешно!"
    log_success "Файл: $BUILD_DIR/friends-scanner (размер: $FILE_SIZE)"
    
    echo ""
    log_info "Для запуска:"
    echo "  ./$BUILD_DIR/friends-scanner"
    echo "  ./$BUILD_DIR/friends-scanner -port=8080"
    
    echo ""
    log_info "Для установки как системный сервис см. ubuntu-build-instructions.md"
else
    log_error "Ошибка сборки!"
    exit 1
fi
