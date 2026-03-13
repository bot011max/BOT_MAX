package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/bot011max/BOT_MAX/internal/api"
	"github.com/bot011max/BOT_MAX/internal/models"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Структура для входящих обновлений от Telegram
type TelegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		MessageID int `json:"message_id"`
		From      struct {
			ID           int    `json:"id"`
			IsBot        bool   `json:"is_bot"`
			FirstName    string `json:"first_name"`
			LastName     string `json:"last_name"`
			Username     string `json:"username"`
			LanguageCode string `json:"language_code"`
		} `json:"from"`
		Chat struct {
			ID        int    `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Username  string `json:"username"`
			Type      string `json:"type"`
		} `json:"chat"`
		Date int    `json:"date"`
		Text string `json:"text"`
	} `json:"message"`
}

// Структура для отправки сообщений
type SendMessageRequest struct {
	ChatID    int    `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

var (
	db           *gorm.DB
	telegramToken string
)

func initDB() {
	// Загружаем .env
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем переменные окружения")
	}

	// Подключение к БД
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "postgres"),
		getEnv("DB_NAME", "medical_bot"),
	)
	
	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}
	
	// Получаем токен Telegram из переменных окружения
	telegramToken = os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		log.Fatal("TELEGRAM_TOKEN не задан в .env файле")
	}
	
	log.Println("Telegram bot: база данных подключена")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Отправка сообщения в Telegram [citation:6]
func sendTelegramMessage(chatID int, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", telegramToken)
	
	message := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}
	
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API вернул статус %d", resp.StatusCode)
	}
	
	return nil
}

// Обработка входящих сообщений
func handleTelegramUpdate(w http.ResponseWriter, r *http.Request) {
	var update TelegramUpdate
	
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	
	// Если нет сообщения - игнорируем
	if update.Message == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	chatID := update.Message.Chat.ID
	text := update.Message.Text
	
	log.Printf("Получено сообщение от %d: %s", chatID, text)
	
	// Обработка команд
	switch text {
	case "/start":
		response := "👋 Добро пожаловать в Медицинского бота!\n\n"
		response += "Я помогу вам:\n"
		response += "• Отслеживать приём лекарств 💊\n"
		response += "• Записывать симптомы 📝\n"
		response += "• Получать напоминания о визитах 📅\n\n"
		response += "Для начала работы зарегистрируйтесь через веб-версию."
		
		sendTelegramMessage(chatID, response)
		
	case "/help":
		response := "📋 Доступные команды:\n"
		response += "/start - Начать работу\n"
		response += "/help - Показать эту справку\n"
		response += "/medications - Мои лекарства\n"
		response += "/appointments - Мои визиты"
		
		sendTelegramMessage(chatID, response)
		
	case "/medications":
		// Здесь нужно будет получить лекарства пользователя по его Telegram ID
		// Для этого нужно связать Telegram ID с аккаунтом в системе
		response := "💊 Ваши лекарства:\n"
		response += "• Парацетамол 500 мг - 3 раза в день\n"
		response += "• Амоксициллин 500 мг - 2 раза в день\n\n"
		response += "Чтобы связать Telegram с вашим аккаунтом, войдите в веб-версию."
		
		sendTelegramMessage(chatID, response)
		
	default:
		response := "Я не понимаю эту команду. Напишите /help для списка команд."
		sendTelegramMessage(chatID, response)
	}
	
	w.WriteHeader(http.StatusOK)
}

func main() {
	initDB()
	
	// Создаём маршрут для вебхуков Telegram
	http.HandleFunc("/webhook/telegram", handleTelegramUpdate)
	
	// Запускаем сервер на порту 8081
	port := ":8081"
	log.Printf("Telegram bot слушает на порту%s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
