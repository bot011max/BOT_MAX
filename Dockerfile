# Этап 1: Сборка приложения
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Копируем файлы для управления зависимостями
COPY go.mod go.sum ./

# Загружаем все зависимости
# Флаг -x показывает подробный вывод (можно убрать)
RUN go mod download -x

# Копируем весь исходный код
COPY . .

# Собираем приложение (статическая сборка)
RUN go build -o medical-bot -ldflags="-w -s" ./cmd/server/main.go

# Этап 2: Финальный образ
FROM alpine:latest

# Устанавливаем необходимые сертификаты и временную зону
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Копируем скомпилированный бинарный файл из этапа сборки
COPY --from=builder /app/medical-bot .

# Копируем веб-файлы (если нужны)
COPY --from=builder /app/web ./web

# Открываем порт
EXPOSE 8080

# Запускаем приложение
CMD ["./medical-bot"]
