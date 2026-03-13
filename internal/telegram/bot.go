package telegram

import (
    "log"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "gorm.io/gorm"
)

// TelegramBot главная структура бота
type TelegramBot struct {
    api      *tgbotapi.BotAPI
    db       *gorm.DB
    handlers map[string]CommandHandler
}

// CommandHandler тип для обработчиков команд
type CommandHandler func(update tgbotapi.Update, user *TelegramUser, session *TelegramSession)

// Services временная заглушка (можно удалить, если не используется)
type Services struct {
    // здесь будут сервисы
}

// NewTelegramBot создает нового бота
func NewTelegramBot(token string, db *gorm.DB) (*TelegramBot, error) {
    api, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return nil, err
    }
    
    api.Debug = false
    
    bot := &TelegramBot{
        api:      api,
        db:       db,
        handlers: make(map[string]CommandHandler),
    }
    
    bot.registerHandlers()
    
    log.Printf("Бот @%s успешно запущен", api.Self.UserName)
    
    return bot, nil
}

// ------------------------------------------------------------------------
// РЕГИСТРАЦИЯ ОБРАБОТЧИКОВ
// ------------------------------------------------------------------------

func (b *TelegramBot) registerHandlers() {
    // Команды
    b.handlers["start"] = b.handleStart
    b.handlers["help"] = b.handleHelp
    b.handlers["login"] = b.handleLogin
    b.handlers["medications"] = b.handleMedications
    b.handlers["appointments"] = b.handleAppointments
    b.handlers["symptoms"] = b.handleSymptoms
    b.handlers["today"] = b.handleToday
    b.handlers["settings"] = b.handleSettings
    
    // Callback-обработчики
    b.handlers["menu_medications"] = b.handleMenuMedications
    b.handlers["menu_appointments"] = b.handleMenuAppointments
    b.handlers["menu_symptoms"] = b.handleMenuSymptoms
    b.handlers["menu_analyses"] = b.handleMenuAnalyses
    b.handlers["menu_doctors"] = b.handleMenuDoctors
    b.handlers["menu_settings"] = b.handleMenuSettings
    
    b.handlers["med_add"] = b.handleMedAdd
    b.handlers["med_list"] = b.handleMedList
    b.handlers["med_take"] = b.handleMedTake
    b.handlers["med_stats"] = b.handleMedStats
    
    b.handlers["symptom_"] = b.handleSymptomSelect
    b.handlers["symptom_custom"] = b.handleSymptomCustom
    b.handlers["intensity_"] = b.handleIntensity
    
    b.handlers["settings_link"] = b.handleSettingsLink
    b.handlers["settings_notifications"] = b.handleSettingsNotifications
    b.handlers["settings_profile"] = b.handleSettingsProfile
    
    b.handlers["back_main"] = b.handleBackMain
    b.handlers["confirm_yes_"] = b.handleConfirmYes
    b.handlers["confirm_no_"] = b.handleConfirmNo
    b.handlers["time_"] = b.handleTimeSelect
}

// ------------------------------------------------------------------------
// ЗАПУСК БОТА
// ------------------------------------------------------------------------

// StartWebhook запускает бота в webhook режиме (для продакшена)
func (b *TelegramBot) StartWebhook(webhookURL string) error {
    webhookConfig, err := tgbotapi.NewWebhook(webhookURL)
    if err != nil {
        return err
    }
    
    _, err = b.api.Request(webhookConfig)
    if err != nil {
        return err
    }
    
    return nil
}

// StartPolling запускает бота в polling режиме (для разработки)
func (b *TelegramBot) StartPolling() error {
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    
    updates := b.api.GetUpdatesChan(u)
    
    for update := range updates {
        b.processUpdate(update)
    }
    
    return nil
}

// ------------------------------------------------------------------------
// ОБРАБОТКА ВХОДЯЩИХ СООБЩЕНИЙ
// ------------------------------------------------------------------------

func (b *TelegramBot) processUpdate(update tgbotapi.Update) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Паника при обработке: %v", r)
        }
    }()
    
    telegramID := int64(0)
    
    // Определяем ID пользователя
    if update.Message != nil {
        telegramID = update.Message.From.ID
    } else if update.CallbackQuery != nil {
        telegramID = update.CallbackQuery.From.ID
    }
    
    if telegramID == 0 {
        return
    }
    
    // Получаем или создаем пользователя
    user, err := b.getOrCreateUser(update)
    if err != nil {
        log.Printf("Ошибка получения пользователя: %v", err)
        return
    }
    
    // Получаем или создаем сессию
    session, err := b.getOrCreateSession(telegramID)
    if err != nil {
        log.Printf("Ошибка получения сессии: %v", err)
        return
    }
    
    // Обрабатываем сообщение или callback
    if update.Message != nil {
        b.handleMessage(update, user, session)
    } else if update.CallbackQuery != nil {
        b.handleCallback(update, user, session)
    }
}

