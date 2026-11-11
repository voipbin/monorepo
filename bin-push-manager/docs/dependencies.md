# bin-push-manager Dependencies

## Upstream Services (consumed via RabbitMQ RPC)

`bin-talk-manager` (`TalkV1ParticipantList`), `bin-conversation-manager` (`ConversationV1ConversationGet`) and `bin-contact-manager` (`ContactV1CaseGet`) through `requesthandler`, to find the recipients of an event. The request handler registers its metrics, so it is built once per process in `cmd/push-manager`.

## Events Subscribed

| Publisher | Event | Handler |
|-----------|-------|---------|
| `agent-manager` | `agent_deleted` | Delete all devices of the agent |
| `customer-manager` | `customer_deleted` | Delete all devices of the customer |
| `talk-manager` | `chatmessage_created` | `chat_mentioned`, `chat_message` |
| `conversation-manager` | `conversation_message_created` | `conversation_message` |
| `contact-manager` | `case_note_created` | `case_note` |
| `contact-manager` | `case_assigned` | `case_assigned` |

## Events Published

None.

## Infrastructure Dependencies

| Dependency | Use |
|-----------|-----|
| RabbitMQ | RPC request queue and event intake |
| MySQL | `push_devices`, `push_kind_prefs`, `push_chat_mutes` (owned, migrated at startup) |
| FCM HTTP v1 | Notification delivery (service account file) |

## Monorepo Module Dependencies

Key local imports:
- `monorepo/bin-common-handler` - sockhandler, utilhandler, databasehandler, migrationhandler
- `monorepo/bin-agent-manager` - agent event model
- `monorepo/bin-customer-manager` - customer event model
- `monorepo/bin-talk-manager` - chat message and participant models, and `message.ExtractAgentMentions`
- `monorepo/bin-conversation-manager` - conversation message and conversation models
- `monorepo/bin-contact-manager` - case, case note and `CaseAssignedEvent` models

**Deploy together.** `ExtractAgentMentions` (and so the mention expression) is compiled into this service. The CI path filter works per service directory, so a change of the expression in `bin-talk-manager/models/message/mention.go` alone does not rebuild this service: deploy `bin-push-manager` together with `bin-talk-manager` when the expression changes, or the mention notifications follow the old expression.

## Reverse Dependencies

- `bin-common-handler` `requesthandler` carries the `PushV1*` client methods.
- `bin-api-manager` registers and deletes devices for the signed-in agent, and reads and changes the notification settings (`/service_agents/me/push_settings`, `/service_agents/me/push_mutes/{chat_id}`).
- `bin-contact-manager` publishes `case_assigned`; `bin-talk-manager` owns the mention expression.
