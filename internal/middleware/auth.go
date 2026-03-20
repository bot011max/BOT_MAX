package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
            c.Abort()
            return
        }

        c.Set("user_id", "test-user")
        c.Next()
    }
}

func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Next()
    }
}

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Set("request_id", "req-123")
        c.Next()
    }
}

func LoggerMiddleware(logger interface{}) gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
    }
}
