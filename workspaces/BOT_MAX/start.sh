echo '#!/bin/bash

echo "====================================="
echo "🚀 ЗАПУСК TELEGRAM-БОТА"
echo "====================================="

# Цвета для вывода
GREEN="\033[0;32m"
YELLOW="\033[1;33m"
RED="\033[0;31m"
NC="\033[0m"

# Функция проверки ошибок
check_error() {
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Ошибка!${NC}"
        exit 1
    fi
}

# Шаг 1: Проверяем PostgreSQL
echo -e "${YELLOW}📦 Шаг 1: Проверка PostgreSQL...${NC}"
if ! docker ps | grep -q "postgres"; then
    echo -e "${YELLOW}⚠️  PostgreSQL не запущен. Запускаем...${NC}"
    docker-compose up -d postgres
    check_error
    echo -e "${GREEN}✅ PostgreSQL запущен${NC}"
    echo -e "${YELLOW}⏳ Ожидание инициализации...${NC}"
    sleep 5
else
    echo -e "${GREEN}✅ PostgreSQL уже запущен${NC}"
fi

# Шаг 2: Проверяем .env
echo -e "${YELLOW}🔑 Шаг 2: Проверка .env...${NC}"
if [ ! -f .env ]; then
    echo -e "${RED}❌ Файл .env не найден!${NC}"
    echo -e "${YELLOW}📝 Создайте .env из .env.example:${NC}"
    echo "cp .env.example .env"
    exit 1
fi

# Шаг 3: Проверяем токен
if ! grep -q "TELEGRAM_TOKEN" .env; then
    echo -e "${RED}❌ TELEGRAM_TOKEN не найден в .env!${NC}"
    exit 1
fi

# Шаг 4: Запускаем бота
echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}🚀 Запуск бота...${NC}"
echo -e "${GREEN}=====================================${NC}"
echo -e "${YELLOW}📝 Для остановки нажмите Ctrl+C${NC}"
echo ""

go run cmd/telegram-bot/main.go
' > start.sh && chmod +x start.sh && echo "✅ Файл start.sh создан и готов к запуску! Запусти командой: ./start.sh"
