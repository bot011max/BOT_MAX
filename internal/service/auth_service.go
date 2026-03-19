package service

import (
    "crypto/subtle"
    "errors"
    "time"
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "golang.org/x/crypto/bcrypt"
    "github.com/pquerna/otp/totp"
    "github.com/pquerna/otp"
    "github.com/skip2/go-qrcode"

    "github.com/bot011max/medical-bot/internal/models"
    "github.com/bot011max/medical-bot/internal/repository"
    "github.com/bot011max/medical-bot/internal/security"
)

// AuthService - сервис аутентификации
type AuthService struct {
    userRepo      *repository.UserRepository
    subRepo       *repository.SubscriptionRepository
    audit         *security.AuditLogger
    crypto        *security.CryptoService
    jwtSecret     []byte
    jwtExpiry     time.Duration
    refreshExpiry time.Duration
}

// NewAuthService - создание сервиса
func NewAuthService(userRepo *repository.UserRepository, subRepo *repository.SubscriptionRepository, 
                    audit *security.AuditLogger, crypto *security.CryptoService, jwtSecret string) *AuthService {
    return &AuthService{
        userRepo:      userRepo,
        subRepo:       subRepo,
        audit:         audit,
        crypto:        crypto,
        jwtSecret:     []byte(jwtSecret),
        jwtExpiry:     15 * time.Minute,      // короткие JWT
        refreshExpiry: 7 * 24 * time.Hour,    // refresh token на неделю
    }
}

// Claims - кастомные claims для JWT
type Claims struct {
    UserID         string   `json:"user_id"`
    Email          string   `json:"email"`
    Role           string   `json:"role"`
    Subscription   string   `json:"subscription"`
    TwoFACompleted bool     `json:"twofa_completed"`
    jwt.RegisteredClaims
}

// RegisterRequest - запрос на регистрацию
type RegisterRequest struct {
    Email     string `json:"email" binding:"required,email"`
    Password  string `json:"password" binding:"required,min=12,max=50"`
    FirstName string `json:"first_name" binding:"required"`
    LastName  string `json:"last_name" binding:"required"`
    Phone     string `json:"phone" binding:"required,e164"` // формат +79991234567
    Role      string `json:"role" binding:"required,oneof=patient doctor clinic"`
}

// RegisterResponse - ответ на регистрацию
type RegisterResponse struct {
    UserID        string `json:"user_id"`
    Email         string `json:"email"`
    TwoFactorQR   string `json:"two_factor_qr,omitempty"` // base64 PNG
    TwoFactorURL  string `json:"two_factor_url,omitempty"`
    Subscription  string `json:"subscription"`
}

// Register - регистрация нового пользователя
func (s *AuthService) Register(req *RegisterRequest, ip string) (*RegisterResponse, error) {
    // 1. Проверка сложности пароля (NIST 800-63B)
    if err := s.validatePassword(req.Password); err != nil {
        return nil, err
    }

    // 2. Хеширование пароля (bcrypt + соль)
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        s.audit.LogSecurity("PASSWORD_HASH_FAILED", "", ip, map[string]interface{}{
            "email": security.HashEmail(req.Email),
        })
        return nil, errors.New("failed to hash password")
    }

    // 3. Создание пользователя
    user := &models.User{
        Email:        strings.ToLower(req.Email),
        PasswordHash: string(hashedPassword),
        FirstName:    req.FirstName,
        LastName:     req.LastName,
        Role:         req.Role,
        SubscriptionTier: "free",
    }

    // 4. Шифрование телефона (AES-256)
    encryptionKey, err := s.crypto.GetDataEncryptionKey()
    if err != nil {
        return nil, err
    }
    if err := user.EncryptPhone(req.Phone, encryptionKey); err != nil {
        return nil, err
    }

    // 5. Генерация 2FA секрета
    key, err := totp.Generate(totp.GenerateOpts{
        Issuer:      "MedicalBot",
        AccountName: req.Email,
    })
    if err != nil {
        return nil, err
    }
    user.TwoFactorSecret = key.Secret()

    // 6. Сохранение в БД (в транзакции)
    if err := s.userRepo.Create(user); err != nil {
        if strings.Contains(err.Error(), "duplicate key") {
            return nil, errors.New("email already registered")
        }
        return nil, err
    }

    // 7. Создание подписки free
    sub := &models.Subscription{
        UserID:      user.ID,
        Tier:        "free",
        StartedAt:   time.Now(),
        ExpiresAt:   time.Now().AddDate(1, 0, 0), // на год
        AutoRenew:   false,
    }
    
    // Копируем лимиты из конфига
    limits := models.SubscriptionLimits["free"]
    sub.MaxPatients = limits["max_patients"].(int)
    sub.MaxReminders = limits["max_reminders"].(int)
    sub.MaxAnalyses = limits["max_analyses"].(int)
    sub.StorageYears = limits["storage_years"].(int)
    sub.PriceMonthly = limits["price_monthly"].(float64)
    sub.PriceYearly = limits["price_yearly"].(float64)
    sub.Features = limits["features"].(map[string]bool)

    if err := s.subRepo.Create(sub); err != nil {
        return nil, err
    }

    // 8. Генерация QR-кода для 2FA
    qr, err := qrcode.Encode(key.URL(), qrcode.Medium, 256)
    if err != nil {
        return nil, err
    }

    // 9. Аудит
    s.audit.LogAccess("USER_REGISTERED", user.ID.String(), ip, map[string]interface{}{
        "email": security.HashEmail(req.Email),
        "role":  req.Role,
    })

    return &RegisterResponse{
        UserID:       user.ID.String(),
        Email:        user.Email,
        TwoFactorQR:  base64.StdEncoding.EncodeToString(qr),
        TwoFactorURL: key.URL(),
        Subscription: "free",
    }, nil
}

