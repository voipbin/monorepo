# bin-email-manager — Dependencies

## Events Subscribed

This service has **no SubscribeHandler** and subscribes to no external events. Provider delivery status is received via inbound `POST /v1/hooks` webhooks.

## Events Published

| Exchange | Event Type | Trigger |
|----------|-----------|---------|
| `bin-manager.email-manager.event` | `email.EventTypeCreated` | Email record created |
| `bin-manager.email-manager.event` | `email.EventTypeDeleted` | Email soft-deleted |
| `bin-manager.email-manager.event` | `email.EventTypeUpdated` | Email status updated |

## Outbound RPC Calls (made during request processing)

| Target Service | When | Purpose |
|----------------|------|---------|
| `bin-storage-manager` | During email send (if attachments present) | Fetch file URLs for attachment resolution |

## Local Monorepo Dependencies

| Module | Purpose |
|--------|---------|
| `monorepo/bin-common-handler` | sockhandler, requesthandler, notifyhandler, databasehandler |
| `monorepo/bin-storage-manager` | BucketFile models for attachment resolution |
| `monorepo/bin-billing-manager` | Billing models (indirect dependency) |
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
| `monorepo/bin-flow-manager` | Flow models (indirect dependency) |
| `monorepo/bin-hook-manager` | Hook models; hooks proxied through bin-hook-manager |
| `monorepo/bin-message-manager` | Message models (indirect dependency) |
| `monorepo/bin-number-manager` | Number models (indirect dependency) |
| `monorepo/bin-outdial-manager` | Outdial models (indirect dependency) |
| `monorepo/bin-pipecat-manager` | Pipecat models (indirect dependency) |
| `monorepo/bin-queue-manager` | Queue models (indirect dependency) |
| `monorepo/bin-rag-manager` | RAG models (indirect dependency) |
| `monorepo/bin-registrar-manager` | Registrar models (indirect dependency) |
| `monorepo/bin-route-manager` | Route models (indirect dependency) |
| `monorepo/bin-tag-manager` | Tag models (indirect dependency) |
| `monorepo/bin-timeline-manager` | Timeline models (indirect dependency) |
| `monorepo/bin-transcribe-manager` | Transcribe models (indirect dependency) |
| `monorepo/bin-transfer-manager` | Transfer models (indirect dependency) |
| `monorepo/bin-tts-manager` | TTS models (indirect dependency) |
| `monorepo/bin-webhook-manager` | Webhook models (indirect dependency) |

## External Dependencies

| Service | Purpose |
|---------|---------|
| MySQL | Persistent storage for email records |
| Redis | Email lookup cache |
| RabbitMQ | RPC request queue + event publishing |
| SendGrid API | Primary email delivery provider |
| Mailgun API | Secondary email delivery provider (failover) |

## Services That Depend on This Service

- `bin-billing-manager` — subscribes to `email.EventTypeCreated` to charge accounts for email sends
- `bin-api-manager` — routes email management API calls
- `bin-hook-manager` — forwards provider delivery webhook events to `POST /v1/hooks`
