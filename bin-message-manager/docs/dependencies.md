# bin-message-manager — Dependencies

## Events Subscribed

This service has **no SubscribeHandler** and subscribes to no external events. Provider delivery status is received via inbound `POST /v1/hooks` webhooks.

## Events Published

| Exchange | Event Type | Trigger |
|----------|-----------|---------|
| `bin-manager.message-manager.event` | `message.EventTypeMessageCreated` | Message created |
| `bin-manager.message-manager.event` | `message.EventTypeMessageDeleted` | Message soft-deleted |
| `bin-manager.message-manager.event` | `message.EventTypeMessageUpdated` | Message or target status updated |

## Outbound RPC Calls (made during request processing)

| Target Service | When | Purpose |
|----------------|------|---------|
| `bin-billing-manager` | Before SMS send | Validate customer has sufficient balance |

## Local Monorepo Dependencies

| Module | Purpose |
|--------|---------|
| `monorepo/bin-common-handler` | sockhandler, requesthandler, notifyhandler, databasehandler |
| `monorepo/bin-billing-manager` | Billing RPC for balance validation before send |
| `monorepo/bin-number-manager` | Phone number models (indirect dependency) |
| `monorepo/bin-hook-manager` | Webhook models; hooks proxied through bin-hook-manager |
| `monorepo/bin-agent-manager` | Agent models (indirect dependency) |
| `monorepo/bin-call-manager` | Call models (indirect dependency) |
| `monorepo/bin-direct-manager` | Direct models (indirect dependency) |
| `monorepo/bin-contact-manager` | Contact models (indirect dependency) |
| `monorepo/bin-talk-manager` | Talk models (indirect dependency) |
| `monorepo/bin-campaign-manager` | Campaign models (indirect dependency) |
| `monorepo/bin-ai-manager` | AI models (indirect dependency) |
| `monorepo/bin-conference-manager` | Conference models (indirect dependency) |
| `monorepo/bin-conversation-manager` | Conversation models (indirect dependency) |
| `monorepo/bin-customer-manager` | Customer models (indirect dependency) |
| `monorepo/bin-email-manager` | Email models (indirect dependency) |
| `monorepo/bin-flow-manager` | Flow models (indirect dependency) |
| `monorepo/bin-message-manager` | Self-reference for shared message models |
| `monorepo/bin-outdial-manager` | Outdial models (indirect dependency) |
| `monorepo/bin-pipecat-manager` | Pipecat models (indirect dependency) |
| `monorepo/bin-queue-manager` | Queue models (indirect dependency) |
| `monorepo/bin-rag-manager` | RAG models (indirect dependency) |
| `monorepo/bin-registrar-manager` | Registrar models (indirect dependency) |
| `monorepo/bin-route-manager` | Route models (indirect dependency) |
| `monorepo/bin-storage-manager` | Storage models (indirect dependency) |
| `monorepo/bin-tag-manager` | Tag models (indirect dependency) |
| `monorepo/bin-timeline-manager` | Timeline models (indirect dependency) |
| `monorepo/bin-transcribe-manager` | Transcribe models (indirect dependency) |
| `monorepo/bin-transfer-manager` | Transfer models (indirect dependency) |
| `monorepo/bin-tts-manager` | TTS models (indirect dependency) |
| `monorepo/bin-webhook-manager` | Webhook models (indirect dependency) |

## External Dependencies

| Service | Purpose |
|---------|---------|
| MySQL | Persistent storage for messages and targets |
| Redis | Message lookup cache |
| RabbitMQ | RPC request queue + event publishing |
| MessageBird API | Primary SMS gateway |
| Telnyx API | Secondary SMS gateway (fallback) |

## Services That Depend on This Service

- `bin-billing-manager` — subscribes to `message.EventTypeMessageCreated` to charge accounts for SMS
- `bin-api-manager` — routes message API calls
- `bin-hook-manager` — forwards provider delivery webhook events to `POST /v1/hooks`
