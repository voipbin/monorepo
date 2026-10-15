# bin-webchat-manager Operations

## Schema Migration Failures

Migrations run at startup, before the RPC listener starts. A failure exits the process with a non-zero code.

| Symptom (log) | Meaning | Action |
|---|---|---|
| `Dirty database version N. Fix and force version.` | A previous run stopped inside migration N (failure, kill) | Stop the crash-looping container. Check whether migration N's statement was applied. `webchat-control migrate force <N-1>` if not (for N=1 use `force -- -1`), `force <N>` if so (`force -- -1` resets to unversioned), then start again. See the migration section of `CLAUDE.md` |
| `can't acquire lock` | Another instance is migrating for longer than 10 seconds | The restart policy retries; no action unless it persists |
| `Error 1205 ... Lock wait timeout` | Only for ALTER style migrations after the baseline (the baseline does not block on an existing table): a long transaction holds the table; the migration left the database dirty | Find the long transaction, then force `N-1` and restart |
| `CREATE command denied ... webchat_schema_migrations` | The database account cannot create tables | Grant CREATE to the service account |
| `db version N is ahead of this image (max M); skipping migrations` (warning) | An older image started after a newer one migrated | Expected during rollback; no action |

`webchat-control migrate status` shows the applied version, the image version and the dirty flag (it creates the version table if missing). `webchat-control` reads `DATABASE_DSN` and opens only the database. In the runtime image (distroless, no shell) run it as a one-off container with the same image and environment: `docker run --rm --entrypoint ./webchat-control <image> migrate status`.

## Order on a New Database

Create the database with `CHARACTER SET utf8 COLLATE utf8_general_ci` (as the install scripts do). On a new database run the Alembic stream first and start the service afterwards. The reverse order is not supported: if the service creates the webchat tables first, `alembic upgrade head` fails with `Table 'webchat_widgets' already exists`. The service account also needs permission to create tables (for the version table).

## Debugging Guide

```bash
# Build
cd bin-webchat-manager && go build -o ./bin/ ./cmd/...

# Migration state of a local database
DATABASE_DSN='<user>:<password>@tcp(127.0.0.1:3306)/<database>' ./bin/webchat-control migrate status

# Run tests
go test ./...

# Lint
golangci-lint run -v --timeout 5m

# Generate mocks
go generate ./...
```

## Deployment

bin-webchat-manager deploys via Komodo (VOIP-1347 Tier 1 rollout, following the VOIP-1342/bin-call-manager pilot pattern).

- **Stack definition:** `bin-webchat-manager/komodo/docker-compose.yml`.
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes the built image tag, then `.circleci/scripts/komodo-api-deploy.sh` pushes the file's content to Komodo and triggers a deploy, gated by the `bin-webchat-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:** [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md) (in the monorepo root, not this service's own `docs/`).

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
