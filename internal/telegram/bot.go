package telegram

import (
    "fmt"
    "log"
    "strings"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "github.com/redis/go-redis/v9"

    "github.com/bot011max/medical-bot/internal/models"
    "github.com/bot011max/medical-bot/internal/repository"
    "github.com/bot011max/medical-bot/internal/service"
    "github.com/bot011max/medical-bot/internal/security"
)

// Bot - структура Telegram бота
type Bot struct {
    api          *tgbotapi.BotAPI
    userRepo     *repository.UserRepository
    subRepo      *repository.SubscriptionRepository
    authService  *service.AuthService
    redis        *redis.Client
    audit        *security.AuditLogger
    commands     map[string]Command
}

// Command - структура команды
type Command struct {
    Handler     func(update tgbotapi.Update, user *TelegramUser)
    Description string
    AuthRequired bool
    RoleAllowed  []string
}

// TelegramUser - пользователь Telegram
type TelegramUser struct {
    ID           int64
    UserID       string
    Username     string
    FirstName    string
    LastName     string
    State        string
    TempData     map[string]interface{}
}

// NewBot - создание нового бота
func NewBot(token string, userRepo *repository.UserRepository, subRepo *repository.SubscriptionRepository,
            authService *service.AuthService, redis *redis.Client, audit *security.AuditLogger) (*Bot, error) {
    
    api, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return nil, err
    }
    
    bot := &Bot{
        api:         api,
        userRepo:    userRepo,
        subRepo:     subRepo,
        authService: authService,
        redis:       redis,
        audit:       audit,
        commands:    make(map[string]Command),
    }
    
    bot.registerCommands()
    
    log.Printf("Telegram bot @%s started", api.Self.UserName)
    
    return bot, nil
}

// registerCommands - регистрация всех команд
func (b *Bot) registerCommands() {
    // Публичные команды
    b.commands["start"] = Command{
        Handler:      b.handleStart,
        Description:  "Начать работу с ботом",
        AuthRequired: false,
    }
    
    b.commands["help"] = Command{
        Handler:      b.handleHelp,
        Description:  "Показать справку",
        AuthRequired: false,
    }
    
    b.commands["login"] = Command{
        Handler:      b.handleLogin,
        Description:  "Войти в аккаунт",
        AuthRequired: false,
    }
    
    // Команды для авторизованных
    b.commands["profile"] = Command{
        Handler:      b.handleProfile,
        Description:  "Мой профиль",
        AuthRequired: true,
    }
    
    b.commands["medications"] = Command{
        Handler:      b.handleMedications,
        Description:  "Мои лекарства",
        AuthRequired: true,
        RoleAllowed:  []string{"patient", "doctor", "clinic"},
    }
    
    b.commands["remind"] = Command{
        Handler:      b.handleRemind,
        Description:  "Установить напоминание",
        AuthRequired: true,
        RoleAllowed:  []string{"patient"},
    }
    
    b.commands["appointments"] = Command{
        Handler:      b.handleAppointments,
        Description:  "Мои приемы",
        AuthRequired: true,
    }
    
    b.commands["symptoms"] = Command{
        Handler:      b.handleSymptoms,
        Description:  "Записать симптомы",
        AuthRequired: true,
        RoleAllowed:  []string{"patient"},
    }
    
    b.commands["patients"] = Command{
        Handler:      b.handlePatients,
        Description:  "Мои пациенты",
        AuthRequired: true,
        RoleAllowed:  []string{"doctor", "clinic"},
    }
    
    b.commands["subscribe"] = Command{
        Handler:      b.handleSubscribe,
        Description:  "Управление подпиской",
        AuthRequired: true,
    }
    
    b.commands["analytics"] = Command{
        Handler:      b.handleAnalytics,
        Description:  "Аналитика",
        AuthRequired: true,
        RoleAllowed:  []string{"doctor", "clinic"},
    }
}

// StartPolling - запуск бота в режиме polling
func (b *Bot) StartPolling() {
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    
    updates := b.api.GetUpdatesChan(u)
    
    for update := range updates {
        if update.Message != nil {
            b.handleMessage(update)
        } else if update.CallbackQuery != nil {
            b.handleCallback(update)
        }
    }
}

