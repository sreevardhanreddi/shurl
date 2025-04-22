# shurl

A modern, high-performance URL shortening service built with Go.

## Features

- Shorten long URLs to compact, easy-to-share links
- Custom URL aliases (optional)
- Visit analytics and tracking
- RESTful API
- Web UI for link management

## Technology Stack

- **Backend**: Go 1.23
- **Web Framework**: Gin
- **Database**: PostgreSQL 16
- **Containerization**: Docker & Docker Compose
- **Development Tools**: CompileDaemon for hot-reloading

## Prerequisites

- Docker and Docker Compose
- Make (for convenient command execution)
- Go 1.23 (for local development)

## Environment Configuration

This application uses environment variables for configuration. You need to create a `.env` file based on the `.env.example` template:

- `.env.example` - Contains all required environment variables with placeholder or default values.
- `.env` - Your local configuration file (should be added to `.gitignore`). Copy `.env.example` to `.env` and fill in your actual values.

### Required Environment Variables

```
POSTGRES_URI=postgres://postgres:postgres@postgres:5432/postgres?sslmode=disable
PORT=8080
APP_USERNAME=admin
APP_PASSWORD=admin
```

## Quick Start

### Development Environment

1. Clone the repository:

2. Create a `.env` file with the development environment variables.

3. Start the development environment:

   ```bash

   # using the development-specific compose file
   docker compose -f docker-compose.dev.yml up
   ```

4. Access the application at `http://localhost:8080`

### Production Deployment

1. Create a `.env` file with production configuration.

2. Build and start the production environment:

   ```bash
   docker compose -f docker-compose.prod.yml up -d
   ```

3. Run database migrations:
   ```bash
   docker compose -f docker-compose.prod.yml --profile tools run db-migrate
   ```

## Available Make Commands

```
make dev        - Run the application in development mode with hot reload
make run        - Run the application
make build      - Build the application
make clean      - Clean build artifacts
make test       - Run tests
make lint       - Run linter
make fmt        - Format code
make docker-dev - Start development environment
make docker-prod - Start production environment
make migrate-up - Run database migrations up
make migrate-down - Rollback last migration
make migrate-create - Create new migration
```

## API Documentation

### Shorten a URL

```
POST /api/generate
```

**Request Body:**

```json
{
  "url": "https://example.com/very-long-url-that-needs-shortening",
  "custom_alias": "myalias", // Optional: 3-6 alphanumeric characters
  "expires_at": "2025-12-31T23:59:59Z" // Optional: RFC3339 format, must be in the future
}
```

**Success Response (201 Created):**

```json
{
  "status": "success",
  "message": "URL created successfully",
  "data": {
    "id": 1,
    "url": "https://example.com/very-long-url-that-needs-shortening",
    "code": "myalias", // or generated code like "aBcDeF"
    "visits_count": 0,
    "created_at": "2025-04-22T10:00:00Z",
    "updated_at": "2025-04-22T10:00:00Z",
    "expires_at": "2025-12-31T23:59:59Z" // null if not provided
  }
}
```

**Error Responses:**

- `400 Bad Request`: If validation fails (e.g., invalid URL, alias format, expired date) or if a custom alias already exists.
- `500 Internal Server Error`: If there's a database issue or failure to generate a unique alias.

The application provides detailed error responses for validation issues:

```json
{
  "errors": [
    {
      "field": "url",
      "message": "Must be a valid URL",
      "location": "body",
      "value": "not-a-valid-url"
    }
  ]
}
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
