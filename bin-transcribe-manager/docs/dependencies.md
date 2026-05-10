# bin-transcribe-manager Dependencies

## Upstream Service Dependencies

Services this service calls via RabbitMQ RPC (from `go.mod` replace directives):

| Service | Purpose |
|---------|---------|
| `bin-common-handler` | Shared models (identity, outline), sockhandler, requesthandler, notifyhandler |
| `bin-call-manager` | External media start/stop for WebSocket audio streams |
| `bin-agent-manager` | Agent identity |
| `bin-billing-manager` | Billing event reporting |
| `bin-campaign-manager` | Campaign context |
| `bin-ai-manager` | AI integration |
| `bin-conference-manager` | Conference context |
| `bin-conversation-manager` | Conversation context |
| `bin-contact-manager` | Contact identity |
| `bin-customer-manager` | Tenant/customer lookups |
| `bin-direct-manager` | Direct channel context |
| `bin-email-manager` | Email integration |
| `bin-flow-manager` | Flow context |
| `bin-hook-manager` | Webhook delivery |
| `bin-message-manager` | Messaging integration |
| `bin-number-manager` | Phone number context |
| `bin-outdial-manager` | Outbound dial context |
| `bin-pipecat-manager` | AI voice pipeline |
| `bin-queue-manager` | Queue context |
| `bin-rag-manager` | RAG/knowledge base |
| `bin-registrar-manager` | SIP registrar |
| `bin-route-manager` | Routing decisions |
| `bin-storage-manager` | File storage |
| `bin-tag-manager` | Tagging |
| `bin-talk-manager` | Chat integration |
| `bin-timeline-manager` | Event audit log |
| `bin-transcribe-manager` | Self-reference (shared models) |
| `bin-transfer-manager` | Call transfer |
| `bin-tts-manager` | TTS |
| `bin-webhook-manager` | Webhook dispatch |

## External Service Dependencies

| Service | Purpose |
|---------|---------|
| Google Cloud Speech-to-Text | Primary STT provider (ADC auth, LINEAR16 8 kHz) |
| AWS Transcribe Streaming | Fallback STT provider (access key auth, PCM 8 kHz) |

## Event Queue Subscriptions

| Queue | Events Consumed |
|-------|----------------|
| `bin-manager.call-manager.event` | `call_hangup` — finalize associated transcription session |
| `bin-manager.customer-manager.event` | `customer_deleted` — cascade delete customer transcriptions |

## Infrastructure Dependencies

| Component | Purpose |
|-----------|---------|
| MySQL | Transcribe session and transcript persistence |
| Redis | Session metadata cache |
| RabbitMQ | RPC transport (shared + per-pod queues) + event subscriptions |
| Asterisk WebSocket (`chan_websocket`) | Audio frame delivery from Asterisk to streaming handler |

## Downstream Consumers

| Queue | Events Published |
|-------|-----------------|
| `bin-manager.transcribe-manager.event` | `transcribe_created`, `transcribe_done`, `transcribe_progressing` |

Consumers include `bin-timeline-manager` (audit log) and any other service subscribing to transcription events.
