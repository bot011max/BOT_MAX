cat > internal/models/telegram.go << 'EOF'
package models

import (
    "time"
)

type TelegramUser struct {
    ID           uint      `json:"id" gorm:"primaryKey"`
    UserID       string    `json:"user_id" gorm:"index"`
    TelegramID   int64     `json:"telegram_id" gorm:"uniqueIndex;not null"`
    ChatID       int64     `json:"chat_id"`
    Username     string    `json:"username"`
    FirstName    string    `json:"first_name"`
    LastName     string    `json:"last_name"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
EOF

# internal/api/doctor.go
cat > internal/api/doctor.go << 'EOF'
package api

import (
    "github.com/gin-gonic/gin"
)

func GetDoctors(c *gin.Context) {
    c.JSON(200, gin.H{"message": "Doctors endpoint"})
}
EOF
