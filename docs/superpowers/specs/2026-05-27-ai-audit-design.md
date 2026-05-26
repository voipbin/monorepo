# AI Audit Feature — Design Spec

**Date:** 2026-05-27
**Status:** Draft

---

## Overview

VoIPbin currently stores the full conversation history, tool call logs, and prompt snapshots for every AI call. However, there is no way to evaluate the quality of an AI's behaviour during a call.

This feature adds an on-demand audit capability: a user triggers an audit for a completed AI call, and VoIPbin calls an LLM (Gemini) to evaluate each AI participant's performance. The result — an overall score, per-dimension ratings, and a freeform narrative — is stored and retrievable via API.

---

## Goals

- Allow users to evaluate how well their AI assistants performed during a call
- Evaluate each AI participant independently (one audit record per AI per call)
- Track which prompt version was active during the evaluated call
- Store results persistently; support re-runs (overwrites previous result)
- Support audit output in the same language as the conversation, with per-request override

---

## Non-Goals

- Automatic post-call auditing (on-demand only)
- Call-level aggregate audit (per-AI only)
- User-configurable evaluator model (fixed: Gemini)
- Real-time or streaming audit results

---

## Data Model

### New table: `ai_aiaudits`

One row per `(aicall_id, ai_id)` pair. Re-running an audit upserts (overwrites) the existing row.

| Field | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key |
| `customer_id` | UUID | Owner (copied from aicall) |
| `aicall_id` | UUID | Which call was audited |
| `ai_id` | UUID | Which AI participant was evaluated |
| `prompt_history_id` | UUID | Exact prompt version active during the call (from `metadata.prompt_snapshots`) |
| `status` | enum | `progressing`, `completed`, `failed` |
| `overall_score` | int | 1–5, independently assessed by LLM (null until completed) |
| `evaluation` | JSON | Full evaluation output (null until completed) |
| `language` | string | BCP47 code used for audit output (e.g. `en-US`, `ko-KR`) |
| `error` | string | Failure reason if status is `failed` (null otherwise) |
| `tm_create` | timestamp | When audit was requested |
| `tm_update` | timestamp | When result was last written |
| `tm_delete` | timestamp | Soft-delete (`9999-01-01` for active records) |

**Unique constraint:** `(aicall_id, ai_id)` — one audit per AI per call.

### `evaluation` JSON schema

```json
{
  "overall_score": 4,
  "dimensions": {
    "helpfulness":     { "score": 4, "reason": "Addressed the billing question clearly on first attempt." },
    "accuracy":        { "score": 3, "reason": "Gave correct account info but misstated the refund policy." },
    "tone":            { "score": 5, "reason": "Consistently warm and professional throughout." },
    "goal_completion": { "score": 4, "reason": "Issue resolved, though it required two clarifying turns." },
    "tool_usage":      null
  },
  "summary": "The AI handled a billing inquiry competently with a warm tone. Its strongest moment was quickly identifying the account. Its weakest was misstating the refund window. Consider adding refund policy details to the init_prompt."
}
```

**Dimension definitions:**

| Dimension | What it measures |
|---|---|
| `helpfulness` | Did the AI address the caller's needs effectively? Were responses useful and actionable? |
| `accuracy` | Was the information correct and relevant? Did it avoid hallucination or wrong answers? |
| `tone` | Was the communication style appropriate — professional, empathetic, on-brand? |
| `goal_completion` | Was the primary objective of the interaction achieved by the end of the call? |
| `tool_usage` | Were tools invoked at the right moments with correct arguments? Set to `null` if no tools were used. |

**`overall_score`** is assessed by the LLM independently — it is NOT an average of the dimension scores. The LLM applies its own contextual weighting (e.g. a failed `goal_completion` may outweigh a perfect `tone`). The top-level `overall_score` DB column mirrors the value inside `evaluation` for easy querying without JSON parsing.

**`summary`** is a 3–5 sentence freeform narrative covering: overall impression, the strongest moment in the conversation, the weakest moment, and one concrete suggestion for improving the AI's prompt or behaviour.

---

