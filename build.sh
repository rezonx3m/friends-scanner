#!/bin/bash

# Скрипт для сборки бекенда friends-scanner
# Использование: ./build.sh [target_os] [target_arch]
# Примеры:
#   ./build.sh                    # сборка для текущей платформы
#   ./build.sh linux amd64       # сборка для Linux x64
#   ./build.sh windows amd64     # сборка для Windows x64
#   ./build.sh darwin arm64      # сборка для macOS ARM64

set -e  # Остановить выполнение при ошибке

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода сообщений
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

# Функция для создания символических ссылок на статические файлы
create_static_symlink() {
    local build_dir="$1"
    local static_link="$build_dir/static"
    
    if [ -d "static" ]; then
        log_info "Создаем символическую ссылку на статические файлы..."
        
        # Удаляем существующую папку/ссылку static в build директории
        if [ -e "$static_link" ]; then
            if [ -L "$static_link" ]; then
                log_info "Удаляем существующую символическую ссылку..."
                rm "$static_link"
            elif [ -d "$static_link" ]; then
                log_info "Удаляем существующую папку static..."
                rm -rf "$static_link"
            fi
        fi
        
        # Создаем относительную символическую ссылку
        # Переходим в build директорию и создаем ссылку на ../static
        (cd "$build_dir" && ln -s "../static" "static")
        
        if [ -L "$static_link" ]; then
            log_success "Относительная символическая ссылка создана: $static_link -> ../static"
        else
            log_error "Не удалось создать символическую ссылку"
            # Fallback: копируем файлы как раньше
            log_info "Используем копирование файлов как резервный вариант..."
            cp -r static "$build_dir/"
            log_success "Статические файлы скопированы в $build_dir/static/"
        fi
    else
        log_warning "Папка static не найдена"
    fi
}

# Проверяем, что Go установлен
if ! command -v go &> /dev/null; then
    log_error "Go не установлен. Пожалуйста, установите Go и повторите попытку."
    exit 1
fi

# Получаем версию Go
GO_VERSION=$(go version | awk '{print $3}')
log_info "Используется $GO_VERSION"

# Определяем целевую платформу
TARGET_OS=${1:-$(go env GOOS)}
TARGET_ARCH=${2:-$(go env GOARCH)}

log_info "Целевая платформа: $TARGET_OS/$TARGET_ARCH"

# Определяем имя выходного файла
BINARY_NAME="friends-scanner"
if [ "$TARGET_OS" = "windows" ]; then
    BINARY_NAME="${BINARY_NAME}.exe"
fi

# Создаем папку для сборки если её нет
BUILD_DIR="build"
if [ ! -d "$BUILD_DIR" ]; then
    mkdir -p "$BUILD_DIR"
    log_info "Создана папка $BUILD_DIR"
fi

# Путь к выходному файлу
OUTPUT_PATH="$BUILD_DIR/${BINARY_NAME}"
if [ "$TARGET_OS" != "$(go env GOOS)" ] || [ "$TARGET_ARCH" != "$(go env GOARCH)" ]; then
    OUTPUT_PATH="$BUILD_DIR/${BINARY_NAME}_${TARGET_OS}_${TARGET_ARCH}"
    if [ "$TARGET_OS" = "windows" ]; then
        OUTPUT_PATH="${OUTPUT_PATH}.exe"
    fi
fi

log_info "Начинаем сборку..."

# Устанавливаем зависимости
log_info "Загружаем зависимости..."
go mod download

# Проверяем код
log_info "Проверяем код..."
go vet ./...

# Сборка
log_info "Собираем бинарный файл: $OUTPUT_PATH"

# Устанавливаем переменные окружения для кросс-компиляции
export GOOS=$TARGET_OS
export GOARCH=$TARGET_ARCH

# Включаем CGO для SQLite
export CGO_ENABLED=1

# Для кросс-компиляции с CGO может потребоваться дополнительная настройка
if [ "$TARGET_OS" != "$(go env GOOS)" ]; then
    log_warning "Кросс-компиляция с CGO может требовать дополнительных инструментов"
    case $TARGET_OS in
        "linux")
            if [ "$TARGET_ARCH" = "amd64" ]; then
                export CC=x86_64-linux-gnu-gcc
            fi
            ;;
        "windows")
            if [ "$TARGET_ARCH" = "amd64" ]; then
                export CC=x86_64-w64-mingw32-gcc
            fi
            ;;
    esac
fi

# Выполняем сборку
go build -ldflags="-s -w" -o "$OUTPUT_PATH" .

# Проверяем результат
if [ -f "$OUTPUT_PATH" ]; then
    FILE_SIZE=$(du -h "$OUTPUT_PATH" | cut -f1)
    log_success "Сборка завершена успешно!"
    log_success "Файл: $OUTPUT_PATH (размер: $FILE_SIZE)"
    
    # Создаем символические ссылки на статические файлы
    create_static_symlink "$BUILD_DIR"
    
    echo ""
    log_info "Для запуска используйте:"
    if [ "$TARGET_OS" = "$(go env GOOS)" ] && [ "$TARGET_ARCH" = "$(go env GOARCH)" ]; then
        echo "  ./$OUTPUT_PATH"
        echo "  ./$OUTPUT_PATH -port=3000"
    else
        echo "  Перенесите файл $OUTPUT_PATH на целевую систему и запустите"
    fi
else
    log_error "Ошибка сборки!"
    exit 1
fi
