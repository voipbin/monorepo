# bin-ai-manager Dependencies

## Local Monorepo Dependencies

All resolved via `replace` directives in `go.mod`.

| Module | Purpose |
|--------|---------|
| `bin-common-handler` | Shared transport (sockhandler, requesthandler, notifyhandler, databasehandler, utilhandler) |
| `bin-call-manager` | Call models and RPC client for telephony operations |
| `bin-pipecat-manager` | Pipecat session models; target of per-pod routing for real-time audio |
| `bin-conversation-manager` | Conversation models for chat-type AIcall references |
| `bin-flow-manager` | Flow models; AI call sessions are triggered and managed via flow-manager actions |
| `bin-agent-manager` | Agent models |
| `bin-billing-manager` | Billing events for AI call duration/usage |
| `bin-campaign-manager` | Campaign models |
| `bin-conference-manager` | Conference bridge models used by AIcall |
| `bin-contact-manager` | Contact models |
| `bin-customer-manager` | Customer identity and auth |
| `bin-direct-manager` | Direct channel models |
| `bin-email-manager` | Email sending (via `send_email` tool) |
| `bin-hook-manager` | Hook/event models |
| `bin-message-manager` | SMS/MMS sending (via `send_message` tool) |
| `bin-number-manager` | Phone number models |
| `bin-outdial-manager` | Outdial campaign models |
| `bin-queue-manager` | Queue models |
| `bin-rag-manager` | RAG (Retrieval-Augmented Generation) knowledge base |
| `bin-registrar-manager` | SIP registrar models |
| `bin-route-manager` | Call routing models |
| `bin-storage-manager` | File storage for recordings |
| `bin-tag-manager` | Tag models |
| `bin-talk-manager` | Talk/agent interaction models |
| `bin-timeline-manager` | Timeline event models |
| `bin-transcribe-manager` | Speech transcription models |
| `bin-transfer-manager` | Call transfer models |
| `bin-tts-manager` | Text-to-speech models |
| `bin-webhook-manager` | Webhook dispatch client |

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| MySQL | Persistent storage (AI configs, AIcalls, messages, summaries) |
| Redis | Cache for AI/AIcall lookups; session state |
| RabbitMQ | RPC transport and event pub/sub |
| OpenAI API | LLM inference (also used for Grok via base URL override) |
| Google Dialogflow API | Dialogflow CX/ES integration |
| Various LLM/STT/TTS provider APIs | Anthropic, AWS Bedrock, Azure OpenAI, Cerebras, DeepSeek, etc. |

## RabbitMQ Queue Names

| Queue | Direction | Purpose |
|-------|-----------|---------|
| `bin-manager.ai-manager.request` | Inbound | RPC requests from other services |
| `bin-manager.ai-manager.event` | Outbound | AI lifecycle events |
| `bin-manager.call-manager.event` | Subscribed | Call hangup / conference events |
| `bin-manager.transcribe-manager.event` | Subscribed | Transcription results |
| `bin-manager.tts-manager.event` | Subscribed | TTS lifecycle events |
| `bin-manager.pipecat-manager.event` | Subscribed | Pipecat session events |
