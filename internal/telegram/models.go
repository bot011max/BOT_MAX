package telegram

import (
    "time"
)

// TelegramUser связывает пользователя системы с Telegram
type TelegramUser struct {
    ID           uint      `json:"id" gorm:"primaryKey"`
    UserID       string    `json:"user_id" gorm:"index;not null"`
    TelegramID   int64     `json:"telegram_id" gorm:"uniqueIndex;not null"`
    ChatID       int64     `json:"chat_id" gorm:"not null"`
    Username     string    `json:"username"`
    FirstName    string    `json:"first_name"`
    LastName     string    `json:"last_name"`
    Email        string    `json:"email"`
    LanguageCode string    `json:"language_code"`
    IsActive     bool      `json:"is_active" gorm:"default:true"`
    AuthToken    string    `json:"auth_token"`
    TokenExpires *time.Time `json:"token_expires"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// TelegramSession сессия диалога с пользователем
type TelegramSession struct {
    ID           uint      `json:"id" gorm:"primaryKey"`
    TelegramID   int64     `json:"telegram_id" gorm:"index;not null"`
    State        string    `json:"state"`
    TempData     string    `json:"temp_data"`
    LastMessageID int      `json:"last_message_id"`
    LastCommand  string    `json:"last_command"`
    UpdatedAt    time.Time `json:"updated_at"`
}

// Reminder напоминание для отправки в Telegram
type Reminder struct {
    ID           uint      `json:"id" gorm:"primaryKey"`
    UserID       string    `json:"user_id" gorm:"index;not null"`
    TelegramID   int64     `json:"telegram_id" gorm:"index"`
    Message      string    `json:"message"`
    ScheduledFor time.Time `json:"scheduled_for" gorm:"index"`
    SentAt       *time.Time `json:"sent_at"`
    Status       string    `json:"status" gorm:"default:'pending'"`
    RetryCount   int       `json:"retry_count" gorm:"default:0"`
    CreatedAt    time.Time `json:"created_at"`
}

// Константы состояний
const (
    StateNone               = "none"
    StateAwaitingAuth       = "awaiting_auth"
    StateAwaitingSymptoms   = "awaiting_symptoms"
    StateAwaitingMedication = "awaiting_medication"
    StateAwaitingDosage     = "awaiting_dosage"
    StateAwaitingFrequency  = "awaiting_frequency"
    StateAwaitingDuration   = "awaiting_duration"
)
