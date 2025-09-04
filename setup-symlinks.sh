#!/bin/bash

# Скрипт для настройки символических ссылок в build папках
# Использование: ./setup-symlinks.sh [clean|setup|status]

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

# Функция для создания символической ссылки
create_symlink() {
    local build_dir="$1"
    local static_link="$build_dir/static"
    
    if [ ! -d "$build_dir" ]; then
        log_info "Создаем папку $build_dir..."
        mkdir -p "$build_dir"
    fi
    
    if [ -d "static" ]; then
        log_info "Настраиваем символическую ссылку в $build_dir..."
        
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
            return 0
        else
            log_error "Не удалось создать символическую ссылку"
            return 1
        fi
    else
        log_error "Папка static не найдена в текущей директории"
        return 1
    fi
}

# Функция для удаления символических ссылок
clean_symlinks() {
    local build_dirs=("build" "build-linux")
    
    for build_dir in "${build_dirs[@]}"; do
        local static_link="$build_dir/static"
        
        if [ -L "$static_link" ]; then
            log_info "Удаляем символическую ссылку $static_link..."
            rm "$static_link"
            log_success "Символическая ссылка удалена: $static_link"
        elif [ -d "$static_link" ]; then
            log_warning "Найдена папка (не ссылка): $static_link"
            echo "Хотите удалить эту папку? (y/N)"
            read -r response
            if [[ "$response" =~ ^[Yy]$ ]]; then
                rm -rf "$static_link"
                log_success "Папка удалена: $static_link"
            fi
        else
            log_info "Ссылка не найдена: $static_link"
        fi
    done
}

# Функция для проверки статуса ссылок
check_status() {
    local build_dirs=("build" "build-linux")
    
    echo ""
    log_info "Статус символических ссылок:"
    echo ""
    
    for build_dir in "${build_dirs[@]}"; do
        local static_link="$build_dir/static"
        
        echo -n "  $static_link: "
        
        if [ -L "$static_link" ]; then
            local target=$(readlink "$static_link")
            # Для относительных ссылок проверяем существование через саму ссылку
            if [ -d "$static_link" ]; then
                echo -e "${GREEN}✓ Относительная ссылка работает${NC} -> $target"
            else
                echo -e "${RED}✗ Битая ссылка${NC} -> $target"
            fi
        elif [ -d "$static_link" ]; then
            echo -e "${YELLOW}⚠ Обычная папка${NC} (не ссылка)"
        elif [ -e "$static_link" ]; then
            echo -e "${RED}✗ Неизвестный тип файла${NC}"
        else
            echo -e "${YELLOW}- Не существует${NC}"
        fi
    done
    
    echo ""
    
    # Проверяем исходную папку static
    if [ -d "static" ]; then
        local file_count=$(find static -type f | wc -l)
        log_success "Исходная папка static содержит $file_count файлов"
    else
        log_error "Исходная папка static не найдена!"
    fi
}

# Функция для настройки всех ссылок
setup_all_symlinks() {
    local build_dirs=("build" "build-linux")
    local success_count=0
    
    for build_dir in "${build_dirs[@]}"; do
        if create_symlink "$build_dir"; then
            ((success_count++))
        fi
    done
    
    echo ""
    if [ $success_count -eq ${#build_dirs[@]} ]; then
        log_success "Все символические ссылки настроены успешно!"
    else
        log_warning "Настроено $success_count из ${#build_dirs[@]} ссылок"
    fi
}

# Основная логика
case "${1:-setup}" in
    "clean")
        log_info "Очистка символических ссылок..."
        clean_symlinks
        ;;
    "setup")
        log_info "Настройка символических ссылок..."
        setup_all_symlinks
        ;;
    "status")
        check_status
        ;;
    *)
        echo "Использование: $0 [clean|setup|status]"
        echo ""
        echo "Команды:"
        echo "  setup  - Создать символические ссылки (по умолчанию)"
        echo "  clean  - Удалить символические ссылки"
        echo "  status - Показать статус ссылок"
        exit 1
        ;;
esac

echo ""
