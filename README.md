# SubscriptionLedger

REST API service for aggregating user online subscription data.

## Stack

- **Go 1.22** + **Gin**
- **PostgreSQL 16**
- **pgx/v5**
- **goose** (migrations)
- **swaggo/swag** (Swagger)
- **zap** (logging)
- **Docker Compose**

---

## Running with Docker

```bash
cp .env.example .env
docker compose up --build
```

Service: http://localhost:8080  
Swagger UI: http://localhost:8080/swagger/index.html

---

## Running locally

**Prerequisites:** Go 1.22+, PostgreSQL, swag CLI

```bash
go install github.com/swaggo/swag/cmd/swag@latest
export PATH=$PATH:$(go env GOPATH)/bin

swag init -g cmd/app/main.go -o docs --parseInternal --parseDependency
go mod tidy
go run ./cmd/app
```

Configure DB credentials in `config.yaml` or `.env`.

---

## API Reference

### Create subscription
```
POST /api/v1/subscriptions
```
```json
{
  "service_name": "Yandex Plus",
  "price": 400,
  "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
  "start_date": "07-2025",
  "end_date": "12-2025"
}
```

### Get subscription
```
GET /api/v1/subscriptions/:id
```

### List subscriptions
```
GET /api/v1/subscriptions?user_id=<uuid>&service_name=<string>
```

### Update subscription
```
PATCH /api/v1/subscriptions/:id
```
```json
{
  "price": 500,
  "end_date": "06-2026"
}
```

### Delete subscription
```
DELETE /api/v1/subscriptions/:id
```

### Total cost for a period
```
GET /api/v1/subscriptions/total-cost?period_from=01-2025&period_to=12-2025&user_id=<uuid>&service_name=<string>
```

All date fields use `MM-YYYY` format. `user_id` and `service_name` filters are optional.

**Response:**
```json
{
  "total_cost": 4800
}
```

### Health check
```
GET /health
```

---

## Project structure

```
.
├── cmd/app/main.go
├── internal/
│   ├── config/        # YAML + env config
│   ├── handler/       # HTTP handlers
│   ├── middleware/    # Request logging
│   ├── model/         # Models and DTOs
│   ├── repository/    # Database layer (pgx)
│   └── service/       # Business logic
├── migrations/        # SQL migrations (goose)
├── docs/              # Generated Swagger docs
├── config.yaml
├── .env.example
├── Dockerfile
└── docker-compose.yml
```
