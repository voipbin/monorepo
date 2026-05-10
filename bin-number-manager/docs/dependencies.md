# bin-number-manager — Dependencies

## Events Subscribed

| Queue | Publisher | Purpose |
|-------|-----------|---------|
| `bin-manager.customer-manager.event` | bin-customer-manager | Release all numbers when a customer is deleted |
| `bin-manager.flow-manager.event` | bin-flow-manager | Clear flow references on numbers when a flow is deleted |

## Events Published

| Exchange | Event Type | Trigger |
|----------|-----------|---------|
| `bin-manager.number-manager.event` | `number.EventTypeNumberCreated` | Number purchased successfully |
| `bin-manager.number-manager.event` | `number.EventTypeNumberDeleted` | Number released/deleted |

## Outbound RPC Calls (made during request processing)

| Target Service | When | Purpose |
|----------------|------|---------|
| `bin-billing-manager` | Before number purchase | Validate customer has sufficient balance |

## Local Monorepo Dependencies

| Module | Purpose |
|--------|---------|
| `monorepo/bin-common-handler` | sockhandler, requesthandler, notifyhandler, databasehandler |
| `monorepo/bin-customer-manager` | Customer event models (for `customer_deleted` cascade) |
| `monorepo/bin-flow-manager` | Flow event models (for `flow_deleted` cleanup) |
| `monorepo/bin-billing-manager` | Billing RPC for balance validation |
| `monorepo/bin-call-manager` | Call models (indirect dependency) |
| `monorepo/bin-agent-manager` | Agent models (indirect dependency) |
| `monorepo/bin-direct-manager` | Direct models (indirect dependency) |
| `monorepo/bin-contact-manager` | Contact models (indirect dependency) |
| `monorepo/bin-talk-manager` | Talk models (indirect dependency) |
| `monorepo/bin-campaign-manager` | Campaign models (indirect dependency) |
| `monorepo/bin-ai-manager` | AI models (indirect dependency) |
| `monorepo/bin-conference-manager` | Conference models (indirect dependency) |
| `monorepo/bin-conversation-manager` | Conversation models (indirect dependency) |
| `monorepo/bin-email-manager` | Email models (indirect dependency) |
| `monorepo/bin-hook-manager` | Hook models (indirect dependency) |
| `monorepo/bin-message-manager` | Message models (indirect dependency) |
| `monorepo/bin-number-manager` | Self-reference for shared number models |
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
| MySQL | Persistent storage for numbers and provider mappings |
| Redis | Number lookup cache |
| RabbitMQ | RPC request queue + event pub/sub |
| Telnyx API | Number purchase, release, and available number search |
| Twilio API | Number purchase, release, and available number search |

## Services That Depend on This Service

- `bin-billing-manager` — subscribes to `number.EventTypeNumberCreated`/`Deleted` to charge accounts
- `bin-call-manager` — looks up `call_flow_id` for inbound call routing via number lookup
- `bin-message-manager` — looks up `message_flow_id` for inbound SMS routing
- `bin-api-manager` — routes number management API calls
