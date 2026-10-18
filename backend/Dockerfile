# Stage 1: Build binary statically
FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Compilação otimizada para produção: estática, sem símbolos de debug (-s -w), sem CGO
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /build/api ./cmd/api

# Stage 2: Runtime Image (Minimal, Hardened & Non-root)
FROM alpine:3.21 AS runner

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S -g 10001 stockpulse && \
    adduser -S -G stockpulse -u 10001 stockpulse

WORKDIR /app

COPY --from=builder --chown=stockpulse:stockpulse /build/api /app/api

USER stockpulse:stockpulse

EXPOSE 8080

CMD ["/app/api"]
