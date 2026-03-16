package main

import (
    "fmt"
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
    
    // Получаем параметры БД из окружения (или используем значения по умолчанию)
    dbHost := os.Getenv("DB_HOST")
    if dbHost == "" {
        dbHost = "localhost"
    }
    
    dbPort := os.Getenv("DB_PORT")
    if dbPort == "" {
        dbPort = "5432"
    }
    
    dbUser := os.Getenv("DB_USER")
    if dbUser == "" {
        dbUser = "postgres"
    }
    
    dbPassword := os.Getenv("DB_PASSWORD")
    if dbPassword == "" {
        dbPassword = "postgres"
    }
    
    dbName := os.Getenv("DB_NAME")
    if dbName == "" {
        dbName = "medical_bot"
    }
    
    // Формируем строку подключения с переменными
    dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
        dbHost, dbUser, dbPassword, dbName, dbPort)
    
    log.Printf("Подключение к БД: host=%s dbname=%s", dbHost, dbName)
    
    // Подключаемся к БД
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Ошибка подключения к БД:", err)
    }
    
    log.Println("База данных успешно подключена")
    
    // Автоматическая миграция
    if err := db.AutoMigrate(&telegram.TelegramUser{}, &telegram.TelegramSession{}, &telegram.Reminder{}); err != nil {
        log.Fatal("Ошибка миграции:", err)
    }
    
    log.Println("Миграции успешно применены")
    
    // Создаем бота
    bot, err := telegram.NewTelegramBot(token, db)
    if err != nil {
        log.Fatal("Ошибка создания бота:", err)
    }
    
    // Запускаем в polling-режиме
    log.Println("Бот запущен в polling-режиме")
    go bot.StartPolling()
    
    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Завершение работы...")
}
