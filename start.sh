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
# Шаг 3.5: Исправление foreign key в базе данных
# =====================================
echo -e "\n${YELLOW}🔧 Шаг 3.5: Проверка и исправление foreign key...${NC}"

# Проверяем, существует ли проблемный foreign key
FK_EXISTS=$(docker exec bot_max-postgres-1 psql -U postgres -d medical_bot -t -c "
SELECT COUNT(*) FROM information_schema.table_constraints 
WHERE constraint_name = 'telegram_users_user_id_fkey' AND table_name = 'telegram_users';
" | xargs)

if [ "$FK_EXISTS" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Обнаружен проблемный foreign key. Исправляю...${NC}"
    
    # Удаляем старый foreign key и меняем тип поля
    docker exec bot_max-postgres-1 psql -U postgres -d medical_bot -c "
    BEGIN;
    -- Удаляем foreign key
    ALTER TABLE telegram_users DROP CONSTRAINT IF EXISTS telegram_users_user_id_fkey;
    -- Меняем тип поля на text
    ALTER TABLE telegram_users ALTER COLUMN user_id TYPE text;
    -- Возвращаем foreign key (опционально)
    ALTER TABLE telegram_users 
    ADD CONSTRAINT telegram_users_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
    COMMIT;
    "
    echo -e "${GREEN}✅ Foreign key успешно исправлен${NC}"
else
    echo -e "${GREEN}✅ Foreign key в порядке${NC}"
fi

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
