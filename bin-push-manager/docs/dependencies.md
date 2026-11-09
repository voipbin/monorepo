# bin-push-manager Dependencies

## Upstream Services (consumed via RabbitMQ RPC)

None. The service answers requests and consumes events only.

## Events Subscribed

| Publisher | Event | Handler |
|-----------|-------|---------|
| `agent-manager` | `agent_deleted` | Delete all devices of the agent |
| `customer-manager` | `customer_deleted` | Delete all devices of the customer |

## Events Published

None.

## Infrastructure Dependencies

| Dependency | Use |
|-----------|-----|
| RabbitMQ | RPC request queue and event intake |
| MySQL | `push_devices` (owned, migrated at startup) |
| FCM HTTP v1 | Notification delivery (service account file) |

## Monorepo Module Dependencies

Key local imports:
- `monorepo/bin-common-handler` - sockhandler, utilhandler, databasehandler, migrationhandler
- `monorepo/bin-agent-manager` - agent event model
- `monorepo/bin-customer-manager` - customer event model

## Reverse Dependencies

- `bin-common-handler` `requesthandler` carries the `PushV1*` client methods.
- `bin-api-manager` registers and deletes devices for the signed-in agent.
