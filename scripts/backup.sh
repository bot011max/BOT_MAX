#!/bin/bash
# Автоматическое резервное копирование с шифрованием

BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/medical_bot_${DATE}.enc"

echo "📦 Создание резервной копии..."

# Бэкап базы данных
docker exec medical-postgres pg_dump -U postgres medical_bot > /tmp/db_dump.sql

# Бэкап конфигурации
tar czf /tmp/config.tar.gz config/

# Бэкап логов
tar czf /tmp/logs.tar.gz /var/log/medical-bot/

# Объединение и шифрование
cat /tmp/db_dump.sql /tmp/config.tar.gz /tmp/logs.tar.gz | \
    openssl enc -aes-256-cbc -salt -pbkdf2 \
    -pass file:secrets/master_key.txt \
    -out ${BACKUP_FILE}

# Очистка
rm /tmp/db_dump.sql /tmp/config.tar.gz /tmp/logs.tar.gz

echo "✅ Резервная копия создана: ${BACKUP_FILE}"
