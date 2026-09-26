# ─── Build stage ────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Pure Go build — no CGO!
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /srga \
    ./cmd/srga

# ─── Runtime stage — scratch! ───────────────────────────
FROM scratch

COPY --from=builder /srga /srga
COPY --from=builder /app/migrations /migrations
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

VOLUME ["/data"]
EXPOSE 4017

ENV PORT=4017
ENV DATABASE_PATH=/data/srga.db

ENTRYPOINT ["/srga"]
