# Design: Sync aimessage_intermediate with TTS

**Date:** 2026-04-02
**Branch:** NOJIRA-Sync-aimessage-intermediate-with-tts

## Problem Statement

The current `aimessage_intermediate` events are batched on a fixed 200ms timer, which has no relation to when TTS actually speaks the text. This causes a disconnect between what's displayed on screen and what the user hears. Additionally, `aimessage_created` fires on `bot-llm-stopped` (LLM done generating), which happens before TTS finishes speaking — the final event doesn't reflect when speech delivery is complete.

## Goal

Synchronize intermediate and final events with TTS output:
1. **Intermediate events** fire when TTS receives text chunks (`bot-tts-text`), matching natural sentence/phrase boundaries.
2. **Final event** fires when TTS finishes speaking (`bot-tts-stopped`), containing only the text TTS actually received (not the full LLM output).
3. **Text-based calls** (no TTS) continue working with the existing 200ms timer and LLM-stopped trigger.

## Requirements

- For voice calls: intermediate events driven by `bot-tts-text` RTVI events, final event driven by `bot-tts-stopped`.
- For text-based calls: intermediate events driven by 200ms timer, final event driven by `bot-llm-stopped`.
- Mode auto-detected by the flush goroutine based on whether `bot-tts-text` events arrive.
- `aimessage_created` content: TTS-received text (voice) or full LLM text (text calls).
- On barge-in (user interrupts TTS), `aimessage_created` contains only what TTS received before interruption.
- No Python-side changes needed (`bot-tts-text` enabled by default in Pipecat >= 0.0.101).

## Current Flow

```
Python LLM (streaming tokens)
  | RTVIBotLLMText (per-token)
Go Pipecat Manager (runner.go)
  | Sends token to llmTokenChan
  | Flush goroutine batches tokens, flushes every 200ms
  | Publishes "message_bot_llm_intermediate" events via RabbitMQ
  | On BotLLMStopped: flush remaining, publish "message_bot_llm" with full LLM text
AI Manager (subscribehandler)
  | Intermediate: forward as "aimessage_intermediate" webhook (NO DB)
  | Final: persist to DB, publish "aimessage_created" webhook
```

## Proposed Flow

```
Python LLM + TTS pipeline
  | RTVIBotLLMText (per-token) - always
  | RTVIBotTTSText (per-sentence) - voice calls only
  | RTVIBotLLMStopped - always
  | RTVIBotTTSStopped - voice calls only

Go Pipecat Manager (runner.go)
  | BotLLMText: accumulate tokens (for text-mode fallback)
  | BotTTSText: publish intermediate, accumulate TTS text (voice mode)
  |   — OR (text calls): timer publishes intermediates from LLM tokens
  | BotLLMStopped: if text mode -> publish final with LLM text
  | BotTTSStopped: publish final with TTS-received text only
```

## Architecture

### Dual-Mode Flush Goroutine

The existing flush goroutine is extended to support two modes, auto-detected at runtime:

**Text mode** (no TTS in pipeline):
- 200ms timer batches LLM tokens into intermediate events (current behavior).
- `bot-llm-stopped` triggers drain + final event with full LLM text.

**TTS mode** (voice calls):
- `bot-tts-text` events trigger intermediate events with natural sentence chunks.
- Timer is effectively disabled (no LLM token delta accumulated in TTS mode).
- `bot-tts-stopped` triggers final event with accumulated TTS text only.
- `bot-llm-stopped` is noted but does NOT trigger the final event.

Mode switches from text to TTS on the first `bot-tts-text` event received.

### Session State

```go
type Session struct {
    // ... existing fields ...

    // LLM intermediate event flush coordination (existing, modified).
    LLMTokenChan chan string   // buffered channel for LLM tokens (cap 64)
    LLMStopChan  chan struct{} // closed when bot-llm-stopped received
    LLMDoneChan  chan struct{} // closed when flush goroutine completes
    LLMFlushing  bool         // whether flush goroutine is running
    LLMMessageID uuid.UUID    // pre-generated message UUID for current generation

    // TTS sync channels (new).
    TTSTextChan  chan string   // TTS text chunks from read loop (cap 16)
    TTSStopChan  chan struct{} // closed when bot-tts-stopped received
}
```

### Flush Goroutine Pseudocode

