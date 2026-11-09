# bin-push-manager Domain

## Domain Entities

### Device
One installed app of an agent that can receive notifications.

Fields:
- `id` - UUID, primary key
- `customer_id` - owning customer UUID
- `agent_id` - owning agent UUID
- `platform` - `android`
- `token` - the push token. A credential: stored as `varbinary(512)`, unique across the table, never serialized
- `tm_create` - creation timestamp
- `tm_update` - last refresh timestamp, used for the 30 day expiry

### Notification
Not stored. A kind (`call_incoming`, `call_cancelled`), a string data map and a target (an agent, or one device). The result is `SendResult{device_count, sent_count}`.

## Key Business Rules

- **One owner per token.** Registering a token that belongs to another agent or customer removes the old row and creates the caller's.
- **Idempotent registration.** Registering a token the agent has refreshes `tm_update` and returns the same device. A concurrent duplicate insert starts the registration again.
- **Device cap.** An agent keeps at most 10 devices. The one with the oldest refresh is removed first.
- **Expiry.** A device whose `tm_update` is older than 30 days is not listed and not addressed. The app refreshes the registration on every start.
- **Hard delete.** There is no soft delete: a token is a credential.
- **Cascade delete.** `agent_deleted` and `customer_deleted` delete the matching devices.
- **Reserved data key.** The data key `kind` is set by this service. A caller that sends it gets a 400.
