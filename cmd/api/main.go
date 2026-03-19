package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
    "github.com/bot011max/medical-bot/internal/api"
    "github.com/bot011max/medical-bot/internal/middleware"
    "github.com/bot011max/medical-bot/internal/security"
    "github.com/bot011max/medical-bot/pkg/logger"
)

func main() {
    // Загрузка .env
    if err := godotenv.Load(); err != nil {
        log.Println("⚠️ .env file not found, using environment variables")
    }

    // Инициализация логгера
    logLevel := logger.INFO
    if os.Getenv("DEBUG") == "true" {
        logLevel = logger.DEBUG
    }
    appLogger := logger.NewLogger(logLevel, "/var/log/api.log")
    appLogger.Info("Starting API server...")

    // Инициализация системы безопасности (ВОЕННЫЙ УРОВЕНЬ)
    armor := security.NewAbsoluteArmor(appLogger)
    if err := armor.Init(); err != nil {
        appLogger.Fatal("Failed to initialize security: %v", err)
    }

    // Настройка Gin
    gin.SetMode(gin.ReleaseMode)
    r := gin.New()
    r.Use(gin.Recovery())
    r.Use(middleware.SecurityHeaders())
    r.Use(armor.ProtectRequest()) // WAF + Rate Limiting + IDS
    r.Use(middleware.RequestID())
    r.Use(middleware.LoggerMiddleware(appLogger))

    // Health check
    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "status": "ok",
            "time":   time.Now().Unix(),
        })
    })

    // Метрики для Prometheus
    r.GET("/metrics", armor.MetricsHandler())

    // API маршруты
    api.SetupRoutes(r, armor, appLogger)

    // Настройка HTTP сервера
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    srv := &http.Server{
        Addr:         ":" + port,
        Handler:      r,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Graceful shutdown
    go func() {
        appLogger.Info("Server starting on port %s", port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            appLogger.Fatal("Server error: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    appLogger.Info("Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        appLogger.Fatal("Server forced to shutdown: %v", err)
    }

    appLogger.Info("Server stopped")
}
