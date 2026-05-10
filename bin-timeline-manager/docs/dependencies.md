# bin-timeline-manager Dependencies

## Upstream Service Dependencies

Direct RPC call dependencies (from `go.mod` replace directives):

| Service | Purpose |
|---------|---------|
| `bin-common-handler` | Shared models and handlers (sockhandler, requesthandler) |
| `bin-direct-manager` | Direct channel context for event correlation |
| `bin-rag-manager` | RAG/knowledge base context |
| `bin-contact-manager` | Contact entity lookups |

## Event Queue Subscriptions

This service subscribes (read-only) to event queues from 27 services:

| Queue | Publishing Service |
|-------|--------------------|
| `bin-manager.ai-manager.event` | AI manager |
| `bin-manager.agent-manager.event` | Agent manager |
| `asterisk.all.event` | Asterisk media server |
| `bin-manager.billing-manager.event` | Billing manager |
| `bin-manager.call-manager.event` | Call manager |
| `bin-manager.campaign-manager.event` | Campaign manager |
| `bin-manager.conference-manager.event` | Conference manager |
| `bin-manager.contact-manager.event` | Contact manager |
| `bin-manager.conversation-manager.event` | Conversation manager |
| `bin-manager.customer-manager.event` | Customer manager |
| `bin-manager.email-manager.event` | Email manager |
| `bin-manager.flow-manager.event` | Flow manager |
| `bin-manager.message-manager.event` | Message manager |
| `bin-manager.number-manager.event` | Number manager |
| `bin-manager.outdial-manager.event` | Outdial manager |
| `bin-manager.pipecat-manager.event` | Pipecat manager |
| `bin-manager.queue-manager.event` | Queue manager |
| `bin-manager.registrar-manager.event` | Registrar manager |
| `bin-manager.route-manager.event` | Route manager |
| `bin-manager.sentinel-manager.event` | Sentinel manager |
| `bin-manager.storage-manager.event` | Storage manager |
| `bin-manager.tag-manager.event` | Tag manager |
| `bin-manager.talk-manager.event` | Talk manager |
| `bin-manager.transcribe-manager.event` | Transcribe manager |
| `bin-manager.transfer-manager.event` | Transfer manager |
| `bin-manager.tts-manager.event` | TTS manager |
| `bin-manager.webhook-manager.event` | Webhook manager |

## Infrastructure Dependencies

| Component | Purpose |
|-----------|---------|
| ClickHouse | Primary event storage (time-series, high-throughput inserts) |
| RabbitMQ | RPC transport (request queue) and event fan-in (27 subscription queues) |
| Homer API | SIP analysis and PCAP retrieval (external service) |
| GCS | Google Cloud Storage bucket (for PCAP/capture archival) |

## Downstream Consumers

This service publishes no events. It is a read-only query service and write-only ingestion sink.
