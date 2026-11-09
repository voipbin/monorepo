# bin-push-manager Architecture

## Component Overview

`bin-push-manager` stores the push devices of agents and delivers push notifications to them through FCM. It has no cache and publishes no events.

```
cmd/push-manager/        - Daemon entry point (pflag/Viper)
cmd/push-control/        - Migration CLI (database only)
pkg/listenhandler/       - RabbitMQ RPC request router (regex URI dispatch)
pkg/subscribehandler/    - Event subscriber (agent_deleted, customer_deleted)
pkg/devicehandler/       - Device registration, expiry, per-agent cap, cascade delete
pkg/notificationhandler/ - Target selection, parallel delivery, metrics
pkg/pushhandler/         - Sender interface and the FCM HTTP v1 sender
pkg/dbhandler/           - MySQL access
models/device/           - Device, WebhookMessage (no token)
models/notification/     - Notification kinds and the send result
migrations/              - Embedded golang-migrate SQL files (service-owned schema of push_devices)
```

## Request Routing

Requests arrive on the RabbitMQ queue `bin-manager.push-manager.request`.

| Pattern | Operations |
|---------|-----------|
| `/v1/devices$` | POST (register or refresh) |
| `/v1/devices\\?(.*)$` | GET (list with body filters and pagination) |
| `/v1/devices/<uuid>$` | GET, DELETE |
| `/v1/notifications$` | POST (send) |

Errors: an unparsable body is a plain 400, an invalid field is a typed 400 (`INVALID_*` reasons), a missing device is a typed 404 (`DEVICE_NOT_FOUND`), anything else is a 500.

### Events Subscribed

The queue `bin-manager.push-manager.subscribe` is bound to the global topic exchange `bin-manager.event` with two patterns, pinned by `pkg/subscribehandler/binding_golden_test.go`:

| Pattern | Purpose |
|---------|---------|
| `agent-manager.agent.*.deleted` | Delete the devices of the agent |
| `customer-manager.customer.*.deleted` | Delete the devices of the customer |

### Events Published

None.

## Delivery

`POST /v1/notifications` selects the live devices (refreshed within 30 days) of the agent, or the one device named by `device_id`, and delivers to each one in its own goroutine with a 5 second limit. The result is `{device_count, sent_count}`. An FCM `UNREGISTERED` answer deletes the device and counts as not sent. Any other failure keeps the device and counts as not sent. A caller treats `sent_count: 0` as "no push was delivered" and a transport error or timeout of the RPC as "unknown".
