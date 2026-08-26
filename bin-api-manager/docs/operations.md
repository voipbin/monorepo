# bin-api-manager Operations

## Configuration

Runtime configuration is provided via CLI flags and/or environment variables, defined in `cmd/api-manager/main.go` and `internal/config/main.go`. **The fields marked "(env var only)" below are environment-variable-only** — `internal/config.LoadGlobalConfig()` runs before Cobra parses `argv`, so CLI flags for those fields are inert; set them via env var.

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
| `-ssl_privkey_base64` | `SSL_PRIVKEY_BASE64` | No | — | Base64-encoded SSL private key |
| `-prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | No | `/metrics` | Prometheus metrics path |
| `-prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | No | `:2112` | Prometheus listen address |
| — (env var only) | `RATE_LIMIT_AUTH_PUBLIC_RPS` | No | `10` | Rate limit, requests/second per IP, for unauthenticated `/auth/*` routes. `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_AUTH_PUBLIC_BURST` | No | `20` | Burst size for `RATE_LIMIT_AUTH_PUBLIC_RPS`. `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_AUTH_PROTECTED_RPS` | No | `10` | Rate limit, requests/second per IP, for `/auth/unregister` and `/auth/delegate`. `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_AUTH_PROTECTED_BURST` | No | `20` | Burst size for `RATE_LIMIT_AUTH_PROTECTED_RPS`. `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_V1_RPS` | No | `200` | Rate limit, requests/second per IP, for the full authenticated `v1.0` API surface. `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_V1_BURST` | No | `400` | Burst size for `RATE_LIMIT_V1_RPS`. `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_PROVISIONING_PUBLIC_RPS` | No | `5` | Rate limit, requests/second per IP, for unauthenticated `/provisioning/*` routes (extension QR provisioning). `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_PROVISIONING_PUBLIC_BURST` | No | `10` | Burst size for `RATE_LIMIT_PROVISIONING_PUBLIC_RPS`. `<=0` disables the tier. |
| — (env var only) | `API_PUBLIC_BASE_URL` | No | `https://api.voipbin.net` | Public base URL of this API, used to build absolute URLs handed to external clients (e.g. the extension provisioning URL). Override for self-hosted deployments. |
| — (env var only) | `RATE_LIMIT_CUSTOMER_V1_RPS` | No | `16.7` | Redis-backed rate limit, requests/second per customer, shared by agent and accesskey identities (tier `v1_customer`, `v1` route group only). `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_CUSTOMER_V1_BURST` | No | `33` | Burst size for `RATE_LIMIT_CUSTOMER_V1_RPS`. `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_CUSTOMER_V1_DIRECT_RPS` | No | `50` | Redis-backed rate limit, requests/second per customer, for direct (resource-scoped) identities (tier `v1_customer_direct`). `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_CUSTOMER_V1_DIRECT_BURST` | No | `100` | Burst size for `RATE_LIMIT_CUSTOMER_V1_DIRECT_RPS`. `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_CUSTOMER_V1_DELEGATE_RPS` | No | `8.3` | Redis-backed rate limit, requests/second per customer, for delegate identities (tier `v1_customer_delegate`). `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_CUSTOMER_V1_DELEGATE_BURST` | No | `16` | Burst size for `RATE_LIMIT_CUSTOMER_V1_DELEGATE_RPS`. `<=0` disables the tier. |
| — (env var only) | `RATE_LIMIT_CUSTOMER_REDIS_TIMEOUT_MS` | No | `50` | Timeout budget, in milliseconds, for the customer rate limiter's Redis round trip. On timeout the request fails open (proceeds). This is a jitter-tolerant starting value, not a measured p99 — tune after observing production latency. |

SSL certificates are passed as base64-encoded values to allow injection via Kubernetes secrets without multi-line PEM issues.

### AudioSocket listen vs. advertise address