// LoginRequest - запрос на вход
type LoginRequest struct {
    Email       string `json:"email" binding:"required,email"`
    Password    string `json:"password" binding:"required"`
    TwoFactor   string `json:"two_factor_code,omitempty"`
}

// LoginResponse - ответ на вход
type LoginResponse struct {
    AccessToken     string `json:"access_token"`
    RefreshToken    string `json:"refresh_token"`
    TokenType       string `json:"token_type"`
    ExpiresIn       int64  `json:"expires_in"`
    TwoFactorRequired bool `json:"two_factor_required,omitempty"`
    UserID          string `json:"user_id"`
    Role            string `json:"role"`
    Subscription    string `json:"subscription"`
}

// Login - вход пользователя
func (s *AuthService) Login(req *LoginRequest, ip, userAgent string) (*LoginResponse, error) {
    // 1. Поиск пользователя
    user, err := s.userRepo.FindByEmail(strings.ToLower(req.Email))
    if err != nil {
        // Защита от timing attack - всегда одинаковое время
        bcrypt.CompareHashAndPassword([]byte("fakehash"), []byte(req.Password))
        time.Sleep(100 * time.Millisecond)
        
        s.audit.LogSecurity("LOGIN_FAILED", "", ip, map[string]interface{}{
            "email": security.HashEmail(req.Email),
            "reason": "user_not_found",
        })
        return nil, errors.New("invalid credentials")
    }

    // 2. Проверка блокировки
    if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
        return nil, errors.New("account locked, try again later")
    }

    // 3. Проверка пароля (constant-time сравнение)
    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
        user.LoginAttempts++
        
        // Блокировка после 5 неудачных попыток
        if user.LoginAttempts >= 5 {
            lockUntil := time.Now().Add(30 * time.Minute)
            user.LockedUntil = &lockUntil
        }
        s.userRepo.Update(user)
        
        s.audit.LogSecurity("LOGIN_FAILED", user.ID.String(), ip, map[string]interface{}{
            "reason": "wrong_password",
            "attempts": user.LoginAttempts,
        })
        return nil, errors.New("invalid credentials")
    }

    // 4. Сброс счетчика попыток
    user.LoginAttempts = 0
    user.LockedUntil = nil
    now := time.Now()
    user.LastLoginAt = &now
    s.userRepo.Update(user)

    // 5. Проверка 2FA
    if user.TwoFactorEnabled {
        if req.TwoFactor == "" {
            return &LoginResponse{
                TwoFactorRequired: true,
                UserID:           user.ID.String(),
            }, nil
        }
        
        // Верификация TOTP
        valid := totp.Validate(req.TwoFactor, user.TwoFactorSecret)
        if !valid {
            s.audit.LogSecurity("2FA_FAILED", user.ID.String(), ip, nil)
            return nil, errors.New("invalid 2FA code")
        }
    }

    // 6. Получение подписки
    sub, err := s.subRepo.GetActiveByUserID(user.ID)
    if err != nil {
        // Если нет активной, создаем free
        sub = s.createDefaultSubscription(user.ID)
    }

    // 7. Создание JWT токена
    accessToken, err := s.generateJWT(user, sub, false)
    if err != nil {
        return nil, err
    }

    // 8. Создание refresh токена
    refreshToken, err := s.generateRefreshToken(user)
    if err != nil {
        return nil, err
    }

    // 9. Аудит успешного входа
    s.audit.LogAccess("USER_LOGIN", user.ID.String(), ip, map[string]interface{}{
        "user_agent": userAgent,
        "method":     user.TwoFactorEnabled,
    })

    return &LoginResponse{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        TokenType:    "Bearer",
        ExpiresIn:    int64(s.jwtExpiry.Seconds()),
        UserID:       user.ID.String(),
        Role:         user.Role,
        Subscription: sub.Tier,
    }, nil
}

