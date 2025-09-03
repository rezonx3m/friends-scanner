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
