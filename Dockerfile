# Stage 1: Сборка
FROM golang:1.24 AS builder
WORKDIR /tonbot
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -v -o /app ./cmd/app/  # Лучше указать конкретный пакет

# Stage 2: Запуск
FROM gcr.io/distroless/base-debian12
COPY --from=builder /app /app
CMD ["/app"]