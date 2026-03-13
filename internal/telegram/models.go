package telegram

import (
    "time"
)

// TelegramUser связывает пользователя системы с Telegram
type TelegramUser struct {
    ID           uint      `json:"id" gorm:"primaryKey"`
    UserID       string    `json:"user_id" gorm:"index;not null"`        // ID в системе
    TelegramID   int64     `json:"telegram_id" gorm:"uniqueIndex;not null"` // ID в Telegram
    ChatID       int64     `json:"chat_id" gorm:"not null"`              // ID чата
    Username     string    `json:"username"`                              // @username
    FirstName    string    `json:"first_name"`
    LastName     string    `json:"last_name"`
    Email        string    `json:"email"`                                 // Добавлено поле Email
    LanguageCode string    `json:"language_code"`
    IsActive     bool      `json:"is_active" gorm:"default:true"`
    AuthToken    string    `json:"auth_token"`                           // Токен для привязки аккаунта
    TokenExpires *time.Time `json:"token_expires"`                        // Срок действия токена
    
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// остальные структуры без изменений
