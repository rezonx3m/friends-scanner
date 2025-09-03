#!/bin/bash

# Скрипт для сборки Docker образа для Ubuntu/Linux
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

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Проверяем, что Docker установлен
if ! command -v docker &> /dev/null; then
    log_error "Docker не установлен. Пожалуйста, установите Docker и повторите попытку."
    exit 1
fi

# Имя образа
IMAGE_NAME="friends-scanner"
TAG=${1:-"latest"}
FULL_IMAGE_NAME="$IMAGE_NAME:$TAG"

log_info "Собираем Docker образ: $FULL_IMAGE_NAME"

# Сборка образа
docker build -t "$FULL_IMAGE_NAME" .

if [ $? -eq 0 ]; then
    log_success "Docker образ собран успешно: $FULL_IMAGE_NAME"
    
    # Показываем размер образа
    IMAGE_SIZE=$(docker images "$FULL_IMAGE_NAME" --format "table {{.Size}}" | tail -n 1)
    log_info "Размер образа: $IMAGE_SIZE"
    
    echo ""
    log_info "Для запуска используйте:"
    echo "  docker run -p 8080:8080 $FULL_IMAGE_NAME"
    echo "  docker run -p 3000:8080 $FULL_IMAGE_NAME  # на порту 3000"
    echo ""
    log_info "Или используйте docker-compose:"
    echo "  docker-compose up -d"
    echo "  docker-compose --profile with-nginx up -d  # с Nginx"
    
    echo ""
    log_info "Для экспорта образа в tar файл:"
    echo "  docker save $FULL_IMAGE_NAME > friends-scanner.tar"
    echo "  # Затем на Ubuntu: docker load < friends-scanner.tar"
    
else
    log_error "Ошибка сборки Docker образа!"
    exit 1
fi
