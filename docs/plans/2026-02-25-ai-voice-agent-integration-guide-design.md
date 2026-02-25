# Design: AI Voice Agent Integration Guide (RST Documentation)

## Date: 2026-02-25

## Problem Statement

VoIPBin documentation currently lacks a guide for customers who want to build custom AI voice agents using VoIPBin's individual `/speakings` (TTS) and `/transcribes` (STT) APIs. The existing `ai_overview.rst` documents only the managed `ai_talk` pipeline. Customers who want to use their own AI backend need a separate integration guide.

## Approach

Add a new documentation section "AI Voice Agent Integration" under "AI & Automation" in `index.rst`, following the standard 2-file RST pattern (overview + tutorial) plus an index file.

## Files to Create

### 1. `ai_voice_agent_integration.rst` (Index)
- Toctree that includes overview and tutorial files.

### 2. `ai_voice_agent_integration_overview.rst` (Overview)

Structure:
1. **AI Context block** — Complexity: Medium, Cost: Chargeable (STT + TTS per-minute), Async: Yes
2. **What is Custom AI Voice Agent Integration** — Use VoIPBin's STT/TTS APIs individually with your own AI backend. Contrast with managed `ai_talk`.
3. **Architecture Flow** — ASCII diagram: Caller ↔ VoIPBin (SIP/RTP) ↔ Customer AI Backend. Three-step loop: (1) transcript_created webhook, (2) AI processes, (3) POST /speakings/{id}/say
4. **API Components** — `/transcribes` for STT (ref: transcribe_overview), `/speakings` for TTS (new content since no speaking docs exist yet)
5. **Integration Workflow** — 6-step cycle: create call → start transcribe → receive transcripts → AI process → create speaking + say → repeat
6. **Voice Detection (Barge-in)** — How to handle caller interruptions: flush/stop current speaking, new transcript arrives
7. **When to Use This vs ai_talk** — Comparison table
8. **Best Practices** — Latency, error handling, session lifecycle
9. **Troubleshooting** — Common issues with cause/fix pairs

### 3. `ai_voice_agent_integration_tutorial.rst` (Tutorial)

Structure:
1. **Prerequisites** — API token, phone number, webhook URL
2. **Step 1: Create an outbound call** — `POST /calls` with curl example
3. **Step 2: Start transcription on the call** — `POST /transcribes` with reference_type: "call"
4. **Step 3: Receive transcript events** — Webhook payload example (transcript_created)
5. **Step 4: Process with your AI backend** — Pseudo-code for LLM call
6. **Step 5: Create a speaking session** — `POST /speakings` with reference_type: "call"
7. **Step 6: Send AI response as speech** — `POST /speakings/{id}/say`
8. **Step 7: Handle ongoing conversation** — Loop, flush, stop patterns
9. **Complete example** — End-to-end flow combining all steps

### 4. `index.rst` (Modify)
- Add `ai_voice_agent_integration` under "AI & Automation" section after `ai`.

## Key Technical Details (Verified from Codebase)

### Speaking API
- `POST /v1/speakings` — Create session (reference_type: "call", reference_id: call UUID)
- `POST /v1/speakings/{id}/say` — Send text to speak (body: {"text": "..."})
- `POST /v1/speakings/{id}/stop` — Stop session
- `POST /v1/speakings/{id}/flush` — Flush queued text
- Status: initiating → active → stopped
- Providers: elevenlabs, deepgram, openai, aws, google, etc.
- Events: speaking_started, speaking_stopped

### Transcribe API
- `POST /v1/transcribes` — Create session (reference_type: "call", reference_id: call UUID)
- `POST /v1/transcribes/{id}/stop` — Stop session
- Status: progressing → done
- Providers: gcp, aws
- Events: transcribe_created, transcribe_progressing, transcribe_done, transcript_created

### Webhook Payload Format (transcript_created)
```json
{
    "type": "transcript_created",
    "data": {
        "id": "UUID",
        "transcribe_id": "UUID",
        "direction": "in",
        "message": "transcribed text",
        "tm_transcript": "timestamp",
        "tm_create": "timestamp"
    }
}
```

## Trade-offs

- **Decided:** Overview + Tutorial 2-file split follows existing documentation patterns
- **Not included:** Speaking struct reference doc (can be added later as separate task)
- **Cross-references:** Link to existing transcribe_overview.rst and call docs rather than duplicating content

## Verification

- Build docs with Sphinx after writing RST files
- Verify cross-references resolve correctly
- Check ASCII diagrams render properly
