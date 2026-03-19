# Design: Add Speaking RST Documentation

**Date:** 2026-03-20
**Branch:** NOJIRA-Add-speaking-rst-documentation

## Problem Statement

The Speaking (TTS) API has no dedicated documentation pages in the RST docs (`bin-api-manager/docsdev/source/`). Speaking is only mentioned inside `quickstart_realtime.rst` as part of a combined workflow. Users looking for Speaking-specific reference, lifecycle details, provider options, or advanced usage have no dedicated resource page.

## Approach

Create dedicated Speaking documentation following the established 3-file pattern (overview, struct, tutorial), plus update index and cross-references.

### Source Material

- Tutorial markdown at `/home/pchero/gitvoipbin/tmp/tutorials/tutorials.md` — an "AI Agent Speak & Listen" tutorial covering curl, CLI, and SDK examples
- Speaking WebhookMessage from `bin-tts-manager/models/speaking/webhook.go` — 11 external fields (PodID excluded)
- Speaking Status from `bin-tts-manager/models/speaking/status.go` — initiating, active, stopped
- Streaming types from `bin-tts-manager/models/streaming/streaming.go` — Direction (in/out/both), ReferenceType (call/confbridge), VendorName (elevenlabs/gcp/aws)
- Existing `quickstart_realtime.rst` — already covers the basic create → say → stop workflow
- Existing `transcribe_*.rst` — peer resource for formatting reference

### Decision: Drop Combined Tutorial File

The original tutorial markdown shows Speaking + Transcribe together as one workflow. We will NOT create a separate `speaking_listen_tutorial.rst` because:
1. `quickstart_realtime.rst` already covers the combined Speaking + Transcribe workflow
2. A cross-resource tutorial inside `speaking.rst` would create ownership bias
3. SDK examples (Python ~170 lines, JS ~130 lines, Go ~260 lines) would bloat the Speaking page

Instead, `speaking_overview.rst` will cross-reference `quickstart_realtime.rst` for the combined workflow.

### Decision: Tutorial Differentiates from Quickstart

`speaking_tutorial.rst` will NOT repeat the basic "create → say → stop" hello-world path (that's in quickstart). Instead it focuses on:
- Provider selection (elevenlabs vs gcp vs aws) with examples
- Voice ID configuration
- Direction options (in vs out vs both) with use cases
- Flush queue operation
- Speaking on conferences (confbridge reference type)
- Lifecycle management and status polling patterns

## Files to Create

| File | Purpose | Estimated Lines |
|------|---------|-----------------|
| `speaking.rst` | Wrapper — includes overview, struct, tutorial | ~15 |
| `speaking_overview.rst` | Conceptual: lifecycle, providers, directions, reference types, best practices | ~250 |
| `speaking_struct_speaking.rst` | Struct reference (11 WebhookMessage fields with types, provenance, enums) | ~150 |
| `speaking_tutorial.rst` | Advanced tutorial: providers, voice IDs, directions, conferences, flush, lifecycle | ~300 |

## Files to Modify

| File | Change |
|------|--------|
| `index.rst` | Add `speaking` to "Voice & Real-Time" toctree after `transcribe` |
| `transcribe_overview.rst` | Add cross-reference to Speaking in "Related Documentation" section |

## Conventions to Follow

- **Auth:** `?token=<your-token>` in all curl examples
- **Code blocks:** `.. code::` (not `.. code-block::`)
- **Heading levels:** Wrapper `****`, overview sections `====`, subsections `----`, sub-subsections `++++`
- **AI blocks:** `.. note:: **AI Context**` in overview, `.. note:: **AI Implementation Hint**` for gotchas
- **Data provenance:** Every ID field states its source endpoint
- **Strict typing:** UUID, E.164, enum, ISO 8601 — no vague terms
- **Struct docs match WebhookMessage only** — no PodID
- **Include order:** Overview → Struct → Tutorial (matches recording.rst, talk.rst)
- **Cross-references:** `:ref:` with section anchors

## WebhookMessage Fields (for struct doc)

| JSON Field | Go Type | Description |
|-----------|---------|-------------|
| id | uuid.UUID | Unique speaking session identifier |
| customer_id | uuid.UUID | Owner customer |
| reference_type | streaming.ReferenceType | "call" or "confbridge" |
| reference_id | uuid.UUID | ID of referenced call or conference |
| language | string | TTS language code (e.g., "en-US") |
| provider | string | TTS provider: "elevenlabs", "gcp", "aws" |
| voice_id | string | Provider-specific voice identifier |
| direction | streaming.Direction | "in", "out", or "both" |
| status | Status | "initiating", "active", or "stopped" |
| tm_create | *time.Time | Creation timestamp (ISO 8601) |
| tm_update | *time.Time | Last update timestamp (ISO 8601) |
| tm_delete | *time.Time | Soft-delete timestamp (ISO 8601) |

**NOT documented:** PodID (internal Kubernetes routing field, stripped by ConvertWebhookMessage)

## Risks

- **Redoc URL may not exist:** `https://api.voipbin.net/redoc/#tag/Speaking` — depends on OpenAPI spec having a Speaking tag. Need to verify during implementation.
- **Provider details may be incomplete:** The tutorial lists providers but actual capabilities per provider (voice options, language support) may need research.
