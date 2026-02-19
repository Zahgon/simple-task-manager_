# Simple Task Manager

A RESTful task management API built with Go, following clean architecture principles.

## Architecture

```
cmd/app/              → Entry point
internal/
├── config/           → Configuration (koanf, env vars)
├── logger/           → Structured logging (zerolog)
├── database/         → Database migrations (golang-migrate)
├── app/              → Dependency injection and application lifecycle
├── domain/
│   ├── entities/     → Domain models
│   ├── ports/        → Interfaces (repository contracts)
│   └── usecase/      → Business logic
└── adapters/
    ├── http/         → HTTP handlers, router, middleware, DTOs
    └── tasks/        → PostgreSQL repository implementation
migrations/           → SQL migration files
```

Dependencies flow inward: `handlers → use cases → ports ← repository`. The domain layer has no external dependencies.

## Tech Stack

- **Go 1.25**
- **Gin** — HTTP framework
- **PostgreSQL** — Database
- **zerolog** — Structured logging
- **koanf** — Configuration management
- **golang-migrate** — Database migrations
- **testify** — Testing assertions

## Getting Started

### Prerequisites

- Go 1.25+
- Docker and Docker Compose

### Run with Docker

```bash
docker compose up --build
```

This starts PostgreSQL and the API. Migrations run automatically on startup.

### Run Locally

Start only the database:

```bash
docker compose up postgres
```

Run the app:

```bash
go run ./cmd/app
```

### Configuration

All settings are configured via environment variables. See [`.env.example`](.env.example) for the full list.

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8080` | HTTP server port |
| `SERVER_READ_TIMEOUT` | `10s` | Read timeout |
| `SERVER_WRITE_TIMEOUT` | `10s` | Write timeout |
| `SERVER_SHUTDOWN_TIMEOUT` | `5s` | Graceful shutdown timeout |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | Database user |
| `DB_PASSWORD` | `postgres` | Database password |
| `DB_NAME` | `tasks` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode |
| `DB_MIGRATIONS_PATH` | `file://migrations` | Path to migration files |
| `LOG_LEVEL` | `info` | Log level (trace, debug, info, warn, error, fatal) |

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/tasks` | List all tasks (with optional filters) |
| `GET` | `/tasks/:id` | Get a task by ID |
| `POST` | `/tasks` | Create a new task |
| `PUT` | `/tasks/:id` | Update a task |
| `DELETE` | `/tasks/:id` | Delete a task |

### Filter tasks

```
GET /tasks?completed=true&dueBefore=2026-12-31T00:00:00Z&dueAfter=2026-01-01T00:00:00Z
```

### Create a task

```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Buy groceries",
    "description": "Milk, eggs, bread",
    "due_date": "2026-03-01T10:00:00Z"
  }'
```

### Update a task

```bash
curl -X PUT http://localhost:8080/tasks/<id> \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Buy groceries",
    "description": "Milk, eggs, bread, butter",
    "completed": true,
    "due_date": "2026-03-01T10:00:00Z"
  }'
```

### Delete a task

```bash
curl -X DELETE http://localhost:8080/tasks/<id>
```

## Testing

Run unit tests:

```bash
go test ./internal/domain/usecase/tasks/ -v
```

Run integration tests (requires a running PostgreSQL):

```bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/tasks?sslmode=disable" \
  go test ./internal/adapters/tasks/ -v
```

## Migrations

Migrations run automatically on application startup via golang-migrate. Migration files are in the `migrations/` directory.

To add a new migration, create a pair of files:

```
migrations/002_add_index.up.sql
migrations/002_add_index.down.sql
```

The app will apply any pending migrations on the next restart.