```
func runLLMIntermediateFlush(se, messageID):
    ticker = 200ms
    defer close(LLMDoneChan)

    var llmFullText string    // from LLM tokens (text mode)
    var ttsFullText string    // from TTS chunks (TTS mode)
    var deltaBuffer string    // for timer intermediates (text mode only)
    var sequence int
    var ttsReceived bool

    for:
        select:
        case token := <-LLMTokenChan:
            if !ttsReceived:
                llmFullText += token
                deltaBuffer += token

        case <-ticker.C:
            if !ttsReceived && deltaBuffer != "":
                sequence++
                publishIntermediate(deltaBuffer, sequence)
                deltaBuffer = ""

        case ttsText := <-TTSTextChan:
            ttsReceived = true
            ttsFullText += ttsText
            sequence++
            publishIntermediate(ttsText, sequence)

        case <-LLMStopChan:
            if !ttsReceived:
                // Text mode: drain remaining tokens + publish final
                drain LLMTokenChan -> llmFullText, deltaBuffer
                if deltaBuffer != "":
                    sequence++; publishIntermediate(deltaBuffer)
                publishFinal(llmFullText)
                return
            // TTS mode: note LLM done, keep running for TTS events

        case <-TTSStopChan:
            // TTS done: publish final with TTS-received text
            publishFinal(ttsFullText)
            return

        case <-ctx.Done():
            // Session teardown: drain both channels, publish partial
            drain LLMTokenChan -> llmFullText
            drain TTSTextChan -> ttsFullText
            finalText = ttsFullText if ttsReceived else llmFullText
            if finalText != "":
                publishFinal(context.Background(), finalText)
            return
```

### Read Loop Changes

```go
case RTVIFrameTypeBotLLMText:
    // Same as today: spawn goroutine on first token, send subsequent tokens
    // Added: create TTSTextChan and TTSStopChan alongside existing channels
    if !se.LLMFlushing {
        // Lazy cleanup from previous generation
        // (handles text-mode calls where bot-llm-stopped didn't wait)
        // ... check LLMDoneChan if needed ...

        se.LLMMessageID = h.utilHandler.UUIDCreate()
        se.LLMTokenChan = make(chan string, 64)
        se.LLMStopChan = make(chan struct{})
        se.LLMDoneChan = make(chan struct{})
        se.TTSTextChan = make(chan string, 16)  // NEW
        se.TTSStopChan = make(chan struct{})     // NEW
        se.LLMFlushing = true
        go h.runLLMIntermediateFlush(se, se.LLMMessageID)
    }
    // Non-blocking send (same as today)
    select {
    case se.LLMTokenChan <- token:
    default:
        log.Warn("LLM token channel full")
    }

case RTVIFrameTypeBotTTSText:  // NEW
    if se.LLMFlushing {
        select {
        case se.TTSTextChan <- ttsText:
        case <-se.LLMDoneChan:
            // Goroutine already exited (text mode finished first)
        }
    }

case RTVIFrameTypeBotLLMStopped:
    if se.LLMFlushing {
        close(se.LLMStopChan)
        // DON'T wait — goroutine may be in TTS mode waiting for TTSStopChan
    }

case RTVIFrameTypeBotTTSStopped:  // NEW
    if se.LLMFlushing {
        close(se.TTSStopChan)
        <-se.LLMDoneChan    // safe: goroutine exits on TTSStopChan
        se.LLMFlushing = false
    }
```

### Event Ordering

