# Variables
BINARY_NAME=bin
SRC_DIR=./src
MIGRATION_DIR=./src/migrations
POSTGRES_URI?=postgres://postgres:postgres@localhost:5432/shurl?sslmode=disable

# Default target
.DEFAULT_GOAL := help

# Help target
help:
	@echo "Available commands:"
	@echo "  make dev        - Run the application in development mode with hot reload"
	@echo "  make run        - Run the application"
	@echo "  make build      - Build the application"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make test       - Run tests"
	@echo "  make lint       - Run linter"
	@echo "  make fmt        - Format code"
	@echo "  make docker-dev - Start development environment"
	@echo "  make docker-prod - Start production environment"
	@echo "  make migrate-up - Run database migrations up"
	@echo "  make migrate-down - Rollback last migration"
	@echo "  make migrate-create - Create new migration"

# Development
dev:
	CompileDaemon --directory=$(SRC_DIR) --build="go build -o $(BINARY_NAME)" --command="$(SRC_DIR)/$(BINARY_NAME)"

run:
	go run $(SRC_DIR)/main.go

# Build
build:
	go build -o $(BINARY_NAME) $(SRC_DIR)/main.go

clean:
	rm -f $(BINARY_NAME)
	go clean

# Testing
test:
	go test -v ./...

# Code Quality
lint:
	golangci-lint run

fmt:
	go fmt ./...

# Docker
docker-dev:
	docker compose -f docker-compose.dev.yml up --build

docker-dev-down:
	docker compose -f docker-compose.dev.yml down --volumes

docker-prod:
	docker compose -f docker-compose.prod.yml up --build

docker-clean:
	docker compose down --volumes --remove-orphans
	docker system prune -f

# Database Migrations
migrate-up:
	@if [ -z "$(POSTGRES_URI)" ]; then \
		echo "Error: POSTGRES_URI is not set"; \
		exit 1; \
	fi
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path $(MIGRATION_DIR) \
		-database $(POSTGRES_URI) \
		-verbose up

migrate-down:
	@if [ -z "$(POSTGRES_URI)" ]; then \
		echo "Error: POSTGRES_URI is not set"; \
		exit 1; \
	fi
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path $(MIGRATION_DIR) \
		-database $(POSTGRES_URI) \
		-verbose down 1

migrate-version:
	@if [ -z "$(POSTGRES_URI)" ]; then \
		echo "Error: POSTGRES_URI is not set"; \
		exit 1; \
	fi
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path $(MIGRATION_DIR) \
		-database $(POSTGRES_URI) \
		-verbose version

migrate-force:
	@if [ -z "$(POSTGRES_URI)" ]; then \
		echo "Error: POSTGRES_URI is not set"; \
		exit 1; \
	fi
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		-path $(MIGRATION_DIR) \
		-database $(POSTGRES_URI) \
		-verbose force 2

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: name parameter is required. Usage: make migrate-create name=migration_name"; \
		exit 1; \
	fi
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest \
		create -ext sql -dir $(MIGRATION_DIR) -seq $(name)

.PHONY: help dev run build clean test lint fmt docker-dev docker-dev-down docker-prod docker-clean migrate-up migrate-down migrate-version migrate-force migrate-create

