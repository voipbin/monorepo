# bin-hook-manager Dependencies

## Local Monorepo Dependencies

All resolved via `replace` directives in `go.mod`.

| Module | Purpose |
|--------|---------|
| `bin-common-handler` | `requesthandler` for RabbitMQ publish; shared models and utilities |
| `bin-agent-manager` | Agent models (indirect) |
| `bin-billing-manager` | Billing models (indirect) |
| `bin-call-manager` | Call models (indirect) |
| `bin-campaign-manager` | Campaign models (indirect) |
| `bin-ai-manager` | AI models (indirect) |
| `bin-conference-manager` | Conference models (indirect) |
| `bin-conversation-manager` | Conversation models (indirect) |
| `bin-customer-manager` | Customer models (indirect) |
| `bin-direct-manager` | Direct channel models (indirect) |
| `bin-email-manager` | Email models (indirect) |
| `bin-flow-manager` | Flow models (indirect) |
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
| `bin-contact-manager` | Contact models (indirect) |
| `bin-webhook-manager` | Webhook models (indirect) |
| `bin-hook-manager` | Self-reference for models |

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| RabbitMQ | Forwarding webhook payloads to internal service queues |
| MySQL | Connected at startup (currently not actively used on request path) |
| SSL certificates | HTTPS termination (passed via env vars as base64) |

## RabbitMQ Queues Used

`bin-hook-manager` **publishes to** destination service request queues (not an event exchange):

| Target queue | Webhook type |
|-------------|-------------|
| `bin-manager.email-manager.request` | Email webhooks |
| `bin-manager.message-manager.request` | SMS/MMS webhooks |
| `bin-manager.conversation-manager.request` | Conversation webhooks |

This service does **not** have its own inbound RabbitMQ queue.
