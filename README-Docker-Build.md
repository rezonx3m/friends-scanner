# 🐳 Сборка для Ubuntu через Docker

## Быстрый старт

Для сборки Linux бинарника локально в Docker и извлечения артефактов:

```bash
./build-docker-extract.sh
```

## Что получится

После сборки в папке `build-linux/` будут:

- `friends-scanner-linux` - исполняемый файл для Linux
- `static/` - папка со статическими файлами
- `start.sh` - скрипт для удобного запуска

## Развертывание на Ubuntu сервере

### 1. Копирование файлов

```bash
# Скопируйте папку build-linux на Ubuntu сервер
scp -r build-linux/ user@your-server:/opt/friends-scanner/
```

### 2. Запуск

```bash
# На Ubuntu сервере
cd /opt/friends-scanner
./start.sh -port=8080

# Или напрямую
./friends-scanner-linux -port=8080
```

### 3. Как системный сервис

```bash
# Создайте systemd unit файл
sudo tee /etc/systemd/system/friends-scanner.service > /dev/null <<EOF
[Unit]
Description=Friends Scanner Service
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/friends-scanner
ExecStart=/opt/friends-scanner/friends-scanner-linux -port=8080
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Запустите сервис
sudo systemctl daemon-reload
sudo systemctl enable friends-scanner
sudo systemctl start friends-scanner

# Проверьте статус
sudo systemctl status friends-scanner
```

## Архитектуры

Текущий скрипт собирает для архитектуры Docker хоста:
- На Apple Silicon (M1/M2) → ARM64 Linux
- На Intel Mac → x86_64 Linux

## Файлы

- `Dockerfile.build` - Dockerfile только для сборки
- `build-docker-extract.sh` - Скрипт сборки и извлечения
- `build-linux/` - Папка с готовыми артефактами

## Размер

Собранный бинарник: ~7.6MB (статически оптимизирован)
