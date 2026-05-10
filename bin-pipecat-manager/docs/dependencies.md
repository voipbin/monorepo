# bin-pipecat-manager Dependencies

## Local Monorepo Dependencies

All resolved via `replace` directives in `go.mod`.

| Module | Purpose |
|--------|---------|
| `bin-common-handler` | Shared transport (sockhandler, requesthandler, notifyhandler, circuit breaker) |
| `bin-ai-manager` | AI call models; target service for tool execution RPCs |
| `bin-call-manager` | External media models; RPC to create Asterisk WebSocket endpoint |
| `bin-agent-manager` | Agent models |
| `bin-billing-manager` | Billing events |
| `bin-campaign-manager` | Campaign models |
| `bin-conference-manager` | Conference models |
| `bin-contact-manager` | Contact models |
| `bin-conversation-manager` | Conversation models |
| `bin-customer-manager` | Customer identity |
| `bin-direct-manager` | Direct channel models |
| `bin-email-manager` | Email models |
| `bin-flow-manager` | Flow models |
| `bin-hook-manager` | Hook models |
| `bin-message-manager` | Message models |
| `bin-number-manager` | Number models |
| `bin-outdial-manager` | Outdial models |
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
| `bin-webhook-manager` | Webhook models |

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| MySQL | Persistent pipecatcall records |
| Redis | Session cache |
| RabbitMQ | RPC transport (shared + per-pod queues) |
| Asterisk `chan_websocket` | Real-time audio streaming (WebSocket external media) |
| Python `pipecat-ai` | STT/LLM/TTS pipeline execution |
| `soxr` (system lib) | Audio resampling (safety net; not on hot path) |
| OpenAI API | LLM inference (also Grok compatible) |
| Deepgram API | STT |
| Cartesia / ElevenLabs / Google TTS | TTS providers |

## RabbitMQ Queue Names

| Queue | Direction | Notes |
|-------|-----------|-------|
| `bin-manager.pipecat-manager.request` | Inbound | Shared queue for create/get |
| `bin-manager.pipecat-manager.request.<POD_IP>` | Inbound | Per-pod volatile queue for stop/message/ping |
| `bin-manager.pipecat-manager.event` | Outbound | Session lifecycle events |

## Python Dependencies

Managed via `scripts/pipecat/requirements.txt`:
- `pipecat-ai` — core voice pipeline framework
- `fastapi`, `uvicorn` — HTTP server for Go → Python communication
- LLM SDK libraries (openai, google-generativeai, anthropic, etc.)
- STT/TTS provider SDKs
