# bin-webchat-manager: Session PageURL/Referrer capture

Status: DRAFT (round 0)
Author: Hermes (CPO)
Date: 2026-07-22

## 1. Problem

pchero asked whether it's possible to tell which site/page a webchat visitor
came in from. Today it is not: `bin-webchat-manager`'s `Session` model
(`models/session/session.go`) has no field for the embedding page's URL, and
the widget embed runtime
(`monorepo-javascript/square-admin/src/webchat-widget-runtime/client.js`)
never reads `document.referrer` or `window.location.href` in the first
place. A customer running one Widget on multiple pages (landing page,
pricing page, blog) currently cannot distinguish which page produced a
given session -- `widget_id` alone doesn't carry that granularity.

(Scope note: a related idea -- persisting Peer/Local `commonaddress.Address`
values on Session, mirroring the just-shipped Case/Interaction pattern --
was evaluated and REJECTED for this design. See §6 "Rejected alternative".)

## 2. Goal / Non-goals

**Goal:** capture the page URL the widget was embedded on at the moment a
Session is created, store it, and surface it to square-admin's Sessions
list/detail so an admin can see where each visitor's conversation started.

**Non-goals:**
- Per-message page tracking (a visitor navigating between pages mid-session
  does not re-fire this capture; it is a one-time, session-creation-time
  fact, mirroring how `widget_id` itself is set once at creation and never
  updated).
- Full referrer chain / UTM campaign parsing. This design captures the raw
  page URL only; marketing-attribution-grade parsing (UTM params, ad click
  IDs) is a separate, larger feature and explicitly out of scope here.
- Server-side URL validation beyond a length cap and scheme allowlist (see
  §5). We do not attempt to verify the URL actually belongs to a domain
  the customer registered anywhere in this design.

## 3. What gets captured, and where

Two independent browser-side signals exist; this design captures **one**
of them, deliberately:

