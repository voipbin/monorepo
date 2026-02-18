# Quickstart AI-Native Improvements Design

**Date:** 2026-02-19
**Status:** Approved
**Scope:** bin-api-manager/docsdev/source/quickstart*.rst + quickstart_events.rst (new)

## Problem Statement

The quickstart documentation was assessed against the AI-Native RST Writing Guidelines (5 commandments from `bin-api-manager/CLAUDE.md`). The assessment found:

- **Excellent:** `quickstart_authentication.rst`, `quickstart_transcribe.rst` — follow nearly all rules
- **Gaps in 4 files:** missing troubleshooting sections, incomplete enum listings, no AI hints
- **Missing content:** no headless API signup flow, no event delivery introduction (webhook + websocket)

## Goal

Bring all quickstart pages to AI-native standard and add two missing sections: headless signup and event delivery (webhook + websocket).

## Changes

### New quickstart flow

```
Signup → Auth → Events (NEW) → Call → Queue → Transcribe → What's Next
```

### File-by-file changes

#### 1. `quickstart.rst` (edit)

Add `.. include:: quickstart_events.rst` between authentication and call includes.

#### 2. `quickstart_signup.rst` (rewrite)

Current state: 17-line UI-only walkthrough. No AI hint, no troubleshooting, no API flow.

New structure:
- **Intro:** Two ways to sign up — Admin Console (UI) or API (headless)
- **Sign up via Admin Console:** Existing 6-step UI flow (kept as-is)
- **Sign up via API (Headless):** NEW
  - `POST /auth/signup` with email (Required), name, phone_number, etc. (all Optional)
  - Response: `temp_token` (String) for headless verification
  - `POST /auth/complete-signup` with `temp_token` + 6-digit OTP `code` from email
  - Response: `customer_id` (UUID) + `accesskey` (with token for immediate API access)
  - Full curl examples for both requests + responses
- **AI Implementation Hint:** Headless path preferred for automated systems. `accesskey.token` returned from complete-signup can be used immediately — no need to call `POST /auth/login`. Always returns 200 on signup to prevent email enumeration — empty `temp_token` may mean email already registered.
- **Troubleshooting:**
  - 200 with empty `temp_token`: Email may already be registered or invalid
  - 400 on complete-signup: Invalid/expired `temp_token` or wrong OTP code
  - 429 on complete-signup: Too many attempts (max 5 per `temp_token`) — must re-signup

Source: OpenAPI specs at `bin-openapi-manager/openapi/paths/auth/signup.yaml`, `complete-signup.yaml`

#### 3. `quickstart_events.rst` (NEW file)

Introduces both event delivery mechanisms before the practical tutorials.

Structure:
- **Intro:** VoIPBIN notifies you in real-time. Two delivery methods: Webhook (HTTP push) and WebSocket (persistent connection).
- **Comparison table:**

  | | Webhook | WebSocket |
  |---|---|---|
  | Connection | Stateless HTTP pushes | Persistent bidirectional |
  | Setup | Register URL via `POST /webhooks` | Connect to `wss://` endpoint |
  | Best for | Server-side integrations, CI/CD | Real-time dashboards, AI agents |
  | Requires | Public HTTPS endpoint | WebSocket client library |

- **AI Implementation Hint:** For AI agents, WebSocket preferred (no public server needed, lower latency). Use Webhook for persistent servers processing events asynchronously.
- **Webhook section:**
  - Create webhook: `POST /webhooks` with `uri`, `method`, `event_types`
  - Curl example with response
  - Note: event type naming uses underscore in delivery (`call_created`)
  - Cross-ref to full webhook docs
- **WebSocket section:**
  - Connect: `wss://api.voipbin.net/v1.0/ws?token=<token>`
  - Subscribe: `{"type": "subscribe", "topics": ["customer_id:<id>:call:*"]}`
  - Minimal Python example (~20 lines): connect, subscribe, print events
  - AI hint: topic format `<scope>:<scope_id>:<resource>:<resource_id>`, wildcard `*`, reconnection on drop
  - Cross-ref to full websocket docs

No troubleshooting in this page — defer to full webhook/websocket docs.

#### 4. `quickstart_sandbox.rst` (edit — add troubleshooting)

Add troubleshooting section at the end:
- **Docker not running:** Cause: Docker daemon not started or Docker Compose v2 not installed. Fix: Verify with `docker compose version`.
- **Port 8443 in use:** Cause: Another service using the port. Fix: Check with `ss -tlnp | grep 8443`.
- **SSL certificate error:** Cause: Sandbox uses self-signed cert. Fix: Add `-k` flag to curl.
- **401 with default credentials:** Cause: Sandbox not fully initialized. Fix: Run `voipbin> init` again.

#### 5. `quickstart_call.rst` (edit — add status enum + troubleshooting)

After the response example, add brief call status lifecycle:
- `dialing` → `ringing` → `progressing` → `hangup` (normal flow)
- `canceling` → `hangup` (caller cancels)
- `terminating` → `hangup` (system terminates)
- Cross-ref to `:ref:`Call overview <call-overview>``

Add troubleshooting section:
- **400 Bad Request:** Source number not owned by account or not E.164. Fix: Verify via `GET /numbers`.
- **404 Not Found (flow_id):** Flow doesn't exist. Fix: Verify via `GET /flows`.
- **Call status immediately "hangup":** Destination unreachable or source has no provider. Fix: Use virtual numbers (`+899` prefix) for testing.

#### 6. `quickstart_queue.rst` (edit — add troubleshooting)

Add troubleshooting section:
- **400 Bad Request:** `tag_ids` invalid UUID or missing fields. Fix: Verify via `GET /tags`.
- **Queue created but calls not routed:** No agents with matching tags online. Fix: Verify agents via `GET /agents` and check `available` status.
- **Callers timing out immediately:** `timeout_wait` set too low (milliseconds, not seconds). Fix: 100 seconds = `100000`.

## Files NOT changed

- `quickstart_authentication.rst` — Already meets AI-native standard
- `quickstart_transcribe.rst` — Already meets AI-native standard (gold standard example)
- `websocket_overview.rst` — Out of scope (noted: `transcription` resource type may need verification)
- `websocket_tutorial.rst` — Out of scope (noted: timestamps use 2024, should be 2026)
- `websocket_struct.rst` — Out of scope

## Implementation notes

- After editing RST files, rebuild HTML: `cd bin-api-manager/docsdev && python3 -m sphinx -M html source build`
- Force-add built HTML: `git add -f bin-api-manager/docsdev/build/`
- Commit both RST sources and built HTML together
