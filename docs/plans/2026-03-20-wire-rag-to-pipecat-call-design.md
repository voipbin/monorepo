# Wire RAG to Pipecat Call — Design Document

**Date:** 2026-03-20
**Branch:** NOJIRA-Wire-rag-to-pipecat-call

## Problem Statement

RAG (Retrieval-Augmented Generation) and pipecat voice calls exist as independent systems. There is no way for an AI assistant during a live pipecat call to search a customer's knowledge base. Users want their AI agents to answer questions grounded in uploaded documents, FAQs, and product guides.

## Approach

Add a `search_knowledge` tool to the existing tool infrastructure. The AI model gets a `rag_id` field. When configured, the tool becomes available during pipecat calls. The LLM decides when to search and passes a query string. AI Manager queries RAG Manager via existing RPC, and returns structured results (text + metadata) to the LLM.

**Key decisions:**
- Tool-based (LLM controls when to search) — not automatic per-utterance
- Per-AI configuration (`rag_id` on the AI model) — not per-flow
- Tool execution in AI Manager — follows existing tool dispatch pattern
- Explicit query parameter — LLM rephrases for better retrieval
- Structured response with metadata — document name, section title, relevance score, text

## End-to-End Data Flow

```
User speaks → STT → LLM sees user text
  → LLM calls search_knowledge(query="pricing plans")
  → Python → Go pipecat-manager → AI Manager (AIV1AIcallToolExecute RPC)
  → aicallhandler.ToolHandle() dispatches to toolHandleSearchKnowledge()
    → Fetches AI record to get RagID
    → Calls reqHandler.RagV1RagQuery(ragID, "pricing plans", 5)
    → RAG Manager embeds query, vector search, returns chunks with text
    → AI Manager formats structured response
  → Response flows back: AI Manager → pipecat-manager → Python → LLM context
  → LLM formulates verbal answer using RAG context
  → TTS → User hears answer
```

No new RPC endpoints. No new WebSocket protocols. No Python runner changes.

## Changes by Service

### 1. bin-rag-manager (2 files)

**Add `Text` field to query response.** Currently `query.Source` returns only metadata (DocumentID, DocumentName, SectionTitle, RelevanceScore) — the LLM needs the actual chunk text.

- `models/query/main.go` — Add `Text string` field to `Source` struct
- `pkg/raghandler/query.go` — Populate `Text` from chunk: `sources[i].Text = c.Text`

Backward-compatible additive change.

### 2. bin-ai-manager (10 files)

**AI model — add `RagID` field:**
- `models/ai/main.go` — Add `RagID uuid.UUID` with `db:"rag_id,uuid"` tag, placed after `EngineKey`
- `models/ai/field.go` — Add `FieldRagID Field = "rag_id"`
- `models/ai/webhook.go` — Add `RagID uuid.UUID` to `WebhookMessage`, update `ConvertWebhookMessage()`

**Tool constants:**
- `models/tool/main.go` — Add `ToolNameSearchKnowledge ToolName = "search_knowledge"`, add to `AllToolNames`
- `models/message/tool.go` — Add `FunctionCallNameSearchKnowledge FunctionCallName = "search_knowledge"`

**Tool definition:**
- `pkg/toolhandler/definitions.go` — Add tool with name `search_knowledge`, single parameter `query` (string, required), description guiding LLM on when to use it

**Tool execution:**
- `pkg/aicallhandler/tool.go` — Add `toolHandleSearchKnowledge()` handler and dispatch map entry:
  1. Parse `query` from arguments
  2. Get AI record via `h.aiHandler.Get()` to find `RagID`
  3. Defensive check: `RagID == uuid.Nil` → `fillFailed`
  4. Call `h.reqHandler.RagV1RagQuery(ragID, query, 5)`
  5. Format structured response: `[Source N: "doc" > "section" (relevance: 0.XX)]` + text
  6. Return via `fillSuccess(res, "rag", ragID, formattedText)`

**AIHandler interface:**
- `pkg/aihandler/main.go` — Add `ragID uuid.UUID` parameter to `Create()` and `Update()`
- `pkg/aihandler/ai.go` — Implementation sets `ai.RagID = ragID`

**ListenHandler:**
- `pkg/listenhandler/` — Unmarshal `rag_id` from RPC request body, pass to aiHandler

### 3. bin-common-handler (4 files)

