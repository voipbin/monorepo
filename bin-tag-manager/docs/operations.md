# bin-tag-manager Operations

## Common Failure Modes

| Symptom | Likely Cause | Resolution |
|---------|-------------|-----------|
| Tag create returns conflict | Duplicate tag name for same customer | Use a different name; names must be unique per customer |
| Cascading delete incomplete | `subscribehandler` missed `customer_deleted` event | Check RabbitMQ dead-letter queue; re-publish event |
| Stale tag data after update | Redis cache not invalidated | Restart pod; cache has TTL and will expire |
| Tag list returns 0 items | Soft-delete sentinel query not filtering correctly | Check `tm_delete` column value in DB |
| RabbitMQ connection refused | `RABBITMQ_ADDRESS` misconfigured | Verify secret and network policy |

## Schema Migration Failures

Migrations run at startup, before the service listens. A failure exits the process with a non-zero code.

| Symptom (log) | Meaning | Action |
|---|---|---|
| `Dirty database version N. Fix and force version.` | A previous run stopped inside migration N (failure, kill) | Stop the crash-looping container. Check whether migration N's statement was applied. `tag-control migrate force <N-1>` if not (for N=1 use `force -- -1`), `force <N>` if so (`force -- -1` resets to unversioned), then start again. See the migration section of `CLAUDE.md` |
| `can't acquire lock` | Another instance is migrating for longer than 10 seconds | Restart policy retries; no action unless it persists |
| `Error 1205 ... Lock wait timeout` | Only for ALTER style migrations after the baseline (the baseline does not block on an existing table): a long transaction holds the table; the migration left the database dirty | Find the long transaction, then force `N-1` and restart |
| `CREATE command denied ... tag_schema_migrations` | The database account cannot create tables | Grant CREATE to the service account |
| `db version N is ahead of this image (max M); skipping migrations` (warning) | An older image started after a newer one migrated | Expected during rollback; no action |

A failed first migration shows as version 1 dirty; only a failed down of the first migration is shown as version 0 not dirty (the library hides the marker for an unversioned database), in which case `up` still refuses with `Dirty database version` and the table above applies. `tag-control migrate status` shows the applied version, the image version and the dirty flag (it creates the version table if missing).

## Order on a New Database

On a new database run the Alembic stream first and start the service afterwards. The reverse order is not supported: if the service creates `tag_tags` first, `alembic upgrade head` fails at the `tags` to `tag_tags` rename with `Table 'tag_tags' already exists` (checked by `scripts/verify-service-migrations.sh`). The service account also needs permission to create tables (for the version table).

## Debugging Guide

```bash
# Pod logs
kubectl logs -n voipbin -l app=tag-manager --tail=100

# Admin CLI — list tags for a customer
./bin/tag-control tag list --customer_id <uuid> --limit 50

# Get a specific tag
./bin/tag-control tag get --id <uuid>

# Create a tag directly
./bin/tag-control tag create --customer_id <uuid> --name "my-tag"

# Delete a tag
./bin/tag-control tag delete --id <uuid>

# Build
cd bin-tag-manager && go build -o ./bin/ ./cmd/...

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Lint
golangci-lint run -v --timeout 5m

# Generate mocks
go generate ./...
```

## Deployment

bin-tag-manager deploys via Komodo (VOIP-1347 Tier 1 rollout, following the
VOIP-1342/bin-call-manager pilot pattern) instead of the older SSH +
`versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-tag-manager/komodo/docker-compose.yml` (git is
  the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-tag-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md)
  (in the monorepo root, not this service's own `docs/`).

## Configuration

| Flag / Env | Description | Default |
|-----------|-------------|---------|
| `DATABASE_DSN` | MySQL connection string | `testid:testpassword@tcp(127.0.0.1:3306)/test` |
| `RABBITMQ_ADDRESS` | RabbitMQ server | `amqp://guest:guest@localhost:5672` |
| `REDIS_ADDRESS` | Redis server | `127.0.0.1:6379` |
| `REDIS_PASSWORD` | Redis auth | empty |
| `REDIS_DATABASE` | Redis DB index | `1` |
| `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Prometheus Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `tag_manager_receive_request_process_time` | Histogram | `type`, `method` | RPC request processing duration |
| `tag_manager_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing duration |
