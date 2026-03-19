# 🏥 Медицинский бот (Military Grade Security)

[![Go Version](https://img.shields.io/badge/Go-1.22-blue)](https://golang.org/)
[![Security](https://img.shields.io/badge/Security-Military%20Grade-red)](https://github.com/bot011max/medical-bot/security)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## 📋 О ПРОЕКТЕ

Telegram-бот для отслеживания лекарств и записи симптомов с **военным уровнем защиты**.

### 🛡️ БЕЗОПАСНОСТЬ (MILITARY GRADE)

- ✅ **Квантово-устойчивая криптография** - защита от квантовых компьютеров
- ✅ **Многофакторная биометрия** - голос, лицо, отпечатки, ЭКГ
- ✅ **Аппаратное шифрование** - HSM, TPM, Secure Enclave
- ✅ **WAF + IDS + IPS** - обнаружение и блокировка атак в реальном времени
- ✅ **Адаптивный rate limiting** - защита от DDoS и брутфорса
- ✅ **Полный аудит** - все действия логируются в неизменяемом хранилище
- ✅ **Автоматическое восстановление** - бэкапы и откат при компрометации
- ✅ **Мертвая хватка** - самоуничтожение данных при попытке взлома

### ✨ ВОЗМОЖНОСТИ

- 💊 **Напоминания о лекарствах** - никогда не пропусти прием
- 📝 **Голосовой ввод симптомов** - просто расскажи, что болит
- 📅 **Запись к врачу** - интеграция с медицинскими учреждениями
- 📸 **Анализ фото рецептов** - распознавание лекарств по фото
- 🔐 **Безопасное хранение** - все данные зашифрованы

## 🚀 БЫСТРЫЙ СТАРТ

### Предварительные требования

- [Docker](https://docs.docker.com/get-docker/) и Docker Compose
- [Git](https://git-scm.com/downloads)
- Telegram Bot Token (получить у [@BotFather](https://t.me/botfather))

### Установка

```bash
# 1. Клонируем репозиторий
git clone git@github.com:bot011max/medical-bot.git
cd medical-bot

# 2. Инициализируем безопасность (создаются ключи и сертификаты)
chmod +x scripts/init-security.sh
./scripts/init-security.sh

# 3. Редактируем .env.production и добавляем TELEGRAM_TOKEN
nano .env.production

# 4. Запускаем проект
docker-compose -f deployments/docker-compose.yml --env-file .env.production up -d

# 5. Проверяем, что всё работает
curl http://localhost:8080/health
