mkdir -p internal/middleware

# Создаем файл auth middleware
cat > internal/middleware/auth.go << 'EOF'
package middleware

import (
    "net/http"
    "os"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

func AuthRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
            c.Abort()
            return
        }

        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
            c.Abort()
            return
        }

        tokenString := parts[1]
        secret := []byte(os.Getenv("JWT_SECRET"))
        if len(secret) == 0 {
            secret = []byte("test-secret")
        }

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, jwt.ErrSignatureInvalid
            }
            return secret, nil
        })

        if err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            c.Abort()
            return
        }

        if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
            c.Set("user_id", claims["user_id"])
            c.Set("user_role", claims["role"])
            c.Next()
        } else {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
            c.Abort()
            return
        }
    }
}

func RoleRequired(roles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole := c.GetString("user_role")
        
        for _, role := range roles {
            if userRole == role {
                c.Next()
                return
            }
        }
        
        c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
        c.Abort()
    }
}

func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Next()
    }
}

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = generateRequestID()
        }
        c.Set("request_id", requestID)
        c.Header("X-Request-ID", requestID)
        c.Next()
    }
}

func generateRequestID() string {
    return "req-" + randomString(8)
}

func randomString(n int) string {
    const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, n)
    for i := range b {
        b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
    }
    return string(b)
}

func AuditLogMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Логирование запросов
        c.Next()
    }
}
EOF
