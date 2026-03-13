package api

import (
    "net/http"
    "os"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"

    "BOT_MAX/internal/models"
)

type AuthHandler struct {
    db *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
    return &AuthHandler{db: db}
}

type RegisterRequest struct {
    Email     string `json:"email" binding:"required,email"`
    Password  string `json:"password" binding:"required,min=6"`
    FirstName string `json:"first_name" binding:"required"`
    LastName  string `json:"last_name" binding:"required"`
    Role      string `json:"role" binding:"required,oneof=patient doctor"`
}

type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

type UserResponse struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    FirstName string    `json:"first_name"`
    LastName  string    `json:"last_name"`
    Role      string    `json:"role"`
    CreatedAt time.Time `json:"created_at"`
}

// Register регистрация нового пользователя
func (h *AuthHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // Хешируем пароль
    hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
        return
    }

    // Начинаем транзакцию
    tx := h.db.Begin()

    // Создаем пользователя
    user := models.User{
        ID:        uuid.New(),
        Email:     req.Email,
        Password:  string(hash),
        FirstName: req.FirstName,
        LastName:  req.LastName,
        Role:      req.Role,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    if err := tx.Create(&user).Error; err != nil {
        tx.Rollback()
        c.JSON(http.StatusInternalServerError, gin.H{"error": "user already exists"})
        return
    }

    // Создаем профиль в зависимости от роли
    if req.Role == "patient" {
        patient := models.Patient{
            ID:        uuid.New(),
            UserID:    user.ID,
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        }
        if err := tx.Create(&patient).Error; err != nil {
            tx.Rollback()
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
    } else if req.Role == "doctor" {
        doctor := models.Doctor{
            ID:        uuid.New(),
            UserID:    user.ID,
            CreatedAt: time.Now(),
            UpdatedAt: time.Now(),
        }
        if err := tx.Create(&doctor).Error; err != nil {
            tx.Rollback()
            c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
            return
        }
    }

    tx.Commit()

    c.JSON(http.StatusCreated, gin.H{
        "success": true,
        "data": UserResponse{
            ID:        user.ID.String(),
            Email:     user.Email,
            FirstName: user.FirstName,
            LastName:  user.LastName,
            Role:      user.Role,
            CreatedAt: user.CreatedAt,
        },
    })
}

// Login вход в систему
func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var user models.User
    if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
        return
    }

    // Создаем JWT токен
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": user.ID.String(),
        "role":    user.Role,
        "exp":     time.Now().Add(24 * time.Hour).Unix(),
    })

    tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": gin.H{
            "token": tokenString,
            "user": UserResponse{
                ID:        user.ID.String(),
                Email:     user.Email,
                FirstName: user.FirstName,
                LastName:  user.LastName,
                Role:      user.Role,
                CreatedAt: user.CreatedAt,
            },
        },
    })
}

// Profile получение профиля текущего пользователя
func (h *AuthHandler) Profile(c *gin.Context) {
    userID := c.GetString("user_id")
    
    var user models.User
    if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": UserResponse{
            ID:        user.ID.String(),
            Email:     user.Email,
            FirstName: user.FirstName,
            LastName:  user.LastName,
            Role:      user.Role,
            CreatedAt: user.CreatedAt,
        },
    })
}