// handleMessage - обработка текстовых сообщений
func (b *Bot) handleMessage(update tgbotapi.Update) {
    msg := update.Message
    chatID := msg.Chat.ID
    text := strings.TrimSpace(msg.Text)
    
    // Получаем или создаем пользователя
    tgUser := b.getOrCreateTelegramUser(update)
    
    // Проверка на команду
    if strings.HasPrefix(text, "/") {
        command := strings.ToLower(strings.Split(text, " ")[0][1:])
        
        if cmd, exists := b.commands[command]; exists {
            // Проверка авторизации
            if cmd.AuthRequired && tgUser.UserID == "" {
                b.sendMessage(chatID, "🔐 Для этой команды нужно авторизоваться. Используйте /login")
                return
            }
            
            // Проверка роли (если указана)
            if len(cmd.RoleAllowed) > 0 && tgUser.UserID != "" {
                allowed := false
                user, _ := b.userRepo.FindByID(tgUser.UserID)
                for _, role := range cmd.RoleAllowed {
                    if user.Role == role {
                        allowed = true
                        break
                    }
                }
                if !allowed {
                    b.sendMessage(chatID, "⛔ У вас нет прав для этой команды")
                    return
                }
            }
            
            cmd.Handler(update, tgUser)
        } else {
            b.sendMessage(chatID, "❌ Неизвестная команда. Напишите /help")
        }
        return
    }
    
    // Обработка состояний диалога
    if tgUser.State != "" {
        b.handleState(update, tgUser)
        return
    }
    
    // По умолчанию
    b.sendMessage(chatID, "Я не понимаю. Используйте /help для списка команд")
}

// handleStart - команда /start
func (b *Bot) handleStart(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    
    welcome := "👋 Добро пожаловать в Медицинского бота!\n\n"
    welcome += "🤖 Я помогу вам:\n"
    welcome += "• 💊 Отслеживать прием лекарств\n"
    welcome += "• 📝 Записывать симптомы\n"
    welcome += "• 📅 Напоминать о визитах к врачу\n"
    welcome += "• 📊 Анализировать ваше здоровье\n\n"
    
    if user.UserID == "" {
        welcome += "Для начала работы авторизуйтесь:\n/login - войти в аккаунт\n"
    } else {
        dbUser, _ := b.userRepo.FindByID(user.UserID)
        welcome += fmt.Sprintf("✅ Вы авторизованы как %s %s (%s)\n", 
            dbUser.FirstName, dbUser.LastName, dbUser.Role)
        welcome += "/help - список команд"
    }
    
    b.sendMessage(chatID, welcome)
}

// handleHelp - команда /help
func (b *Bot) handleHelp(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    
    help := "📋 <b>Доступные команды</b>\n\n"
    
    for cmd, info := range b.commands {
        if !info.AuthRequired || user.UserID != "" {
            help += fmt.Sprintf("/%s - %s\n", cmd, info.Description)
        }
    }
    
    b.sendMessage(chatID, help)
}

// handleLogin - команда /login
func (b *Bot) handleLogin(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    
    if user.UserID != "" {
        b.sendMessage(chatID, "✅ Вы уже авторизованы")
        return
    }
    
    // Генерируем временный код
    code := security.GenerateNumericCode(6)
    
    // Сохраняем в Redis на 10 минут
    b.redis.Set(update.Context(), fmt.Sprintf("tg_auth:%d", chatID), code, 10*time.Minute)
    
    msg := "🔐 <b>Авторизация</b>\n\n"
    msg += "Ваш код: <code>" + code + "</code>\n\n"
    msg += "Введите этот код в веб-версии в разделе 'Привязать Telegram'\n"
    msg += "Код действителен 10 минут"
    
    b.sendMessage(chatID, msg)
}

