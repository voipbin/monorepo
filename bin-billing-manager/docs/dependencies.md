# bin-billing-manager — Dependencies

## Upstream Services (RPC calls made by this service)

This service makes outbound RPC calls to retrieve context for billing operations. Exact RPC targets are resolved at runtime via RabbitMQ.

## Events Subscribed

| Queue | Publisher | Purpose |
|-------|-----------|---------|
| `bin-manager.call-manager.event` | bin-call-manager | Track call billing lifecycle |
| `bin-manager.message-manager.event` | bin-message-manager | Bill SMS messages |
| `bin-manager.email-manager.event` | bin-email-manager | Bill email sends |
| `bin-manager.customer-manager.event` | bin-customer-manager | Create/delete billing accounts on customer lifecycle |
| `bin-manager.number-manager.event` | bin-number-manager | Bill number purchases and renewals |
| `bin-manager.tts-manager.event` | bin-tts-manager | Bill TTS usage |

## Events Published

This service publishes no outbound events. All state changes are reflected via RPC responses and database mutations only.

## Local Monorepo Dependencies

The following packages are imported via `replace` directives in `go.mod`:

| Module | Purpose |
|--------|---------|
| `monorepo/bin-common-handler` | sockhandler, requesthandler, notifyhandler, databasehandler |
| `monorepo/bin-call-manager` | Call event models consumed by subscribehandler |
| `monorepo/bin-message-manager` | Message event models consumed by subscribehandler |
| `monorepo/bin-customer-manager` | Customer event models consumed by subscribehandler |
| `monorepo/bin-number-manager` | Number event models consumed by subscribehandler |
| `monorepo/bin-tts-manager` | TTS event models consumed by subscribehandler |
| `monorepo/bin-email-manager` | Email event models consumed by subscribehandler |
| `monorepo/bin-agent-manager` | Agent models (indirect dependency) |
| `monorepo/bin-contact-manager` | Contact models (indirect dependency) |
| `monorepo/bin-talk-manager` | Talk models (indirect dependency) |
| `monorepo/bin-campaign-manager` | Campaign models (indirect dependency) |
| `monorepo/bin-ai-manager` | AI models (indirect dependency) |
| `monorepo/bin-conference-manager` | Conference models (indirect dependency) |
| `monorepo/bin-conversation-manager` | Conversation models (indirect dependency) |
| `monorepo/bin-flow-manager` | Flow models (indirect dependency) |
| `monorepo/bin-hook-manager` | Hook models (indirect dependency) |
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
| `monorepo/bin-webhook-manager` | Webhook models (indirect dependency) |
| `monorepo/bin-direct-manager` | Direct models (indirect dependency) |
| `monorepo/bin-billing-manager` | Self-reference for shared billing models |

## External Dependencies

| Service | Purpose |
|---------|---------|
| MySQL | Persistent storage for accounts, billings, failed_events |
| Redis | Account lookup cache |
| RabbitMQ | RPC request queue + event subscriptions |
| Paddle | Payment gateway for subscription management and balance top-ups |

## Services That Depend on This Service

Virtually every service in the monorepo calls billing-manager before creating billable resources:
- `bin-api-manager` — routes billing API calls
- `bin-call-manager` — validates balance before call creation
- `bin-message-manager` — validates balance before SMS send
- `bin-email-manager` — validates balance before email send
- `bin-number-manager` — charges on number purchase/renewal
