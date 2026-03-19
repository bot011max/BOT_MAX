#!/bin/bash
# Восстановление из резервной копии

if [ -z "$1" ]; then
    echo "❌ Укажите файл для восстановления"
    echo "Использование: $0 <backup_file>"
    exit 1
fi

BACKUP_FILE=$1

if [ ! -f "$BACKUP_FILE" ]; then
    echo "❌ Файл не найден: $BACKUP_FILE"
    exit 1
fi

echo "🔄 Восстановление из бэкапа: $BACKUP_FILE"
echo "⚠️  ВНИМАНИЕ: Все текущие данные будут перезаписаны!"
read -p "Продолжить? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
fi

# Дешифрование
echo "  🔓 Дешифрование..."
openssl enc -aes-256-cbc -d -salt -pbkdf2 \
    -pass file:secrets/master_key.txt \
    -in ${BACKUP_FILE} -out /tmp/restore.tar

# Распаковка
cd /tmp
tar -xf restore.tar

# Восстановление базы данных
echo "  💾 Восстановление PostgreSQL..."
docker exec -i medical-postgres psql -U postgres medical_bot < db_dump.sql

# Восстановление конфигурации
echo "  ⚙️  Восстановление конфигурации..."
tar -xf config.tar.gz -C / 2>/dev/null || true

# Восстановление SSL сертификатов
echo "  🔐 Восстановление SSL..."
tar -xf ssl.tar.gz -C / 2>/dev/null || true

# Очистка
rm -f /tmp/restore.tar /tmp/db_dump.sql /tmp/config.tar.gz /tmp/ssl.tar.gz

echo "✅ Восстановление завершено!"
