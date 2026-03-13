.PHONY: run build test docker-up docker-down migrate clean

run:
	go run cmd/server/main.go

build:
	go build -o medical-bot cmd/server/main.go

test:
	go test -v ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

migrate:
	docker-compose exec postgres psql -U postgres -d medical_bot -f /docker-entrypoint-initdb.d/001_init.sql

clean:
	rm -f medical-bot
	docker-compose down -v

deps:
	go mod download
	go mod tidy

lint:
	go fmt ./...
	go vet ./...
