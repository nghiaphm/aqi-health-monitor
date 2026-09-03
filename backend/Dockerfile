# ─────────────────────────────
# Stage 1: Build
# ─────────────────────────────
FROM golang:1.26-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

# ─────────────────────────────
# Stage 2a: API image
# ─────────────────────────────
FROM alpine:3.20 AS api
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/api /usr/local/bin/api
COPY migrations ./migrations
EXPOSE 8000
ENTRYPOINT ["api"]

# ─────────────────────────────
# Stage 2b: Worker image
# ─────────────────────────────
FROM alpine:3.20 AS worker
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/worker /usr/local/bin/worker
ENTRYPOINT ["worker"]