The AudioSocket/ExternalMedia listener (port 9000, `pkg/streamhandler/`, `cmd/api-manager/main.go`) binds to all interfaces unconditionally (`:9000`) -- this is no longer configurable and is not part of the `Config` struct above. Separately, the *advertise* address -- the host:port handed to Asterisk via `CallV1ExternalMediaStart` so it can dial back -- is resolved by `internal/nethandler.AdvertiseIP()` (a package internal to this service -- see the bin-common-handler admission rule in the root CLAUDE.md for why it is not a shared library package):

1. `POD_IP` env var, if set -- returned verbatim (Kubernetes Downward API `status.podIP`, or an explicit Docker Compose override).
2. Otherwise, auto-detected as the first non-loopback IPv4 address found on the host's network interfaces (in Komodo/Compose deployments, the container's `production` network interface).
3. If neither resolves, the service fails fast at startup (`log.Fatalf`) rather than advertising an empty/unusable host.

`cmd/api-manager/main.go`'s `getAddressAdvertiseAudiosock()` additionally validates the resolved value is a well-formed IP (`net.ParseIP`) and joins it with the port via `net.JoinHostPort` -- not a bare `fmt.Sprintf("%s:%d", ...)` -- so an IPv6 `POD_IP` is bracketed correctly (e.g. `[2001:db8::1]:9000`) instead of producing an unparseable string.

This replaces the old `listen_ip_audiosock` flag (bound to `POD_IP`), which incorrectly fed the same value into both the listen and advertise addresses -- see `pkg/streamhandler/main.go`'s `advertiseAddress` field and `cmd/api-manager/main.go`'s `getAddressListenAudiosock`/`getAddressAdvertiseAudiosock`.

The per-customer rate limiter uses a dedicated `*redis.Client` (`cmd/api-manager/main.go`, `runDaemon`), constructed from the same `REDIS_ADDRESS`/`REDIS_PASSWORD`/`REDIS_DATABASE` values as the cache client but connected separately — `pkg/cachehandler` is intentionally not reused for this (see `pkg/ratelimithandler`). On Kubernetes, ensure the Redis instance's `maxmemory-policy` is `noeviction` or `volatile-*`; an `allkeys-lru` policy can cause rate-limit keys to be evicted under memory pressure, which degrades to the same fail-open behavior as a Redis timeout.

---

## Prometheus Metrics

