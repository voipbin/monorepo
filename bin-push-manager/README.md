# bin-push-manager

Stores the push devices of agents and delivers push notifications to them through FCM. A device is one installed app of an agent. The first notification kinds are the incoming call (`call_incoming`) and its end (`call_cancelled`).

## Key Concepts

- **Device**: one push token of one agent (`id`, `customer_id`, `agent_id`, `platform`). The token is a credential: it is never part of a response, an event or a log line.
- **Registration**: registering a token that the agent has already refreshes it. A token that belongs to another agent or customer moves to the caller. An agent keeps at most 10 devices (the oldest refresh is removed). A device that was not refreshed for 30 days is not used.
- **Notification**: a request to deliver one kind of notification with string data to the live devices of an agent, or to one device. FCM `UNREGISTERED` removes the device.

## Public RPC Entrypoints

| Pattern | Operations |
|---------|-----------|
| `POST /v1/devices` | Register or refresh a device |
| `GET /v1/devices` | List devices (filters in the body) |
| `GET /v1/devices/<id>` | Get a device |
| `DELETE /v1/devices/<id>` | Delete a device |
| `POST /v1/notifications` | Send a notification |

## Dependencies

- **MySQL**: `push_devices` (owned by this service, migrated at startup)
- **RabbitMQ**: listen queue `bin-manager.push-manager.request`; subscribes to `agent_deleted` and `customer_deleted`
- **FCM**: HTTP v1 API, service account file from `FCM_SERVICE_ACCOUNT_FILE`

## Local Development

```bash
cd bin-push-manager
go build -o ./bin/ ./cmd/...
go test ./...
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
