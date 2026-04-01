# Design: aimessage_intermediate Event

**Date:** 2026-04-02
**Branch:** NOJIRA-Add-aimessage-intermediate-event

## Problem Statement

When the LLM generates a response during an AI call, the system only publishes a single `aimessage_created` event after the entire response is complete. External clients (web/mobile apps) have no way to show the LLM's response as it's being generated — they must wait for the full message, which can take several seconds.

## Goal

Introduce a new `aimessage_intermediate` webhook event that streams delta text chunks to external clients as the LLM generates tokens, enabling real-time "typing" display.

## Requirements

- **Delta-only content**: Each intermediate event carries only the new tokens since the last event, not the full accumulated text.
- **Time-based batching**: Tokens are batched at 200ms intervals to balance responsiveness vs. event volume (~5 events/sec per conversation).
- **Transient (no DB)**: Intermediate events are fire-and-forget via webhook. Only the final `aimessage_created` is persisted to the database.
- **Shared ID**: All intermediate events and the final `aimessage_created` share the same pre-generated UUID, allowing clients to correlate them.
- **Metadata**: Each intermediate event includes: ID, aicall_id, activeflow_id, role, delta content, and a sequence number for ordering/gap detection.

## Current Flow

```
Python LLM (streaming tokens)
  ↓ RTVIBotLLMText (per-token)
Go Pipecat Manager (runner.go)
  ↓ Accumulates ALL tokens in se.LLMBotText
  ↓ On BotLLMStopped: publish ONE "message_bot_llm" event with full text
AI Manager (subscribehandler)
  ↓ Persist to DB, publish "aimessage_created" webhook
External client
  ↓ Receives complete message only after generation finishes
```

## Proposed Flow

```
Python LLM (streaming tokens)
  ↓ RTVIBotLLMText (per-token)
Go Pipecat Manager (runner.go)
  ↓ Sends token to llmTokenChan
  ↓ Flush goroutine batches tokens, flushes every 200ms
  ↓ Publishes "message_bot_llm_intermediate" events via RabbitMQ
  ↓ On BotLLMStopped: flush remaining, publish "message_bot_llm" with same UUID
AI Manager (subscribehandler)
  ↓ Intermediate: forward as "aimessage_intermediate" webhook (NO DB)
  ↓ Final: persist to DB with pre-generated UUID, publish "aimessage_created" webhook
External client
  ↓ Receives aimessage_intermediate (delta text, sequence number)
  ↓ Receives aimessage_created (final complete message, same ID)
```

## Architecture

### Flush Goroutine (Per-Generation, Channel-Based)

The core design introduces a per-generation flush goroutine spawned lazily on the first `RTVIBotLLMText` token. This avoids concurrency issues with the single-threaded WebSocket read loop.

**Read loop (existing single goroutine):**
```
RTVIBotLLMText:
  if no flush goroutine running:
    generate message UUID
    create llmTokenChan (buffered, cap 64), llmStopChan, llmDoneChan
    spawn flush goroutine
  send token to llmTokenChan

RTVIBotLLMStopped:
  close(llmStopChan)    // signal stop
  <-llmDoneChan          // wait for flush + final publish
  reset state (channels, goroutine flag)
```

**Flush goroutine (per generation, owns all text state):**
```
local state: fullText, deltaBuffer, sequence, ticker (200ms), UUID

select:
  case token := <-llmTokenChan:
    fullText += token
    deltaBuffer += token

  case <-ticker.C:
    if deltaBuffer != "":
      sequence++
      publish "message_bot_llm_intermediate" with {UUID, delta, sequence}
      deltaBuffer = ""

  case <-llmStopChan:
    if deltaBuffer != "":
      sequence++
      publish last intermediate with {UUID, delta, sequence}
    publish "message_bot_llm" with {UUID, fullText}
    close(llmDoneChan)
    return

  case <-ctx.Done():
    drain remaining tokens from llmTokenChan
    if fullText != "":
      publish "message_bot_llm" with {UUID, fullText} using context.Background()
    close(llmDoneChan)
    return
```

### Why Channel-Based (Not Mutex)

- The WebSocket read loop is single-threaded per session. Adding a ticker goroutine that shares buffer state would require a mutex on every token write.
- Channel-based approach keeps all text state local to the flush goroutine — zero shared mutable state.
- The flush goroutine naturally handles ordering: last intermediate always published before the final event.
- Fits existing Go idioms in the codebase.

### Event Ordering Guarantee

The flush goroutine publishes both intermediate and final events sequentially. This ensures:
1. All intermediate events are published in sequence order.
2. The last intermediate is always published before `message_bot_llm`.
3. No race between the read loop and flush goroutine on the final publish.

