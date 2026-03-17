#!/bin/bash

# =====================================
# АВТОМАТИЧЕСКИЙ ЗАПУСК TELEGRAM-БОТА
# =====================================

# Цвета для вывода
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=====================================${NC}"
echo -e "${BLUE}🚀 ЗАПУСК TELEGRAM-БОТА${NC}"
echo -e "${BLUE}=====================================${NC}"

# Функция проверки ошибок
check_error() {
    if [ $? -ne 0 ]; then
        echo -e "${RED}❌ Ошибка!${NC}"
        exit 1
    fi
}

# =====================================
# Шаг 1: Проверка Docker
# =====================================
echo -e "\n${YELLOW}🔍 Шаг 1: Проверка Docker...${NC}"

if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker не установлен!${NC}"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose не установлен!${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Docker OK${NC}"

# =====================================
# Шаг 2: Проверка .env
# =====================================
echo -e "\n${YELLOW}🔑 Шаг 2: Проверка .env...${NC}"

if [ ! -f .env ]; then
    echo -e "${YELLOW}⚠️ Файл .env не найден. Создаю из .env.example...${NC}"
    cp .env.example .env
    echo -e "${GREEN}✅ .env создан${NC}"
    echo -e "${RED}⚠️ Нужно отредактировать .env и добавить TELEGRAM_TOKEN!${NC}"
    exit 1
fi

# Проверяем наличие TELEGRAM_TOKEN
if ! grep -q "TELEGRAM_TOKEN" .env; then
    echo -e "${RED}❌ TELEGRAM_TOKEN не найден в .env!${NC}"
    exit 1
fi

echo -e "${GREEN}✅ .env OK${NC}"

# =====================================
# Шаг 3: Остановка старых контейнеров
# =====================================
echo -e "\n${YELLOW}🛑 Шаг 3: Остановка старых контейнеров...${NC}"
docker-compose down 2>/dev/null
echo -e "${GREEN}✅ Готово${NC}"

# =====================================
# Шаг 4: Запуск
# =====================================
echo -e "\n${YELLOW}🚀 Шаг 4: Запуск контейнеров...${NC}"
docker-compose up -d --build
check_error
echo -e "${GREEN}✅ Контейнеры запущены${NC}"

# =====================================
# Шаг 5: Ожидание и проверка
# =====================================
echo -e "\n${YELLOW}⏳ Шаг 5: Ожидание инициализации (5 сек)...${NC}"
sleep 5

# Проверяем статус контейнеров
echo -e "\n${YELLOW}📊 Статус контейнеров:${NC}"
docker-compose ps

# Проверяем логи Telegram-бота
echo -e "\n${YELLOW}📋 Последние логи Telegram-бота:${NC}"
docker-compose logs --tail=20 telegram-bot | grep -E "успешно запущен|Ошибка" || echo "Логов пока нет"

# Проверяем логи веб-сервера
echo -e "\n${YELLOW}📋 Последние логи веб-сервера:${NC}"
docker-compose logs --tail=5 app | grep -E "Сервер запущен|Ошибка" || echo "Логов пока нет"

# =====================================
# Итог
# =====================================
echo -e "\n${GREEN}=====================================${NC}"
echo -e "${GREEN}✅ ВСЕ КОНТЕЙНЕРЫ ЗАПУЩЕНЫ!${NC}"
echo -e "${GREEN}=====================================${NC}"
echo -e "\n📱 Telegram-бот: @NEW_lorhelper_bot"
echo -e "🌐 Веб-сервер: http://localhost:8080"
echo -e "\n📝 Для просмотра логов:"
echo -e "   docker-compose logs -f telegram-bot"
echo -e "   docker-compose logs -f app"
echo -e "\n🛑 Для остановки: docker-compose down"
