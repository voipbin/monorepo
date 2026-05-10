# bin-api-manager Operations

## Configuration

Runtime configuration is provided via CLI flags (which can also be set as environment variables). Defined in `cmd/api-manager/main.go`.

| Flag | Env Var | Required | Default | Description |
|------|---------|----------|---------|-------------|
| `-dsn` | `DATABASE_DSN` | Yes | — | MySQL connection string |
| `-rabbit_addr` | `RABBITMQ_ADDRESS` | Yes | — | RabbitMQ address |
| `-redis_addr` | `REDIS_ADDRESS` | Yes | — | Redis address |
| `-redis_pass` | `REDIS_PASSWORD` | No | `""` | Redis password |
| `-redis_db` | `REDIS_DATABASE` | No | `0` | Redis database index |
| `-jwt_key` | `JWT_KEY` | Yes | — | JWT signing key (HMAC secret) |
| `-gcp_project_id` | `GCP_PROJECT_ID` | No | — | GCP project for storage |
| `-gcp_bucket_name` | `GCP_BUCKET_NAME` | No | — | GCS bucket for media/recordings |
| `-ssl_cert_base64` | `SSL_CERT_BASE64` | No | — | Base64-encoded SSL certificate |
| `-ssl_privkey_base64` | `SSL_PRIVATE_BASE64` | No | — | Base64-encoded SSL private key |
| `-listen_ip_audiosock` | `LISTEN_IP_AUDIOSOCK` | No | `""` | Audiosocket listener IP (AI audio streaming) |
| `-prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | No | `/metrics` | Prometheus metrics path |
| `-prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | No | `:2112` | Prometheus listen address |

SSL certificates are passed as base64-encoded values to allow injection via Kubernetes secrets without multi-line PEM issues.

---

## Prometheus Metrics

Metrics are exposed on the configured listen address (default `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `api_manager_receive_request_process_time` | Histogram | `method`, `path` | HTTP request processing latency |
| `api_manager_subscribe_event_process_time` (equivalent: `receive_subscribe_event_process_time`) | Histogram | `publisher`, `type` | RabbitMQ event processing latency |
| `api_manager_websocket_connections` | Gauge | — | Active WebSocket connections |

Circuit-breaker metrics for each RabbitMQ RPC target are also registered under the `api_manager_*` namespace by `bin-common-handler/pkg/requesthandler`. See [docs/patterns/circuit-breaker.md](../docs/patterns/circuit-breaker.md).

---

## Common Failure Modes

### Backend Service Unavailable

**Symptom:** `503 Service Unavailable` or `504 Gateway Timeout` responses for specific resource groups (e.g., all `/calls/*` fail but `/agents/*` succeed).

**Cause:** The corresponding backend manager is down or not consuming from RabbitMQ.

**Diagnosis:**
1. Check RabbitMQ queue depth for the affected service: `bin-manager.<service>.request` queue filling up indicates the consumer is down.
2. Check backend service pod logs in GKE.
3. Check circuit-breaker metrics — if `api_manager_circuit_breaker_state{target="<service>"}` is `open`, RPC calls are failing fast.

**Resolution:** Restart the affected backend service pod. Circuit breaker resets automatically after the configured timeout.

### Authentication Failures

**Symptom:** `401 AUTHENTICATION_REQUIRED` or `401 INVALID_CREDENTIALS` for requests that should be valid.

**Cause options:**
- JWT token expired (tokens have a configurable TTL)
- JWT signing key mismatch (JWT_KEY environment variable changed)
- Accesskey expired or deleted
- Cookie/header not set correctly

**Diagnosis:**
- Check that `JWT_KEY` matches the key used to issue the token.
- Decode the JWT (e.g., jwt.io) and verify `exp` claim is in the future.
- For accesskeys: query the database for the key and check `tm_expire`, `tm_delete`.

### Account Frozen (403 ACCOUNT_FROZEN)

**Symptom:** Authenticated requests return `403 ACCOUNT_FROZEN` across all endpoints.

**Cause:** The customer account has been frozen (e.g., payment overdue, admin action).

**Resolution:** The response body includes `deletion_scheduled_at` and `recovery_endpoint`. The customer must use `DELETE /auth/unregister` to self-service, or contact support.

### Rate Limiting / RabbitMQ Backpressure

**Symptom:** Requests succeed but with high latency (1-5s+); eventually timeout.

**Cause:** RabbitMQ is overloaded or backend service is processing slowly.

**Diagnosis:**
- Check RabbitMQ management UI for queue depths and consumer counts.
- Check `api_manager_receive_request_process_time` histogram for p99 latency spike.
- Check circuit-breaker state metrics.

### OpenAPI Schema Drift

**Symptom:** Requests succeed but response fields are missing or have wrong types; API validator tests fail.

**Cause:** Backend service `WebhookMessage` struct changed but `bin-openapi-manager/openapi/openapi.yaml` was not updated.

**Resolution:** Compare the backend service's `models/<entity>/webhook.go` against the OpenAPI schema. Update the schema and regenerate. See `bin-openapi-manager/CLAUDE.md` for the full procedure.

---

## Debugging Guide

### Check service health

```bash
# Health check via api-control CLI
./bin/api-control health

# Check metrics endpoint
curl http://localhost:2112/metrics | grep api_manager
```

### Trace a specific request

All error responses include a `request_id` field in the JSON envelope. Set this request ID in the `X-Request-ID` header to correlate logs across services.

```bash
# Grep logs by request ID
kubectl logs -l app=api-manager | grep <request_id>
```

### Check RabbitMQ queue status

```bash
# Via kubectl exec into RabbitMQ pod
rabbitmqctl list_queues name messages consumers

# Key queues to watch:
# bin-manager.call-manager.request
# bin-manager.flow-manager.request
# bin-manager.agent-manager.request
# (etc for each backend service)
```

### Check circuit breaker state

```bash
curl http://localhost:2112/metrics | grep circuit_breaker_state
```

States: `0` = closed (healthy), `1` = open (failing fast), `0.5` = half-open (probing).

### Reproduce auth issues locally

```bash
# Decode JWT claims (without verification)
echo "<jwt_token>" | cut -d. -f2 | base64 -d | jq .

# Key claims to check:
# exp: expiry timestamp
# type: "agent" or "direct"
# agent.customer_id: customer UUID
# agent.permission: permission bitmask
```

---

## Key Dependencies

| Dependency | Purpose |
|-----------|---------|
| `gin-gonic/gin` | HTTP router and middleware |
| `golang-jwt/jwt` | JWT parsing and validation |
| `go-redis/redis` | Redis client (via cachehandler) |
| `amqp091-go` | RabbitMQ client (via bin-common-handler) |
| `pebbe/zmq4` | ZeroMQ bindings (requires `libzmq` system library) |
| `cloud.google.com/go/storage` | GCS for recordings and media files |
| `oapi-codegen` (build tool) | Generates `gens/openapi_server/gen.go` from OpenAPI spec |

### System prerequisites

ZMQ requires native libraries. Install on Debian/Ubuntu:

```bash
apt update && apt install -y pkg-config libzmq5 libzmq3-dev libczmq4 libczmq-dev
```
