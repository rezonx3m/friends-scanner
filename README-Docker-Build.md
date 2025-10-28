# 🐳 Сборка для Ubuntu через Docker

Этот документ описывает процесс сборки Linux бинарника приложения Friends QR Scanner в Docker контейнере и его развертывание на Ubuntu сервере.

## Быстрый старт

Для сборки Linux бинарника локально в Docker и извлечения артефактов:

```bash
./build-docker-extract.sh
```

Скрипт автоматически:
- Определит вашу архитектуру (ARM64/x86_64)
- Соберет бинарник для Linux x86_64 (AMD64)
- Извлечет артефакты в папку `build-linux/`
- Создаст скрипт запуска `start.sh`
- Настроит симлинк на статические файлы

### Особенности сборки

- **Apple Silicon (M)**: Использует эмуляцию x86_64, первая сборка может занять время
- **Intel Mac/Linux**: Нативная сборка, быстрая
- **BuildKit**: Включен для ускорения и кэширования слоев
- **Multi-stage build**: Оптимизированный Dockerfile с раздельными этапами

## Что получится

После сборки в папке `build-linux/` будут:

- `friends-scanner-linux` - исполняемый файл для Linux x86_64 (AMD64)
- `static/` - симлинк на папку со статическими файлами (HTML, CSS, JS)
- `start.sh` - скрипт для удобного запуска с параметрами

## Развертывание на Ubuntu сервере

### 1. Копирование файлов

```bash
# Скопируйте папку build-linux на Ubuntu сервер
scp -r build-linux/ user@your-server:/opt/friends-scanner/

# Или используйте rsync для более быстрой передачи
rsync -avz build-linux/ user@your-server:/opt/friends-scanner/
```

### 2. Запуск

```bash
# На Ubuntu сервере
cd /opt/friends-scanner
./start.sh -port=8080

# Или напрямую
./friends-scanner-linux -port=8080
```

Приложение будет доступно по адресу: `http://your-server:8080`

### 3. Настройка как системный сервис

Для автоматического запуска при загрузке системы:

```bash
# Создайте systemd unit файл
sudo tee /etc/systemd/system/friends-scanner.service > /dev/null <<EOF
[Unit]
Description=Friends QR Scanner Service
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/friends-scanner
ExecStart=/opt/friends-scanner/friends-scanner-linux -port=8080
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Безопасность
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/friends-scanner

[Install]
WantedBy=multi-user.target
EOF

# Запустите сервис
sudo systemctl daemon-reload
sudo systemctl enable friends-scanner
sudo systemctl start friends-scanner

# Проверьте статус
sudo systemctl status friends-scanner

# Просмотр логов
sudo journalctl -u friends-scanner -f
```

### 4. Настройка Nginx (опционально)

Для проксирования через Nginx с SSL:

```nginx
server {
    listen 80;
    server_name scanner.yourdomain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Архитектура сборки

Скрипт собирает бинарник для **Linux x86_64 (AMD64)** независимо от архитектуры хоста:

- **На Apple Silicon (M)**: Использует `--platform=linux/amd64` с эмуляцией
- **На Intel Mac**: Нативная сборка для x86_64
- **На Linux x86_64**: Нативная сборка

### Почему x86_64?

- Максимальная совместимость с большинством серверов Ubuntu
- Стабильная поддержка CGO и SQLite
- Предсказуемая производительность

## Структура файлов

```
build-linux/
├── friends-scanner-linux    # Исполняемый файл (~15-20 MB)
├── start.sh                 # Скрипт запуска
├── static/                  # Симлинк на ../static/
└── db.sqlite               # Создается автоматически при первом запуске
```

## Технические детали

### Dockerfile.build

- **Base image**: `golang:1.23-bullseye`
- **Platform**: `linux/amd64`
- **CGO**: Включен для SQLite
- **Оптимизация**: `-ldflags="-s -w"` для уменьшения размера
- **Multi-stage**: Раздельные этапы для сборки и артефактов

### Зависимости

Бинарник статически линкован с:
- SQLite3 (через CGO)
- Go runtime
- Стандартная библиотека Go

### Размер

- Бинарник: ~15-20 MB (с оптимизацией)
- Статические файлы: ~100 KB
- База данных: растет с данными

## Обновление приложения

Для обновления на сервере:

```bash
# 1. Соберите новую версию локально
./build-docker-extract.sh

# 2. Скопируйте на сервер
scp build-linux/friends-scanner-linux user@your-server:/opt/friends-scanner/

# 3. Перезапустите сервис
ssh user@your-server "sudo systemctl restart friends-scanner"
```

## Устранение неполадок

### Ошибка "permission denied"

```bash
chmod +x friends-scanner-linux
chmod +x start.sh
```

### Ошибка "cannot open shared object file"

Убедитесь, что используется Ubuntu/Debian с glibc:

```bash
ldd friends-scanner-linux
```

### База данных не создается

Проверьте права на запись в директории:

```bash
ls -la /opt/friends-scanner/
chmod 755 /opt/friends-scanner/
```

### Порт уже занят

Измените порт в параметрах запуска:

```bash
./start.sh -port=8081
```

## Файлы проекта

- `Dockerfile.build` - Dockerfile только для сборки (без runtime)
- `build-docker-extract.sh` - Скрипт автоматической сборки и извлечения
- `build-linux/` - Папка с готовыми артефактами для развертывания

## Альтернативный способ: Docker на сервере

Если предпочитаете использовать Docker на сервере:

```bash
# Скопируйте весь проект
scp -r . user@your-server:/opt/friends-scanner/

# На сервере запустите через docker-compose
cd /opt/friends-scanner
docker-compose up -d
```

Подробнее см. основной [README.md](README.md)