// generateJWT - создание JWT токена
func (s *AuthService) generateJWT(user *models.User, sub *models.Subscription, twoFACompleted bool) (string, error) {
    claims := &Claims{
        UserID:         user.ID.String(),
        Email:          user.Email,
        Role:           user.Role,
        Subscription:   sub.Tier,
        TwoFACompleted: twoFACompleted,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "medical-bot",
            Subject:   user.ID.String(),
            ID:        uuid.New().String(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}

// generateRefreshToken - создание refresh токена
func (s *AuthService) generateRefreshToken(user *models.User) (string, error) {
    // Refresh token - долгоживущий, хранится в БД
    tokenID := uuid.New().String()
    expiresAt := time.Now().Add(s.refreshExpiry)
    
    // Сохраняем в БД
    if err := s.userRepo.SaveRefreshToken(user.ID, tokenID, expiresAt); err != nil {
        return "", err
    }

    // JWT для refresh токена
    claims := jwt.RegisteredClaims{
        ExpiresAt: jwt.NewNumericDate(expiresAt),
        IssuedAt:  jwt.NewNumericDate(time.Now()),
        ID:        tokenID,
        Subject:   user.ID.String(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(s.jwtSecret)
}

// RefreshToken - обновление access токена
func (s *AuthService) RefreshToken(refreshToken string, ip string) (*LoginResponse, error) {
    // 1. Парсинг токена
    token, err := jwt.ParseWithClaims(refreshToken, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
        return s.jwtSecret, nil
    })
    if err != nil {
        return nil, errors.New("invalid refresh token")
    }

    claims, ok := token.Claims.(*jwt.RegisteredClaims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid refresh token")
    }

    // 2. Проверка в БД
    userID, err := s.userRepo.ValidateRefreshToken(claims.ID)
    if err != nil {
        s.audit.LogSecurity("REFRESH_TOKEN_INVALID", "", ip, map[string]interface{}{
            "token_id": claims.ID,
        })
        return nil, errors.New("refresh token expired")
    }

    // 3. Получение пользователя
    user, err := s.userRepo.FindByID(userID)
    if err != nil {
        return nil, err
    }

    // 4. Получение подписки
    sub, _ := s.subRepo.GetActiveByUserID(user.ID)

    // 5. Создание нового access токена
    accessToken, err := s.generateJWT(user, sub, true)
    if err != nil {
        return nil, err
    }

    return &LoginResponse{
        AccessToken:  accessToken,
        TokenType:    "Bearer",
        ExpiresIn:    int64(s.jwtExpiry.Seconds()),
        UserID:       user.ID.String(),
        Role:         user.Role,
        Subscription: sub.Tier,
    }, nil
}

// validatePassword - проверка сложности пароля (NIST 800-63B)
func (s *AuthService) validatePassword(password string) error {
    if len(password) < 12 {
        return errors.New("password must be at least 12 characters")
    }
    
    hasUpper := false
    hasLower := false
    hasDigit := false
    hasSpecial := false
    
    for _, char := range password {
        switch {
        case 'A' <= char && char <= 'Z':
            hasUpper = true
        case 'a' <= char && char <= 'z':
            hasLower = true
        case '0' <= char && char <= '9':
            hasDigit = true
        default:
            hasSpecial = true
        }
    }
    
    if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
        return errors.New("password must contain uppercase, lowercase, digit and special character")
    }
    
    return nil
}

// createDefaultSubscription - создание подписки по умолчанию
func (s *AuthService) createDefaultSubscription(userID uuid.UUID) *models.Subscription {
    sub := &models.Subscription{
        UserID:    userID,
        Tier:      "free",
        StartedAt: time.Now(),
        ExpiresAt: time.Now().AddDate(1, 0, 0),
        AutoRenew: false,
    }
    
    limits := models.SubscriptionLimits["free"]
    sub.MaxPatients = limits["max_patients"].(int)
    sub.MaxReminders = limits["max_reminders"].(int)
    sub.MaxAnalyses = limits["max_analyses"].(int)
    sub.StorageYears = limits["storage_years"].(int)
    sub.PriceMonthly = limits["price_monthly"].(float64)
    sub.PriceYearly = limits["price_yearly"].(float64)
    sub.Features = limits["features"].(map[string]bool)
    
    return sub
}
