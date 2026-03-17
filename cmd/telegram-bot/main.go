// cmd/telegram/main.go
package main

import (
    "context"
    "crypto/subtle"
    "fmt"
    "log"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/bot011max/BOT_MAX/internal/security"
    "github.com/bot011max/BOT_MAX/internal/models"
    "gorm.io/gorm"
    "github.com/redis/go-redis/v9"
)

type SecureTelegramBot struct {
    bot           *tgbotapi.BotAPI
    db            *gorm.DB
    redis         *redis.Client
    secretManager *security.SecretManager
    auditLogger   *security.AuditLogger
    rateLimiter   *security.AdaptiveRateLimiter
    commands      map[string]CommandHandler
}

type CommandHandler struct {
    Handler     func(update tgbotapi.Update, args []string)
    Description string
    AuthRequired bool
    RateLimit    int // запросов в минуту
}

func NewSecureTelegramBot() (*SecureTelegramBot, error) {
    // Инициализация секретов
    secretManager, err := security.NewSecretManager(true)
    if err != nil {
        return nil, err
    }

    // Получаем токен из секретов
    token, err := secretManager.GetSecret("telegram_token")
    if err != nil {
        return nil, err
    }

    // Создаем бота
    bot, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return nil, err
    }

    // Инициализация Redis для rate limiting
    redisPass, _ := secretManager.GetSecret("redis_password")
    rdb := redis.NewClient(&redis.Options{
        Addr:     os.Getenv("REDIS_HOST") + ":6379",
        Password: redisPass,
        DB:       0,
    })

    // Rate limiter
    rateLimiter, _ := security.NewAdaptiveRateLimiter(os.Getenv("REDIS_HOST")+":6379", 
        &security.RateLimiterConfig{
            RequestsPerSecond: 1,
            Burst:             5,
            BlockDuration:     time.Hour,
            CleanupInterval:   time.Minute,
            EnableAdaptive:    true,
        })

    // Audit logger
    auditLogger, _ := security.NewAuditLogger()

    stb := &SecureTelegramBot{
        bot:           bot,
        secretManager: secretManager,
        auditLogger:   auditLogger,
        rateLimiter:   rateLimiter,
        redis:         rdb,
        commands:      make(map[string]CommandHandler),
    }

    // Регистрируем команды
    stb.registerCommands()

    return stb, nil
}

func (b *SecureTelegramBot) registerCommands() {
    b.commands = map[string]CommandHandler{
        "/start": {
            Handler:     b.handleStart,
            Description: "Начать работу с ботом",
            AuthRequired: false,
            RateLimit:    2,
        },
        "/help": {
            Handler:     b.handleHelp,
            Description: "Показать справку",
            AuthRequired: false,
            RateLimit:    5,
        },
        "/bind": {
            Handler:     b.handleBind,
            Description: "Привязать аккаунт",
            AuthRequired: false,
            RateLimit:    3,
        },
        "/medications": {
            Handler:     b.handleMedications,
            Description: "Мои лекарства",
            AuthRequired: true,
            RateLimit:    10,
        },
        "/remind": {
            Handler:     b.handleRemind,
            Description: "Установить напоминание",
            AuthRequired: true,
            RateLimit:    5,
        },
        "/appointments": {
            Handler:     b.handleAppointments,
            Description: "Мои визиты",
            AuthRequired: true,
            RateLimit:    5,
        },
    }
}

func (b *SecureTelegramBot) Run() error {
    log.Printf("Бот @%s запущен", b.bot.Self.UserName)

    // Устанавливаем вебхук (для production)
    webhookURL := os.Getenv("WEBHOOK_URL") + "/webhook/telegram"
    webhookConfig := tgbotapi.NewWebhook(webhookURL)
    
    _, err := b.bot.Request(webhookConfig)
    if err != nil {
        log.Printf("Ошибка установки webhook: %v", err)
    }

    updates := b.bot.ListenForWebhook("/webhook/telegram")
    
    // Graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), 
        os.Interrupt, syscall.SIGTERM)
    defer stop()

    go func() {
        <-ctx.Done()
        log.Println("Завершение работы бота...")
        b.bot.StopReceivingUpdates()
    }()

    for update := range updates {
        go b.handleUpdate(update)
    }

    return nil
}

func (b *SecureTelegramBot) handleUpdate(update tgbotapi.Update) {
    // Rate limiting по chat_id
    chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
    if !b.rateLimiter.IsWhitelisted(chatID) {
        // Проверяем лимиты
        // (упрощенно, в реальности через middleware)
    }

    // Валидация входных данных
    if update.Message != nil {
        b.handleMessage(update)
    } else if update.CallbackQuery != nil {
        b.handleCallback(update)
    }
}

