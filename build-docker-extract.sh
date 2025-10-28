#!/bin/bash

# Скрипт для сборки Linux бинарника в Docker и извлечения артефактов
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

# Имя образа для сборки
BUILD_IMAGE="friends-scanner-builder"
CONTAINER_NAME="friends-scanner-build-container"

# Создаем папку для Linux артефактов
LINUX_BUILD_DIR="build-linux"
mkdir -p "$LINUX_BUILD_DIR"

log_info "Собираем Linux бинарники в Docker..."

# Удаляем старый контейнер если существует
docker rm -f "$CONTAINER_NAME" 2>/dev/null || true

# Определяем текущую архитектуру
ARCH=$(uname -m)

# Собираем образ для сборки и извлекаем артефакты
log_info "Создаем Docker образ для сборки..."

# Показываем предупреждение об эмуляции только на ARM64
if [[ "$ARCH" == "arm64" || "$ARCH" == "aarch64" ]]; then
    log_warning "Обнаружена архитектура ARM64 (Apple Silicon)"
    log_warning "Используется эмуляция x86_64, это может занять время при первой сборке"
    log_info "Последующие сборки будут быстрее благодаря кэшированию слоев"
fi

# Используем BuildKit для ускорения сборки с кэшированием
DOCKER_BUILDKIT=1 docker build \
    -f Dockerfile.build \
    --target builder \
    --progress=plain \
    -t "$BUILD_IMAGE" \
    .

if [ $? -ne 0 ]; then
    log_error "Ошибка сборки Docker образа!"
    exit 1
fi

# Создаем контейнер для извлечения артефактов
log_info "Создаем контейнер для извлечения артефактов..."
docker create --name "$CONTAINER_NAME" "$BUILD_IMAGE"

# Извлекаем артефакты из контейнера
log_info "Извлекаем собранный бинарник..."
docker cp "$CONTAINER_NAME:/app/friends-scanner-linux" "$LINUX_BUILD_DIR/" 2>/dev/null || log_error "Не удалось извлечь бинарник"

# Проверяем статические файлы (симлинк уже должен существовать)
if [ -L "$LINUX_BUILD_DIR/static" ]; then
    log_info "Симлинк на статические файлы уже существует"
elif [ -d "static" ]; then
    log_info "Создаем симлинк на статические файлы..."
    ln -s ../static "$LINUX_BUILD_DIR/static"
fi

# Создаем простой запускающий скрипт
cat > "$LINUX_BUILD_DIR/start.sh" << 'EOF'
#!/bin/bash

BINARY="./friends-scanner-linux"

if [ ! -f "$BINARY" ]; then
    echo "Бинарный файл $BINARY не найден!"
    exit 1
fi

# Делаем исполняемым
chmod +x "$BINARY"

# Запускаем с переданными параметрами
echo "Запускаем $BINARY $@"
exec "$BINARY" "$@"
EOF

chmod +x "$LINUX_BUILD_DIR/start.sh"

# Делаем бинарник исполняемым
chmod +x "$LINUX_BUILD_DIR"/friends-scanner-linux 2>/dev/null || true

# Удаляем контейнер
docker rm "$CONTAINER_NAME"

# Опционально удаляем образ сборки
read -p "Удалить Docker образ сборки? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    docker rmi "$BUILD_IMAGE"
    log_info "Docker образ удален"
fi

# Показываем результат
log_success "Сборка завершена!"
echo ""
log_info "Артефакты сохранены в папке: $LINUX_BUILD_DIR/"
ls -la "$LINUX_BUILD_DIR/"

echo ""
log_info "Для запуска на Linux сервере:"
echo "  1. Скопируйте папку $LINUX_BUILD_DIR на Ubuntu сервер"
echo "  2. Запустите: cd $LINUX_BUILD_DIR && ./start.sh -port=8080"
echo ""
log_info "Или запустите напрямую:"
echo "  ./friends-scanner-linux -port=8080"

# Показываем размеры файлов
echo ""
log_info "Размеры файлов:"
du -h "$LINUX_BUILD_DIR"/friends-scanner-linux 2>/dev/null || echo "Нет бинарного файла"
