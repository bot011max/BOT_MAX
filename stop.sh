#!/bin/bash

echo "🛑 Останавливаем бота..."

# Находим процесс бота и убиваем его
pkill -f "go run cmd/telegram-bot/main.o"
pkill -f "tmp/go-build"

echo "✅ Бот остановлен"

# Спрашиваем, нужно ли остановить PostgreSQL
read -p "Остановить PostgreSQL? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    docker-compose down
    echo "✅ PostgreSQL остановлен"
fi
