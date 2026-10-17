# Multi-stage Dockerfile for Lukla Operations Engine Backend
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/server cmd/api/main.go

# Final Minimal Runtime Stage
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root application user
RUN adduser -D -u 1000 appuser

COPY --from=builder /app/server /app/server
COPY --from=builder /app/migrations /app/migrations

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/server"]
