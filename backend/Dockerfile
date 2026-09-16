# Stage 1: Build binary
FROM golang:alpine AS builder

WORKDIR /build

# Install CA certificates for secure external calls
RUN apk add --no-cache ca-certificates git

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Compile optimized static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /build/pulsepoll-server cmd/server/main.go

# Stage 2: Minimal production image
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S pulsepoll && adduser -S pulsepoll -G pulsepoll

COPY --from=builder /build/pulsepoll-server /app/pulsepoll-server

USER pulsepoll:pulsepoll

EXPOSE 8080

ENTRYPOINT ["/app/pulsepoll-server"]
