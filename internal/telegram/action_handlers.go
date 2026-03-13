 package telegram

import (
    "encoding/json"
    "fmt"
    "strings"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Добавление лекарства
func (b *TelegramBot) handleMedAdd(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    
    if user.UserID == "" {
        b.requireAuth(chatID)
        return
    }
    
    session.State = StateAwaitingMedication
    session.TempData = ""
    b.db.Save(session)
    
    msg := "💊 Введите название лекарства:"
    b.sendMessage(chatID, msg, nil)
}

func (b *TelegramBot) handleMedicationInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    text := update.Message.Text
    
    session.State = StateAwaitingDosage
    session.TempData = text
    b.db.Save(session)
    
    msg := fmt.Sprintf("💊 Лекарство: <b>%s</b>\n\n", text)
    msg += "Введите дозировку (например: 500 мг):"
    b.sendMessage(chatID, msg, nil)
}

func (b *TelegramBot) handleDosageInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    dosage := update.Message.Text
    medication := session.TempData
    
    session.State = StateAwaitingFrequency
    session.TempData = medication + "|" + dosage
    b.db.Save(session)
    
    msg := fmt.Sprintf("💊 <b>%s</b> %s\n\n", medication, dosage)
    msg += "Как часто принимать? (например: 3 раза в день)"
    b.sendMessage(chatID, msg, nil)
}

func (b *TelegramBot) handleFrequencyInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    frequency := update.Message.Text
    data := strings.Split(session.TempData, "|")
    medication := data[0]
    dosage := data[1]
    
    session.State = StateAwaitingDuration
    session.TempData = medication + "|" + dosage + "|" + frequency
    b.db.Save(session)
    
    msg := fmt.Sprintf("💊 <b>%s</b> %s\n", medication, dosage)
    msg += fmt.Sprintf("📅 Частота: %s\n\n", frequency)
    msg += "Сколько дней принимать? (например: 7 дней или постоянно)"
    b.sendMessage(chatID, msg, nil)
}

func (b *TelegramBot) handleDurationInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    duration := update.Message.Text
    data := strings.Split(session.TempData, "|")
    medication := data[0]
    dosage := data[1]
    frequency := data[2]
    
    // Сохраняем в БД
    b.saveMedication(user.UserID, medication, dosage, frequency, duration)
    
    session.State = StateNone
    session.TempData = ""
    b.db.Save(session)
    
    msg := fmt.Sprintf("✅ Лекарство <b>%s</b> добавлено!\n\n", medication)
    msg += "Я буду напоминать вам о приёме."
    
    b.sendMessage(chatID, msg, MedicationsMenu(true))
}

// Выбор симптома
func (b *TelegramBot) handleSymptomSelect(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    data := update.CallbackQuery.Data
    symptom := strings.TrimPrefix(data, "symptom_")
    
    session.TempData = symptom
    b.db.Save(session)
    
    msg := fmt.Sprintf("📝 Симптом: <b>%s</b>\n\n", symptom)
    msg += "Оцените интенсивность по шкале от 1 до 10:"
    
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, msg, IntensityMenu(symptom))
}

func (b *TelegramBot) handleIntensity(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    data := update.CallbackQuery.Data
    parts := strings.Split(data, "_")
    intensity := parts[1]
    symptom := parts[2]
    
    // Сохраняем в дневник
    b.saveSymptom(user.UserID, symptom, intensity)
    
    msg := fmt.Sprintf("✅ Симптом <b>%s</b> (интенсивность %s/10) записан!", symptom, intensity)
    
    session.State = StateNone
    session.TempData = ""
    b.db.Save(session)
    
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, msg, SymptomsMenu())
}

func (b *TelegramBot) handleSymptomCustom(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    
    session.State = StateAwaitingSymptoms
    b.db.Save(session)
    
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "📝 Опишите ваш симптом:", nil)
}

func (b *TelegramBot) handleSymptomsInput(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.Message.Chat.ID
    symptom := update.Message.Text
    
    session.State = StateNone
    session.TempData = symptom
    b.db.Save(session)
    
    msg := fmt.Sprintf("📝 Симптом: <b>%s</b>\n\n", symptom)
    msg += "Оцените интенсивность по шкале от 1 до 10:"
    
    b.sendMessage(chatID, msg, IntensityMenu(symptom))
}

// Отметка приёма
func (b *TelegramBot) handleMedTake(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    
    medications := b.getTodayMedications(user.UserID)
    if len(medications) == 0 {
        b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "💊 Сегодня нет лекарств для приёма", MedicationsMenu(false))
        return
    }
    
    // Создаем клавиатуру с лекарствами
    var rows [][]tgbotapi.InlineKeyboardButton
    for _, med := range medications {
        if !med.Taken {
            row := []tgbotapi.InlineKeyboardButton{
                tgbotapi.NewInlineKeyboardButtonData(
                    fmt.Sprintf("%s %s", med.Name, med.Dosage),
                    fmt.Sprintf("take_%d", med.ID),
                ),
            }
            rows = append(rows, row)
        }
    }
    
    if len(rows) == 0 {
        b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "✅ Все лекарства на сегодня приняты!", MedicationsMenu(true))
        return
    }
    
    rows = append(rows, []tgbotapi.InlineKeyboardButton{
        tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back_main"),
    })
    
    keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, "💊 Выберите лекарство, которое приняли:", keyboard)
}

// Привязка аккаунта
func (b *TelegramBot) handleSettingsLink(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    
    if user.UserID != "" {
        msg := fmt.Sprintf("✅ Аккаунт уже привязан к пользователю %s %s", user.FirstName, user.LastName)
        b.editMessage(chatID, update.CallbackQuery.Message.MessageID, msg, SettingsMenu())
        return
    }
    
    // Генерируем код для привязки
    code := fmt.Sprintf("%06d", rand.Intn(1000000))
    expires := time.Now().Add(30 * time.Minute)
    
    user.AuthToken = code
    user.TokenExpires = &expires
    b.db.Save(user)
    
    msg := "🔗 Для привязки аккаунта:\n\n"
    msg += "1. Зайдите в веб-версию\n"
    msg += "2. Перейдите в настройки профиля\n"
    msg += "3. Введите этот код:\n\n"
    msg += fmt.Sprintf("<b>%s</b>\n\n", code)
    msg += "Код действителен 30 минут"
    
    b.editMessage(chatID, update.CallbackQuery.Message.MessageID, msg, nil)
}
