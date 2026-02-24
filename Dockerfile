# ==============================================================================
# Stage 1: Build
# ==============================================================================
FROM golang:1.25-alpine AS builder

# Зависимости для сборки
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Копируем go.mod и go.sum отдельно — слой кешируется пока зависимости не изменились
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники и собираем
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/tester \
    ./cmd/app/main.go

# ==============================================================================
# Stage 2: Runtime
# ==============================================================================
FROM alpine:3.20

# Минимальные runtime зависимости
RUN apk add --no-cache ca-certificates tzdata curl

# Создаём непривилегированного пользователя
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Копируем бинарник из builder
COPY --from=builder /app/tester .

# Директория для конфигов — будет перекрыта volume-монтированием
RUN mkdir -p /app/confs
RUN chown -R appuser:appgroup /app

EXPOSE 8080

# Healthcheck
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["./tester"]