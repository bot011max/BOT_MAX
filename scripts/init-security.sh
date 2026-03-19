#!/bin/bash
# Инициализация безопасности - ЗАПУСКАТЬ ПЕРВЫМ!

set -e

echo "🔐 ИНИЦИАЛИЗАЦИЯ ВОЕННОЙ ЗАЩИТЫ"
echo "================================"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 1. Создание директории для секретов
echo -e "\n${YELLOW}1. Создание секретов...${NC}"
mkdir -p secrets
chmod 700 secrets

# Генерация мастер-ключа (32 байта)
MASTER_KEY=$(openssl rand -base64 32)
echo -n $MASTER_KEY > secrets/master_key.txt
echo -e "${GREEN}✅ Мастер-ключ создан${NC}"

# Генерация JWT секрета
JWT_SECRET=$(openssl rand -base64 64)
echo -n $JWT_SECRET > secrets/jwt_secret.txt
echo -e "${GREEN}✅ JWT секрет создан${NC}"

# Генерация пароля PostgreSQL
POSTGRES_PASSWORD=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-24)
echo -n $POSTGRES_PASSWORD > secrets/postgres_password.txt
echo -e "${GREEN}✅ PostgreSQL пароль создан${NC}"

# Генерация пароля Redis
REDIS_PASSWORD=$(openssl rand -base64 24 | tr -d "=+/" | cut -c1-24)
echo -n $REDIS_PASSWORD > secrets/redis_password.txt
echo -e "${GREEN}✅ Redis пароль создан${NC}"

# 2. Создание SSL сертификатов
echo -e "\n${YELLOW}2. Генерация SSL сертификатов...${NC}"
mkdir -p config/nginx/ssl

openssl req -x509 -nodes -days 365 -newkey rsa:4096 \
    -keyout config/nginx/ssl/privkey.pem \
    -out config/nginx/ssl/fullchain.pem \
    -subj "/C=RU/ST=Moscow/L=Moscow/O=MedicalBot/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"

chmod 600 config/nginx/ssl/*.pem
echo -e "${GREEN}✅ SSL сертификаты созданы${NC}"

# 3. Создание .env файла
echo -e "\n${YELLOW}3. Создание .env.production...${NC}"

cat > .env.production << EOF
# ==========================================
# PRODUCTION КОНФИГУРАЦИЯ
# ==========================================

# База данных
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=${POSTGRES_PASSWORD}
DB_NAME=medical_bot

# Redis
REDIS_HOST=redis
REDIS_PASSWORD=${REDIS_PASSWORD}

# JWT
JWT_SECRET=${JWT_SECRET}
JWT_EXPIRY=900

# Telegram
TELEGRAM_TOKEN=your-telegram-bot-token
WEBHOOK_URL=https://your-domain.com
WEBHOOK_SECRET=$(openssl rand -base64 32)

# Мониторинг
GRAFANA_PASSWORD=$(openssl rand -base64 16)

# Криптография
MASTER_KEY=${MASTER_KEY}
EOF

echo -e "${GREEN}✅ .env.production создан${NC}"

# 4. Настройка прав доступа
echo -e "\n${YELLOW}4. Настройка прав доступа...${NC}"
chmod 600 secrets/*.txt
chmod 600 .env.production

echo -e "\n${GREEN}✅ ИНИЦИАЛИЗАЦИЯ ЗАВЕРШЕНА!${NC}"
echo -e "\n⚠️  ВАЖНО: Отредактируйте .env.production и добавьте TELEGRAM_TOKEN"
echo -e "⚠️  Сохраните мастер-ключ в безопасном месте!"