**Voice calls (TTS mode):**
1. `bot-llm-text` tokens stream → goroutine accumulates (doesn't publish in TTS mode)
2. `bot-tts-text` chunks → goroutine publishes intermediates (sequence 1, 2, 3...)
3. `bot-llm-stopped` → goroutine notes LLM done, keeps running
4. More `bot-tts-text` → more intermediates
5. `bot-tts-stopped` → goroutine publishes final, exits

**Text calls (no TTS):**
1. `bot-llm-text` tokens → goroutine accumulates + timer publishes intermediates
2. `bot-llm-stopped` → goroutine drains + publishes final, exits
3. No TTS events arrive

### Barge-in Handling

On user interruption:
1. Pipecat sends InterruptionFrame → Go sends FLUSH_MEDIA to Asterisk
2. TTS stops → `bot-tts-stopped` arrives
3. Goroutine publishes final with `ttsFullText` (only what TTS received)
4. Read loop resets state
5. New user speech → new generation starts cleanly

The `aimessage_created` event contains only the text TTS received before interruption, not the full LLM output. Unspoken LLM text is discarded.

### Multiple Generations Per Session

Each generation cycle:
1. First `bot-llm-text` spawns new goroutine with fresh channels, UUID, state.
2. Generation completes via `bot-tts-stopped` (voice) or `bot-llm-stopped` (text).
3. State resets. Next `bot-llm-text` starts a new cycle.

For text-mode calls, a lazy cleanup check at the start of `bot-llm-text` ensures the previous goroutine has exited (via non-blocking `LLMDoneChan` check).

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| Very short response (< 1 TTS chunk) | LLM might finish before TTS gets text. Goroutine in text mode → publishes via `bot-llm-stopped`. Late `bot-tts-text` safely ignored. |
| Empty LLM response | No goroutine spawned. No events published. |
| Call ends mid-generation | `ctx.Done()` ��� goroutine drains both channels, publishes partial final with `ttsFullText` or `llmFullText`. |
| Barge-in during TTS | `bot-tts-stopped` fires → final event has only TTS-received text. |
| `bot-tts-text` after goroutine exits | Non-blocking send detects `LLMDoneChan` closed, safely ignores. |
| `bot-tts-stopped` before `bot-llm-stopped` | Goroutine exits on `TTSStopChan`. Late `bot-llm-stopped` is a no-op (`LLMFlushing` already false). |
| `bot-llm-stopped` before first `bot-tts-text` | Goroutine in text mode → publishes final via LLM path. |
| Text-based call (no TTS) | Timer-driven intermediates, final on `bot-llm-stopped`. Same as current behavior. |
| Multiple rapid generations | Lazy cleanup on next `bot-llm-text` ensures previous goroutine finished. |

## Test Cases

### Text mode (existing behavior, updated)
1. **Tokens batched and final on LLM stop** — Send tokens, wait for timer tick, signal LLM stop. Verify intermediates and final with correct `llmFullText`.
2. **Stop without tick flushes all** — Send tokens and immediately stop. Verify drain + intermediate + final.
3. **Context cancellation publishes partial** — Send tokens, cancel context. Verify partial final event.
4. **No text = no events** — Cancel immediately with no tokens. Verify no events.
5. **Multiple generations reset state** — Two sequential generation cycles. Verify independent UUIDs and text.

### TTS mode (new)
6. **TTS text chunks produce intermediates** — Send TTS text chunks, verify intermediates with correct sequence and content.
7. **TTS stop publishes final with TTS text only** — Send LLM tokens + TTS chunks + TTS stop. Verify final contains `ttsFullText`, not `llmFullText`.
8. **Barge-in: partial TTS text in final** — Send LLM tokens + partial TTS chunks + TTS stop (before all LLM text spoken). Verify final contains only received TTS text.
9. **LLM stop before TTS stop (normal)** — Send LLM stop, then more TTS chunks, then TTS stop. Verify goroutine keeps running and publishes correctly.
10. **Context cancellation in TTS mode** — Send TTS chunks, cancel context. Verify final with partial `ttsFullText`.

### Mode switching
11. **Auto-detect TTS mode on first TTS text** — Send LLM tokens (timer fires), then TTS text. Verify timer intermediates stop and TTS intermediates begin.
12. **LLM stop before first TTS text** — LLM finishes before any TTS chunk. Verify text-mode behavior (final from LLM tokens).

### Read loop integration
13. **Non-blocking TTS text send after goroutine exits** — Goroutine exits via LLM stop (text mode), then TTS text arrives. Verify no blocking.
14. **TTS stop after goroutine already exited** — Goroutine exits via text mode, then TTS stop arrives. Verify no panic.
15. **Metadata correctness** — Verify intermediate events carry correct ID, customer_id, pipecatcall_id, reference fields.

## Files to Change

| File | Change |
|------|--------|
| `bin-pipecat-manager/models/pipecatcall/session.go` | Add `TTSTextChan chan string`, `TTSStopChan chan struct{}` |
| `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` | Evolve flush goroutine to dual-mode, add `bot-tts-text` + `bot-tts-stopped` cases, make `bot-llm-stopped` non-blocking |
| `bin-pipecat-manager/pkg/pipecatcallhandler/runner_flush_test.go` | Rewrite/extend tests for dual-mode goroutine (15 test cases above) |

**No changes needed:**
- Python side (`bot-tts-text` enabled by default in Pipecat >= 0.0.101)
- AI Manager (same event types and webhook structs)
- OpenAPI specs (no API-facing changes)

## Trade-offs

- **TTS-received text vs full LLM text in final event**: Voice calls store only what TTS received. On barge-in, unspoken LLM text is lost from conversation history. This is intentional — the stored message reflects what the user experienced.
- **Dual-mode complexity**: The flush goroutine handles both text and TTS modes. This is more complex than a single mode, but avoids breaking text-based AI calls.
- **No explicit TTS mode detection**: Mode is auto-detected by the arrival of `bot-tts-text` events. For very short responses where LLM finishes before TTS receives text, the goroutine may use text mode even for voice calls. This is acceptable — the text is the same either way for such short responses.
- **Event ordering**: Intermediate and final events are published via `go notifyHandler.PublishEvent(...)` (fire-and-forget goroutines). Theoretical out-of-order risk, but practically negligible due to natural timing gaps between events.