// handleProfile - команда /profile
func (b *Bot) handleProfile(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    
    dbUser, err := b.userRepo.FindByID(user.UserID)
    if err != nil {
        b.sendMessage(chatID, "❌ Ошибка получения профиля")
        return
    }
    
    sub, _ := b.subRepo.GetActiveByUserIDStr(user.UserID)
    
    profile := fmt.Sprintf("👤 <b>Профиль</b>\n\n")
    profile += fmt.Sprintf("Имя: %s %s\n", dbUser.FirstName, dbUser.LastName)
    profile += fmt.Sprintf("Email: %s\n", dbUser.Email)
    profile += fmt.Sprintf("Роль: %s\n", dbUser.Role)
    
    if sub != nil {
        profile += fmt.Sprintf("\n📦 <b>Подписка: %s</b>\n", sub.Tier)
        profile += fmt.Sprintf("Пациентов: %d\n", sub.MaxPatients)
        profile += fmt.Sprintf("Напоминаний: %d/мес\n", sub.MaxReminders)
        profile += fmt.Sprintf("Действует до: %s", sub.ExpiresAt.Format("02.01.2006"))
    }
    
    b.sendMessage(chatID, profile)
}

// handleMedications - команда /medications
func (b *Bot) handleMedications(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    
    // Получаем лекарства из БД
    // TODO: добавить репозиторий лекарств
    
    meds := "💊 <b>Ваши лекарства</b>\n\n"
    meds += "• Парацетамол 500 мг - 3 раза/день\n"
    meds += "• Амоксициллин 500 мг - 2 раза/день\n"
    meds += "• Витамин D - 1 раз/день\n\n"
    meds += "Последний прием: 2 часа назад"
    
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("✅ Отметить прием", "med_take"),
            tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "med_stats"),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("➕ Добавить лекарство", "med_add"),
        ),
    )
    
    b.sendMessageWithKeyboard(chatID, meds, keyboard)
}

// handleRemind - команда /remind
func (b *Bot) handleRemind(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    args := strings.Split(update.Message.Text, " ")[1:]
    
    if len(args) < 2 {
        b.sendMessage(chatID, "❌ Использование: /remind [название] [время]\nПример: /remind Амоксициллин 09:00")
        return
    }
    
    // Сохраняем напоминание
    // TODO: сохранить в БД
    
    b.sendMessage(chatID, "✅ Напоминание создано!")
}

// handleAppointments - команда /appointments
func (b *Bot) handleAppointments(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    
    appointments := "📅 <b>Ближайшие приемы</b>\n\n"
    appointments += "• 25 марта 15:00 - Терапевт\n"
    appointments += "• 28 марта 10:30 - Стоматолог\n\n"
    appointments += "Показаны 2 из 5 записей"
    
    b.sendMessage(chatID, appointments)
}

// handleSymptoms - команда /symptoms
func (b *Bot) handleSymptoms(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("🤕 Головная боль", "symptom_headache"),
            tgbotapi.NewInlineKeyboardButtonData("🌡️ Температура", "symptom_temp"),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("🤧 Кашель", "symptom_cough"),
            tgbotapi.NewInlineKeyboardButtonData("😷 Боль в горле", "symptom_throat"),
        ),
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("✏️ Свой вариант", "symptom_custom"),
        ),
    )
    
    b.sendMessageWithKeyboard(chatID, "📝 Выберите симптом:", keyboard)
}

// handlePatients - команда /patients (для врачей)
func (b *Bot) handlePatients(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    
    patients := "👥 <b>Мои пациенты</b>\n\n"
    patients += "• Иван Петров - последний визит 2 дня назад\n"
    patients += "• Мария Сидорова - последний визит 5 дней назад\n"
    patients += "• Алексей Иванов - новое назначение\n\n"
    patients += "Всего: 12 пациентов"
    
    b.sendMessage(chatID, patients)
}

// handleSubscribe - команда /subscribe
func (b *Bot) handleSubscribe(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    
    sub, _ := b.subRepo.GetActiveByUserIDStr(user.UserID)
    
    msg := "💳 <b>Управление подпиской</b>\n\n"
    
    if sub != nil {
        msg += fmt.Sprintf("Текущий тариф: <b>%s</b>\n", sub.Tier)
        msg += fmt.Sprintf("Действует до: %s\n", sub.ExpiresAt.Format("02.01.2006"))
        msg += fmt.Sprintf("Автопродление: %v\n\n", sub.AutoRenew)
    }
    
    msg += "Доступные тарифы:\n"
    msg += "• Free - 0₽/мес (1 пациент, 10 напоминаний)\n"
    msg += "• Patient Pro - 299₽/мес (5 пациентов, безлимит)\n"
    msg += "• Doctor Pro - 1490₽/мес (100 пациентов, аналитика)\n"
    msg += "• Clinic - 9900₽/мес (1000+ пациентов, API)\n\n"
    msg += "Для смены тарифа перейдите в веб-версию"
    
    keyboard := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonURL("🔑 Перейти на сайт", "https://your-domain.com/subscription"),
        ),
    )
    
    b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

