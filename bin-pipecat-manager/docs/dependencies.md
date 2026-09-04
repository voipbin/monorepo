# bin-pipecat-manager Dependencies

## Local Monorepo Dependencies

All resolved via `replace` directives in `go.mod`.

| Module | Purpose |
|--------|---------|
| `bin-common-handler` | Shared transport (sockhandler, requesthandler, notifyhandler, circuit breaker) |
| `bin-ai-manager` | AI call models; target service for tool execution RPCs; also the source of the Insight-tool whitelist (see note below) |
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

**Redeploy trigger, no source change required:** `pkg/toolhandler/main.go`'s
`GetByNames` filters every tool through `amai.AllowedToolNames(aiType)`
(`bin-ai-manager/models/ai/tool_validation.go`, which derives from
`bin-ai-manager/models/tool`'s `AllInsightToolNames`/`AllToolNames`). Because
that whitelist is a Go value compiled into this service's binary via the
local module `replace` above (not fetched at runtime), **adding or removing a
tool from those lists in `bin-ai-manager` has no effect here until this
service is rebuilt and redeployed** -- even though this service's own source
is untouched. (Tool *definitions* -- name/description/parameters/`run_llm`
-- are different: those are fetched at runtime via `AIV1ToolList` RPC on a
5-minute ticker, so they propagate without a redeploy. Only the whitelist
membership is compile-time.) There is no ordering requirement between the
two services' deploys -- whichever lags simply filters the new tool out
(or doesn't yet know its definition), with no error either way -- but until
both have deployed, the tool is not usable by an Insight AI. See VOIP-1455 (`docs/plans/2026-09-04-insight-assistant-emit-info-card-design.md`
at the monorepo root, §1.1) for the incident that surfaced this: `bin-ai-manager`
deployed a new whitelist entry (`emit_info_card`) and this service was never
redeployed to pick it up, so the new tool silently had no effect for roughly
a day until someone noticed and triggered a manual redeploy.

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
