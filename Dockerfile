FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g cmd/app/main.go -o docs --parseInternal --parseDependency

RUN CGO_ENABLED=0 GOOS=linux go build -o /SubscriptionLedger ./cmd/app

# -------

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /SubscriptionLedger .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/docs ./docs

EXPOSE 8080

CMD ["./SubscriptionLedger"]
