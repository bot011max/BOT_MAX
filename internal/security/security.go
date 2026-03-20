package security

import (
    "crypto/rand"
    "encoding/hex"
    "log"
    "time"
)

func GenerateRandomString(length int) string {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "default-secret"
    }
    return hex.EncodeToString(bytes)[:length]
}

func GenerateNumericCode(length int) string {
    const digits = "0123456789"
    result := make([]byte, length)
    for i := range result {
        result[i] = digits[time.Now().UnixNano()%int64(len(digits))]
    }
    return string(result)
}

func SecurityAlert(alertType string, details map[string]interface{}) {
    log.Printf("SECURITY ALERT: %s - %v", alertType, details)
}

func InitAuditLogger() error {
    log.Println("Audit logger initialized")
    return nil
}
