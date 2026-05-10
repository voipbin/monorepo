# bin-tts-manager Dependencies

## Upstream Service Dependencies

Services this service calls via RabbitMQ RPC (from `go.mod` replace directives):

| Service | Purpose |
|---------|---------|
| `bin-common-handler` | Shared models (identity, outline), sockhandler, requesthandler, notifyhandler |
| `bin-call-manager` | External media models for streaming setup (AudioSocket endpoint registration) |
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
| `bin-pipecat-manager` | AI voice pipeline integration |
| `bin-queue-manager` | Queue context |
| `bin-rag-manager` | RAG/knowledge base |
| `bin-registrar-manager` | SIP registrar |
| `bin-route-manager` | Routing decisions |
| `bin-storage-manager` | File storage |
| `bin-tag-manager` | Tagging |
| `bin-talk-manager` | Chat integration |
| `bin-timeline-manager` | Event audit log |
| `bin-transcribe-manager` | Transcription |
| `bin-transfer-manager` | Call transfer |
| `bin-tts-manager` | Self-reference (shared models) |
| `bin-webhook-manager` | Webhook dispatch |

## External Service Dependencies

| Service | Purpose |
|---------|---------|
| Google Cloud TTS | Primary batch synthesis provider (ADC auth, EU regional endpoint) |
| AWS Polly | Fallback batch synthesis provider (access key/secret auth) |
| ElevenLabs | Real-time streaming TTS via WebSocket |

## Infrastructure Dependencies

| Component | Purpose |
|-----------|---------|
| MySQL | TTS session persistence |
| Redis | TTS metadata cache |
| RabbitMQ | RPC transport (shared + per-pod queues) |
| `/shared-data` volume | Audio file storage shared with Python HTTP sidecar |
| Python HTTP sidecar (port 80) | Serves generated audio files from `/shared-data` |
| AudioSocket TCP :8080 | Asterisk media delivery to streaming sessions |

## Downstream Consumers

This service publishes no events via SubscribeHandler. No event queue is published to by this service (no `events_published` in extractor data).