- `pkg/requesthandler/main.go` — Add `ragID uuid.UUID` to `AIV1AICreate` and `AIV1AIUpdate` interface signatures
- `pkg/requesthandler/ai_ais.go` — Add `ragID` to request structs and function implementations
- `pkg/requesthandler/mock_main.go` — Regenerate via `go generate`
- `pkg/requesthandler/ai_ais_test.go` — Update 2 test call sites with new parameter

No changes needed for RAG RPC — `RagV1RagQuery` already exists with correct signature. Queue routing uses hardcoded constant `QueueNameRagRequest`.

### 4. bin-openapi-manager (3 files)

- `openapi/openapi.yaml` — Add `rag_id` to `AIManagerAI` schema after `engine_key`:
  ```yaml
  rag_id:
    type: string
    format: uuid
    x-go-type: string
    description: "The knowledge base ID for the search_knowledge tool. Returned from the `GET /rags` response."
    example: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  ```
- `openapi/paths/ais/main.yaml` — Add `rag_id` to create request schema (NOT required)
- `openapi/paths/ais/id.yaml` — Add `rag_id` to update request schema (NOT required)

### 5. bin-api-manager (5+ files)

- `pkg/servicehandler/main.go` — Add `ragID uuid.UUID` to `AICreate` and `AIUpdate` interface
- `pkg/servicehandler/ai.go` — Add `ragID` param + **RAG ownership validation**:
  ```go
  if ragID != uuid.Nil {
      rag, err := h.reqHandler.RagV1RagGet(ctx, ragID)
      // verify rag.CustomerID == a.CustomerID
  }
  ```
- `pkg/servicehandler/ai_test.go` — Update mock expectations
- Generated code — Regenerate from OpenAPI spec
- `docsdev/source/ai_struct_ai.rst` — Add `rag_id` field documentation, rebuild HTML

### 6. bin-pipecat-manager (1 file)

- `pkg/pipecatcallhandler/runner.go` — In `runnerStartScript()`, after resolving tools via `GetByNames(ai.ToolNames)`, filter out `search_knowledge` if `ai.RagID == uuid.Nil`:
  ```go
  if ai.RagID == uuid.Nil {
      // remove search_knowledge from tools slice
  }
  ```
  Same filtering applies in team flow where per-member tools are resolved.

### 7. bin-dbscheme-manager (1 file)

New Alembic migration:
```python
def upgrade():
    op.execute("ALTER TABLE ai_ais ADD COLUMN rag_id binary(16) AFTER engine_key")

def downgrade():
    op.execute("ALTER TABLE ai_ais DROP COLUMN rag_id")
```

### 8. api-validator (1 file)

Add tests verifying `rag_id` is accepted in AI create/update and returned in AI get.

## Security

- **Cross-tenant RAG access prevention:** API layer validates RAG ownership when setting `rag_id`. The servicehandler checks that the RAG's `CustomerID` matches the authenticated agent's `CustomerID`.
- **Defense-in-depth:** Tool handler defensively checks `RagID != uuid.Nil` even though filtering should prevent registration.
- **No new external attack surface:** Everything flows through existing authenticated tool pipeline.

## Deployment Order

1. **bin-rag-manager** — Add `Text` to query response (backward compatible)
2. **Database migration** — Add `rag_id` column (nullable, existing AIs unaffected)
3. **bin-common-handler** — RPC signature changes, mock regeneration
4. **bin-ai-manager** — Model, tools, handler, aiHandler changes
5. **bin-openapi-manager** — Schema update, regenerate
6. **bin-api-manager** — ServiceHandler + validation, regenerate, RST docs
7. **bin-pipecat-manager** — Tool filtering
8. **api-validator** — Tests

## What Does NOT Change

- Flow Manager — RAG is per-AI config, not per-flow
- Python pipecat runner — tools flow through existing pipeline
- RAG Manager query logic — just adding a field to response struct
- WebSocket/protobuf protocol — no changes
- Existing 9 tools — unaffected

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| RAG query adds ~200ms latency per tool call | Only when LLM chooses to search, bounded by topK=5 |
| RagID points to deleted RAG | Tool returns graceful error, LLM falls back to general knowledge |
| bin-common-handler signature change | Only AIV1AICreate/Update affected, only api-manager calls them |
| Migration before code deploy gap | `rag_id` is nullable, existing code ignores it |
| Cross-tenant data leak via rag_id | API layer validates RAG ownership at create/update time |

## File Change Summary

~27 files across 7 services + api-validator.