The read loop's `BotLLMStopped` handler sends a stop signal and blocks on `<-llmDoneChan` until the flush goroutine completes all publishing.

### Session Teardown Mid-Generation

If the call ends while the LLM is generating (context cancelled):
- The flush goroutine exits via `case <-ctx.Done()`.
- Remaining tokens are drained from `llmTokenChan`.
- If any text was accumulated, a final `message_bot_llm` event is published using `context.Background()` (since the original context is cancelled) to preserve the partial LLM response in conversation history for the summary handler.
- The ticker is stopped (deferred `ticker.Stop()`).
- `llmDoneChan` is closed, unblocking the read loop for cleanup.

### Multiple Generations Per Session

A session supports multiple user→bot exchanges. Each generation cycle:
1. First `RTVIBotLLMText` spawns a new flush goroutine with fresh UUID, sequence, and buffers.
2. `BotLLMStopped` signals completion, read loop waits, then resets.
3. Next `RTVIBotLLMText` starts a new cycle.

## Changes By Service

### bin-pipecat-manager

**models/message/event.go** — Add event type:
```go
EventTypeBotLLMIntermediate string = "message_bot_llm_intermediate"
```

**models/message/main.go** — Add `Sequence` field to `Message`:
```go
type Message struct {
    identity.Identity
    PipecatcallID            uuid.UUID
    PipecatcallReferenceType pipecatcall.ReferenceType
    PipecatcallReferenceID   uuid.UUID
    ActiveflowID             uuid.UUID
    Text                     string
    Sequence                 int  `json:"sequence,omitempty"`  // NEW: for intermediate events
}
```

**models/pipecatcall/session.go** — Add flush goroutine coordination fields:
```go
type Session struct {
    // ... existing fields ...

    // LLM intermediate event flush coordination
    LLMTokenChan   chan string    // buffered channel for LLM tokens
    LLMStopChan    chan struct{}  // signals flush goroutine to stop
    LLMDoneChan    chan struct{}  // closed when flush goroutine completes
    LLMFlushing    bool          // whether flush goroutine is running
    LLMMessageID   uuid.UUID     // pre-generated message UUID for current generation
}
```

**pkg/pipecatcallhandler/runner.go** — Modify `receiveMessageFrameTypeMessage`:
- `RTVIBotLLMText` case: spawn flush goroutine on first token, send subsequent tokens to channel.
- `RTVIBotLLMStopped` case: signal stop, wait for completion, reset state.
- New method: `runLLMIntermediateFlush(se, messageUUID)` — the flush goroutine.

### bin-ai-manager

**models/message/event.go** — Add event type:
```go
EventTypeMessageIntermediate string = "aimessage_intermediate"
```

**models/message/webhook.go** — Add intermediate webhook struct:
```go
type IntermediateWebhookMessage struct {
    identity.Identity
    AIcallID     uuid.UUID `json:"aicall_id,omitempty"`
    ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
    Role         Role      `json:"role"`
    Content      string    `json:"content"`
    Direction    Direction `json:"direction"`
    Sequence     int       `json:"sequence"`
}

func (h *IntermediateWebhookMessage) CreateWebhookEvent() ([]byte, error) {
    return json.Marshal(h)
}
```

**pkg/messagehandler/main.go** — Add interface method:
```go
EventPMMessageBotLLMIntermediate(ctx context.Context, evt *pmmessage.Message)
```

**pkg/messagehandler/event.go** — Add handler (webhook only, no DB):
```go
func (h *messageHandler) EventPMMessageBotLLMIntermediate(ctx context.Context, evt *pmmessage.Message) {
    // Build IntermediateWebhookMessage from evt
    // Publish via h.notifyHandler.PublishWebhookEvent (NO DB write)
}
```

**pkg/messagehandler/db.go** — Modify `Create()` to accept optional pre-generated ID:
```go
func (h *messageHandler) Create(
    ctx context.Context,
    id uuid.UUID,          // NEW: pre-generated ID, uuid.Nil to auto-generate
    customerID uuid.UUID,
    // ... rest unchanged
) (*message.Message, error) {
    if id == uuid.Nil {
        id = h.utilHandler.UUIDCreate()
    }
    // ... rest unchanged
}
```

All existing `Create()` callers updated to pass `uuid.Nil` as the first argument.

**pkg/subscribehandler/main.go** — Add routing case:
```go
case m.Publisher == string(commonoutline.ServiceNamePipecatManager) && m.Type == string(pmmessage.EventTypeBotLLMIntermediate):
    err = h.processEventPMMessageBotLLMIntermediate(ctx, m)
```

