BACKUP_DIR="/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/medical_bot_${DATE}.enc"

echo "📦 Создание резервной копии..."

# Создание директории для бэкапов
mkdir -p $BACKUP_DIR

# Бэкап базы данных
echo "  💾 Бэкап PostgreSQL..."
docker exec medical-postgres pg_dump -U postgres medical_bot > /tmp/db_dump.sql

# Бэкап конфигурации
echo "  ⚙️  Бэкап конфигурации..."
tar -czf /tmp/config.tar.gz config/

# Бэкап логов
echo "  📋 Бэкап логов..."
tar -czf /tmp/logs.tar.gz /var/log/medical-bot/ 2>/dev/null || true

# Бэкап SSL сертификатов
echo "  🔐 Бэкап SSL сертификатов..."
tar -czf /tmp/ssl.tar.gz config/nginx/ssl/ 2>/dev/null || true

# Объединение и шифрование
echo "  🔒 Шифрование бэкапа..."
cat /tmp/db_dump.sql /tmp/config.tar.gz /tmp/logs.tar.gz /tmp/ssl.tar.gz 2>/dev/null | \
    openssl enc -aes-256-cbc -salt -pbkdf2 \
    -pass file:secrets/master_key.txt \
    -out ${BACKUP_FILE}

# Очистка
rm -f /tmp/db_dump.sql /tmp/config.tar.gz /tmp/logs.tar.gz /tmp/ssl.tar.gz

# Проверка размера
SIZE=$(du -h ${BACKUP_FILE} | cut -f1)
echo -e "\n✅ Резервная копия создана: ${BACKUP_FILE} (${SIZE})"

# Удаление старых бэкапов (старше 30 дней)
find $BACKUP_DIR -name "medical_bot_*.enc" -mtime +30 -delete
