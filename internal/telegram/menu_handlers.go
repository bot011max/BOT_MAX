package telegram

import (
    "encoding/json"
    "fmt"
    "strings"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *TelegramBot) handleMenuMedications(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.handleMedications(update, user, session)
}

func (b *TelegramBot) handleMenuAppointments(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.handleAppointments(update, user, session)
}

func (b *TelegramBot) handleMenuSymptoms(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.handleSymptoms(update, user, session)
}

func (b *TelegramBot) handleMenuAnalyses(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    
    if user.UserID == "" {
        b.requireAuth(chatID)
        return
    }
    
    msg := "🔬 Результаты анализов\n\n"
    msg += "Загрузите фото или PDF с результатами анализа, и я распознаю их!"
    
    b.sendMessage(chatID, msg, nil)
}

func (b *TelegramBot) handleMenuDoctors(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    
    if user.UserID == "" {
        b.requireAuth(chatID)
        return
    }
    
    doctors := b.getUserDoctors(user.UserID)
    
    if len(doctors) == 0 {
        msg := "👨‍⚕️ У вас пока нет привязанных врачей."
        b.sendMessage(chatID, msg, nil)
        return
    }
    
    msg := "👨‍⚕️ Ваши врачи:\n\n"
    for i, doc := range doctors {
        msg += fmt.Sprintf("%d. <b>%s</b>\n", i+1, doc.FullName)
        msg += fmt.Sprintf("   🏥 %s\n", doc.Specialty)
        msg += fmt.Sprintf("   📞 %s\n\n", doc.Phone)
    }
    
    b.sendMessage(chatID, msg, nil)
}

func (b *TelegramBot) handleMenuSettings(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.handleSettings(update, user, session)
}

func (b *TelegramBot) handleBackMain(update tgbotapi.Update, user *TelegramUser, session *TelegramSession) {
    chatID := update.CallbackQuery.Message.Chat.ID
    b.showMainMenu(chatID)
}

// Показать главное меню
func (b *TelegramBot) showMainMenu(chatID int64) {
    msg := "🏠 Главное меню\n\nВыберите раздел:"
    b.sendMessage(chatID, msg, MainMenu())
}
