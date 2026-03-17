// internal/security/secrets.go
package security

import (
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "io/ioutil"
    "os"
    "strings"
    "sync"
    "time"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/secretsmanager"
    "golang.org/x/crypto/argon2"
)

type SecretManager struct {
    mu           sync.RWMutex
    secrets      map[string]*Secret
    useVault     bool
    vaultClient  *secretsmanager.SecretsManager
    rotationTime time.Duration
}

type Secret struct {
    Value     string
    Version   int
    CreatedAt time.Time
    ExpiresAt time.Time
}

func NewSecretManager(useCloudVault bool) (*SecretManager, error) {
    sm := &SecretManager{
        secrets:      make(map[string]*Secret),
        useVault:     useCloudVault,
        rotationTime: 30 * 24 * time.Hour, // 30 дней
    }

    if useCloudVault {
        sess, err := session.NewSession(&aws.Config{
            Region: aws.String(os.Getenv("AWS_REGION")),
        })
        if err != nil {
            return nil, err
        }
        sm.vaultClient = secretsmanager.New(sess)
    }

    // Загружаем секреты из файлов при старте
    if err := sm.loadSecretsFromFiles(); err != nil {
        return nil, err
    }

    // Запускаем ротацию секретов
    go sm.rotateSecretsPeriodically()

    return sm, nil
}

func (sm *SecretManager) loadSecretsFromFiles() error {
    files, err := ioutil.ReadDir("/run/secrets")
    if os.IsNotExist(err) {
        // Используем environment для разработки
        return sm.loadFromEnv()
    }
    if err != nil {
        return err
    }

    for _, file := range files {
        if file.IsDir() {
            continue
        }
        data, err := ioutil.ReadFile("/run/secrets/" + file.Name())
        if err != nil {
            continue
        }
        secretName := strings.TrimSuffix(file.Name(), ".txt")
        sm.secrets[secretName] = &Secret{
            Value:     strings.TrimSpace(string(data)),
            Version:   1,
            CreatedAt: time.Now(),
            ExpiresAt: time.Now().Add(sm.rotationTime),
        }
    }
    return nil
}

func (sm *SecretManager) loadFromEnv() error {
    // Для разработки используем переменные окружения
    envVars := []string{"JWT_SECRET", "TELEGRAM_TOKEN", "DB_PASSWORD"}
    for _, envVar := range envVars {
        if value := os.Getenv(envVar); value != "" {
            sm.secrets[strings.ToLower(envVar)] = &Secret{
                Value:     value,
                Version:   1,
                CreatedAt: time.Now(),
                ExpiresAt: time.Now().Add(sm.rotationTime),
            }
        }
    }
    return nil
}

func (sm *SecretManager) GetSecret(name string) (string, error) {
    sm.mu.RLock()
    secret, exists := sm.secrets[name]
    sm.mu.RUnlock()

    if !exists {
        if sm.useVault {
            return sm.getFromVault(name)
        }
        return "", fmt.Errorf("secret %s not found", name)
    }

    // Проверяем, не истек ли секрет
    if time.Now().After(secret.ExpiresAt) {
        go sm.rotateSecret(name)
    }

    return secret.Value, nil
}

func (sm *SecretManager) getFromVault(name string) (string, error) {
    input := &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(name),
    }

    result, err := sm.vaultClient.GetSecretValue(input)
    if err != nil {
        return "", err
    }

    return *result.SecretString, nil
}

func (sm *SecretManager) rotateSecret(name string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    // Генерируем новый секрет
    newSecret, err := GenerateRandomString(32)
    if err != nil {
        return err
    }

    if sm.useVault {
        // Обновляем в Vault
        input := &secretsmanager.UpdateSecretInput{
            SecretId:     aws.String(name),
            SecretString: aws.String(newSecret),
        }
        if _, err := sm.vaultClient.UpdateSecret(input); err != nil {
            return err
        }
    }

    // Обновляем в памяти
    sm.secrets[name] = &Secret{
        Value:     newSecret,
        Version:   sm.secrets[name].Version + 1,
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(sm.rotationTime),
    }

    // Логируем ротацию
    AuditLog("SECRET_ROTATED", "system", map[string]interface{}{
        "secret_name": name,
        "version":     sm.secrets[name].Version,
    })

    return nil
}

func (sm *SecretManager) rotateSecretsPeriodically() {
    ticker := time.NewTicker(24 * time.Hour)
    for range ticker.C {
        for name, secret := range sm.secrets {
            if time.Now().After(secret.ExpiresAt.Add(-7 * 24 * time.Hour)) {
                // Скоро истекает, начинаем ротацию
                go sm.rotateSecret(name)
            }
        }
    }
}

// Генерация криптостойких случайных строк
func GenerateRandomString(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// Argon2id хеширование паролей
func HashPassword(password string) (string, error) {
    salt := make([]byte, 16)
    if _, err := rand.Read(salt); err != nil {
        return "", err
    }

    hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    
    // Формат: argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
    encoded := fmt.Sprintf("argon2id$v=19$m=65536,t=1,p=4$%s$%s",
        base64.RawStdEncoding.EncodeToString(salt),
        base64.RawStdEncoding.EncodeToString(hash))
    
    return encoded, nil
}

func VerifyPassword(password, encodedHash string) bool {
    // Парсинг и верификация Argon2id
    parts := strings.Split(encodedHash, "$")
    if len(parts) != 6 {
        return false
    }

    salt, err := base64.RawStdEncoding.DecodeString(parts[4])
    if err != nil {
        return false
    }

    hash, err := base64.RawStdEncoding.DecodeString(parts[5])
    if err != nil {
        return false
    }

    computedHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
    return subtle.ConstantTimeCompare(hash, computedHash) == 1
}
