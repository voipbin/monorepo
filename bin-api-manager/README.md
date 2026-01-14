# bin-api-manager

RESTful API gateway for the VoIPBIN platform. Handles authentication, routing, and API requests to various backend microservices.

## Overview

`bin-api-manager` serves as the public-facing API for the entire VoIPBIN system. It:
- Provides REST API endpoints for external clients
- Authenticates requests via JWT tokens or access keys
- Routes requests to backend managers via RabbitMQ
- Serves API documentation (Swagger UI, ReDoc)
- Serves developer documentation (Sphinx-based docs)

## Prerequisites

### System Dependencies
```bash
# ZMQ libraries (required for messaging)
apt update && apt install -y pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
```

### Go Setup
```bash
# Configure access to private Go modules (if needed)
git config --global url."https://${GL_DEPLOY_USER}:${GL_DEPLOY_TOKEN}@gitlab.com".insteadOf "https://gitlab.com"
export GOPRIVATE="gitlab.com/voipbin"
```

## Building

```bash
# Download dependencies
go mod download
go mod vendor

# Build the service
go build -o api-manager ./cmd/api-manager

# Or build all commands
go build ./cmd/...
```

## Running

### Basic Usage
```bash
./api-manager \
  -dsn "user:password@tcp(host:3306)/database" \
  -rabbit_addr "amqp://guest:guest@localhost:5672" \
  -redis_addr "localhost:6379" \
  -redis_db 1 \
  -jwt_key "your-secret-key"
```

### Configuration Options

| Flag | Description | Default |
|------|-------------|---------|
| `-dsn` | MySQL connection string | `testid:testpassword@tcp(127.0.0.1:3306)/test` |
| `-rabbit_addr` | RabbitMQ address | `amqp://guest:guest@localhost:5672` |
| `-redis_addr` | Redis address | `127.0.0.1:6379` |
| `-redis_db` | Redis database number | `1` |
| `-redis_password` | Redis password | _(empty)_ |
| `-jwt_key` | JWT signing key | `voipbin` |
| `-gcp_project_id` | GCP project ID | `project` |
| `-gcp_bucket_name` | GCP bucket name | `bucket` |
| `-gcp_credential` | GCP credential file path | `./credential.json` |
| `-ssl_cert_base64` | Base64-encoded SSL certificate | _(empty)_ |
| `-ssl_private_base64` | Base64-encoded SSL private key | _(empty)_ |

### SSL Configuration

The service accepts SSL certificates as base64-encoded strings for containerized environments:

```bash
# Generate base64-encoded certificates
cat your-cert.pem | base64 -w 0
cat your-privkey.pem | base64 -w 0

# Use in command
./api-manager \
  -ssl_cert_base64 "$(cat cert.pem | base64 -w 0)" \
  -ssl_private_base64 "$(cat privkey.pem | base64 -w 0)"
```

## Testing

### Unit Tests
```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -coverprofile cp.out -v ./...
go tool cover -html=cp.out -o cp.html
go tool cover -func=cp.out

# Run specific package tests
go test -v ./pkg/servicehandler/...
```

### API Testing

**Health Check:**
```bash
curl -k https://api.voipbin.net/ping
```

**Authentication:**
```bash
curl -k -X POST https://api.voipbin.net/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"your-username","password":"your-password"}'
```

**API Requests:**
```bash
# Using JWT token in header
curl -k https://api.voipbin.net/v1.0/flows \
  -H "Authorization: Bearer <your-token>"

# Using access key in query parameter
curl -k https://api.voipbin.net/v1.0/flows?accesskey=<your-access-key>
```

### Linting
```bash
# Run go vet
go vet ./...

# Run golangci-lint
golangci-lint run -v --timeout 5m
```

## Documentation

This service maintains two types of documentation:

### 1. API Documentation (Swagger/OpenAPI)

Located at `https://api.voipbin.net/`:
- **Swagger UI**: `/swagger/index.html`
- **ReDoc**: `/redoc/index.html`

**Regenerating Swagger Docs:**
```bash
# Install swag CLI tool
go install github.com/swaggo/swag/cmd/swag@latest

# Format and generate Swagger documentation
swag fmt
swag init --parseDependency --parseInternal -g cmd/api-manager/main.go -o docsapi
```

### 2. Developer Documentation (Sphinx)

Located in `docsdev/` directory, built with Sphinx. Provides comprehensive guides, tutorials, and references for the entire VoIPBIN platform.

**Prerequisites:**
```bash
# Create Python virtual environment
python3 -m venv .venv_docs
source .venv_docs/bin/activate

# Install Sphinx and extensions
pip install sphinx sphinx-rtd-theme sphinx-wagtail-theme sphinxcontrib-youtube
```

**Building Documentation:**
```bash
cd docsdev

# Build HTML documentation
make html

# Or use sphinx-build directly
sphinx-build -M html source build
```

**Output:**
- Generated HTML in `build/html/`
- Open `build/html/index.html` in your browser

**Using Docker:**
```bash
cd docsdev
docker run --rm -v $(pwd):/documents sphinxdoc/sphinx make html
```

**Important:** The generated HTML files in `build/html/` are committed to the repository because the API manager serves them directly via HTTP.

## Project Structure

```
bin-api-manager/
├── cmd/
│   └── api-manager/         # Main application entry point
├── pkg/                     # Core packages
│   ├── servicehandler/      # Business logic coordinator
│   ├── dbhandler/           # Database operations
│   ├── cachehandler/        # Redis caching
│   ├── streamhandler/       # Audio streaming
│   ├── websockhandler/      # WebSocket connections
│   └── subscribehandler/    # RabbitMQ subscriptions
├── lib/                     # HTTP layer
│   ├── middleware/          # JWT authentication
│   └── service/             # HTTP endpoint handlers
├── models/                  # Data structures
├── gens/openapi_server/     # Generated OpenAPI code
├── docsapi/                 # Swagger documentation
├── docsdev/                 # Developer documentation (Sphinx)
│   ├── source/              # RST source files
│   └── build/               # Generated HTML (committed)
└── README.md
```

## Architecture

This service acts as an API gateway:
1. Receives HTTP requests from external clients
2. Validates JWT tokens or access keys
3. Sends RabbitMQ RPC requests to backend managers
4. Returns responses to clients

Backend managers include:
- `bin-call-manager`: Call routing and control
- `bin-flow-manager`: IVR flow execution
- `bin-conference-manager`: Conference management
- `bin-ai-manager`: AI features (STT, TTS, summarization)
- And 20+ other microservices

See [CLAUDE.md](./CLAUDE.md) for detailed architecture and development guidelines.

## Live Services

- **API Gateway**: https://api.voipbin.net/
- **API Documentation (Swagger)**: https://api.voipbin.net/swagger/index.html
- **API Documentation (ReDoc)**: https://api.voipbin.net/redoc/index.html
- **Developer Documentation**: https://docs.voipbin.net/ _(if deployed separately)_
- **Admin Console**: https://admin.voipbin.net/
- **Agent Interface**: https://talk.voipbin.net/

## Development

For detailed development guidelines, including:
- Inter-service communication patterns
- Adding new API endpoints
- Database operations
- Testing patterns
- API schema validation

See [CLAUDE.md](./CLAUDE.md).

## License

Copyright © 2024 VoIPBIN. All rights reserved.
