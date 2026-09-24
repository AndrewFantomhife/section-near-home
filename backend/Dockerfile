# Этап 1: компиляция Go-приложения
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bot-server ./cmd/bot/main.go


# Этап 2: минимальный финальный образ
FROM alpine:3.20

RUN apk --no-cache add ca-certificates \
    && adduser -D -u 10001 appuser

WORKDIR /app

COPY --from=builder /bot-server .

USER appuser

EXPOSE 8080

CMD ["./bot-server"]