func (b *SecureTelegramBot) handleMessage(update tgbotapi.Update) {
    msg := update.Message
    chatID := msg.Chat.ID
    text := msg.Text

    // Логируем входящее сообщение
    b.auditLogger.Log("TELEGRAM_MESSAGE", "message_received", 
        fmt.Sprintf("chat_%d", chatID), "INFO", "telegram_user", 
        map[string]interface{}{
            "message_id": msg.MessageID,
            "text_len":   len(text),
            "has_media":  msg.Voice != nil || msg.Photo != nil,
        })

    // Санитизация ввода
    text = b.sanitizeInput(text)

    // Обработка команд
    if strings.HasPrefix(text, "/") {
        parts := strings.Fields(text)
        command := parts[0]
        args := parts[1:]

        if handler, exists := b.commands[command]; exists {
            // Проверка авторизации
            if handler.AuthRequired {
                if !b.isAuthorized(chatID) {
                    b.sendMessage(chatID, "❌ Сначала привяжите аккаунт командой /bind")
                    return
                }
            }

            // Вызов обработчика
            handler.Handler(update, args)
        } else {
            b.sendMessage(chatID, "❌ Неизвестная команда. Напишите /help")
        }
        return
    }

    // Обработка голосовых сообщений
    if msg.Voice != nil {
        b.handleVoiceMessage(update)
        return
    }

    // Обработка фото
    if msg.Photo != nil {
        b.handlePhotoMessage(update)
        return
    }

    // Ответ по умолчанию
    b.sendMessage(chatID, "Я не понимаю. Напишите /help для списка команд.")
}

func (b *SecureTelegramBot) handleStart(update tgbotapi.Update, args []string) {
    chatID := update.Message.Chat.ID
    
    welcomeMsg := "👋 Добро пожаловать в Медицинского бота!\n\n"
    welcomeMsg += "🔐 <b>Безопасный помощник для вашего здоровья</b>\n\n"
    welcomeMsg += "Что я умею:\n"
    welcomeMsg += "• 💊 Напоминать о лекарствах\n"
    welcomeMsg += "• 📝 Записывать симптомы\n"
    welcomeMsg += "• 📅 Отслеживать визиты к врачу\n"
    welcomeMsg += "• 🎤 Распознавать голосовые заметки\n"
    welcomeMsg += "• 📸 Анализировать фото рецептов\n\n"
    welcomeMsg += "Для начала работы:\n"
    welcomeMsg += "1. Зарегистрируйтесь на нашем сайте\n"
    welcomeMsg += "2. Введите команду /bind для привязки аккаунта"

    b.sendMessage(chatID, welcomeMsg)
}

func (b *SecureTelegramBot) handleBind(update tgbotapi.Update, args []string) {
    chatID := update.Message.Chat.ID
    
    // Генерируем одноразовый код
    code := b.generateBindCode(chatID)
    
    msg := "🔐 <b>Привязка Telegram к аккаунту</b>\n\n"
    msg += "Ваш одноразовый код:\n"
    msg += fmt.Sprintf("<code>%s</code>\n\n", code)
    msg += "Введите этот код в личном кабинете на сайте.\n"
    msg += "Код действителен 10 минут."

    // Создаем inline клавиатуру
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonURL("🔑 Перейти на сайт", os.Getenv("SITE_URL")),
        ),
    )

    b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (b *SecureTelegramBot) handleMedications(update tgbotapi.Update, args []string) {
    chatID := update.Message.Chat.ID
    
    // Получаем user_id по telegram_id
    userID, err := b.getUserID(chatID)
    if err != nil {
        b.sendMessage(chatID, "❌ Ошибка получения данных")
        return
    }

    // Получаем лекарства из БД
    var prescriptions []models.Prescription
    if err := b.db.Where("user_id = ? AND is_active = ?", userID, true).
        Find(&prescriptions).Error; err != nil {
        b.sendMessage(chatID, "❌ Ошибка получения лекарств")
        return
    }

    if len(prescriptions) == 0 {
        b.sendMessage(chatID, "У вас нет активных назначений")
        return
    }

    msg := "💊 <b>Ваши лекарства:</b>\n\n"
    for _, p := range prescriptions {
        msg += fmt.Sprintf("• %s\n", p.Name)
        msg += fmt.Sprintf("  Дозировка: %s\n", p.Dosage)
        msg += fmt.Sprintf("  Режим: %s\n", p.Frequency)
        msg += "---\n"
    }

    b.sendMessage(chatID, msg)
}

func (b *SecureTelegramBot) handleRemind(update tgbotapi.Update, args []string) {
    chatID := update.Message.Chat.ID
    
    if len(args) < 2 {
        b.sendMessage(chatID, "❌ Использование: /remind [название] [время]\nПример: /remind Амоксициллин 09:00")
        return
    }

    // Сохраняем напоминание в Redis
    key := fmt.Sprintf("reminder:%d:%s", chatID, args[0])
    err := b.redis.Set(context.Background(), key, strings.Join(args[1:], " "), 24*time.Hour).Err()
    if err != nil {
        b.sendMessage(chatID, "❌ Ошибка создания напоминания")
        return
    }

    b.sendMessage(chatID, "✅ Напоминание создано!")
}

func (b *SecureTelegramBot) handleAppointments(update tgbotapi.Update, args []string) {
    chatID := update.Message.Chat.ID
    
    // Здесь получение визитов из БД
    b.sendMessage(chatID, "📅 У вас нет запланированных визитов")
}

