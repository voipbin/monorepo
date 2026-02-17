# Quickstart Transcribe Guide Design

**Date:** 2026-02-18
**Status:** Approved

## Problem Statement

The existing quickstart documentation covers signup, authentication, making calls, and queues, but lacks a guided walkthrough for real-time transcription — one of VoIPBIN's key features. New users have no quick path to see speech-to-text working end-to-end.

## Approach

Add a new `quickstart_transcribe.rst` file included in the existing `quickstart.rst` page (same pattern as `quickstart_call.rst` and `quickstart_queue.rst`). The guide walks the user through a complete scenario:

1. Create a flow with `answer` + `transcribe_start` + `talk` + `sleep`
2. Create a virtual number and assign the flow to it
3. Subscribe to transcribe events via WebSocket
4. Make a call to the virtual number
5. Receive real-time transcription events (WebSocket)
6. Alternative: receive events via webhook
7. Verify transcription results via `GET /transcripts`

## Design Decisions

- **Virtual numbers only** — free, no provider cost, safe for quickstart
- **Flow-first approach** — teaches creating a flow and assigning it to a number, rather than inline actions on the call
- **WebSocket as primary event delivery** — shows real-time experience; webhook as alternative
- **`sleep` action** — gives 30 seconds for the TTS output to be transcribed and for the user to observe events
- **Follows AI-native RST writing guidelines** — provenance, strict typing, AI hints, troubleshooting

## File Changes

1. **Create:** `bin-api-manager/docsdev/source/quickstart_transcribe.rst`
2. **Edit:** `bin-api-manager/docsdev/source/quickstart.rst` — add `.. include:: quickstart_transcribe.rst` before the "What's Next" section
3. **Rebuild:** HTML docs via `sphinx-build`

## RST Structure

```
.. _quickstart_transcribe:

Transcribe
==========
  Intro paragraph
  Prerequisites (references auth section)

  Step 1: Create a flow
    - POST /flows with answer + transcribe_start + talk + sleep
    - Full request + response example

  Step 2: Create a virtual number and assign the flow
    - POST /numbers (virtual)
    - PUT /numbers/{id} with call_flow_id set to the flow

  Step 3: Subscribe to transcribe events via WebSocket
    - Connect to wss://api.voipbin.net/v1.0/ws?token=...
    - Send subscribe message for transcribe:* topic

  Step 4: Make a call to the virtual number
    - POST /calls with source=owned number, destination=virtual number

  Step 5: Receive real-time transcription events
    - Show example WebSocket event payload (transcript_created)

  Receive events via Webhook (alternative)
    - POST /webhooks to create webhook for transcript events
    - Show webhook payload example

  Step 6: Verify transcription results
    - GET /transcripts?transcribe_id=...
    - Show response with in/out direction

  Troubleshooting
    - 400, 404, no transcripts generated, etc.
```