Metrics are exposed on the configured listen address (default `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `api_manager_receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | RabbitMQ event processing latency |
| `api_manager_pubsub_dropped_message_total` | Counter | — | In-process pub/sub messages dropped because a subscriber buffer was full |
| `api_manager_rate_limit_allowed_total` | Counter | `tier` | Requests allowed by the rate limiter, by tier (`auth_public`, `auth_protected`, `v1`, `provisioning_public`) |
| `api_manager_rate_limit_rejected_total` | Counter | `tier` | Requests rejected (429) by the rate limiter, by tier |

Note: an HTTP request-latency histogram and a WebSocket-connection-count gauge were previously (incorrectly) documented here; neither is registered anywhere in `bin-api-manager` and both have been removed from this table.

Circuit-breaker metrics for each RabbitMQ RPC target are also registered under the `api_manager_*` namespace by `bin-common-handler/pkg/requesthandler`. See [docs/patterns/circuit-breaker.md](../docs/patterns/circuit-breaker.md).

---

## Common Failure Modes

### AudioSocket Advertise Address Resolution Failure (startup crash loop)

**Symptom:** The container crashes immediately after starting and enters a restart/crash loop -- it never reaches a healthy state, so no other failure mode below applies.

**Cause:** `POD_IP` is set to a value that is not a valid IP address (e.g. a hostname string), or (with `POD_IP` unset) no non-loopback IPv4 address could be auto-detected on any network interface. Either way, `internal/nethandler.AdvertiseIP()` / `cmd/api-manager/main.go`'s `getAddressAdvertiseAudiosock()` cannot resolve a usable AudioSocket advertise address, and the service fails fast rather than silently handing Asterisk an empty or unusable host (see the "AudioSocket listen vs. advertise address" subsection above).

**Log signal:** `log.Fatalf` in `cmd/api-manager/main.go`'s `run()` prints, then the process exits:
```
Could not resolve the audiosocket advertise address. err: the resolved audiosocket advertise address is not a valid ip. ip: <value>
```
(or, when no address could be auto-detected at all: `err: could not resolve the audiosocket advertise ip: could not auto-detect advertise ip: no non-loopback ipv4 address found on any network interface`)

**Resolution:** Either unset `POD_IP` so the service auto-detects an address from the container's network interfaces, or set `POD_IP` to a valid, routable IPv4/IPv6 address (not a hostname).

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

### RabbitMQ Backpressure

**Symptom:** Requests succeed but with high latency (1-5s+); eventually timeout.

**Cause:** RabbitMQ is overloaded or backend service is processing slowly.

**Diagnosis:**
- Check RabbitMQ management UI for queue depths and consumer counts.
- Check circuit-breaker state metrics.

### Rate Limiting

**Symptom:** Requests return `429 RATE_LIMIT_EXCEEDED` (no `Retry-After` header is returned).

**Cause:** The client IP exceeded one of four per-IP, in-memory token-bucket tiers enforced by `lib/middleware/ratelimit.go`:

| Tier | Routes | Default | Runs before |
|------|--------|---------|-------------|
| `auth_public` | Unauthenticated `/auth/*` (login, signup, password reset, email-verify, boot) | 10 req/s, burst 20 | (no auth) |
| `auth_protected` | `/auth/unregister`, `/auth/delegate` | 10 req/s, burst 20 | `Authenticate()` |
| `v1` | Entire authenticated `v1.0` API surface (~346 routes) | 200 req/s, burst 400 | `Authenticate()` |
| `provisioning_public` | Unauthenticated `/provisioning/*` (extension QR provisioning) | 5 req/s, burst 10 | (no auth) |

Each tier is independently tunable via the `RATE_LIMIT_*` environment variables in the Configuration section above; setting a tier's RPS or burst to `<=0` disables it (unlimited pass-through) — this is the safe rollback lever if a limit turns out to be too aggressive, and does **not** require a redeploy.

**Important caveats:**
- **Per-pod, not global.** The limiter is in-memory per pod, not Redis-backed. A single client IP's effective ceiling scales with replica count and load-balancer hashing — it is not a hard global ceiling. Do not treat the documented defaults as an exact contractual number.
- **Client IP depends on the Cloudflare header.** `c.ClientIP()` trusts `CF-Connecting-IP` (see `cmd/api-manager/main.go`). If a request reaches a pod without that header (e.g. direct access bypassing Cloudflare), the IP falls back to the raw connection IP, which could be a shared LB/node address — collapsing many distinct clients into one bucket. This is the most likely real-world trigger for an unexpected rate-limit incident from this change.
- **This is a blast-radius safety valve, not a per-customer anti-abuse quota.** Per-IP bucketing cannot distinguish customers behind a shared NAT/proxy, and a determined abuser can rotate IPs. Real anti-abuse enforcement requires per-customer/per-accesskey quotas with shared (Redis-backed) state, which is a separate, larger initiative tracked outside this change.
- **A disabled tier's allowed/rejected ratio reads as 0/0, not "0 = healthy."** Both `api_manager_rate_limit_allowed_total{tier}` and `api_manager_rate_limit_rejected_total{tier}` are pre-initialized to 0 at startup for every configured tier (including disabled ones), so a disabled tier shows no allowed traffic either — don't read that as "no traffic is flowing."

**Diagnosis:**
- Check `api_manager_rate_limit_rejected_total{tier=...}` and `api_manager_rate_limit_allowed_total{tier=...}` to see which tier is rejecting and how it compares to allowed volume.
- If legitimate traffic is being rejected, raise (or temporarily disable) the affected tier's `RATE_LIMIT_*_RPS`/`RATE_LIMIT_*_BURST` env vars — no redeploy required, just a config/env change and pod restart.

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
| `cloud.google.com/go/storage` | GCS for recordings and media files |
| `oapi-codegen` (build tool) | Generates `gens/openapi_server/gen.go` from OpenAPI spec |

### System prerequisites

None. The service is pure Go (no cgo); the internal pub/sub is the in-process `pkg/pubsubhandler` package.

---

## Deployment (Komodo)

Komodo-managed (VOIP-1362, Tier 5 — final service in the install/→Komodo
rollout, migrated last per plan given this service's blast radius: the
sole public HTTP/WebSocket entry point, JWT auth). Deployed via
`.circleci/scripts/render-image-tag.sh` + `.circleci/scripts/komodo-api-deploy.sh`
from `komodo/docker-compose.yml`.

**Fundamentally different cutover from every other `bin-*-manager`
service in this rollout**: every other service communicates purely over
RabbitMQ RPC, so a new Komodo container and the old `install/` container
could both run as competing consumers on the same queue simultaneously —
zero-downtime overlap. `bin-api-manager` instead serves public HTTPS
traffic directly; `install/`'s `docker-compose.yml.dist` publishes it to
the host at `0.0.0.0:8443:443`. Two containers cannot both bind the same
host port, so the RabbitMQ-style overlap trick does not apply here.

**Resolution — route through Caddy instead of publishing a host port**:
`monorepo-etc`'s `infra-caddy` already reverse-proxies `api.voipbin.net`
to this service over the `production` network (`infra-caddy/config/Caddyfile`'s
`reverse_proxy https://voipbin-api-mgr:443`, confirmed live). The Komodo
compose file below therefore:
- Has **no `ports:` block at all** — reachable only via Caddy inside
  `production`, never a host-published port.
- Still self-terminates TLS (`SSL_CERT_BASE64`/`SSL_PRIVKEY_BASE64` from
  the already-existing `[[BIN_MANAGER__SSL_CERT_API_BASE64]]`/
  `[[BIN_MANAGER__SSL_PRIVKEY_API_BASE64]]` secrets) — Caddy connects to
  it over HTTPS, same as it already does for `voipbin-hook-manager`.
- Uses `container_name: voipbin-api-manager` (the fleet's new-service
  naming convention), not the old `voipbin-api-mgr` — `infra-caddy`'s
  Caddyfile target must be updated to match in a companion
  `monorepo-etc` PR.

**Cutover sequence (short downtime, by design — see VOIP-1362)**:
1. Deploy this Stack via CI. The new `voipbin-api-manager` container
   starts and can be fully verified (DB/Redis/RabbitMQ connectivity, TLS
   cert load, GCP credential load) **with zero live-traffic impact** —
   Caddy is still pointed at the old `voipbin-api-mgr` the whole time.
2. Once verified, get explicit go-ahead for the actual cutover window,
   then: stop/remove the old `voipbin-api-mgr` container → merge/deploy
   the companion `monorepo-etc` Caddyfile PR (new target
   `voipbin-api-manager`) → confirm `api.voipbin.net` responds again.
   The downtime window is exactly the gap between those two steps, not
   the whole migration.
3. Comment out (not delete) the `api-manager` block in the live
   `install/docker-compose.yml` on bm-nyc-01, same as every other
   migrated service, to prevent a `container_name`/host-port collision on
   a future host rebuild.

**GCP credential file**: same Docker Compose environment-sourced
`secrets:` block as `bin-storage-manager` — see that service's
`docs/operations.md` for the full mechanism. `GCP_SA_JSON` comes from
`komodo/environment.env`, passed as `komodo-api-deploy.sh`'s 3rd
argument.

**`SSL_PRIVKEY_BASE64` env var name note**: the Configuration table above
lists `SSL_PRIVATE_BASE64` — that is stale/incorrect. The actual env var
`internal/config/main.go` binds via viper is `SSL_PRIVKEY_BASE64`,
matching `install/`'s own `docker-compose.yml.dist`. The Komodo compose
file uses the correct (source-verified) name.

No healthcheck block: distroless (`gcr.io/distroless/static-debian12`),
same as every other distroless `bin-*-manager` service (VOIP-1342 pilot
established the fleet-standard `wget` healthcheck can never pass on this
image).
