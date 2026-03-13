package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// KeyboardBuilder помогает создавать клавиатуры
type KeyboardBuilder struct {
    buttons [][]tgbotapi.InlineKeyboardButton
}

// Главное меню
func MainMenu() tgbotapi.InlineKeyboardMarkup {
    buttons := [][]tgbotapi.InlineKeyboardButton{
        {
            tgbotapi.NewInlineKeyboardButtonData("💊 Лекарства", "menu_medications"),
            tgbotapi.NewInlineKeyboardButtonData("📅 Приемы", "menu_appointments"),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("📝 Симптомы", "menu_symptoms"),
            tgbotapi.NewInlineKeyboardButtonData("📊 Анализы", "menu_analyses"),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("👨‍⚕️ Врачи", "menu_doctors"),
            tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "menu_settings"),
        },
    }
    return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// Меню лекарств
func MedicationsMenu(hasActive bool) tgbotapi.InlineKeyboardMarkup {
    buttons := [][]tgbotapi.InlineKeyboardButton{
        {
            tgbotapi.NewInlineKeyboardButtonData("➕ Добавить", "med_add"),
            tgbotapi.NewInlineKeyboardButtonData("📋 Список", "med_list"),
        },
    }
    
    if hasActive {
        buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
            tgbotapi.NewInlineKeyboardButtonData("✅ Принял", "med_take"),
            tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", "med_stats"),
        })
    }
    
    buttons = append(buttons, []tgbotapi.InlineKeyboardButton{
        tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back_main"),
    })
    
    return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// Меню симптомов
func SymptomsMenu() tgbotapi.InlineKeyboardMarkup {
    commonSymptoms := []string{"Головная боль", "Температура", "Кашель", "Тошнота", "Слабость", "Боль в животе"}
    
    var rows [][]tgbotapi.InlineKeyboardButton
    for i := 0; i < len(commonSymptoms); i += 2 {
        row := []tgbotapi.InlineKeyboardButton{}
        row = append(row, tgbotapi.NewInlineKeyboardButtonData(commonSymptoms[i], "symptom_"+commonSymptoms[i]))
        if i+1 < len(commonSymptoms) {
            row = append(row, tgbotapi.NewInlineKeyboardButtonData(commonSymptoms[i+1], "symptom_"+commonSymptoms[i+1]))
        }
        rows = append(rows, row)
    }
    
    rows = append(rows, []tgbotapi.InlineKeyboardButton{
        tgbotapi.NewInlineKeyboardButtonData("✏️ Свой вариант", "symptom_custom"),
    })
    rows = append(rows, []tgbotapi.InlineKeyboardButton{
        tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back_main"),
    })
    
    return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// Меню с интенсивностью
func IntensityMenu(symptom string) tgbotapi.InlineKeyboardMarkup {
    buttons := [][]tgbotapi.InlineKeyboardButton{
        {
            tgbotapi.NewInlineKeyboardButtonData("😊 1-2", "intensity_1_"+symptom),
            tgbotapi.NewInlineKeyboardButtonData("😐 3-4", "intensity_2_"+symptom),
            tgbotapi.NewInlineKeyboardButtonData("😕 5-6", "intensity_3_"+symptom),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("😟 7-8", "intensity_4_"+symptom),
            tgbotapi.NewInlineKeyboardButtonData("😫 9-10", "intensity_5_"+symptom),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back_symptoms"),
        },
    }
    return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// Меню подтверждения
func ConfirmationMenu(action string) tgbotapi.InlineKeyboardMarkup {
    buttons := [][]tgbotapi.InlineKeyboardButton{
        {
            tgbotapi.NewInlineKeyboardButtonData("✅ Да", "confirm_yes_"+action),
            tgbotapi.NewInlineKeyboardButtonData("❌ Нет", "confirm_no_"+action),
        },
    }
    return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// Меню выбора времени
func TimeMenu(action string) tgbotapi.InlineKeyboardMarkup {
    times := []string{"08:00", "09:00", "10:00", "12:00", "14:00", "16:00", "18:00", "20:00", "22:00"}
    
    var rows [][]tgbotapi.InlineKeyboardButton
    for i := 0; i < len(times); i += 3 {
        row := []tgbotapi.InlineKeyboardButton{}
        for j := 0; j < 3 && i+j < len(times); j++ {
            row = append(row, tgbotapi.NewInlineKeyboardButtonData(times[i+j], "time_"+times[i+j]+"_"+action))
        }
        rows = append(rows, row)
    }
    
    rows = append(rows, []tgbotapi.InlineKeyboardButton{
        tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back"),
    })
    
    return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// Меню настроек
func SettingsMenu() tgbotapi.InlineKeyboardMarkup {
    buttons := [][]tgbotapi.InlineKeyboardButton{
        {
            tgbotapi.NewInlineKeyboardButtonData("🔔 Уведомления", "settings_notifications"),
            tgbotapi.NewInlineKeyboardButtonData("🌐 Язык", "settings_language"),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("👤 Профиль", "settings_profile"),
            tgbotapi.NewInlineKeyboardButtonData("🔗 Привязать аккаунт", "settings_link"),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back_main"),
        },
    }
    return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}
