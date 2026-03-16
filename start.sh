cat > start.sh << 'EOF'
#!/bin/bash

# =====================================
# 🚀 АВТОМАТИЧЕСКИЙ ЗАПУСК TELEGRAM-БОТА
# =====================================

# Цвета для красивого вывода
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
# Шаг 1: Проверка PostgreSQL
# =====================================
echo -e "\n${YELLOW}📦 Шаг 1: Проверка PostgreSQL...${NC}"

if ! docker ps | grep -q "postgres"; then
    echo -e "${YELLOW}⚠️  PostgreSQL не запущен. Запускаем...${NC}"
    docker-compose down 2>/dev/null
    docker-compose up -d postgres
    check_error
    echo -e "${GREEN}✅ PostgreSQL запущен${NC}"
    echo -e "${YELLOW}⏳ Ожидание инициализации (5 сек)...${NC}"
    sleep 5
else
    echo -e "${GREEN}✅ PostgreSQL уже запущен${NC}"
fi

# =====================================
# Шаг 2: Проверка .env
# =====================================
echo -e "\n${YELLOW}🔑 Шаг 2: Проверка .env...${NC}"

if [ ! -f .env ]; then
    echo -e "${RED}❌ Файл .env не найден!${NC}"
    echo -e "${YELLOW}📝 Создаю .env из .env.example...${NC}"
    cp .env.example .env
    check_error
    echo -e "${GREEN}✅ Файл .env создан${NC}"
    echo -e "${RED}⚠️  Нужно отредактировать .env и добавить TELEGRAM_TOKEN!${NC}"
    exit 1
fi

# Проверяем наличие TELEGRAM_TOKEN
if ! grep -q "TELEGRAM_TOKEN" .env; then
    echo -e "${RED}❌ TELEGRAM_TOKEN не найден в .env!${NC}"
    echo -e "${YELLOW}📝 Добавьте строку: TELEGRAM_TOKEN=ваш_токен_от_BotFather${NC}"
    exit 1
fi

# Загружаем токен для проверки
source .env
if [ -z "$TELEGRAM_TOKEN" ] || [ "$TELEGRAM_TOKEN" = "your-telegram-bot-token" ]; then
    echo -e "${RED}❌ TELEGRAM_TOKEN не установлен или имеет значение по умолчанию!${NC}"
    exit 1
fi
echo -e "${GREEN}✅ TELEGRAM_TOKEN найден${NC}"

# =====================================
# Шаг 3: Проверка зависимостей
# =====================================
echo -e "\n${YELLOW}📚 Шаг 3: Проверка зависимостей Go...${NC}"

if [ ! -f go.mod ]; then
    echo -e "${RED}❌ go.mod не найден!${NC}"
    exit 1
fi

echo -e "${YELLOW}📦 Синхронизация зависимостей (go mod tidy)...${NC}"
go mod tidy
check_error
echo -e "${GREEN}✅ Зависимости в порядке${NC}"

# =====================================
# Шаг 4: Запуск бота
# =====================================
echo -e "\n${GREEN}=====================================${NC}"
echo -e "${GREEN}✅ ВСЕ ПРОВЕРКИ ПРОЙДЕНЫ!${NC}"
echo -e "${GREEN}=====================================${NC}"
echo -e "${GREEN}🚀 Запуск бота...${NC}"
echo -e "${GREEN}=====================================${NC}"
echo -e "${YELLOW}📝 Для остановки нажмите Ctrl+C${NC}\n"

go run cmd/telegram-bot/main.go

# Эта строка выполнится после остановки бота
echo -e "\n${YELLOW}🛑 Бот остановлен${NC}"
EOF

chmod +x start.sh

echo -e "\n${GREEN}✅ Скрипт start.sh успешно создан!${NC}"
echo -e "${YELLOW}Для запуска выполните: ./start.sh${NC}"
