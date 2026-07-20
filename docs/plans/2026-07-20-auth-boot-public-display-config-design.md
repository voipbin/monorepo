# POST /auth/boot: general-purpose `public_display_config` field for resource-scoped public display data

- Ticket: VOIP-1268
- Sibling ticket: SQUARE-15 ([webchat-widget-runtime] embed widget doesn't reflect saved theme_config)
- Repos in scope: `monorepo` (`bin-api-manager`, `bin-openapi-manager`), `monorepo-javascript` (`square-admin`)
- Status: draft, pending review loop

## 1. Problem

SQUARE-15 investigation found the webchat embed widget runtime has **no code
path at all** that fetches a widget's saved `theme_config` — not a caching or
deploy-lag bug. `square-admin/src/webchat-widget-runtime/index.js`'s
`createEmbeddableEntry()` constructs `WebchatWidget` with only `directHash`;
no theming data is ever threaded in. The result: every customer who
customizes their widget's header title, colors, or logo in the admin
console sees it reflected only in the Live Preview (which renders from local
React form state, not from the server) — the actual embedded widget on their
website always renders with platform defaults.

## 2. Rejected alternatives

### 2a. Bake `theme_config` into the static embed `<script>` snippet

The embed snippet (`square-admin/src/views/webchat_widgets/detail.js:50-51`)
is copy-pasted by the customer into their own site's HTML and is genuinely
static once deployed. Encoding `theme_config` into the snippet at
generation time (e.g. as a `data-theme` attribute) means **any subsequent
theme edit never reaches the already-deployed widget** — the customer would
have to notice the mismatch, regenerate the snippet, and redistribute it to
their own site. This regresses the customer experience versus competitor
products (Intercom/Crisp-class dashboards apply branding changes to already-
deployed widgets without requiring snippet redeployment). Rejected.

### 2b. Open a new unauthenticated `GET` for the widget's full config

`WebchatWidgetGet()` (`bin-api-manager/pkg/servicehandler/webchat_widget.go`)
is intentionally gated behind `IsDirect()` — relaxing that gate to let an
anonymous visitor JWT read the full `Widget` resource would expose fields
beyond cosmetic data (e.g. `SessionFlowID`, `MessageFlowID`) that have no
business being visible to an anonymous website visitor. Rejected in favor of
exposing only the specific, already-vetted-safe subset of data needed.

## 3. Chosen approach: extend `POST /auth/boot`'s response

The widget runtime already calls `POST /auth/boot` exactly once per page
load (`square-admin/src/webchat-widget-runtime/client.js`'s `_doStart()`),
resolving the widget's `direct_hash` into a resource-scoped JWT. This is
already the platform's sanctioned "public read via a non-secret hash"
pattern (`direct_hash` itself is documented as "not a traditional secret" —
`bin-webchat-manager/models/widget/webhook.go:14-20`, referencing VOIP-1264).
Extending this existing call's response, rather than adding a new endpoint,
adds zero new authentication surface.

### 3.1 Why a general field, not `theme_config` specifically

`POST /auth/boot` is a **shared entry point for multiple resource types**
(`bin-api-manager/pkg/servicehandler/boot.go:19-24`'s `directResourceMapping`
currently registers `ai`, `ai_team`, `webchat_widget`). A field literally
named `theme_config` would only make sense for one resource type. The
precedent for "one field, shape varies by a type discriminator" already
exists in this codebase: `bin-conversation-manager`'s `Account.ProviderData`
(`json.RawMessage`, shape varies by `Account.Type`). `BootResponse` already
carries a `ResourceType` discriminator (`boot.go:30`) — reuse it, don't add a
second one.

**Field name: `public_display_config` (not the earlier-discussed
`resource_data`).** This was a required change from the review loop below —
a generically-named field invites a future engineer wiring a fetcher for
`ai`/`ai_team` to stuff non-public data into it by default, since the name
carries no boundary signal. `public_display_config` self-documents: this
field is for data that is safe to show an anonymous, unauthenticated visitor.

### 3.2 Source discipline (mandatory, not just convention)

Any fetcher populating `public_display_config` **must** read from the
resource's `WebhookMessage` / `ConvertWebhookMessage()`-shaped external DTO,
never the raw internal domain struct. For webchat_widget specifically:

```go
// CORRECT — external-safe DTO, already vetted (webhook.go's ThemeConfig
// field is genuinely cosmetic-only; DirectID and other internal-only
// fields are already excluded by ConvertWebhookMessage()).
w, err := h.reqHandler.WebchatV1WidgetGet(ctx, resourceID)
if err == nil && w != nil {
    payload = w.ConvertWebhookMessage().ThemeConfig
}

// WRONG — never do this. The raw Widget struct is not vetted for
// external exposure; a future field added directly to Widget (e.g. an
// internal flag, DirectID, SessionFlowID) would leak silently through
// this path.
// payload = w.ThemeConfig
```

This is a **required code-review checklist item** for this PR and any
future PR adding a fetcher for another `resource_type`: reviewers must
confirm the fetcher reads a `WebhookMessage`-shaped struct, not the internal
model.

**Important scope limit of this rule (found in review, corrected here):**
reading via `ConvertWebhookMessage()` protects against **Widget-level**
internal fields leaking (`DirectID`, `SessionFlowID`, `MessageFlowID` are
all correctly excluded by the converter). It provides **zero** protection
against an unsafe field added directly to `ThemeConfig` itself —
`Widget.ThemeConfig` (`widget.go:54`) and `WebhookMessage.ThemeConfig`
(`webhook.go:32`) are declared as the **identical `*ThemeConfig` pointer
type**, and `ConvertWebhookMessage()` does a bare pointer copy
(`webhook.go:54`, `ThemeConfig: h.ThemeConfig`) with no field-level
filtering inside `ThemeConfig`. A future field added to `ThemeConfig`
would flow through both `w.ThemeConfig` and
`w.ConvertWebhookMessage().ThemeConfig` identically — the converter is not
a safety boundary for `ThemeConfig`'s own contents. **Additional
checklist item**: any new field added to `ThemeConfig` (`widget.go`) must
be independently vetted for anonymous-visitor safety at the time it is
added — it is not filtered by this design's fetcher pattern.

### 3.3 Failure semantics: best-effort, fail-open

The `public_display_config` fetch **must not** be able to fail the overall
`/auth/boot` request. Precedent already exists in this exact service pair:
`bin-webchat-manager/pkg/sessionhandler/create_test.go:158-160`
(`Test_Create_WidgetFetchFails_SessionStillSucceeds`) establishes that a
Widget-fetch failure during session creation is logged and swallowed, not
propagated. Apply the same rule here:

- RPC failure (network error, widget deleted mid-request, timeout): log at
  `Warn` or `Info`, set `public_display_config = nil`, return **HTTP 200**
  with the rest of `BootResponse` populated normally.
- A visitor must never be blocked from opening the chat widget because a
  cosmetic-data lookup hiccuped.
- `/auth/boot` is already bounded by `middleware.RateLimit(10, 20)` applied
  group-wide to the `auth` route group (`bin-api-manager/cmd/api-manager/
  main.go:236-237,245`, `bin-api-manager/lib/middleware/ratelimit.go:69-89`)
  — 10 req/s with a burst of 20, per client IP. `AuthBoot()` today makes 2
  backend RPCs per request (`DirectV1DirectGetByHash`, boot.go:49;
  `CustomerV1CustomerGet`, boot.go:57); this change adds a 3rd
  (`WebchatV1WidgetGet`), a 50% increase in backend RPC fan-out per
  anonymous request. This is not a new class of risk — the existing
  per-IP rate limit bounds inbound HTTP volume identically regardless of
  how many backend RPCs each request triggers — but is worth noting
  explicitly since this endpoint requires no authentication by design.

### 3.4 Type and JSON shape

```go
// BootResponse is the typed response for POST /auth/boot.
type BootResponse struct {
	Token        string      `json:"token"`
	Type         string      `json:"type"`
	ResourceType string      `json:"resource_type"`
	ResourceID   uuid.UUID   `json:"resource_id"`
	CustomerID   uuid.UUID   `json:"customer_id"`
	Expire       string      `json:"expire"`

	// PublicDisplayConfig carries resource-type-scoped, publicly-safe
	// display/cosmetic data for the anonymous client to render with.
	// Shape depends on ResourceType (see resourceDisplayConfigFetchers).
	// Populated best-effort: a fetch failure never fails the boot
	// request itself (see design doc §3.3). nil/omitted for resource
	// types with no registered fetcher.
	PublicDisplayConfig interface{} `json:"public_display_config,omitempty"`
}
```

`interface{}` (not a concrete struct) because the shape genuinely varies —
mirrors `ProviderData`'s untyped-blob precedent for the same reason (§3.1).

### 3.5 Fetcher registration (extensibility point)

```go
// resourceDisplayConfigFetchers maps a direct resource_type to a function
// that resolves its public_display_config payload. Add an entry here when
// a new resource type needs to expose safe, anonymous-visitor-facing
// display data through /auth/boot. Every fetcher MUST read from the
// resource's ConvertWebhookMessage()-shaped external DTO (see design doc
// §3.2) — never the raw internal model struct.
var resourceDisplayConfigFetchers = map[string]func(ctx context.Context, h *serviceHandler, resourceID uuid.UUID) (interface{}, error){
	"webchat_widget": func(ctx context.Context, h *serviceHandler, resourceID uuid.UUID) (interface{}, error) {
		w, err := h.reqHandler.WebchatV1WidgetGet(ctx, resourceID)
		if err != nil {
			return nil, err
		}
		return w.ConvertWebhookMessage().ThemeConfig, nil
	},
}
```

Inside `AuthBoot()`, after `BootResponse` is otherwise fully built (`d` here
is the same `*dmdirect.Direct` record already resolved earlier in
`AuthBoot()` via `h.reqHandler.DirectV1DirectGetByHash`, `boot.go:49`, and
already used to build `res.ResourceType`/`res.ResourceID`):

```go
if fetcher, ok := resourceDisplayConfigFetchers[d.ResourceType]; ok {
	data, ferr := fetcher(ctx, h, d.ResourceID)
	if ferr != nil {
		log.Warnf("Could not fetch public display config (best-effort, continuing). resource_type: %s, err: %v", d.ResourceType, ferr)
	} else {
		res.PublicDisplayConfig = data
	}
}
```

## 4. Frontend wiring (square-admin)

### 4.1 The originally-discussed wiring is impossible as first sketched — corrected here

`WebchatWidget`'s constructor (`widget.js:376-392`) calls
`applyWidgetTheme(this.themeConfig, this.dom)` **synchronously at
construction time**. `WebchatClient` (and the `/auth/boot` call it triggers
via `start()`) is only invoked later, from `open()` (`widget.js:579`) and
`_handleSend()` (`widget.js:531`) — well after the constructor has already
returned. `client.js` has no back-reference to the `WebchatWidget` instance
or its DOM today. Passing `public_display_config` "into the constructor" is
therefore not implementable as originally sketched.

**Fix**: add a new callback, mirroring the existing `onSessionStart` /
`onReconnected` pattern (`widget.js:398-467`), fired once `_doStart()`
resolves the boot response:

```js
// client.js — WebchatClient constructor opts
this.onBootResourceData = opts.onBootResourceData || (() => {})

// client.js — _doStart(), right after the /auth/boot call resolves
const boot = await this._fetchJson(this._apiUrl('/auth/boot'), { ... })
this.token = boot?.token
this.customerId = boot?.customer_id
this.resourceId = boot?.resource_id
if (boot?.public_display_config) {
  this.onBootResourceData(boot.public_display_config)
}
```

```js
// widget.js — WebchatWidget constructor, wiring the new callback
this.client = new WebchatClient({
  ...
  onBootResourceData: (displayConfig) => {
    // Guard against firing after the widget has been torn down: if a
    // visitor closes/destroys the widget while /auth/boot is still
    // in flight, _doStart() still resolves later and would otherwise
    // call applyWidgetTheme() on a detached this.dom. Mirrors the
    // existing this.isOpen-guard idiom already used for the
    // connecting/typing indicators (widget.js:583-593).
    if (this._destroyed) return
    // Re-apply theme with the server-confirmed config, overriding
    // whatever themeConfig (if any) the constructor was called with.
    this.themeConfig = displayConfig
    applyWidgetTheme(this.themeConfig, this.dom)
  },
})
```

**Re-entrancy / re-fire guarantee (verified against actual code, stated
explicitly per review round 1 finding):** `client.js`'s `_startPromise`
de-duplication (`client.js:264-273`) ensures `onBootResourceData` fires at
most once per concurrent `start()` race — multiple callers awaiting
`start()` before it resolves share the same in-flight `_doStart()` call, so
the callback cannot double-fire from that path. Separately, `end()`
(`client.js:768-785`) never resets `this.sessionId` to `null`. Combined
with `start()`'s own guard (`if (!this.client.sessionId) await
this.client.start()`, checked at both `open()` and `_handleSend()`), this
means **`onBootResourceData` fires at most once ever per `WebchatWidget`
instance** — a visitor who closes the widget and reopens it later in the
same page load does NOT re-trigger `/auth/boot`, and therefore does not
re-fetch or re-apply `public_display_config` on reopen. This is a further,
distinct limitation from §4.2's "already-open session" case (this one
covers close-then-reopen within the SAME page load, not a separate visit) —
documented explicitly here and in §4.2, not silently left as an implicit
consequence of the dedup mechanism. Accepted for this ticket's scope: a
full page reload is required to pick up a saved theme change regardless of
whether the widget was closed and reopened first.

### 4.2 Known limitation: theme updates do not reach already-open or reopened sessions

`/auth/boot` is called exactly once per `WebchatWidget` instance, at first
`start()` (`client.js`'s `start()`/`_doStart()`, gated by `if
(!this.client.sessionId)`). Two related limitations follow, both accepted
for this ticket's scope:

- **Already-open session (separate visit):** a visitor who already has the
  widget open when the customer saves a new theme in square-admin will
  **not** see the update until they reload the page.
- **Closed-then-reopened session (same page load):** per §4.1's re-entrancy
  analysis, `end()` never resets `sessionId`, so closing and reopening the
  widget within the same page load does not re-trigger `/auth/boot` either
  — only a full page reload picks up a new theme value.

Both are accepted, explicitly-documented limitations, not oversights:

- Both are a strict improvement over current production, where the embed
  path has **zero** theming input at all (`index.js`'s
  `createEmbeddableEntry()` passes no `themeConfig` whatsoever today).
- Both are a strict improvement over the rejected bake-into-snippet
  approach (§2a), which would freeze the theme permanently at
  snippet-generation time regardless of page reloads.
- A live-push mechanism (a new WS event type + client-side re-apply,
  covering both cases at once) is a plausible follow-up but explicitly out
  of scope for this ticket.

### 4.3 `index.js` still passes no `themeConfig` at construction — unchanged, intentional

`createEmbeddableEntry()` continues to construct `WebchatWidget({ directHash,
document })` with no `themeConfig` argument. The widget renders with
platform defaults initially, then re-themes itself once `_doStart()`
resolves and fires `onBootResourceData`. This is expected — there is no
config available before boot completes, and rendering with a brief default
flash-then-reflow is acceptable (matches the existing `_typingEl`/
`_reconnectingEl` "transient state, reconciled asynchronously" pattern
already used elsewhere in this runtime).

## 5. OpenAPI spec update (documentation-only — no runtime effect)

`POST /auth/boot` is served by a **hand-wired route**
(`bin-api-manager/lib/service/boot.go`'s `PostBoot`), not the OpenAPI-
codegen'd strict handler. The generated `PostAuthBoot`
(`bin-api-manager/server/auth_boot.go:10-19`) is an explicit dead stub that
always returns 404 — its own doc comment states the generated route is
"never called." `bin-api-manager` never imports or constructs
`AuthBootResponse` (the openapi-generated type) outside `gens/` (confirmed
via repo-wide grep, 18 hits, all inside `gens/openapi_server/` and
`gens/openapi_redoc/`).

**Implication: updating `openapi.yaml`'s `AuthBootResponse` schema and
regenerating types has zero effect on `/auth/boot`'s actual runtime
response shape**, which is governed solely by the local `BootResponse`
struct in `bin-api-manager/pkg/servicehandler/boot.go`. The spec update is
still required — it is the source of truth for `docs.voipbin.net` and the
Swagger/ReDoc UIs, and leaving it stale would actively mislead external API
consumers — but must not be assumed to "activate" anything.

**CI dependency (build-time, not runtime):** the `bin-openapi-manager-
validate` CI job (`.circleci/config_work.yml:1347-1372`) runs `go generate
./...` in `bin-openapi-manager` and fails the build if the committed
`gens/models/gen.go` doesn't match what regeneration produces
(`config_work.yml:1369-1371`). Editing `openapi.yaml` without running `go
generate` and committing the regenerated file **will break CI** — this is
a real, narrow consequence distinct from the "zero runtime effect" claim
above, and implementers must run the regen step even though the change has
no effect on `/auth/boot`'s actual behavior. Separately, there is no
automated check anywhere that `AuthBootResponse` (spec type) matches the
real `BootResponse` Go struct's shape going forward — since the generated
route is permanently dead, that drift is invisible to CI and this PR does
not change that pre-existing gap.

```yaml
# bin-openapi-manager/openapi/openapi.yaml — AuthBootResponse schema addition
public_display_config:
  type: object
  nullable: true
  description: >
    Additional, publicly-safe display/cosmetic data scoped to resource_type.
    Present only for resource types with a registered fetcher (currently
    "webchat_widget", carrying the widget's WebchatManagerWidgetThemeConfig
    shape). Omitted or null for resource types without one, or if the
    underlying lookup failed (best-effort; never blocks token issuance).
  example: { "primary_color": "#2563eb", "position": "bottom_right" }
```

## 6. Scope by repo

| Repo | Component | Change |
|---|---|---|
| `monorepo` | `bin-api-manager/pkg/servicehandler/boot.go` | `BootResponse.PublicDisplayConfig` field, `resourceDisplayConfigFetchers` map, best-effort fetch wiring in `AuthBoot()` |
| `monorepo` | `bin-api-manager/pkg/servicehandler/boot_test.go` | New/updated tests: happy path, RPC-failure fail-open, no-fetcher omitempty, §3.2 source-discipline assertion (payload contains only `WebhookMessage`-safe fields) |
| `monorepo` | `bin-openapi-manager/openapi/openapi.yaml`, `openapi/paths/auth/boot.yaml` | Add `public_display_config` to `AuthBootResponse` schema (docs-only, see §5); regenerate via `go generate ./...` to satisfy CI (§5) |
| `monorepo` | `bin-api-manager/docsdev/source/` | Rebuild RST docs (`AuthBootResponse` is user-visible in Swagger/ReDoc — CLAUDE.md's RST Docs Sync rule applies) |
| `monorepo-javascript` | `square-admin/src/webchat-widget-runtime/client.js` | `onBootResourceData` callback, fired from `_doStart()` |
| `monorepo-javascript` | `square-admin/src/webchat-widget-runtime/widget.js` | Wire `onBootResourceData` to re-invoke `applyWidgetTheme()`, guarded against post-destroy firing (§4.1) |
| `monorepo-javascript` | `square-admin/src/webchat-widget-runtime/__tests__/` | New tests: `onBootResourceData` fires and re-themes; destroyed-widget no-op; no re-fire on reopen-after-close (§4.1, §8) |

## 7. Explicitly out of scope

- Live-push theme updates to already-open sessions (§4.2).
- Registering fetchers for `ai`/`ai_team` resource types (no product
  requirement today; the extensibility point exists but is not populated).
- Any change to `WebchatWidgetGet()`'s existing `IsDirect()` gate (§2b).
- Caching the `bin-webchat-manager` RPC result inside `bin-api-manager` or
  `bin-webchat-manager` to reduce per-request fan-out — noted as a future
  consideration if more resource types get fetchers registered and
  `/auth/boot`'s RPC fan-out becomes a measured concern, not addressed here.

## 8. Verification plan

- `bin-api-manager`: unit test for `AuthBoot()` covering (a) `webchat_widget`
  happy path returns `public_display_config` populated from
  `ConvertWebhookMessage().ThemeConfig`, (b) `WebchatV1WidgetGet` RPC failure
  still returns HTTP 200 with `public_display_config` nil and the rest of
  `BootResponse` populated, (c) a resource type with no registered fetcher
  (`ai`/`ai_team`) omits the field entirely (`omitempty`), (d) a regression
  test asserting the fetcher's returned payload contains only fields present
  on `WebhookMessage`/`ThemeConfig` as currently defined — enforcing §3.2's
  source-discipline rule with a real test, not just a code-review checklist
  item.
- `square-admin`: unit test for `client.js`'s `_doStart()` confirming
  `onBootResourceData` fires with the parsed `public_display_config` value,
  and a `widget.js` test confirming `applyWidgetTheme()` is re-invoked with
  the boot-delivered config, overriding any constructor-time default. Also:
  a test confirming `onBootResourceData` firing after `destroy()`/`close()`
  is a no-op (§4.1 guard), and a test confirming closing and reopening the
  widget within the same page load does NOT re-trigger `/auth/boot` or
  re-fire `onBootResourceData` (§4.1 re-entrancy finding, §4.2 documented
  limitation).
- Manual end-to-end: save a theme change in square-admin, open the embed
  widget in a fresh browser tab (simulating a new visitor), confirm the
  updated header title/colors render without any snippet redeployment.
- Full verification workflow (`go mod tidy && go mod vendor && go generate
  ./... && go test ./... && golangci-lint run -v --timeout 5m`) in
  `bin-api-manager` before PR, per root CLAUDE.md.

## 9. Round review disposition

(Filled in after each review round — see `design-first-with-review-loops`
skill's convention of recording verdict + disposition inside the doc
itself, not only in the review transcript.)

### Pre-draft adversarial review (3 parallel angles, run against the verbal proposal before this doc existed)

All three returned **REQUEST CHANGES**; all three sets of required changes
are incorporated into this draft (§3.1 field rename, §3.2 source discipline,
§3.3 failure semantics, §3.4 omitempty, §4.1 callback-based frontend wiring,
§4.2 known-limitation callout, §5 docs-only clarification).

### Round 1 (3 parallel angles: feasibility/correctness, completeness/internal-consistency, adversarial security/production-readiness)

- **Feasibility/correctness angle: APPROVE** with 3 minor citation-accuracy
  fixes (line-number off-by-ones in §3.1, a conflated function-name citation
  in §4.1). All fixed in this revision. No structural or compile-time gaps
  found; confirmed the proposed Go changes are buildable as described.
- **Completeness/internal-consistency angle: REQUEST CHANGES.** Found: (1)
  §3.5 used an undefined variable `d` without introducing it — fixed with an
  inline clarification tying it to the existing `AuthBoot()` flow; (2) §4.1
  never addressed `onBootResourceData` firing after widget teardown (torn-
  down DOM) — fixed with an explicit guard + code comment; (3) §4.1 never
  verified interaction with the existing `_startPromise` dedup and `end()`
  never resetting `sessionId` — fixed with an explicit "Re-entrancy / re-fire
  guarantee" paragraph and a corresponding §4.2 second bullet; (4) §8's
  verification plan didn't cover §3.2's "mandatory" source-discipline rule
  with an actual test, and didn't cover the two new §4.1 findings — fixed by
  adding test requirements (d) in §8's backend list and two new bullets in
  §8's frontend list; (5) §6's scope table omitted test files implied by §8
  — fixed by adding two new rows.
- **Adversarial security/production-readiness angle: REQUEST CHANGES.**
  Found: (1) §3.2's stated safety rationale ("read via ConvertWebhookMessage,
  never raw Widget") does not actually protect against an unsafe field added
  directly to `ThemeConfig` itself, since `Widget.ThemeConfig` and
  `WebhookMessage.ThemeConfig` are the identical `*ThemeConfig` pointer type
  — fixed by adding an explicit "Important scope limit of this rule"
  paragraph correcting the overstated claim and adding a checklist item for
  future `ThemeConfig` field additions; (2) rate-limiting context was
  missing — fixed by adding a paragraph in §3.3 citing the existing
  `RateLimit(10,20)` guard and noting the RPC fan-out increase explicitly;
  (3) the OpenAPI "zero runtime effect" claim, while true, omitted a real CI
  dependency (`bin-openapi-manager-validate` fails if the spec change isn't
  regenerated) — fixed by adding a "CI dependency (build-time, not runtime)"
  paragraph in §5.

All required changes from Round 1 are incorporated above. Proceeding to
Round 2.