- `window.location.href` (the *embedding* page's own URL) -- reliable,
  always present, this is what we want.
- `document.referrer` (the URL of the page that *linked to* the embedding
  page) -- NOT what we want here. `document.referrer` answers "what page
  did the visitor click from to arrive at this page", not "what page is
  the widget currently sitting on". Since the widget is embedded directly
  in the customer's own page (not loaded in an iframe from elsewhere),
  `location.href` already IS the fact pchero is asking for ("which site
  did they come in from" = "which of my pages was the widget open on").
  `document.referrer` would answer a *different*, marketing-attribution
  question (how did the visitor discover this page at all) that is exactly
  the UTM/campaign-tracking territory called out as non-goal in §2.

Renamed accordingly from the earlier "referrer/page_url" framing: this
design captures **`page_url`** only, sourced from `window.location.href`
at boot time, in `webchat-widget-runtime/client.js`'s `_doStart()`
(currently `bin-webchat-manager/pkg/sessionhandler/create.go` is the
server side that would receive it).

## 4. Design

### 4.1 Client: `webchat-widget-runtime/client.js`

In `_doStart()` (client.js:315-319), add `page_url` to the
`POST /webchat_sessions` request body:

```js
const session = await this._fetchJson(this._v1Url('/webchat_sessions'), {
  method: 'POST',
  body: JSON.stringify({
    widget_id: this.resourceId,
    page_url: (typeof window !== 'undefined' && window.location?.href) || undefined,
  }),
})
```

`window.location.href` is always present in a real browser; the
`typeof window !== 'undefined'` guard exists only for this file's existing
test harness (`__tests__/client.test.js` runs under jsdom, where `window`
exists but a defensive check costs nothing and matches this file's
existing style of guarding browser globals). No new dependency, no new
async call -- `location.href` is synchronously available at `_doStart()`
call time.

### 4.2 API contract: `POST /webchat_sessions` request body

New optional field `page_url` (string, not a UUID) on
`PostWebchatSessionsJSONBody`
(`bin-openapi-manager/openapi/paths/webchat_sessions/main.yaml`):

```yaml
page_url:
  type: string
  maxLength: 2048
  description: "The URL of the page the widget was embedded on when this session was created. Captured client-side from window.location.href at session-creation time; not re-captured on subsequent navigation within the same session."
  example: "https://example.com/pricing"
```

Not `required`: a caller invoking `POST /webchat_sessions` directly (the
"authenticated agent/accesskey for admin-side testing" path already
documented in `WebchatSessionCreate`'s comment) has no browser page to
report, and older embed-runtime bundles cached on third-party sites will
keep POSTing without this field indefinitely -- the field must degrade
gracefully to "unknown" (see §4.4), not become a hard requirement.

### 4.3 Backend: `bin-webchat-manager`

**`models/session/session.go`**: add one field.

```go
// PageURL is the page the widget was embedded on when this Session was
// created, captured client-side from window.location.href at
// POST /webchat_sessions time. Best-effort: absent for pre-upgrade embed
// snippets and for sessions created via the admin/accesskey direct-create
// path (no browser page exists in that path). NEVER re-captured on
// mid-session navigation -- this is a session-creation-time fact, exactly
// like WidgetID.
PageURL string `json:"page_url,omitempty" db:"page_url"`
```

Plain `VARCHAR`, not JSON -- unlike Case/Interaction's `Peer`/`Local`
(structured `commonaddress.Address`), a page URL is a single opaque
string with no sub-fields worth decomposing, and no existing generated
column/index depends on parsing it. `omitempty` because the field is
genuinely absent (not "empty string meaning something"), consistent with
`Session.WidgetID uuid.UUID \`json:"widget_id,omitempty"\`` immediately
above it in the same struct.

**`models/session/field.go`**: add `FieldPageURL Field = "page_url"`
after `FieldStatus` (mirrors `WidgetID`/`Status` grouping).

**`models/session/webhook.go`**: add `PageURL` to `WebhookMessage` and to
`ConvertWebhookMessage()` -- this is an externally-visible, non-sensitive
field (the customer's own page URL, not visitor PII beyond what a normal
web server access log already has), no reason to strip it on external
responses.

**`pkg/sessionhandler/create.go`**: `Create()`'s signature gains a
`pageURL string` parameter (threaded through from the API-manager call
site, see §4.4), stored on the `session.Session{}` literal at
construction (line 40-48) alongside `WidgetID`/`Status`.

**Database**: `scripts/database_scripts_test/sessions.sql` (SQLite test
schema) gets a new `page_url TEXT` column; the real migration is a new
Alembic revision in `bin-dbscheme-manager/bin-manager/main/versions/`
(`alembic revision -m "webchat_sessions_add_column_page_url"`), a plain
`ALTER TABLE webchat_sessions ADD COLUMN page_url VARCHAR(2048) NULL`
(matching the OpenAPI `maxLength: 2048` cap) -- no generated column, no
backfill needed (nullable, no existing rows have this data by
definition), no index (this is a display-only field for the admin
console, not a lookup key today -- see §6 non-goal on richer analytics).

### 4.4 `bin-api-manager`: threading `page_url` through

`server/webchat_sessions.go`'s `PostWebchatSessions` parses
`openapi_server.PostWebchatSessionsJSONBody` (which now carries an
optional `PageUrl *string` per the oapi-codegen-generated type) and
passes it to `serviceHandler.WebchatSessionCreate`.

`pkg/servicehandler/main.go`'s `WebchatSessionCreate` interface signature
and `pkg/servicehandler/webchat_session.go`'s implementation both gain a
`pageURL string` parameter, passed straight through to
`h.reqHandler.WebchatV1SessionCreate(ctx, ownerCustomerID, widgetID, pageURL)`
(the RabbitMQ RPC call into `bin-webchat-manager`) -- no new auth/ownership
logic needed, this is inert metadata riding alongside the existing
`widgetID` argument through the exact same call path already handling
agent/accesskey/direct auth branches (webchat_session.go:163-203).

`bin-common-handler/pkg/requesthandler`'s `WebchatV1SessionCreate` request
struct gains the field; mock regeneration required
(`pkg/requesthandler` mocks) per the standard verification workflow.

### 4.5 square-admin: display

**`sessions_list.js`**: no new column added to the default table view (the
existing four-plus-widget columns are already dense enough per the
existing `sessions_list_global.js` design's own column-crowding
reasoning) -- instead, `page_url` becomes a truncated hover-title cell
only when the transcript dialog (`message_timeline.js`) opens for a
selected session, since that's where an admin actually drills into a
single session's detail. Concretely:

- `message_timeline.js` (already receives `session={selectedSession}`
  from both `detail.js` and `sessions_list_global.js`) renders a small
  header line "Started from: `<page_url or "Unknown">`" above the message
  list, using an ellipsis-truncated `<a href={page_url} target="_blank"
  rel="noopener noreferrer">` link so an admin can click through to see
  the actual page.
- `page_url` absent (older sessions, or the accesskey direct-create path)
  renders as plain "Unknown" text, not a broken/empty link.

This avoids adding a `showPageUrlColumn`-style prop-threading exercise to
`sessions_list.js` (which is already carrying `showWidgetColumn` +
`widgetNameMap` complexity from the prior global-list design) for a field
that's genuinely a detail-view fact, not a scannable list-view column
(URLs are long and low-signal at table-row density).

### 4.6 Privacy / documentation

`page_url` is the customer's OWN page (not a third-party site the
visitor was previously on, since we deliberately chose `location.href`
over `document.referrer` -- see §3) -- this is materially LESS sensitive
than a referrer chain would be; it does not reveal anything about the
visitor's browsing history outside the customer's own site. No new
consent/privacy-notice requirement is triggered beyond what operating a
webchat widget already implies (the customer's own site owner already
knows which of their pages embeds the widget).

`bin-api-manager/docsdev/source/webchat_struct_session.rst` gets a new
`page_url` bullet describing the field (mirrors the existing
`tm_last_activity` etc. bullet style) -- required per this repo's
"RST Docs Sync" CLAUDE.md rule since this is a new externally-visible
response field.

## 5. Edge cases

- **`window.location.href` exceeding 2048 chars** (pathological query
  strings): client-side truncation is NOT performed (adds complexity for
  a vanishingly rare case); instead the OpenAPI `maxLength: 2048` combined
  with the DB column's `VARCHAR(2048)` means an oversized value is
  rejected by request validation at `bin-api-manager`'s Gin binding layer
  with a 400 -- and `WebchatSessionCreate`'s caller (the visitor's own
  browser) has no error-recovery UI for a failed session-create beyond
  what already exists for other 400s, so this is treated as an accepted,
  rare failure mode, not specially handled.
- **`javascript:`/`data:` scheme URLs**: `window.location.href` cannot
  itself be one of these (a page can't be navigated to a `javascript:`
  URL and stay loaded), so no server-side scheme allowlist is added; this
  differs from `ThemeConfig.LogoURL`'s existing `https://`-only validation
  (widget.go), which exists because THAT field is customer-supplied
  config, not a same-origin browser fact.
- **Local file testing / `file://` URLs**: captured verbatim, same as any
  other `location.href` value; no special-casing.
- **Widget embedded inside an iframe on the customer's page** (e.g. a
  page builder's preview iframe): `window.location.href` inside that
  iframe context returns the iframe's own URL, which may differ from the
  top-level page URL a real visitor sees in their browser's address bar.
  Reading `window.top.location.href` to escape the iframe was considered
  and REJECTED: cross-origin iframe access throws a `SecurityError`
  (violates same-origin policy) in the common case where the page builder
  serves the preview from a different origin, which would turn a
  same-origin embed's PageURL capture into an uncaught exception. Staying
  with `window.location.href` (the widget script's own execution context)
  is the safe, universally-working choice; this is a known, accepted
  imprecision for the iframe-preview case specifically, not a general
  correctness bug.

## 6. Rejected alternative: Peer/Local on Session

Initially considered adding `Peer`/`Local` (`commonaddress.Address`)
fields to `Session`, mirroring `kase.Case`'s just-shipped pattern
(`bin-contact-manager/models/kase/kase.go`, PR #1130). Rejected after
tracing the actual values that would be stored:

- `Peer.Target` would be `Session.ID` itself (`sessionhandler/create.go`
  line 78's already-computed `peer` local var) -- a webchat visitor has
  no identity beyond the Session ID, so this field would be a tautology
  (Peer.Target == the record's own primary key), carrying zero new
  information.
- `Local.Target` would be `Widget.ID`'s string form (line 77's `self`
  local var) -- already stored, byte-for-byte, as the existing
  `Session.WidgetID` column. Adding `Local` would be a pure duplicate of
  data already on the row, in a different serialization.

Unlike Case's Peer (a real phone number/email -- external identity) and
Local (the DID the call/message arrived on -- real routing information),
neither Session field would carry information not already present.
Format consistency with Case/Interaction was considered as a reason to
add them anyway, but rejected: introducing a column whose only value is
"looks like the other services' pattern" contradicts the project's
stated preference for storing 1st-party immutable facts, not schema
decoration (see pchero's standing principle on this, captured in CPO
memory). `page_url` (this design) is a genuinely new fact Session does
not otherwise carry, which is why it was pursued instead.

## 7. Files touched (implementation checklist)

- `monorepo-javascript/square-admin/src/webchat-widget-runtime/client.js`
- `monorepo-javascript/square-admin/src/webchat-widget-runtime/__tests__/client.test.js`
- `monorepo-javascript/square-admin/public/webchat-widget-runtime.bundle.js` (regenerated via `npm run build:widget`, not hand-edited)
- `monorepo-javascript/square-admin/public/webchat-widget-runtime.esm.js` (regenerated)
- `monorepo-javascript/square-admin/src/views/webchat_widgets/message_timeline.js`
- `monorepo-javascript/square-admin/src/views/webchat_widgets/__tests__/message_timeline.test.js`
- `monorepo/bin-openapi-manager/openapi/paths/webchat_sessions/main.yaml`
- `monorepo/bin-openapi-manager/openapi/openapi.yaml` (`WebchatManagerSession` schema)
- `monorepo/bin-api-manager/server/webchat_sessions.go`
- `monorepo/bin-api-manager/pkg/servicehandler/main.go`
- `monorepo/bin-api-manager/pkg/servicehandler/webchat_session.go`
- `monorepo/bin-api-manager/pkg/servicehandler/mock_main.go` (regenerated)
- `monorepo/bin-api-manager/docsdev/source/webchat_struct_session.rst`
- `monorepo/bin-common-handler/pkg/requesthandler/` (WebchatV1SessionCreate signature + mock)
- `monorepo/bin-webchat-manager/models/session/session.go`
- `monorepo/bin-webchat-manager/models/session/field.go`
- `monorepo/bin-webchat-manager/models/session/webhook.go`
- `monorepo/bin-webchat-manager/pkg/sessionhandler/create.go`
- `monorepo/bin-webchat-manager/pkg/sessionhandler/create_test.go`
- `monorepo/bin-webchat-manager/pkg/dbhandler/session.go` (no logic change expected -- `PrepareFields`/`GetDBFields` are struct-tag-driven)
- `monorepo/bin-webchat-manager/scripts/database_scripts_test/sessions.sql`
- `monorepo/bin-dbscheme-manager/bin-manager/main/versions/<new>_webchat_sessions_add_column_page_url.py`
