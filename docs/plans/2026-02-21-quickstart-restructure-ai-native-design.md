# Quickstart Restructure: AI-Native Design

**Date:** 2026-02-21
**Status:** Proposed

## Problem Statement

The current quickstart documentation covers signup, authentication, events, calls, queue, and transcription as separate sections. While the content is mostly AI-native (explicit types, data provenance, troubleshooting), it has gaps:

1. **Extensions are missing** from the quickstart entirely — no walkthrough for creating an extension and registering a SIP phone
2. **Speakings (real-time TTS API)** are not documented in the quickstart
3. **The transcribe quickstart** uses virtual numbers and flows, which is functional but doesn't demonstrate a truly interactive scenario where a human speaks into a SIP phone and receives real-time TTS responses
4. **Queue section** is included but not essential for getting started — it's better suited as advanced/tutorial content

The goal is to restructure the quickstart into progressive scenarios that guide both humans and AI agents through increasingly interactive use cases, culminating in a real-time voice interaction demo.

## Approach

Restructure in-place (Approach A). Reuse existing content where it's already AI-native, add new content for extensions and speakings, remove queue.

## New Structure

```
quickstart.rst                     (parent — restructured intro + TOC)
├── quickstart_sandbox.rst         (unchanged)
├── quickstart_signup.rst          (unchanged)
├── quickstart_authentication.rst  (unchanged)
├── quickstart_call.rst            (Scenario 1 — minor cleanup)
├── quickstart_events.rst          (Scenario 2 — unchanged)
└── quickstart_realtime.rst        (NEW — Scenario 3)
```

### Removed
- `quickstart_queue.rst` — removed from quickstart includes (file kept for queue tutorial reference)
- `quickstart_transcribe.rst` — replaced by `quickstart_realtime.rst` (file kept for transcribe tutorial reference)

### File Changes

#### `quickstart.rst` (Modified)
- Update intro text to describe 3 progressive scenarios
- Remove `.. include:: quickstart_queue.rst`
- Replace `.. include:: quickstart_transcribe.rst` with `.. include:: quickstart_realtime.rst`
- Update "What's Next" section

#### `quickstart_call.rst` (Minor Edits)
- Keep as Scenario 1: "Your First Call"
- No structural changes — content is already AI-native

#### `quickstart_realtime.rst` (New File)
Title: "Real-Time Voice Interaction"

Steps:

**Step 1: Create an extension**
- `POST /extensions` with name, username, password
- Show full request and response
- AI hint: extension name is used in SIP registration address

**Step 2: Register a SIP phone (Linphone)**
- Brief Linphone setup instructions
- Registration domain: `{customer-id}.registrar.voipbin.net`
- Username/password from the extension created in Step 1
- Verification: show how to confirm registration succeeded

**Step 3: Subscribe to events via WebSocket**
- Connect to `wss://api.voipbin.net/v1.0/ws?token=<token>`
- Subscribe to `call:*` and `transcribe:*` topics
- Python example (reuse from existing quickstart_events.rst pattern)

**Step 4: Make a call to the extension**
- `POST /calls` with:
  - `source.type: "tel"`, `source.target: "<your-number>"`
  - `destinations[0].type: "extension"`, `destinations[0].target_name: "<extension-name>"`
  - Inline actions: `transcribe_start` + `talk` (welcome message) + `sleep` (long duration)
- Show full request and response
- AI hint: the call rings the SIP phone; answer it to start the interactive session

**Step 5: Answer the call and observe transcription**
- When the SIP phone rings, answer it
- The TTS greeting plays (direction: "out")
- Speak into the SIP phone — transcription events arrive (direction: "in")
- Show example WebSocket events

**Step 6: Create a speaking stream**
- `POST /speakings` with:
  - `reference_type: "call"`
  - `reference_id: "<call-id>"` (from Step 4 response)
  - `language: "en-US"`
  - `provider: "elevenlabs"`
  - `direction: "out"`
- Show full request and response
- AI hint: the speaking stream is a persistent TTS channel attached to the call

**Step 7: Speak via TTS API**
- `POST /speakings/{id}/say` with text
- The text is spoken into the call in real-time via ElevenLabs
- Show request and response
- AI hint: you can call `/say` multiple times to queue more speech

**Troubleshooting**
- Extension creation failures
- SIP registration failures (wrong domain, wrong credentials)
- Call not ringing (extension not registered)
- No transcription events (wrong WebSocket topic)
- Speaking creation failures (call not in progressing state)
- Speaking say failures (speaking not active)

## SIP Client Recommendation

Recommend Linphone (open-source, cross-platform) with brief setup instructions:
1. Download from linphone.org
2. Create SIP account with:
   - Username: `<extension-username>`
   - Password: `<extension-password>`
   - Domain: `<customer-id>.registrar.voipbin.net`
3. Verify registration shows "Registered" status

## TTS Provider

Use `elevenlabs` explicitly in examples so they work out-of-the-box.

## RST Compliance

All new content follows the AI-Native RST Writing Guidelines:
- Rule 1: Data provenance for all IDs
- Rule 2: Strict typing (UUID, String, E.164, enum)
- Rule 3: AI Implementation Hints (at least 2-3 per page)
- Rule 4: Explicit state transitions for enums
- Rule 5: Self-correcting error handling with cause-fix pairs

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `quickstart.rst` | Modify | Update intro and includes |
| `quickstart_call.rst` | Minor edit | Ensure Scenario 1 framing |
| `quickstart_realtime.rst` | Create | New Scenario 3 content |

## Build Requirement

After all RST changes, rebuild HTML:
```bash
cd bin-api-manager/docsdev
python3 -m sphinx -M html source build
```

Commit both RST sources and built HTML together.
