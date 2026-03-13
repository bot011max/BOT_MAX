package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/bot011max/BOT_MAX/internal/telegram"
    "github.com/joho/godotenv"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    // Загружаем .env
    if err := godotenv.Load(); err != nil {
        log.Println("Файл .env не найден, используем переменные окружения")
    }
    
    // Получаем токен
    token := os.Getenv("TELEGRAM_TOKEN")
    if token == "" {
        log.Fatal("TELEGRAM_TOKEN не задан")
    }
    
    // Подключаемся к БД
    dsn := "host=localhost user=postgres password=postgres dbname=medical_bot port=5432 sslmode=disable"
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Ошибка подключения к БД:", err)
    }
    
    // Автоматическая миграция
    db.AutoMigrate(&telegram.TelegramUser{}, &telegram.TelegramSession{}, &telegram.Reminder{})
    
    // Создаем бота
    bot, err := telegram.NewTelegramBot(token, db)
    if err != nil {
        log.Fatal("Ошибка создания бота:", err)
    }
    
    // Запускаем в polling-режиме (проще для разработки)
    log.Println("Бот запущен в polling-режиме")
    go bot.StartPolling()
    
    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Завершение работы...")
}