// handleAnalytics - команда /analytics
func (b *Bot) handleAnalytics(update tgbotapi.Update, user *TelegramUser) {
    chatID := update.Message.Chat.ID
    
    analytics := "📊 <b>Аналитика</b>\n\n"
    analytics += "За последние 30 дней:\n"
    analytics += "• Назначений: 45\n"
    analytics += "• Выполнено: 38 (84%)\n"
    analytics += "• Пропущено: 7 (16%)\n\n"
    analytics += "• Активных пациентов: 23\n"
    analytics += "• Новых пациентов: 5\n\n"
    analytics += "• Средняя приверженность: 82%"
    
    b.sendMessage(chatID, analytics)
}

// handleCallback - обработка нажатий на inline-кнопки
func (b *Bot) handleCallback(update tgbotapi.Update) {
    callback := update.CallbackQuery
    chatID := callback.Message.Chat.ID
    
    data := callback.Data
    
    // Отвечаем на callback
    b.api.Send(tgbotapi.NewCallback(callback.ID, ""))
    
    switch data {
    case "med_take":
        b.api.Send(tgbotapi.NewMessage(chatID, "✅ Прием отмечен!"))
    case "med_stats":
        b.api.Send(tgbotapi.NewMessage(chatID, "📊 Статистика: принято 12 из 15"))
    case "med_add":
        b.api.Send(tgbotapi.NewMessage(chatID, "➕ Введите название лекарства:"))
    case "symptom_headache":
        b.api.Send(tgbotapi.NewMessage(chatID, "🤕 Оцените интенсивность (1-10):"))
    case "symptom_temp":
        b.api.Send(tgbotapi.NewMessage(chatID, "🌡️ Введите температуру:"))
    }
}

// handleState - обработка состояний диалога
func (b *Bot) handleState(update tgbotapi.Update, user *TelegramUser) {
    // TODO: реализовать FSM для многошаговых диалогов
}

// getOrCreateTelegramUser - получение или создание пользователя Telegram
func (b *Bot) getOrCreateTelegramUser(update tgbotapi.Update) *TelegramUser {
    var tgID int64
    var username, firstName, lastName string
    
    if update.Message != nil {
        tgID = update.Message.From.ID
        username = update.Message.From.UserName
        firstName = update.Message.From.FirstName
        lastName = update.Message.From.LastName
    } else if update.CallbackQuery != nil {
        tgID = update.CallbackQuery.From.ID
        username = update.CallbackQuery.From.UserName
        firstName = update.CallbackQuery.From.FirstName
        lastName = update.CallbackQuery.From.LastName
    }
    
    // Получаем из Redis или создаем
    key := fmt.Sprintf("tg_user:%d", tgID)
    
    // TODO: получать из Redis
    
    user := &TelegramUser{
        ID:        tgID,
        Username:  username,
        FirstName: firstName,
        LastName:  lastName,
        State:     "",
        TempData:  make(map[string]interface{}),
    }
    
    return user
}

// sendMessage - отправка обычного сообщения
func (b *Bot) sendMessage(chatID int64, text string) {
    msg := tgbotapi.NewMessage(chatID, text)
    msg.ParseMode = "HTML"
    b.api.Send(msg)
}

// sendMessageWithKeyboard - отправка сообщения с клавиатурой
func (b *Bot) sendMessageWithKeyboard(chatID int64, text string, keyboard interface{}) {
    msg := tgbotapi.NewMessage(chatID, text)
    msg.ParseMode = "HTML"
    msg.ReplyMarkup = keyboard
    b.api.Send(msg)
}
