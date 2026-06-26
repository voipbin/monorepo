# bin-conversation-manager Dependencies

## Local Monorepo Dependencies

All resolved via `replace` directives in `go.mod`.

| Module | Purpose |
|--------|---------|
| `bin-common-handler` | Shared transport (sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler) |
| `bin-message-manager` | SMS/MMS models; RPC client for sending messages |
| `bin-flow-manager` | Flow models (conversation variables injected into flow context) |
| `bin-hook-manager` | Hook/webhook models |
| `bin-number-manager` | Phone number models |
| `bin-agent-manager` | Agent models |
| `bin-billing-manager` | Billing event models |
| `bin-call-manager` | Call models |
| `bin-ai-manager` | AI models |
| `bin-campaign-manager` | Campaign models |
| `bin-conference-manager` | Conference models |
| `bin-contact-manager` | Contact models |
| `bin-customer-manager` | Customer identity |
| `bin-direct-manager` | Direct channel models |
| `bin-email-manager` | Email models |
| `bin-outdial-manager` | Outdial models |
| `bin-pipecat-manager` | Pipecat models |
| `bin-queue-manager` | Queue models |
| `bin-rag-manager` | RAG models |
| `bin-registrar-manager` | Registrar models |
| `bin-route-manager` | Route models |
| `bin-storage-manager` | Storage models |
| `bin-tag-manager` | Tag models |
| `bin-talk-manager` | Talk models |
| `bin-timeline-manager` | Timeline models |
| `bin-transcribe-manager` | Transcription models |
| `bin-transfer-manager` | Transfer models |
| `bin-tts-manager` | TTS models |
| `bin-webhook-manager` | Webhook dispatch |

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| MySQL | Conversation, message, account, media records |
| Redis | Cache for account/conversation lookups |
| RabbitMQ | RPC transport and event pub/sub |
| LINE Bot SDK v7 (`github.com/line/line-bot-sdk-go/v7`) | LINE platform webhook verification and message delivery |
| Meta Graph API (`graph.facebook.com`, v19.0) | WhatsApp Cloud API message delivery; inbound webhook HMAC validation and hub challenge verification (called over plain `net/http`, no SDK) |

## RabbitMQ Queue Names

| Queue | Direction | Purpose |
|-------|-----------|---------|
| `bin-manager.conversation-manager.request` | Inbound | RPC requests |
| `bin-manager.conversation-manager.event` | Outbound | Conversation/message lifecycle events |
| `bin-manager.message-manager.event` | Subscribed | Inbound SMS/MMS notification |