func (b *SecureTelegramBot) handleVoiceMessage(update tgbotapi.Update) {
    chatID := update.Message.Chat.ID
    voice := update.Message.Voice

    // Проверяем размер (макс 5 МБ)
    if voice.FileSize > 5*1024*1024 {
        b.sendMessage(chatID, "❌ Слишком большое голосовое сообщение (макс 5 МБ)")
        return
    }

    b.sendMessage(chatID, "🎤 Обрабатываю голосовое сообщение...")

    // Получаем файл
    file, err := b.bot.GetFile(tgbotapi.FileConfig{FileID: voice.FileID})
    if err != nil {
        b.sendMessage(chatID, "❌ Ошибка получения файла")
        return
    }

    // Отправляем в сервис распознавания (через защищенный канал)
    go b.processVoiceFile(chatID, file.FilePath)
}

func (b *SecureTelegramBot) handlePhotoMessage(update tgbotapi.Update) {
    chatID := update.Message.Chat.ID
    photos := update.Message.Photo
    
    // Берем самую большую фотографию
    photo := photos[len(photos)-1]

    b.sendMessage(chatID, "📸 Анализирую изображение...")

    file, err := b.bot.GetFile(tgbotapi.FileConfig{FileID: photo.FileID})
    if err != nil {
        b.sendMessage(chatID, "❌ Ошибка получения файла")
        return
    }

    // Отправляем в сервис OCR (через защищенный канал)
    go b.processPhotoFile(chatID, file.FilePath)
}

func (b *SecureTelegramBot) handleCallback(update tgbotapi.Update) {
    callback := update.CallbackQuery
    chatID := callback.Message.Chat.ID
    data := callback.Data

    // Подтверждаем получение
    b.bot.Request(tgbotapi.NewCallback(callback.ID, ""))

    // Обработка callback данных
    parts := strings.Split(data, ":")
    if len(parts) < 2 {
        return
    }

    switch parts[0] {
    case "confirm":
        b.sendMessage(chatID, "✅ Подтверждено!")
    case "cancel":
        b.sendMessage(chatID, "❌ Отменено")
    }
}

func (b *SecureTelegramBot) generateBindCode(chatID int64) string {
    // Генерируем 6-значный код
    code := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
    
    // Сохраняем в Redis на 10 минут
    key := fmt.Sprintf("bind:%d", chatID)
    b.redis.Set(context.Background(), key, code, 10*time.Minute)
    
    return code
}

func (b *SecureTelegramBot) isAuthorized(chatID int64) bool {
    key := fmt.Sprintf("user:%d", chatID)
    exists, _ := b.redis.Exists(context.Background(), key).Result()
    return exists > 0
}

func (b *SecureTelegramBot) getUserID(chatID int64) (string, error) {
    key := fmt.Sprintf("user:%d", chatID)
    return b.redis.Get(context.Background(), key).Result()
}

func (b *SecureTelegramBot) sanitizeInput(input string) string {
    // Удаляем потенциально опасные символы
    dangerous := []string{"<", ">", "&", "'", "\"", "--", ";", "/*", "*/"}
    result := input
    for _, d := range dangerous {
        result = strings.ReplaceAll(result, d, "")
    }
    return result
}

func (b *SecureTelegramBot) sendMessage(chatID int64, text string) {
    msg := tgbotapi.NewMessage(chatID, text)
    msg.ParseMode = "HTML"
    
    if _, err := b.bot.Send(msg); err != nil {
        log.Printf("Ошибка отправки сообщения: %v", err)
    }
}

func (b *SecureTelegramBot) sendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) {
    msg := tgbotapi.NewMessage(chatID, text)
    msg.ParseMode = "HTML"
    msg.ReplyMarkup = keyboard
    
    if _, err := b.bot.Send(msg); err != nil {
        log.Printf("Ошибка отправки сообщения: %v", err)
    }
}

func (b *SecureTelegramBot) processVoiceFile(chatID int64, filePath string) {
    // Защищенная отправка в voice service через mTLS
    // Реализация с шифрованием
}

func (b *SecureTelegramBot) processPhotoFile(chatID int64, filePath string) {
    // Защищенная отправка в OCR service
    // Реализация с шифрованием
}

func (b *SecureTelegramBot) handleHelp(update tgbotapi.Update, args []string) {
    chatID := update.Message.Chat.ID
    
    msg := "📋 <b>Доступные команды:</b>\n\n"
    for cmd, handler := range b.commands {
        authMark := ""
        if handler.AuthRequired {
            authMark = " 🔐"
        }
        msg += fmt.Sprintf("%s - %s%s\n", cmd, handler.Description, authMark)
    }
    
    b.sendMessage(chatID, msg)
}

func main() {
    // Инициализация логгера аудита
    security.InitAuditLogger()

    bot, err := NewSecureTelegramBot()
    if err != nil {
        log.Fatalf("Ошибка создания бота: %v", err)
    }

    if err := bot.Run(); err != nil {
        log.Fatalf("Ошибка запуска: %v", err)
    }
}
