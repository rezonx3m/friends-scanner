# Сборка friends-scanner на Ubuntu

## Установка зависимостей

```bash
# Обновляем пакеты
sudo apt update

# Устанавливаем Go (если не установлен)
sudo apt install golang-go

# Устанавливаем gcc для CGO (необходимо для SQLite)
sudo apt install build-essential

# Проверяем версию Go
go version
```

## Клонирование и сборка

```bash
# Клонируем проект (или копируем файлы)
git clone <your-repo-url>
cd friends-scanner

# Или если копируете файлы вручную:
# mkdir friends-scanner
# cd friends-scanner
# # Скопируйте все файлы проекта

# Загружаем зависимости
go mod download

# Собираем проект
go build -ldflags="-s -w" -o friends-scanner .

# Или используем наш скрипт (если скопировали)
chmod +x build.sh
./build.sh

# Запускаем
./friends-scanner -port=8080
```

## Создание systemd сервиса (опционально)

```bash
# Создаем пользователя для сервиса
sudo useradd -r -s /bin/false friends-scanner

# Создаем директории
sudo mkdir -p /opt/friends-scanner
sudo mkdir -p /var/lib/friends-scanner

# Копируем файлы
sudo cp friends-scanner /opt/friends-scanner/
sudo cp -r static /opt/friends-scanner/

# Создаем systemd unit файл
sudo tee /etc/systemd/system/friends-scanner.service > /dev/null <<EOF
[Unit]
Description=Friends Scanner Service
After=network.target

[Service]
Type=simple
User=friends-scanner
WorkingDirectory=/opt/friends-scanner
ExecStart=/opt/friends-scanner/friends-scanner -port=8080
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Устанавливаем права
sudo chown -R friends-scanner:friends-scanner /opt/friends-scanner
sudo chown -R friends-scanner:friends-scanner /var/lib/friends-scanner

# Запускаем сервис
sudo systemctl daemon-reload
sudo systemctl enable friends-scanner
sudo systemctl start friends-scanner

# Проверяем статус
sudo systemctl status friends-scanner
```

## Настройка Nginx (опционально)

```bash
# Устанавливаем Nginx
sudo apt install nginx

# Создаем конфигурацию
sudo tee /etc/nginx/sites-available/friends-scanner > /dev/null <<EOF
server {
    listen 80;
    server_name your-domain.com;  # Замените на ваш домен

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

# Активируем сайт
sudo ln -s /etc/nginx/sites-available/friends-scanner /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## Настройка файрвола

```bash
# Разрешаем HTTP трафик
sudo ufw allow 'Nginx Full'
# Или если без Nginx:
sudo ufw allow 8080
```
