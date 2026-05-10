# bin-customer-manager — Dependencies

## Events Subscribed

This service has **no SubscribeHandler** and subscribes to no external events. All state changes are driven entirely by inbound RPC requests.

## Events Published

| Exchange | Event Type | Trigger |
|----------|-----------|---------|
| `bin-manager.customer-manager.event` | `customer_created` | New customer created |
| `bin-manager.customer-manager.event` | `customer_updated` | Customer fields updated |
| `bin-manager.customer-manager.event` | `customer_deleted` | Customer soft-deleted |
| `bin-manager.customer-manager.event` | `accesskey_created` | New access key created |
| `bin-manager.customer-manager.event` | `accesskey_updated` | Access key updated |
| `bin-manager.customer-manager.event` | `accesskey_deleted` | Access key soft-deleted |

## Outbound RPC Calls (made during request processing)

| Target Service | When | Purpose |
|----------------|------|---------|
| `bin-agent-manager` | Customer create/signup | Check username conflict with agent email |
| `bin-billing-manager` | Customer create (optional) | Validate billing account exists if provided |

## Local Monorepo Dependencies

| Module | Purpose |
|--------|---------|
| `monorepo/bin-common-handler` | sockhandler, requesthandler, notifyhandler, databasehandler |
| `monorepo/bin-billing-manager` | Billing account models and RPC for validation |
| `monorepo/bin-agent-manager` | Agent models for username conflict check |
| `monorepo/bin-call-manager` | Call models (indirect dependency) |
| `monorepo/bin-direct-manager` | Direct models (indirect dependency) |
| `monorepo/bin-contact-manager` | Contact models (indirect dependency) |
| `monorepo/bin-talk-manager` | Talk models (indirect dependency) |
| `monorepo/bin-campaign-manager` | Campaign models (indirect dependency) |
| `monorepo/bin-ai-manager` | AI models (indirect dependency) |
| `monorepo/bin-conference-manager` | Conference models (indirect dependency) |
| `monorepo/bin-conversation-manager` | Conversation models (indirect dependency) |
| `monorepo/bin-customer-manager` | Self-reference for shared customer models |
| `monorepo/bin-email-manager` | Email models (indirect dependency) |
| `monorepo/bin-flow-manager` | Flow models (indirect dependency) |
| `monorepo/bin-hook-manager` | Hook models (indirect dependency) |
| `monorepo/bin-message-manager` | Message models (indirect dependency) |
| `monorepo/bin-number-manager` | Number models (indirect dependency) |
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
| MySQL | Persistent storage for customers and access keys |
| Redis | Cache-first reads for customer and access key lookups |
| RabbitMQ | RPC request queue + event publishing |

## Services That Depend on This Service

`bin-customer-manager` is foundational — almost every service in the monorepo depends on it:

- `bin-api-manager` — authenticates requests using access keys; validates customer context
- `bin-billing-manager` — subscribes to `customer_created`/`customer_deleted` to manage billing accounts
- `bin-number-manager` — subscribes to `customer_deleted` to release numbers
- All resource-managing services — scope resources by `customer_id` from this service
