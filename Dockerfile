# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o docuflow .

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary and web assets
COPY --from=builder /app/docuflow .
COPY --from=builder /app/web ./web

# Create directories for runtime data
RUN mkdir -p uploads

EXPOSE 8080

ENTRYPOINT ["./docuflow"]
