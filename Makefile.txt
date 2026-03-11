.PHONY: run build test docker-up docker-down

run:
	go run cmd/server/main.go

build:
	go build -o medical-bot cmd/server/main.go

test:
	go test ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

migrate:
	go run cmd/migrate/main.go