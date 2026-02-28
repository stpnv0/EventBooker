.PHONY: build run test lint mocks migrate-up migrate-down docker-up docker-down clean

BINARY_NAME=server
CMD_PATH=./cmd/server

build:
	go build -o bin/$(BINARY_NAME) $(CMD_PATH)

run: build
	./bin/$(BINARY_NAME)

test:
	go test ./... -count=1 -race

lint:
	golangci-lint run ./...

mocks:
	mockery

migrate-up:
	goose -dir migrations postgres "$(APP_DATABASE_DSN)" up

migrate-down:
	goose -dir migrations postgres "$(APP_DATABASE_DSN)" down

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