func (b *TelegramBot) handleMessage(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    msg := update.Message
    text := msg.Text
    
    // Обработка команд
    if text != "" && text[0] == '/' {
        command := text[1:]
        if handler, ok := b.handlers[command]; ok {
            handler(update, user, session)
        } else {
            b.sendMessage(msg.Chat.ID, "Неизвестная команда. Напишите /help", nil)
        }
        return
    }
    
    // Обработка в зависимости от состояния сессии
    switch session.State {
    case StateAwaitingAuth:
        b.handleAuthInput(update, user, session)
    case StateAwaitingSymptoms:
        b.handleSymptomsInput(update, user, session, text)
    case StateAwaitingMedication:
        b.handleMedicationInput(update, user, session, text)
    case StateAwaitingDosage:
        b.handleDosageInput(update, user, session, text)
    case StateAwaitingFrequency:
        b.handleFrequencyInput(update, user, session, text)
    case StateAwaitingDuration:
        b.handleDurationInput(update, user, session, text)
    default:
        // По умолчанию - показываем меню
        b.showMainMenu(msg.Chat.ID)
    }
}

func (b *TelegramBot) handleCallback(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    callback := update.CallbackQuery
    data := callback.Data
    
    // Отвечаем на callback, чтобы убрать часики
    b.api.Send(tgbotapi.NewCallback(callback.ID, ""))
    
    // Ищем обработчик
    for prefix, handler := range b.handlers {
        if len(data) >= len(prefix) && data[:len(prefix)] == prefix {
            handler(update, user, session)
            return
        }
    }
    
    // Если не нашли, показываем меню
    b.showMainMenu(callback.Message.Chat.ID)
}

// ------------------------------------------------------------------------
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ
// ------------------------------------------------------------------------

func (b *TelegramBot) getOrCreateUser(update tgbotapi.Update) (*TelegramUser, error) {
    var from *tgbotapi.User
    if update.Message != nil {
        from = update.Message.From
    } else if update.CallbackQuery != nil {
        from = update.CallbackQuery.From
    } else {
        return nil, nil
    }
    
    var user TelegramUser
    err := b.db.Where("telegram_id = ?", from.ID).First(&user).Error
    
    if err == nil {
        return &user, nil
    }
    
    // Создаем нового пользователя
    chatID := int64(0)
    if update.Message != nil {
        chatID = update.Message.Chat.ID
    } else if update.CallbackQuery != nil {
        chatID = update.CallbackQuery.Message.Chat.ID
    }
    
    user = TelegramUser{
        TelegramID:   from.ID,
        ChatID:       chatID,
        Username:     from.UserName,
        FirstName:    from.FirstName,
        LastName:     from.LastName,
        IsActive:     true,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }
    
    if err := b.db.Create(&user).Error; err != nil {
        return nil, err
    }
    
    return &user, nil
}

func (b *TelegramBot) getOrCreateSession(telegramID int64) (*TelegramSession, error) {
    var session TelegramSession
    err := b.db.Where("telegram_id = ?", telegramID).First(&session).Error
    
    if err == nil {
        return &session, nil
    }
    
    session = TelegramSession{
        TelegramID: telegramID,
        State:      StateNone,
        UpdatedAt:  time.Now(),
    }
    
    if err := b.db.Create(&session).Error; err != nil {
        return nil, err
    }
    
    return &session, nil
}

func (b *TelegramBot) sendMessage(chatID int64, text string, keyboard interface{}) {
    msg := tgbotapi.NewMessage(chatID, text)
    msg.ParseMode = "HTML"
    
    if keyboard != nil {
        msg.ReplyMarkup = keyboard
    }
    
    if _, err := b.api.Send(msg); err != nil {
        log.Printf("Ошибка отправки сообщения: %v", err)
    }
}

func (b *TelegramBot) editMessage(chatID int64, messageID int, text string, keyboard interface{}) {
    msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
    msg.ParseMode = "HTML"
    
    if keyboard != nil {
        if kb, ok := keyboard.(tgbotapi.InlineKeyboardMarkup); ok {
            replyMarkup := &kb
            msg.ReplyMarkup = replyMarkup
        }
    }
    
    if _, err := b.api.Send(msg); err != nil {
        log.Printf("Ошибка редактирования сообщения: %v", err)
    }
}

