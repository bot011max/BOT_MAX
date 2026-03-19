cat > go.mod << 'EOF'
module github.com/bot011max/medical-bot

go 1.22

require (
    github.com/gin-contrib/cors v1.5.0
    github.com/gin-gonic/gin v1.9.1
    github.com/go-redis/redis/v8 v8.11.5
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/google/uuid v1.6.0
    github.com/joho/godotenv v1.5.1
    github.com/lib/pq v1.10.9
    github.com/prometheus/client_golang v1.19.0
    golang.org/x/crypto v0.21.0
    golang.org/x/time v0.5.0
    gorm.io/driver/postgres v1.5.7
    gorm.io/gorm v1.25.9
)
EOF
