package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// MainMenu создает главное меню
func MainMenu() tgbotapi.InlineKeyboardMarkup {
    buttons := [][]tgbotapi.InlineKeyboardButton{
        {
            tgbotapi.NewInlineKeyboardButtonData("💊 Лекарства", "menu_medications"),
            tgbotapi.NewInlineKeyboardButtonData("📅 Приемы", "menu_appointments"),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("📝 Симптомы", "menu_symptoms"),
            tgbotapi.NewInlineKeyboardButtonData("🔬 Анализы", "menu_analyses"),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("👨‍⚕️ Врачи", "menu_doctors"),
            tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "menu_settings"),
        },
    }
    return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// MedicationsMenu создает меню лекарств
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

// SymptomsMenu создает меню симптомов
func SymptomsMenu() tgbotapi.InlineKeyboardMarkup {
    symptoms := []string{"Головная боль", "Температура", "Кашель", "Тошнота", "Слабость", "Боль в животе"}
    
    var rows [][]tgbotapi.InlineKeyboardButton
    for i := 0; i < len(symptoms); i += 2 {
        row := []tgbotapi.InlineKeyboardButton{}
        row = append(row, tgbotapi.NewInlineKeyboardButtonData(symptoms[i], "symptom_"+symptoms[i]))
        if i+1 < len(symptoms) {
            row = append(row, tgbotapi.NewInlineKeyboardButtonData(symptoms[i+1], "symptom_"+symptoms[i+1]))
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

// SettingsMenu создает меню настроек
func SettingsMenu() tgbotapi.InlineKeyboardMarkup {
    buttons := [][]tgbotapi.InlineKeyboardButton{
        {
            tgbotapi.NewInlineKeyboardButtonData("🔗 Привязать аккаунт", "settings_link"),
            tgbotapi.NewInlineKeyboardButtonData("🔔 Уведомления", "settings_notifications"),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("👤 Профиль", "settings_profile"),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back_main"),
        },
    }
    return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}

// IntensityMenu создает меню выбора интенсивности
func IntensityMenu(symptom string) tgbotapi.InlineKeyboardMarkup {
    buttons := [][]tgbotapi.InlineKeyboardButton{
        {
            tgbotapi.NewInlineKeyboardButtonData("1-2", "intensity_1-2_"+symptom),
            tgbotapi.NewInlineKeyboardButtonData("3-4", "intensity_3-4_"+symptom),
            tgbotapi.NewInlineKeyboardButtonData("5-6", "intensity_5-6_"+symptom),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("7-8", "intensity_7-8_"+symptom),
            tgbotapi.NewInlineKeyboardButtonData("9-10", "intensity_9-10_"+symptom),
        },
        {
            tgbotapi.NewInlineKeyboardButtonData("◀️ Назад", "back_symptoms"),
        },
    }
    return tgbotapi.NewInlineKeyboardMarkup(buttons...)
}
