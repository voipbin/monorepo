# bin-webhook-manager Dependencies

## Local Monorepo Dependencies

All resolved via `replace` directives in `go.mod`.

| Module | Purpose |
|--------|---------|
| `bin-common-handler` | Shared transport (sockhandler, requesthandler, notifyhandler, databasehandler) |
| `bin-customer-manager` | Customer account models; source of webhook URI/method config |
| `bin-agent-manager` | Agent models (indirect) |
| `bin-billing-manager` | Billing models (indirect) |
| `bin-call-manager` | Call models (indirect) |
| `bin-ai-manager` | AI models (indirect) |
| `bin-campaign-manager` | Campaign models (indirect) |
| `bin-conference-manager` | Conference models (indirect) |
| `bin-conversation-manager` | Conversation models (indirect) |
| `bin-direct-manager` | Direct channel models (indirect) |
| `bin-contact-manager` | Contact models (indirect) |
| `bin-email-manager` | Email models (indirect) |
| `bin-flow-manager` | Flow models (indirect) |
| `bin-hook-manager` | Hook models (indirect) |
| `bin-message-manager` | Message models (indirect) |
| `bin-number-manager` | Number models (indirect) |
| `bin-outdial-manager` | Outdial models (indirect) |
| `bin-pipecat-manager` | Pipecat models (indirect) |
| `bin-queue-manager` | Queue models (indirect) |
| `bin-rag-manager` | RAG models (indirect) |
| `bin-registrar-manager` | Registrar models (indirect) |
| `bin-route-manager` | Route models (indirect) |
| `bin-storage-manager` | Storage models (indirect) |
| `bin-tag-manager` | Tag models (indirect) |
| `bin-talk-manager` | Talk models (indirect) |
| `bin-timeline-manager` | Timeline models (indirect) |
| `bin-transcribe-manager` | Transcription models (indirect) |
| `bin-transfer-manager` | Transfer models (indirect) |
| `bin-tts-manager` | TTS models (indirect) |
| `bin-webhook-manager` | Self-reference for models |

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| MySQL | Webhook records |
| Redis | Cache for customer webhook configuration |
| RabbitMQ | RPC transport and event pub/sub |

## RabbitMQ Queue Names

| Queue | Direction | Purpose |
|-------|-----------|---------|
| `bin-manager.webhook-manager.request` | Inbound | RPC requests for webhook dispatch |
| `bin-manager.webhook-manager.event` | Outbound | `webhook_published` events |
| `bin-manager.customer-manager.event` | Subscribed | Cache invalidation on customer changes |
