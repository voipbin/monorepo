# bin-conversation-manager Operations

## Deployment

bin-conversation-manager deploys via Komodo (VOIP-1347 Tier 1 rollout, following the
VOIP-1342/bin-call-manager pilot pattern) instead of the older SSH +
`versions.lock` (`ssh-deploy.sh`) path.

- **Stack definition:** `bin-conversation-manager/komodo/docker-compose.yml` (git is
  the source of truth for structure; Komodo only executes it on
  request).
- **CI path:** `.circleci/scripts/render-image-tag.sh` substitutes
  the built image tag, then `.circleci/scripts/komodo-api-deploy.sh`
  pushes the file's content to Komodo and triggers a deploy, gated
  by the `bin-conversation-manager-deploy` job's poll/running checks.
- **Full design and cutover procedure:**
  [docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md](../../docs/plans/2026-08-18-bin-manager-komodo-rollout-tier1-design.md)
  (in the monorepo root, not this service's own `docs/`).

## Schema Migration Failures

Migrations run at startup, before the service listens. A failure exits the process with a non-zero code.

| Symptom (log) | Meaning | Action |
|---|---|---|
| `Dirty database version N. Fix and force version.` | A previous run stopped inside migration N (failure, kill) | Stop the crash-looping container. Check whether migration N's statement was applied. `conversation-control migrate force <N-1>` if not (for N=1 use `force -- -1`), `force <N>` if so, then start again. See the migration section of `CLAUDE.md` |
| `can't acquire lock` | Another instance is migrating for longer than 10 seconds | The restart policy retries; no action unless it persists |
| `Error 1205 ... Lock wait timeout` | Only for ALTER style migrations after the baseline: a long transaction holds the table; the migration left the database dirty | Find the long transaction, then force `N-1` and restart |
| `CREATE command denied ... conversation_schema_migrations` | The database account cannot create tables | Grant CREATE to the service account |
| `db version N is ahead of this image (max M); skipping migrations` (warning) | An older image started after a newer one migrated | Expected during rollback; no action |

Run the `conversation-control migrate` commands with the DSN in the `DATABASE_DSN` environment variable (for example `DATABASE_DSN=<dsn> conversation-control migrate force 1`); the `--database_dsn` flag is not accepted after `migrate`. `conversation-control migrate status` shows the applied version, the image version and the dirty flag (it creates the version table if missing).

### Order on a New Database

On a new database run the Alembic stream first and start the service afterwards. The reverse order is not supported: if the service creates the conversation tables first, `alembic upgrade head` fails with `Table 'conversation_accounts' already exists`. The new database must be created with the default character set `utf8` (collation `utf8_general_ci`), because the baseline gives no table options. The service account also needs permission to create tables (for the version table).

## Configuration

All flags support equivalent `UPPER_SNAKE_CASE` environment variables.

| Flag | Env | Description | Required |
|------|-----|-------------|----------|
| `rabbitmq_address` | `RABBITMQ_ADDRESS` | RabbitMQ connection URL | yes |
| `database_dsn` | `DATABASE_DSN` | MySQL DSN | yes |
| `redis_address` | `REDIS_ADDRESS` | Redis host:port | yes |
| `redis_password` | `REDIS_PASSWORD` | Redis auth | no |
| `redis_database` | `REDIS_DATABASE` | Redis DB index | no |
| `prometheus_endpoint` | `PROMETHEUS_ENDPOINT` | Metrics path | `/metrics` |
| `prometheus_listen_address` | `PROMETHEUS_LISTEN_ADDRESS` | Metrics listen address | `:2112` |

## Prometheus Metrics

Exposed at `PROMETHEUS_LISTEN_ADDRESS/PROMETHEUS_ENDPOINT` (default `:2112/metrics`).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `account_create_total` | Counter | — | Messaging platform accounts created |
| `receive_request_process_time` | Histogram | `type`, `method` | RPC request latency |
| `receive_subscribe_event_process_time` | Histogram | `publisher`, `type` | Event processing latency |

## CLI Tool: conversation-control

`cmd/conversation-control` — direct DB/cache management (bypasses RabbitMQ). All output is JSON on stdout; logs go to stderr.

```bash
# Conversation commands
./bin/conversation-control conversation get  --id <uuid>
./bin/conversation-control conversation list --customer_id <uuid> [--limit 100] [--token] [--type]

# Account commands
./bin/conversation-control account create --customer_id <uuid> --type <line|sms> --secret <secret> --token <token> [--name] [--detail]
./bin/conversation-control account get    --id <uuid>
./bin/conversation-control account list   --customer_id <uuid> [--limit 100] [--token] [--type]
./bin/conversation-control account update --id <uuid> [--name] [--detail] [--secret] [--token]
./bin/conversation-control account delete --id <uuid>

# Migration commands (database only, DSN from DATABASE_DSN)
./bin/conversation-control migrate status

# Message commands
./bin/conversation-control message get  --id <uuid>
./bin/conversation-control message list --customer_id <uuid> [--limit 100] [--token] [--conversation_id] [--direction] [--status]
```

## Common Commands

```bash
# Build
go build -o bin/conversation-manager ./cmd/conversation-manager

# Test with coverage
go test -coverprofile cp.out -v $(go list ./...)
go tool cover -html=cp.out -o cp.html

# Regenerate mocks
go generate ./...

# Full verification (mandatory before every commit)
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m

# Docker build (from monorepo root)
docker build -f bin-conversation-manager/Dockerfile -t conversation-manager:latest .
```

## Database Setup (local development)

Test schemas in `scripts/database_scripts_test/`:

```bash
mysql -u root -p voipbin < scripts/database_scripts_test/table_conversation_accounts.sql
mysql -u root -p voipbin < scripts/database_scripts_test/table_conversation_conversations.sql
mysql -u root -p voipbin < scripts/database_scripts_test/table_conversation_messages.sql
mysql -u root -p voipbin < scripts/database_scripts_test/table_conversation_medias.sql
```
