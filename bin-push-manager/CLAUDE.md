# bin-push-manager

Stores the push devices and the notification settings of agents and delivers push notifications (FCM) to them.

> Cross-cutting rules (verification workflow, branch/commit format, worktrees, RST sync) live in the root [CLAUDE.md](../CLAUDE.md). This file covers only what is specific to `bin-push-manager`.

## Key facts

- **MySQL only** (no Redis, no cache). No events are published.
- **RabbitMQ queue**: `bin-manager.push-manager.request`
- **Subscribes to**: `agent_deleted` (agent-manager), `customer_deleted` (customer-manager): the devices and the settings are deleted. `chatmessage_created` (talk-manager), `conversation_message_created` (conversation-manager), `case_note_created` and `case_assigned` (contact-manager): the notifications of the new kinds. Delivery is at most once and the event callback always returns nil.
- **Secrets**: the FCM service account is a file (`FCM_SERVICE_ACCOUNT_FILE`). Without a usable file the sender reports "not configured" and every send counts as not sent. The token and the service account content are never logged, returned or put in an error.
- Tokens are stored in `varbinary(512)` (a token is case sensitive and `migrationcheck` allows no other column charset). A token has one owner (unique key).

## Package layout

| Package | Role |
|---------|------|
| `cmd/push-manager` | Daemon entry point |
| `cmd/push-control` | Migration CLI (`migrate status|up|down|force`, database only) |
| `pkg/listenhandler` | RabbitMQ RPC router (regex dispatch) |
| `pkg/subscribehandler` | Event consumer, pool of 32 events |
| `pkg/eventhandler` | Recipients and sends of the event notifications |
| `pkg/settingshandler` | Notification settings, recipient filter |
| `pkg/devicehandler` | Device registration, expiry, cap, cascade delete |
| `pkg/notificationhandler` | Target selection, parallel delivery (5 s per device), metrics |
| `pkg/pushhandler` | `Sender` interface and the FCM HTTP v1 sender |
| `pkg/dbhandler` | MySQL reads and writes |
| `models/device`, `models/notification`, `models/settings` | Domain types; the delivery policy of a kind is in `models/notification/policy.go` |

## Database migrations

The schema of `push_devices`, `push_kind_prefs` and `push_chat_mutes` is owned by this service and applied at startup (golang-migrate). Rules and checks are those of [docs/workflows/service-owned-migrations.md](../docs/workflows/service-owned-migrations.md). Run the dynamic check as `DAEMON_BLOCKS=1 scripts/verify-service-migrations.sh bin-push-manager` (`DAEMON_BLOCKS=1` because this service has no cache to fail on, so the daemon keeps retrying the broker after it migrated). Update `scripts/database_scripts_test/table_push_devices.sql` (the sqlite test schema) by hand when the schema changes.

## Adding a notification kind

A new kind is one constant in `models/notification`, its entry in `IsValidKind` (and in `settingsKinds` if the agent may switch it off), its delivery policy in `policy.go`, and the caller. The sender does not change: the payload is a string map plus the kind. A kind that comes from an event also needs the topic pattern, its case in `subscribehandler` (and the golden test) and a flow in `pkg/eventhandler`. The payload keys of a kind are a contract with the mobile app: change them only together with the app.

`NewRequestHandler` registers metrics and panics when it is built twice in one process: it is built once in `cmd/push-manager`, and the tests of `pkg/eventhandler` use the mock.
