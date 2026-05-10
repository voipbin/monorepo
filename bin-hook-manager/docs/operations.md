# bin-hook-manager Operations

## Configuration

All flags support equivalent `UPPER_SNAKE_CASE` environment variables.

| Flag | Env | Description | Required |
|------|-----|-------------|----------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | RabbitMQ connection URL | yes |
| `database_dsn` | `DATABASE_DSN` | MySQL DSN (connected but not used on request path) | yes |
| `ssl_cert_base64` | `SSL_CERT_BASE64` | Base64-encoded SSL certificate | yes |
| `ssl_privkey_base64` | `SSL_PRIVKEY_BASE64` | Base64-encoded SSL private key | yes |
| `paddle_webhook_secret_key` | `PADDLE_WEBHOOK_SECRET_KEY` | Paddle billing webhook signature key | yes |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

SSL certificates are decoded from base64 at startup and written to `/tmp/`. Ensure `/tmp/` is writable in the container.

## Prometheus Metrics

This service has no custom application metrics in the extracted JSON. Standard Go runtime metrics are available via the Prometheus endpoint.

## CLI Tool: hook-control

`cmd/hook-control` — sends test webhook payloads for end-to-end testing.

```bash
./bin/hook-control send-email        --customer_id <uuid> --email_id <uuid>
./bin/hook-control send-message      --customer_id <uuid> --message_id <uuid>
./bin/hook-control send-conversation --customer_id <uuid> --conversation_id <uuid>
```

## Common Commands

```bash
# Build
go build -o hook-manager ./cmd/hook-manager

# Test
go test ./...
go test -v ./api/v1.0/emails
go test -v ./pkg/servicehandler
go test -run TestEmailsPOST ./api/v1.0/emails

# Regenerate mocks
go generate ./...

# Full verification (mandatory before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```

## Deployment Notes

- Service listens on both `:80` (HTTP) and `:443` (HTTPS) simultaneously
- Must be accessible from external internet — DNS and firewall rules must allow inbound HTTPS
- Logging uses logrus + joonix Stackdriver formatter
- CORS allows all origins — expected behavior for a public webhook receiver
