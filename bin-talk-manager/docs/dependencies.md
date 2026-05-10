# bin-talk-manager Dependencies

## Upstream Service Dependencies

Services this service calls via RabbitMQ RPC (from `go.mod` replace directives):

| Service | Purpose |
|---------|---------|
| `bin-common-handler` | Shared models (identity, sock, outline), sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler |
| `bin-agent-manager` | Agent identity lookups |
| `bin-billing-manager` | Billing event reporting |
| `bin-call-manager` | Call context for talk sessions |
| `bin-campaign-manager` | Campaign context |
| `bin-ai-manager` | AI integration |
| `bin-conference-manager` | Conference context |
| `bin-conversation-manager` | Conversation linkage |
| `bin-contact-manager` | Contact identity |
| `bin-customer-manager` | Tenant/customer lookups |
| `bin-direct-manager` | Direct channel context |
| `bin-email-manager` | Email integration |
| `bin-flow-manager` | Flow context |
| `bin-hook-manager` | Webhook delivery |
| `bin-message-manager` | Generic messaging |
| `bin-number-manager` | Phone number context |
| `bin-outdial-manager` | Outbound dial context |
| `bin-pipecat-manager` | AI voice pipeline |
| `bin-queue-manager` | Queue context |
| `bin-rag-manager` | RAG/knowledge base |
| `bin-registrar-manager` | SIP registrar |
| `bin-route-manager` | Routing decisions |
| `bin-storage-manager` | File storage |
| `bin-tag-manager` | Tagging |
| `bin-timeline-manager` | Event audit log |
| `bin-talk-manager` | Self-reference (shared models) |
| `bin-transcribe-manager` | Transcription |
| `bin-transfer-manager` | Call transfer |
| `bin-tts-manager` | Text-to-speech |
| `bin-webhook-manager` | Webhook dispatch |

## Infrastructure Dependencies

| Component | Purpose |
|-----------|---------|
| MySQL | Primary persistence (`talk_chats`, `talk_participants`, `talk_messages`) |
| Redis | Cache layer for lookups |
| RabbitMQ | RPC transport (request queue + event queue) |

## Downstream Consumers

Services that consume events published by this service:

| Queue | Consumers |
|-------|-----------|
| `bin-manager.talk-manager.event` | `bin-timeline-manager` (stores all events for audit) |