func (b *TelegramBot) requireAuth(chatID int64) {
    msg := "🔐 Для использования этой функции необходимо авторизоваться.\n\nИспользуйте /login"
    b.sendMessage(chatID, msg, nil)
}

func (b *TelegramBot) showMainMenu(chatID int64) {
    msg := "🏠 Главное меню\n\nВыберите раздел:"
    b.sendMessage(chatID, msg, MainMenu())
}

// ------------------------------------------------------------------------
// ЗАГЛУШКИ ДЛЯ НЕДОСТАЮЩИХ МЕТОДОВ (ОБРАБОТЧИКИ)
// ------------------------------------------------------------------------

func (b *TelegramBot) handleStart(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    welcome := "👋 Добро пожаловать!\n\nИспользуйте /help для списка команд"
    b.sendMessage(chatID, welcome, MainMenu())
}

func (b *TelegramBot) handleHelp(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    help := "📋 Доступные команды:\n/start\n/help\n/login"
    b.sendMessage(chatID, help, nil)
}

func (b *TelegramBot) handleLogin(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "🔐 Функция входа в разработке", nil)
}

func (b *TelegramBot) handleMedications(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "💊 Лекарства (в разработке)", nil)
}

func (b *TelegramBot) handleAppointments(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "📅 Приемы (в разработке)", nil)
}

func (b *TelegramBot) handleSymptoms(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "📝 Симптомы (в разработке)", nil)
}

func (b *TelegramBot) handleToday(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "📅 Сегодня (в разработке)", nil)
}

func (b *TelegramBot) handleSettings(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "⚙️ Настройки (в разработке)", SettingsMenu())
}

func (b *TelegramBot) handleMenuMedications(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    b.handleMedications(update, user, session)
}

func (b *TelegramBot) handleMenuAppointments(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    b.handleAppointments(update, user, session)
}

func (b *TelegramBot) handleMenuSymptoms(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    b.handleSymptoms(update, user, session)
}

func (b *TelegramBot) handleMenuAnalyses(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "🔬 Анализы (в разработке)", nil)
}

func (b *TelegramBot) handleMenuDoctors(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "👨‍⚕️ Врачи (в разработке)", nil)
}

func (b *TelegramBot) handleMenuSettings(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    b.handleSettings(update, user, session)
}

func (b *TelegramBot) handleMedAdd(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "➕ Добавление лекарства (в разработке)", nil)
}

func (b *TelegramBot) handleMedList(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "📋 Список лекарств (в разработке)", nil)
}

func (b *TelegramBot) handleMedTake(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "✅ Отметка приема (в разработке)", nil)
}

func (b *TelegramBot) handleMedStats(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "📊 Статистика (в разработке)", nil)
}

func (b *TelegramBot) handleSymptomSelect(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "📝 Выбор симптома (в разработке)", nil)
}

func (b *TelegramBot) handleSymptomCustom(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "✏️ Свой симптом (в разработке)", nil)
}

func (b *TelegramBot) handleIntensity(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "📊 Оценка интенсивности (в разработке)", nil)
}

func (b *TelegramBot) handleSymptomsInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession, text string) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "📝 Симптом записан (в разработке)", nil)
}

func (b *TelegramBot) handleMedicationInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession, text string) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "💊 Лекарство добавлено (в разработке)", nil)
}

func (b *TelegramBot) handleDosageInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession, text string) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "💊 Дозировка сохранена (в разработке)", nil)
}

func (b *TelegramBot) handleFrequencyInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession, text string) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "🕐 Частота сохранена (в разработке)", nil)
}

func (b *TelegramBot) handleDurationInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession, text string) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "📅 Длительность сохранена (в разработке)", nil)
}

func (b *TelegramBot) handleAuthInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    b.sendMessage(chatID, "🔐 Авторизация (в разработке)", nil)
}

func (b *TelegramBot) handleSettingsLink(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "🔗 Привязка аккаунта (в разработке)", nil)
}

func (b *TelegramBot) handleSettingsNotifications(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "🔔 Уведомления (в разработке)", nil)
}

func (b *TelegramBot) handleSettingsProfile(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "👤 Профиль (в разработке)", nil)
}

func (b *TelegramBot) handleBackMain(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.showMainMenu(chatID)
}

func (b *TelegramBot) handleConfirmYes(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "✅ Подтверждено", nil)
}

func (b *TelegramBot) handleConfirmNo(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "❌ Отменено", nil)
}

func (b *TelegramBot) handleTimeSelect(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "🕐 Время выбрано (в разработке)", nil)
}
