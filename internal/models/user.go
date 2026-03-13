package models

import (
    "time"
    "github.com/google/uuid"
)

type User struct {
    ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    Email     string    `json:"email" gorm:"uniqueIndex;not null"`
    Password  string    `json:"-" gorm:"not null"`
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    Role      string    `json:"role" gorm:"index;not null;default:'patient'"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Patient struct {
    ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID    uuid.UUID `json:"user_id" gorm:"index;not null"`
    BirthDate *time.Time `json:"birth_date"`
    Phone     string    `json:"phone"`
    Snils     string    `json:"snils"`
    Polis     string    `json:"polis"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    
    User      User      `json:"user" gorm:"foreignKey:UserID"`
}

type Doctor struct {
    ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    UserID      uuid.UUID `json:"user_id" gorm:"uniqueIndex;not null"`
    Specialty   string    `json:"specialty"`
    LicenseNum  string    `json:"license_num"`
    Experience  int       `json:"experience"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
    
    User        User      `json:"user" gorm:"foreignKey:UserID"`
}
