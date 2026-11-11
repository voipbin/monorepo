# bin-push-manager

Stores the push devices and the notification settings of agents and delivers push notifications to them through FCM. A device is one installed app of an agent. The kinds are the incoming call (`call_incoming`) and its end (`call_cancelled`), and from events of the other services a chat mention (`chat_mentioned`), a chat message (`chat_message`), a customer message in an owned conversation (`conversation_message`), a case assignment (`case_assigned`) and a case note (`case_note`).

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
| `GET`, `PUT /v1/settings` | Read and switch the notification kinds of an agent |
| `PUT`, `DELETE /v1/mutes/<chat-id>` | Mute and unmute a chat for an agent |

## Dependencies

- **MySQL**: `push_devices`, `push_kind_prefs`, `push_chat_mutes` (owned by this service, migrated at startup)
- **RabbitMQ**: listen queue `bin-manager.push-manager.request`; subscribes to `agent_deleted`, `customer_deleted`, `chatmessage_created`, `conversation_message_created`, `case_note_created` and `case_assigned`
- **FCM**: HTTP v1 API, service account file from `FCM_SERVICE_ACCOUNT_FILE`

## Local Development

```bash
cd bin-push-manager
go build -o ./bin/ ./cmd/...
go test ./...
go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
```
