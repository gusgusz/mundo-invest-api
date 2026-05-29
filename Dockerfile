# ─────────────────────────────────────────────
# Stage 1: build
# ─────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

# gcc/musl are required for cgo used by the postgres driver
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Cache deps before copying source (faster rebuilds)
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a fully-static binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/api

# ─────────────────────────────────────────────
# Stage 2: runtime (distroless-like minimal image)
# ─────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/server .

# Non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

EXPOSE 8080

ENTRYPOINT ["./server"]