## API Endpoints

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/v1/aiaudits` | Trigger audit for a call |
| `GET` | `/v1/aiaudits` | List audits (filterable) |
| `GET` | `/v1/aiaudits/{id}` | Get single audit by ID |
| `DELETE` | `/v1/aiaudits/{id}` | Delete an audit record |

### `POST /v1/aiaudits`

Request body:
```json
{
  "aicall_id": "uuid",
  "language": "en-US"
}
```

- `aicall_id` — required
- `language` — optional BCP47 code; defaults to `aicall.stt_language` if omitted

Behaviour:
- Returns `400` if the aicall is not yet `terminated`
- Returns `409` if an audit for any AI in this call is currently `progressing`
- Creates one audit record per AI participant in the aicall (status: `progressing`)
- Spawns one background goroutine per AI to run the Gemini evaluation asynchronously
- Returns `202 Accepted` with the list of created audit records immediately

Response (202):
```json
[
  {
    "id": "uuid",
    "aicall_id": "uuid",
    "ai_id": "uuid",
    "prompt_history_id": "uuid",
    "status": "progressing",
    "language": "en-US",
    ...
  }
]
```

### `GET /v1/aiaudits`

Query parameters:
- `aicall_id` — filter by call
- `ai_id` — filter by AI
- `page_size`, `page_token` — pagination

Returns a paginated list of audit records matching the filters.

### `GET /v1/aiaudits/{id}`

Returns a single audit record by ID.

### `DELETE /v1/aiaudits/{id}`

Soft-deletes the audit record. Note: re-running `POST /v1/aiaudits` for the same call already overwrites the existing result — DELETE is for when the user wants to discard an audit entirely rather than replace it.

---

## Audit Logic

### Per-AI audit job (background goroutine)

1. Load `aicall.metadata["prompt_snapshots"]` → find the snapshot where `ai_id` matches → extract `prompt_history_id` and the exact `prompt` text
2. Load all messages for this aicall where `active_ai_id = ai_id`, ordered by `tm_create`
3. Determine output language: use request `language` if provided, else `aicall.stt_language`
4. Build the Gemini evaluation prompt (see below)
5. Call Gemini, parse the JSON response
6. Upsert audit record: status `completed`, write `evaluation`, write `overall_score`
7. On any error: set status `failed`, write `error` field

### Gemini evaluation prompt

```
You are an AI quality evaluator. You will be given an AI assistant's
system prompt and its conversation transcript. Evaluate the assistant's
performance and return a JSON object.

--- SYSTEM PROMPT ---
{prompt}

--- CONVERSATION TRANSCRIPT ---
[user]: ...
[assistant]: ...
[tool_call]: transfer_call({"destination": "..."}})
[tool_result]: ...
...

--- INSTRUCTIONS ---
Return ONLY valid JSON with this exact structure:
{
  "overall_score": <1-5>,
  "dimensions": {
    "helpfulness":     { "score": <1-5>, "reason": "<1-2 sentences>" },
    "accuracy":        { "score": <1-5>, "reason": "<1-2 sentences>" },
    "tone":            { "score": <1-5>, "reason": "<1-2 sentences>" },
    "goal_completion": { "score": <1-5>, "reason": "<1-2 sentences>" },
    "tool_usage":      <null | { "score": <1-5>, "reason": "<1-2 sentences>" }>
  },
  "summary": "<3-5 sentences>"
}

Score overall_score independently — do not average the dimensions.
Set tool_usage to null if no tools were used in the transcript.
Respond in the following language: {language}
```

### LLM implementation

A new `geminiaudithandler` package inside `bin-ai-manager` handles:
- Building the evaluation prompt
- Calling the Gemini API
- Parsing and validating the JSON response

---

## Error Handling

| Scenario | Behaviour |
|---|---|
| `aicall` not yet `terminated` | `POST /aiaudits` returns `400` |
| An audit for this call is already `progressing` | `POST /aiaudits` returns `409` |
| `aicall` has no messages | Audit runs; Gemini evaluates based on prompt alone; summary notes the empty conversation |
| Gemini returns invalid JSON | Retry once; if still invalid → status `failed`, error: `"invalid response from evaluator"` |
| Gemini API error / timeout | Status `failed`, error: `"evaluator unavailable: <reason>"` |
| Prompt snapshot not found for AI | Status `failed`, error: `"prompt snapshot not found for ai_id"` |

---

## Affected Services

| Service | Changes |
|---|---|
| `bin-ai-manager` | New model `models/aicallaudit/`, new handler `pkg/aicallaudithandler/`, new `geminiaudithandler/`, new RPC handlers |
| `bin-api-manager` | New server handlers for 4 `/aiaudits` endpoints, OpenAPI spec update |
| `bin-dbscheme-manager` | New Alembic migration: `ai_aiaudits` table |
| `bin-api-manager/docsdev` | New RST docs for `aiaudit` resource |

---

## Testing

- **Unit tests** — Gemini prompt builder: verify transcript formatting, tool calls rendered correctly, `null` tool_usage when no tools used
- **Unit tests** — JSON response parser: valid response, invalid JSON, missing fields
- **Integration tests** — audit handler: mock Gemini, verify audit record upserted correctly for single-AI and multi-AI (team) calls
- **API tests** — all 4 endpoints in `bin-api-manager`

---

## Open Questions

None.
