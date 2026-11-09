# bin-push-manager

Stores the push devices of agents and delivers push notifications (FCM) to them.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file covers only what is specific to `bin-push-manager`.

## Key facts

- **MySQL only** (no Redis, no cache). No events are published.
- **RabbitMQ queue**: `bin-manager.push-manager.request`
- **Subscribes to**: `agent_deleted` (agent-manager), `customer_deleted` (customer-manager): the devices are deleted.
- **Secrets**: the FCM service account is a file (`FCM_SERVICE_ACCOUNT_FILE`). Without a usable file the sender reports "not configured" and every send counts as not sent. The token and the service account content are never logged, returned or put in an error.
- Tokens are stored in `varbinary(512)` (a token is case sensitive and `migrationcheck` allows no other column charset). A token has one owner (unique key).

## Package layout

| Package | Role |
|---------|------|
| `cmd/push-manager` | Daemon entry point |
| `cmd/push-control` | Migration CLI (`migrate status|up|down|force`, database only) |
| `pkg/listenhandler` | RabbitMQ RPC router (regex dispatch) |
| `pkg/subscribehandler` | Event consumer |
| `pkg/devicehandler` | Device registration, expiry, cap, cascade delete |
| `pkg/notificationhandler` | Target selection, parallel delivery (5 s per device), metrics |
| `pkg/pushhandler` | `Sender` interface and the FCM HTTP v1 sender |
| `pkg/dbhandler` | MySQL reads and writes |
| `models/device`, `models/notification` | Domain types |

## Database migrations

The schema of `push_devices` is owned by this service and applied at startup (golang-migrate). Rules and checks are those of [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md). This service has no Alembic history: run the dynamic check as `NO_ALEMBIC=1 DAEMON_BLOCKS=1 scripts/verify-service-migrations.sh bin-push-manager` (`DAEMON_BLOCKS=1` because this service has no cache to fail on, so the daemon keeps retrying the broker after it migrated). Update `scripts/database_scripts_test/table_push_devices.sql` (the sqlite test schema) by hand when the schema changes.

## Adding a notification kind

A new kind (for example a chat message) is one constant in `models/notification`, its entry in `IsValidKind`, and the caller. The sender does not change: the payload is a string map plus the kind.
