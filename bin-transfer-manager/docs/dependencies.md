# bin-transfer-manager Dependencies

## Upstream Service Dependencies

Services this service calls via RabbitMQ RPC (from `go.mod` replace directives):

| Service | Purpose |
|---------|---------|
| `bin-common-handler` | Shared models (address, identity, outline), sockhandler, requesthandler, notifyhandler |
| `bin-call-manager` | Confbridge flag management, call muting, MOH, groupcall creation and termination |
| `bin-agent-manager` | Agent identity resolution |
| `bin-billing-manager` | Billing event reporting |
| `bin-campaign-manager` | Campaign context |
| `bin-ai-manager` | AI integration |
| `bin-conference-manager` | Conference context |
| `bin-conversation-manager` | Conversation context |
| `bin-contact-manager` | Contact identity |
| `bin-customer-manager` | Tenant/customer lookups |
| `bin-direct-manager` | Direct channel context |
| `bin-email-manager` | Email integration |
| `bin-flow-manager` | Flow context (outbound call routing) |
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
| `bin-transcribe-manager` | Transcription |
| `bin-transfer-manager` | Self-reference (shared models) |
| `bin-tts-manager` | TTS |
| `bin-webhook-manager` | Webhook dispatch |

## Event Queue Subscriptions

| Queue | Events Consumed |
|-------|----------------|
| `bin-manager.call-manager.event` | `groupcall_progressing`, `groupcall_hangup`, `call_hangup` |

## Infrastructure Dependencies

| Component | Purpose |
|-----------|---------|
| MySQL | Transfer record persistence |
| Redis | Transfer lookups by `transferer_call_id` and `groupcall_id` |
| RabbitMQ | RPC transport + event subscription |

## Downstream Consumers

| Queue | Consumers |
|-------|-----------|
| `bin-manager.transfer-manager.event` | `bin-timeline-manager` (audit log) |
