package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/joho/godotenv"
    "github.com/bot011max/medical-bot/internal/security"
    "github.com/bot011max/medical-bot/internal/telegram"
    "github.com/bot011max/medical-bot/pkg/logger"
)

func main() {
    // Загрузка .env
    if err := godotenv.Load(); err != nil {
        log.Println("⚠️ .env file not found, using environment variables")
    }

    // Инициализация логгера
    appLogger := logger.NewLogger(logger.INFO, "/var/log/telegram.log")
    appLogger.Info("Starting Telegram bot...")

    // Инициализация безопасности
    armor := security.NewAbsoluteArmor(appLogger)
    if err := armor.Init(); err != nil {
        appLogger.Fatal("Failed to initialize security: %v", err)
    }

    // Создание и запуск бота
    bot := telegram.NewBot(armor, appLogger)
    if err := bot.Start(); err != nil {
        appLogger.Fatal("Failed to start bot: %v", err)
    }

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    appLogger.Info("Shutting down bot...")
    bot.Stop()
    appLogger.Info("Bot stopped")
}
