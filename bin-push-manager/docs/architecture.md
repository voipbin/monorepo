# bin-push-manager Architecture

## Component Overview

`bin-push-manager` stores the push devices of agents, stores their notification settings and delivers push notifications to them through FCM. It decides who is notified of a chat message, a conversation message, a case assignment and a case note from the events of the other services. It has no cache and publishes no events.

```
cmd/push-manager/        - Daemon entry point (pflag/Viper)
cmd/push-control/        - Migration CLI (database only)
pkg/listenhandler/       - RabbitMQ RPC request router (regex URI dispatch)
pkg/subscribehandler/    - Event subscriber, bounded pool of 32 events at a time
pkg/eventhandler/        - Recipients and sends of the chat, conversation and case events
pkg/settingshandler/     - Notification settings (kinds off, muted chats), recipient filter, cascade delete
pkg/devicehandler/       - Device registration, expiry, per-agent cap, cascade delete
pkg/notificationhandler/ - Target selection, parallel delivery, metrics
pkg/pushhandler/         - Sender interface and the FCM HTTP v1 sender
pkg/dbhandler/           - MySQL access
models/device/           - Device, WebhookMessage (no token)
models/notification/     - Notification kinds, delivery policy of a kind and the send result
models/settings/         - Notification settings of an agent
migrations/              - Embedded golang-migrate SQL files (service-owned schema of push_devices, push_kind_prefs, push_chat_mutes)
```

## Request Routing

Requests arrive on the RabbitMQ queue `bin-manager.push-manager.request`.

| Pattern | Operations |
|---------|-----------|
| `/v1/devices$` | POST (register or refresh) |
| `/v1/devices\\?(.*)$` | GET (list with body filters and pagination) |
| `/v1/devices/<uuid>$` | GET, DELETE |
| `/v1/notifications$` | POST (send) |
| `^/v1/settings(\\?(.*))?$` | GET (`customer_id`, `agent_id` in the query), PUT (body `customer_id`, `agent_id`, `kinds`) |
| `^/v1/mutes/<uuid>(\\?(.*))?$` | PUT (mute), DELETE (unmute), `customer_id` and `agent_id` in the query |

The settings and mutes routes are anchored at both ends and are matched before the older routes, whose expressions are not anchored at the start. The settings calls answer the whole settings object `{kinds, muted_chat_ids}`. A kind that can not be switched (the call kinds, an unknown one) is a typed 400 `INVALID_KIND` before anything is written.

Errors: an unparsable body is a plain 400, an invalid field is a typed 400 (`INVALID_*` reasons), a missing device is a typed 404 (`DEVICE_NOT_FOUND`), anything else is a 500.

### Events Subscribed

The queue `bin-manager.push-manager.subscribe` is bound to the global topic exchange `bin-manager.event` with six patterns, pinned by `pkg/subscribehandler/binding_golden_test.go`:

| Pattern | Purpose |
|---------|---------|
| `agent-manager.agent.*.deleted` | Delete the devices and the notification settings of the agent |
| `customer-manager.customer.*.deleted` | Delete the devices and the notification settings of the customer |
| `talk-manager.chatmessage.*.created` | `chat_mentioned` and `chat_message` |
| `conversation-manager.conversation.*.message_created` | `conversation_message` |
| `contact-manager.case.*.note_created` | `case_note` |
| `contact-manager.case.*.assigned` | `case_assigned` |

### Events Published

None.

## Notifications from events

Delivery is at most once: a failure is logged and never retried, and the event callback always returns nil so the broker does not deliver the event again (a redelivery would send the notification twice). No notification is stored, and the server does not know whether the app is open. Every `Send` is scoped by the `customer_id` of the event, so an agent id of another customer finds no device.

| Kind | Event | Recipient |
|------|-------|-----------|
| `chat_mentioned` | `chatmessage_created`, type `normal` | The agents mentioned in the text (`message.ExtractAgentMentions`, at most 50, never the author), also when they are not in the chat. A muted chat does not silence it |
| `chat_message` | `chatmessage_created`, type `normal` | The agent participants of the chat except the author and the mentioned agents, so one message gives an agent one notification at most. Only participants: an agent who reads a public chat without being a participant gets none |
| `conversation_message` | `conversation_message_created`, direction `incoming` | The agent who owns the conversation (checked against the customer of the event) |
| `case_assigned` | `case_assigned` | The new owner of the case, also when he assigned it to himself |
| `case_note` | `case_note_created`, author type `agent` with an author id | The agent who owns the case, unless he wrote the note. A note of any other author type is not notified |

The payload holds ids only, never a message text, a name or a number. The keys are a contract with the app:

| Kind | Data keys (besides `kind`) |
|------|----------------------------|
| `chat_mentioned`, `chat_message` | `chat_id`, `message_id` |
| `conversation_message` | `conversation_id`, `message_id` |
| `case_assigned` | `case_id` |
| `case_note` | `case_id`, `note_id` |

Delivery policy of a kind (`models/notification/policy.go`, first values without a measurement behind them): the call kinds are `HIGH` with a 60 second TTL; `chat_mentioned`, `conversation_message` and `case_assigned` are `HIGH` with a 1 hour TTL; `chat_message` and `case_note` are `NORMAL` with a 1 hour TTL. No kind uses an FCM `collapse_key`: FCM keeps only 4 different collapse keys per device and drops the others without a rule. The app replaces a shown notification by using `<kind>:<id>` as its own notification id.

Limits (`pkg/eventhandler`, `pkg/subscribehandler`): 32 events are processed at the same time; one event runs for at most 30 seconds with 8 sends at a time; a chat message goes to at most 200 recipients. The 200 is applied after the author, the mentioned agents, the kind setting and the mutes are removed, and the participants who joined first are cut first (the list is sorted by join time, newest first). A cut is logged and counted in `push_manager_recipient_truncated_total`.

## Notification settings

Everything is on and nothing is muted until an agent changes it. The settings are stored sparsely: a row in `push_kind_prefs(agent_id, kind)` means the kind is off, a row in `push_chat_mutes(agent_id, chat_id)` means the chat is muted, and switching back deletes the row. There is no `enabled` column. Writing the same state again succeeds (insert, and if it fails look the row up). The settings are read from the database for every event, because two replicas share the queue. `PUT /v1/settings` writes the kinds one by one and is not atomic: on a failure it returns an error, and sending the same request again finishes the change. The kinds that can be switched are `chat_mentioned`, `chat_message`, `conversation_message`, `case_assigned` and `case_note`; the call kinds can not be switched off. The settings of a deleted chat stay and match no event.

## Delivery

`POST /v1/notifications` selects the live devices (refreshed within 30 days) of the agent, or the one device named by `device_id`, and delivers to each one in its own goroutine with a 5 second limit. The result is `{device_count, sent_count}`. An FCM `UNREGISTERED` answer deletes the device and counts as not sent. Any other failure keeps the device and counts as not sent. A caller treats `sent_count: 0` as "no push was delivered" and a transport error or timeout of the RPC as "unknown".