**pkg/subscribehandler/pipecat_message.go** — Add handler:
```go
func (h *subscribeHandler) processEventPMMessageBotLLMIntermediate(ctx context.Context, m *sock.Event) error {
    // Unmarshal, forward to messageHandler.EventPMMessageBotLLMIntermediate
}
```

**pkg/messagehandler/event.go** — Modify `EventPMMessageBotLLM` to pass pre-generated ID:
```go
func (h *messageHandler) EventPMMessageBotLLM(ctx context.Context, evt *pmmessage.Message) {
    // Pass evt.ID to Create() instead of letting Create() generate a new one
    tmp, err := h.Create(ctx, evt.ID, evt.CustomerID, ...)
}
```

## Event Payloads

### aimessage_intermediate (webhook)
```json
{
  "id": "abc-123-...",
  "customer_id": "cust-456-...",
  "aicall_id": "call-789-...",
  "activeflow_id": "flow-012-...",
  "role": "assistant",
  "direction": "incoming",
  "content": " help you with",
  "sequence": 2
}
```

### aimessage_created (webhook, unchanged format)
```json
{
  "id": "abc-123-...",
  "customer_id": "cust-456-...",
  "aicall_id": "call-789-...",
  "activeflow_id": "flow-012-...",
  "role": "assistant",
  "direction": "incoming",
  "content": "Sure, I can help you with that. Let me check your account.",
  "tool_calls": [],
  "tm_create": "2026-04-02T10:30:00Z"
}
```

Same `id` in both events allows client correlation.

## Client Integration

Clients concatenate deltas to build the in-progress message, then replace with the final `aimessage_created` content:

```
on aimessage_intermediate:
  displayText[event.id] += event.content
  render displayText[event.id]

on aimessage_created:
  displayText[event.id] = event.content  // authoritative final
  render displayText[event.id]
```

If a client misses intermediate events, it simply waits for `aimessage_created` which contains the complete message.

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| Empty LLM response (no tokens) | No flush goroutine spawned, no intermediates. `BotLLMStopped` publishes empty final (skipped by AI Manager as today). |
| Very short response (< 200ms) | Flush goroutine spawned but ticker doesn't fire. `BotLLMStopped` flushes all tokens as one intermediate, then publishes final. |
| Call ends mid-generation | Context cancelled → flush goroutine drains remaining tokens and publishes final `message_bot_llm` with partial text (via `context.Background()`), preserving conversation history. |
| RabbitMQ slow (backpressure) | Token channel buffered at 64. At ~40 tokens/sec, gives ~1.6s buffer before blocking the WebSocket read loop. |
| Multiple generations per session | Each generation gets its own flush goroutine, UUID, and sequence. Clean reset between generations. |

## Files to Change

| File | Change |
|------|--------|
| `bin-pipecat-manager/models/message/event.go` | Add `EventTypeBotLLMIntermediate` |
| `bin-pipecat-manager/models/message/main.go` | Add `Sequence` field |
| `bin-pipecat-manager/models/pipecatcall/session.go` | Add flush coordination fields |
| `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` | Modify token handling, add flush goroutine |
| `bin-ai-manager/models/message/event.go` | Add `EventTypeMessageIntermediate` |
| `bin-ai-manager/models/message/webhook.go` | Add `IntermediateWebhookMessage` struct |
| `bin-ai-manager/pkg/messagehandler/main.go` | Add `EventPMMessageBotLLMIntermediate` to interface |
| `bin-ai-manager/pkg/messagehandler/db.go` | Modify `Create()` to accept optional ID |
| `bin-ai-manager/pkg/messagehandler/event.go` | Add intermediate handler, modify `EventPMMessageBotLLM` |
| `bin-ai-manager/pkg/subscribehandler/main.go` | Add routing case |
| `bin-ai-manager/pkg/subscribehandler/pipecat_message.go` | Add `processEventPMMessageBotLLMIntermediate` |

## Trade-offs

- **Delta vs full-text**: Delta reduces bandwidth but requires clients to concatenate. Missed events cause gaps until `aimessage_created` arrives with complete text. Acceptable since the final event is authoritative.
- **200ms batching**: Fixed interval, not configurable. Can be made configurable later if needed.
- **Channel buffer size 64**: Handles ~1.6s of backpressure at 40 tokens/sec. If exceeded, tokens are dropped with a warning log (non-blocking send). Dropped tokens will be missing from both intermediate and final events. In practice, the flush goroutine drains the channel every 200ms and `PublishEvent` uses its own `context.Background()` with timeout, making drops extremely unlikely.
- **No DB for intermediates**: Intermediates are lost if webhook delivery fails. Acceptable since `aimessage_created` has the complete message.
