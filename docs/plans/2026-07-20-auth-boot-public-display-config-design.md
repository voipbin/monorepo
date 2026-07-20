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
`resource_data`)** for the DATA itself. **[Superseded by Revision 1, §9
below — `resource_data` is reintroduced as a wrapping ENVELOPE, with
`public_display_config` as a named key inside it, not the top-level field
name.]** The renaming rationale below still holds for why the key itself
must be self-documenting, not generic:

a generically-named field invites a future engineer wiring a fetcher for
`ai`/`ai_team` to stuff non-public data into it by default, since the name
carries no boundary signal. `public_display_config` self-documents: this
key is for data that is safe to show an anonymous, unauthenticated visitor.

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

**Important scope limit of this rule (corrected here):**
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
added — it is not filtered by this design's fetcher pattern. This checklist
item will be backed by an in-code artifact, not prose alone (see the
required `ThemeConfig` struct comment below, to be added during the
implementation phase) — VoIPBin's `monorepo` has no `.github/
PULL_REQUEST_TEMPLATE` to hang a checklist on, so a durable in-code warning
is the only mechanism that will actually reach a future editor at the
point of risk.

**Required implementation-phase change: add a
warning comment directly on the `ThemeConfig` struct definition itself**
(`bin-webchat-manager/models/widget/widget.go:83`, immediately above
`type ThemeConfig struct`), since the existing comment there only
describes the field as "cosmetic, customer-editable" with no mention of
anonymous-visitor exposure. **[Text below updated per Revision 1, §9 —
"field" corrected to "resource_data envelope's public_display_config
key".]**

```go
// ThemeConfig holds cosmetic, customer-editable widget appearance
// settings. Nil/omitted fields fall back to platform defaults.
//
// SECURITY: this struct is serialized verbatim to ANONYMOUS website
// visitors via POST /auth/boot's resource_data envelope's
// public_display_config key (see
// docs/plans/2026-07-20-auth-boot-public-display-config-design.md).
// Any new field added here must be independently vetted as safe for
// unauthenticated public exposure before merge -- it is NOT filtered
// by ConvertWebhookMessage() or any other export boundary.
type ThemeConfig struct {
```

### 3.3 Failure semantics: best-effort, fail-open

The `public_display_config` fetch **must not** be able to fail the overall
`/auth/boot` request. Precedent already exists in this exact service pair:
`bin-webchat-manager/pkg/sessionhandler/create_test.go:158-160`
(`Test_Create_WidgetFetchFails_SessionStillSucceeds`) establishes that a
Widget-fetch failure during session creation is logged and swallowed, not
propagated. Apply the same rule here:

- RPC failure (network error, widget deleted mid-request, timeout): log at
  `Warn` or `Info`, omit the `public_display_config` key from the
  response entirely (see §9.2 for the exact mechanism — the fetcher
  returns an error, so the envelope key is simply never inserted, not
  assigned `nil`), return **HTTP 200** with the rest of `BootResponse`
  populated normally.
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

**[Superseded by Revision 1, §9.1 below — this field became a
`map[string]interface{}` envelope named `resource_data`, with
`public_display_config` nested inside it as a key, rather than a
top-level `interface{}` field on `BootResponse`. The typed-nil trap
described below still applies, re-derived one level deeper in §9.1.]**

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

**Typed-nil trap (must be handled explicitly):**
Go's `encoding/json` `omitempty` only drops an `interface{}` field when the
interface itself is a TRUE nil (no type, no value). A widget that exists
but has no customer-configured theme has `Widget.ThemeConfig == nil` (a
nil `*ThemeConfig`), and `ConvertWebhookMessage().ThemeConfig` returns that
same nil `*ThemeConfig` — assigning a nil TYPED POINTER into the static
`interface{}` field `PublicDisplayConfig` produces a NON-nil interface
value (it carries the type `*ThemeConfig`, value nil). `omitempty` will
NOT drop this — the JSON output would be `"public_display_config": null`,
not an omitted key, for the extremely common "widget exists, no custom
theme set" case.

**Required fix**: the `webchat_widget` fetcher must explicitly normalize a
nil `*ThemeConfig` to a true nil `interface{}` before returning, so the two
cases ("no fetcher registered" and "fetcher ran but the widget has no
custom theme") both correctly omit the key:

```go
"webchat_widget": func(ctx context.Context, h *serviceHandler, resourceID uuid.UUID) (interface{}, error) {
	w, err := h.reqHandler.WebchatV1WidgetGet(ctx, resourceID)
	if err != nil {
		return nil, err
	}
	tc := w.ConvertWebhookMessage().ThemeConfig
	if tc == nil {
		// Explicit nil interface, not a nil *ThemeConfig boxed into
		// interface{} -- see design doc's typed-nil/omitempty note.
		// A boxed typed-nil pointer is a non-nil interface value and
		// omitempty would otherwise serialize it as "public_display_config": null
		// instead of omitting the key.
		return nil, nil
	}
	return tc, nil
},
```

### 3.5 Fetcher registration (extensibility point)

**[Superseded by Revision 1, §9.2 below — the wiring snippet's assignment
target changed from `res.PublicDisplayConfig = data` to inserting into
`res.ResourceData["public_display_config"]`, only when `data != nil`. The
fetcher function signature itself is unchanged.]**

```go
// resourceDisplayConfigFetchers maps a direct resource_type to a function
// that resolves its public_display_config payload. Add an entry here when
// a new resource type needs to expose safe, anonymous-visitor-facing
// display data through /auth/boot. Every fetcher MUST read from the
// resource's ConvertWebhookMessage()-shaped external DTO (see design doc
// §3.2) — never the raw internal model struct. Every fetcher MUST also
// return a true nil interface{} (not a nil-but-typed pointer) when there
// is no data to report, so `omitempty` actually omits the key (see design
// doc §3.4's typed-nil note).
var resourceDisplayConfigFetchers = map[string]func(ctx context.Context, h *serviceHandler, resourceID uuid.UUID) (interface{}, error){
	"webchat_widget": func(ctx context.Context, h *serviceHandler, resourceID uuid.UUID) (interface{}, error) {
		w, err := h.reqHandler.WebchatV1WidgetGet(ctx, resourceID)
		if err != nil {
			return nil, err
		}
		tc := w.ConvertWebhookMessage().ThemeConfig
		if tc == nil {
			return nil, nil
		}
		return tc, nil
	},
}
```

Inside `AuthBoot()`, after `BootResponse` is otherwise fully built (`d` here
is the same `*dmdirect.Direct` record already resolved earlier in
`AuthBoot()` via `h.reqHandler.DirectV1DirectGetByHash`, `boot.go:49`, and
already used to build `res.ResourceType`/`res.ResourceID`). Log level: use
`Infof`, matching the file's existing convention for non-fatal
backend-lookup outcomes (`boot.go:51,59,65,72` all use `Infof` for
lookup-miss cases; there is no existing `Warnf` call anywhere in this
file, and this is the same class of "backend lookup didn't come back
cleanly, not fatal" event). Message format follows the file's existing
terse `"Could not X. err: %v"` convention (compare `boot.go:51`'s "Could
not get direct by hash. err: %v"), not a longer parenthetical-annotated
variant:

```go
if fetcher, ok := resourceDisplayConfigFetchers[d.ResourceType]; ok {
	data, ferr := fetcher(ctx, h, d.ResourceID)
	if ferr != nil {
		log.Infof("Could not fetch public display config. resource_type: %s, err: %v", d.ResourceType, ferr)
	} else {
		res.PublicDisplayConfig = data
	}
}
```

## 4. Frontend wiring (square-admin)

### 4.1 The originally-discussed wiring is impossible as first sketched — corrected here

**[The `boot?.public_display_config` read in the snippet below is
superseded by Revision 1, §9.3 — the boot response's read site changed to
`boot?.resource_data?.public_display_config`. Everything else in this
section (the callback signature, the `_destroyed` teardown guard, the
re-entrancy analysis) is unaffected — see §9.3's own note.]**

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
    // visitor destroys the widget while /auth/boot is still in flight,
    // _doStart() still resolves later and would otherwise call
    // applyWidgetTheme() on a detached this.dom. Mirrors the existing
    // this.isOpen-guard idiom already used for the connecting/typing
    // indicators (widget.js:583-593). Gated on destroy() specifically
    // (not close()) because close() only hides the panel via CSS and
    // leaves this.dom attached -- only destroy() actually detaches it
    // (widget.js:605-612).
    if (this._destroyed) return
    // Re-apply theme with the server-confirmed config, overriding
    // whatever themeConfig (if any) the constructor was called with.
    this.themeConfig = displayConfig
    applyWidgetTheme(this.themeConfig, this.dom)
  },
})
```

**Required wiring for the `_destroyed` flag** (the guard above references a
property that must be introduced; without this explicit instruction, an
implementer could add only the read-site check and leave it permanently
dead code):
- Constructor: initialize `this._destroyed = false` alongside the other
  instance fields set at construction time (`widget.js:376-392`).
- `destroy()` (`widget.js:605-612`): set `this._destroyed = true` as its
  FIRST statement, before any DOM teardown, so a boot response resolving
  concurrently with `destroy()` sees the flag set no matter how the two
  race.
- `close()` (`widget.js:597-603`) must NOT set this flag — it only hides
  the panel via a CSS class and leaves `this.dom` attached, so a boot
  response resolving after `close()` should still be safe to re-theme
  against (the visitor may reopen the same DOM later).

**Re-entrancy / re-fire guarantee (verified against actual code):** `client.js`'s `_startPromise`
de-duplication (`client.js:264-273`) ensures `onBootResourceData` fires at
most once per concurrent `start()` race — multiple callers awaiting
`start()` before it resolves share the same in-flight `_doStart()` call, so
the callback cannot double-fire from that path. Separately, `end()`
(`client.js:768-785`) never resets `this.sessionId` to `null`, and this
design does NOT add such a reset either (see §4.2's explicit rejection of
that idea, for a cost/abuse reason unrelated to re-entrancy). Combined with
`start()`'s own guard (`if (!this.client.sessionId) await
this.client.start()`, checked at both `open()` and `_handleSend()`), this
means **`onBootResourceData` fires at most once ever per `WebchatWidget`
instance** — a visitor who closes the widget and reopens it later in the
same page load does NOT re-trigger `/auth/boot`, and therefore does not
re-fetch or re-apply `public_display_config` on reopen. This is intentional
and documented as an accepted limitation in §4.2, not an oversight — the
alternative (resetting `sessionId` on `close()`) was considered and
rejected because it would also re-trigger `Widget.SessionFlowID`'s Flow
execution on every close/reopen cycle (§4.2).

### 4.2 Known limitations: theme updates do not reach already-open or reopened sessions

`/auth/boot` is called exactly once per `WebchatWidget` instance, at first
`start()` (`client.js`'s `start()`/`_doStart()`, gated by `if
(!this.client.sessionId)`). Two related cases follow, both accepted for
this ticket's scope:

- **Already-open session (separate visit):** a visitor who already has the
  widget open when the customer saves a new theme in square-admin will not
  see the update until they reload the page.
- **Closed-then-reopened session (same page load):** closing and reopening
  the widget within the same page load does not re-trigger `/auth/boot`
  either — only a full page reload picks up a new theme value.

**A `close()`-resets-`sessionId` mitigation was considered and explicitly
REJECTED** (found during review): resetting `this.client.sessionId = null`
in `close()` would make `open()`'s existing guard (`if
(!this.client.sessionId) await this.client.start()`) re-run `start()` on
every reopen, which does correctly re-fetch `public_display_config` — but
it ALSO re-creates a brand-new webchat Session on the server on every
close/reopen cycle. `bin-webchat-manager/pkg/sessionhandler/create.go:
62-93` shows every Session creation checks `Widget.SessionFlowID`, and if
configured, triggers `ConversationV1ConversationCreateAndExecuteFlow` — a
new Conversation plus a full Flow execution, which can include AI Team
calls, agent notifications, or other billing-relevant actions. Before this
change, `open()`'s guard only fires `start()` once per page load, so
`SessionFlowID` fires once per visit. The rejected mitigation would make a
visitor idly toggling the chat bubble open/close re-trigger the configured
flow on every single toggle — `POST /webchat_sessions` is not covered by
the `/auth` group's `RateLimit(10,20)` (confirmed absent from
`bin-api-manager/cmd/api-manager/main.go`'s rate-limited routes), so this
is an uncosted, unbounded amplification path (repeated AI/flow spend,
duplicate agent notifications, "why did I get 5 welcome messages"-class
customer complaints) — a materially worse and more expensive problem than
the cosmetic staleness bug this mitigation was meant to fix. **Do not
implement a `sessionId` reset on `close()`.**

Both accepted limitations above are still improvements over current
production, where the embed path has **zero** theming input at all
(`index.js`'s `createEmbeddableEntry()` passes no `themeConfig` whatsoever
today), and over the rejected bake-into-snippet approach (§2a), which
would freeze the theme permanently at snippet-generation time regardless
of page reloads or reopens. A live-push mechanism (a new WS event type +
client-side re-apply, which would deliver a theme update to an
already-running client WITHOUT creating a new Session or re-triggering
`SessionFlowID`) is the correct fix for both remaining cases and is a
plausible follow-up, but explicitly out of scope for this ticket — see §7.

### 4.3 `index.js` still passes no `themeConfig` at construction — unchanged, intentional

`createEmbeddableEntry()` continues to construct `WebchatWidget({ directHash,
document })` with no `themeConfig` argument. The widget renders with
platform defaults initially, then re-themes itself once `_doStart()`
resolves and fires `onBootResourceData`. This is expected — there is no
config available before boot completes, and rendering with a brief default
flash-then-reflow is acceptable (matches the existing `_typingEl`/
`_reconnectingEl` "transient state, reconciled asynchronously" pattern
already used elsewhere in this runtime).

### 4.4 The already-embedded widget-runtime bundle picks up this change without any customer action

Confirmed the customer-facing distribution path so §2a's rejection
rationale ("bake into snippet requires redeployment") doesn't silently
apply to the JS half of this fix by a different mechanism:

- `square-admin/nginx.conf:28-31` serves `/webchat/embed.js` as an alias to
  `webchat-widget-runtime.bundle.js` with `Cache-Control: public,
  max-age=300` (a 5-minute cache — deliberately shorter than the 1-year
  immutable cache rule at `nginx.conf:20-23` used for other static assets).
- `package.json:13` wires `build:widget` as a `prebuild` step, so every
  `square-admin` deploy regenerates the bundle automatically.
- **Net effect**: once this PR's `square-admin` deploy ships, an
  already-embedded customer `<script src="https://admin.voipbin.net/
  webchat/embed.js">` tag picks up the new runtime automatically, within
  at most 5 minutes, on the visitor's next page load — no customer action
  (snippet regeneration, redeployment) required. This is a structurally
  different mechanism from the direct-hash `data-theme`-baking approach
  rejected in §2a (which would have frozen data at snippet-generation
  time regardless of deploys); here only the STATIC JS CODE is
  cache-bounded, and it already refreshes on every backend deploy cadence.

## 5. OpenAPI spec update (documentation-only — no runtime effect)

**[The bare `public_display_config` top-level schema property below is
superseded by Revision 1, §9.4 — it is now nested inside a `resource_data`
envelope schema. §5's surrounding prose (dead-stub route, docs-only claim,
CI dependency) is unaffected by the revision and still applies.]**

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
  nullable: true
  description: >
    Additional, publicly-safe display/cosmetic data scoped to resource_type.
    Present only for resource types with a registered fetcher (currently
    "webchat_widget", carrying the widget's WebchatManagerWidgetThemeConfig
    shape). The key is OMITTED (not present) for resource types without a
    registered fetcher, when the underlying lookup failed (best-effort;
    never blocks token issuance), and when the widget has no
    customer-configured theme (all fields fall back to platform defaults).
  oneOf:
    - $ref: '#/components/schemas/WebchatManagerWidgetThemeConfig'
```

**Schema typing fix (found in review):** the concrete shape for this
field's only currently-registered case (`webchat_widget`) already has a
first-class named schema in this file —
`WebchatManagerWidgetThemeConfig` (`openapi.yaml:2340-2374`, already
`$ref`'d for the `Widget` resource's own `theme_config` field at
`openapi.yaml:2468`). An earlier draft of this snippet used a bare `type:
object` blob with only a hand-written example — that both discards
real, already-defined field-level typing for external SDK/docs
consumers, and conflicts with `bin-openapi-manager/CLAUDE.md`'s mandatory
rule #1 ("`oneOf` for polymorphism — no `additionalProperties: true` on
type-discriminated objects"; `public_display_config` is explicitly
type-discriminated by `resource_type`, per §3.1). The `oneOf` wrapper
(rather than a bare `$ref`) is deliberate even though there is currently
only one registered fetcher — it is the correct, extensible shape for
when a second resource type's fetcher is registered in the future (§3.5's
extensibility map), avoiding a breaking spec change at that point.

## 6. Scope by repo

**[Rows below describe the pre-Revision-1 flat-field shape in places —
see the "superseded" markers inline on the `boot.go` and `openapi.yaml`
rows, and §9.5 for the full delta list.]**

| Repo | Component | Change |
|---|---|---|
| `monorepo` | `bin-api-manager/pkg/servicehandler/boot.go` | `BootResponse.PublicDisplayConfig` field, `resourceDisplayConfigFetchers` map, best-effort fetch wiring in `AuthBoot()` **[superseded by Revision 1, §9.1/§9.2 — now `BootResponse.ResourceData map[string]interface{}`]** |
| `monorepo` | `bin-api-manager/pkg/servicehandler/boot_test.go` | New/updated tests: happy path, RPC-failure fail-open, no-fetcher omitempty, §3.2 source-discipline assertion (payload contains only `WebhookMessage`-safe fields) |
| `monorepo` | `bin-openapi-manager/openapi/openapi.yaml`, `openapi/paths/auth/boot.yaml` | Add `public_display_config` to `AuthBootResponse` schema (docs-only, see §5); regenerate via `go generate ./...` to satisfy CI (§5) **[superseded by Revision 1, §9.4 — now a `resource_data` envelope schema]** |
| `monorepo` | `bin-api-manager/docsdev/source/` | Rebuild RST docs (`AuthBootResponse` is user-visible in Swagger/ReDoc — CLAUDE.md's RST Docs Sync rule applies) |
| `monorepo-javascript` | `square-admin/src/webchat-widget-runtime/client.js` | `onBootResourceData` callback, fired from `_doStart()` |
| `monorepo-javascript` | `square-admin/src/webchat-widget-runtime/widget.js` | Wire `onBootResourceData` to re-invoke `applyWidgetTheme()`, guarded against post-destroy firing (§4.1). `close()` is NOT modified to reset `sessionId` — see §4.2's explicit rejection of that idea. |
| `monorepo-javascript` | `square-admin/src/webchat-widget-runtime/__tests__/` | New tests: `onBootResourceData` fires and re-themes; destroyed-widget no-op; no re-fire on reopen-after-close (§4.1, §8) |

## 7. Explicitly out of scope

- Live-push theme updates to already-open OR reopened-within-same-page-load
  sessions (§4.2). A `close()`-resets-`sessionId` shortcut was considered
  and explicitly rejected (§4.2) because it would also re-trigger
  `Widget.SessionFlowID`'s Flow execution on every close/reopen cycle — an
  uncosted, unbounded amplification path distinct from the theming
  problem. A real fix requires a live-push WS mechanism that updates an
  already-running client's theme without creating a new Session.
- Registering fetchers for `ai`/`ai_team` resource types (no product
  requirement today; the extensibility point exists but is not populated).
- Any change to `WebchatWidgetGet()`'s existing `IsDirect()` gate (§2b).
- Caching the `bin-webchat-manager` RPC result inside `bin-api-manager` or
  `bin-webchat-manager` to reduce per-request fan-out — noted as a future
  consideration if more resource types get fetchers registered and
  `/auth/boot`'s RPC fan-out becomes a measured concern, not addressed here.

## 8. Verification plan

**[Assertions below reference the pre-Revision-1 flat shape
(`res.PublicDisplayConfig`, bare `public_display_config` in mock
fixtures) — see §9.5 for the exact updated assertions
(`res.ResourceData["public_display_config"]`,
`boot.resource_data.public_display_config`).]**

- `bin-api-manager`: unit test for `AuthBoot()` covering (a) `webchat_widget`
  happy path returns `public_display_config` populated from
  `ConvertWebhookMessage().ThemeConfig`, (b) `WebchatV1WidgetGet` RPC failure
  still returns HTTP 200 with the `resource_data` envelope entirely nil
  (not `{}`, not `{"public_display_config": null}`) and the rest of
  `BootResponse` populated, (c) a resource type with no registered fetcher
  (`ai`/`ai_team`) omits `resource_data` entirely (`omitempty` on the nil
  envelope map), (d) the fetcher SUCCEEDS but the widget has no
  customer-configured theme (`Widget.ThemeConfig == nil`, so
  `ConvertWebhookMessage().ThemeConfig == nil` and the fetcher returns
  `(nil, nil)` per §9.2) — `resource_data` must ALSO be entirely omitted
  in this case, distinct from both (b) and (c). This is the exact wire-
  level scenario that motivated the typed-nil trap finding (§3.4/§9.1) in
  the first place and must have its own explicit test case, not be
  assumed covered by (b) or (c). All three of (b)/(c)/(d) result in the
  same wire output (`resource_data` absent) but exercise three genuinely
  different code paths per §9.2 — `ferr != nil`, the map lookup's `ok ==
  false`, and `data != nil` evaluating false — so each needs its own test.
  (e) §3.2's source-discipline rule enforced via code review against the
  `ThemeConfig` struct's required SECURITY comment (§3.2) rather than a
  runtime reflection-based field-diff test — this codebase has no existing
  precedent for reflection-based struct field comparison, and inventing one
  (embedded structs, JSON tags vs Go field names) is nontrivial enough that
  the code-review-plus-struct-comment combination is the pragmatic choice
  here, not a gap.
- `square-admin`: unit test for `client.js`'s `_doStart()` confirming
  `onBootResourceData` fires with the parsed `public_display_config` value,
  and a `widget.js` test confirming `applyWidgetTheme()` is re-invoked with
  the boot-delivered config, overriding any constructor-time default. Also:
  a test confirming `onBootResourceData` firing after `destroy()`/`close()`
  is a no-op (§4.1 guard), and a test confirming closing and reopening the
  widget within the same page load does NOT re-trigger `/auth/boot` or
  re-fire `onBootResourceData` (§4.1 re-entrancy finding, §4.2 documented
  limitation — this is intentional, not a bug, per §4.2's rejection of the
  `sessionId`-reset mitigation).
- Manual end-to-end: save a theme change in square-admin, open the embed
  widget in a fresh browser tab (simulating a new visitor), confirm the
  updated header title/colors render without any snippet redeployment.
- Full verification workflow (`go mod tidy && go mod vendor && go generate
  ./... && go test ./... && golangci-lint run -v --timeout 5m`) in
  `bin-api-manager` before PR, per root CLAUDE.md.

## 9. Wire contract Revision 1 (2026-07-20, post-8-round-closure, pchero-initiated)

**Trigger**: after the review loop closed (Round 7/Round 8, both clean
APPROVE), pchero reviewed a sample `/auth/boot` response and proposed
wrapping `public_display_config` inside a named `resource_data` envelope,
rather than having it be a top-level `BootResponse` field:

```json
{
  "resource_data": {
    "public_display_config": { "primary_color": "#2563eb", "...": "..." }
  }
}
```

**Rationale accepted**: `BootResponse` currently has a fixed set of
top-level fields; every FUTURE kind of resource-scoped public data (not
just cosmetic/display data) would otherwise require adding a NEW top-level
field to `BootResponse` each time. Wrapping in a `resource_data` envelope
means `BootResponse`'s shape never needs to change again — new kinds of
public data become new named keys inside the envelope. This is a closer
match to the platform's existing `Account.ProviderData` precedent (§3.1)
than the flat-field version was: `ProviderData` is itself an envelope, not
a single-purpose top-level field.

**Confirmed NOT a regression of the §3.1 self-documentation rule**: the
safety-boundary argument for avoiding a generic name (§3.1) applies to
KEYS inside the envelope, not to the envelope's own name. `resource_data`
itself is expected to be generic (it is a container, not a specific
datum); `public_display_config` remains the specific, self-documenting KEY
within it, and any future key added to the envelope must still carry its
own boundary-signaling name (e.g. a hypothetical future
`public_status_flags` key would need the same "public_" self-documentation
discipline `public_display_config` already has).

### 9.1 Updated `BootResponse` struct (supersedes §3.4)

```go
// BootResponse is the typed response for POST /auth/boot.
type BootResponse struct {
	Token        string      `json:"token"`
	Type         string      `json:"type"`
	ResourceType string      `json:"resource_type"`
	ResourceID   uuid.UUID   `json:"resource_id"`
	CustomerID   uuid.UUID   `json:"customer_id"`
	Expire       string      `json:"expire"`

	// ResourceData is a resource-type-scoped envelope for additional,
	// publicly-safe data about the boot-scoped resource. Each entry is a
	// named, self-documenting key (see design doc §3.1/§9) -- do not add
	// bare/generic keys. Currently the only populated key is
	// "public_display_config" (see resourceDisplayConfigFetchers).
	// Populated best-effort: a fetch failure never fails the boot
	// request itself (§3.3). nil/omitted entirely when no fetcher is
	// registered for ResourceType, or when every fetcher for this
	// resource returned nothing.
	ResourceData map[string]interface{} `json:"resource_data,omitempty"`
}
```

`map[string]interface{}` (not a single `interface{}`) because the envelope
is now explicitly multi-key-capable — this is the actual mechanism that
lets future kinds of public data ride in without another `BootResponse`
struct change.

**Typed-nil trap still applies, now one level deeper.** §3.4's typed-nil
finding (a nil `*ThemeConfig` boxed into `interface{}` is a non-nil
interface value) is unaffected by the envelope wrapping — the fetcher
below still normalizes a nil `*ThemeConfig` to a true nil before assigning
it into the map. What changes: `omitempty` on the OUTER `ResourceData`
map only omits the whole `resource_data` key when the map itself is
`nil` — a map containing one key with a `nil` value (e.g. `{"public_display_config":
nil}`) is NOT empty and would still serialize (as `"resource_data":
{"public_display_config": null}`). The wiring below must therefore only
insert the `"public_display_config"` key into the map when the fetcher
actually returned non-nil data, not insert-then-let-omitempty-handle-it.

### 9.2 Updated fetcher wiring (supersedes §3.5)

```go
// resourceDisplayConfigFetchers maps a direct resource_type to a function
// that resolves its public_display_config payload for the resource_data
// envelope (§9). Add an entry here when a new resource type needs to
// expose safe, anonymous-visitor-facing display data through /auth/boot.
// Every fetcher MUST read from the resource's ConvertWebhookMessage()-
// shaped external DTO (§3.2) -- never the raw internal model struct.
// Every fetcher MUST return a true nil interface{} (not a nil-but-typed
// pointer) when there is no data to report (§3.4/§9.1's typed-nil note).
var resourceDisplayConfigFetchers = map[string]func(ctx context.Context, h *serviceHandler, resourceID uuid.UUID) (interface{}, error){
	"webchat_widget": func(ctx context.Context, h *serviceHandler, resourceID uuid.UUID) (interface{}, error) {
		w, err := h.reqHandler.WebchatV1WidgetGet(ctx, resourceID)
		if err != nil {
			return nil, err
		}
		tc := w.ConvertWebhookMessage().ThemeConfig
		if tc == nil {
			return nil, nil
		}
		return tc, nil
	},
}
```

Inside `AuthBoot()`, after `BootResponse` is otherwise fully built (`d` is
the same `*dmdirect.Direct` record already resolved earlier, `boot.go:49`):

```go
if fetcher, ok := resourceDisplayConfigFetchers[d.ResourceType]; ok {
	data, ferr := fetcher(ctx, h, d.ResourceID)
	if ferr != nil {
		log.Infof("Could not fetch public display config. resource_type: %s, err: %v", d.ResourceType, ferr)
	} else if data != nil {
		// Only allocate + populate the envelope when there is
		// something real to put in it -- an empty non-nil map would
		// still serialize as "resource_data": {}, which is different
		// from (and worse than) omitting the key entirely. See §9.1's
		// typed-nil note for why this check must be `data != nil`,
		// not relying on omitempty to clean up after insertion.
		res.ResourceData = map[string]interface{}{"public_display_config": data}
	}
}
```

### 9.3 Updated frontend consumption (supersedes §4.1's boot-response read)

```js
// client.js — _doStart(), right after the /auth/boot call resolves
const boot = await this._fetchJson(this._apiUrl('/auth/boot'), { ... })
this.token = boot?.token
this.customerId = boot?.customer_id
this.resourceId = boot?.resource_id
const displayConfig = boot?.resource_data?.public_display_config
if (displayConfig) {
  this.onBootResourceData(displayConfig)
}
```

Only this one read site changes (`boot?.public_display_config` →
`boot?.resource_data?.public_display_config`); `onBootResourceData`'s own
signature, the `_destroyed` teardown guard, the re-entrancy analysis
(§4.1), and the accepted known limitations (§4.2, including the
`sessionId`-reset rejection) are all UNCHANGED by this revision — they
operate on the value AFTER extraction, which is unaffected by where in the
response shape that value was nested.

### 9.4 Updated OpenAPI schema (supersedes §5's snippet)

```yaml
# bin-openapi-manager/openapi/openapi.yaml — AuthBootResponse schema addition
resource_data:
  type: object
  nullable: true
  description: >
    Resource-type-scoped envelope for additional, publicly-safe data about
    the boot-scoped resource. Each entry is a self-documenting named key;
    currently only "public_display_config" is populated (for
    resource_type "webchat_widget", carrying the widget's
    WebchatManagerWidgetThemeConfig shape). The envelope key itself, and
    any entry inside it, is OMITTED (not present) when there is nothing
    to report -- never present as an empty object.
  properties:
    public_display_config:
      oneOf:
        - $ref: '#/components/schemas/WebchatManagerWidgetThemeConfig'
```

The `oneOf`-for-polymorphism rule (`bin-openapi-manager/CLAUDE.md` rule
#1, cited in §5) is applied to the envelope's `public_display_config`
entry specifically, same as before — the envelope itself (`resource_data`)
is deliberately left as a plain `type: object` with named `properties`
(not `oneOf`), since it is not itself type-discriminated by
`resource_type` — its entries are.

### 9.5 Scope table and test-plan deltas from this revision

- §6's `boot.go` row: implementation now touches `BootResponse.ResourceData`
  (`map[string]interface{}`) instead of `BootResponse.PublicDisplayConfig`
  (`interface{}`) — same file, same PR, no new files.
- §6's `openapi.yaml` row: schema addition is `resource_data` (object with
  named `properties`) instead of a bare `public_display_config` field —
  same file, same PR.
- §8's backend test (a)/(b)/(c) assertions now check
  `res.ResourceData["public_display_config"]` instead of
  `res.PublicDisplayConfig`; §8 now also has an explicit (d) test case for
  the fetcher-succeeds-but-widget-has-no-theme scenario (the exact case
  that motivated §9.1's typed-nil normalization), confirming
  `resource_data` is fully omitted in that case too, not just the
  no-fetcher-registered (c) and fetch-failure (b) cases.
- §8's frontend tests: the `_doStart()` assertion updates to read
  `boot.resource_data.public_display_config` in its mock response fixture,
  per §9.3.

No other section (§1, §2, §3.1-§3.3, §4.1's teardown/re-entrancy logic,
§4.2-§4.4, §7) requires any change — this revision is confined to the
wire-format nesting, not the underlying design decisions.

### 9.6 Revision 1 review disposition

**Round A (2 parallel angles: correctness/feasibility, completeness/
staleness audit):**

- **Correctness/feasibility angle: APPROVE.** Independently compiled §9.1's
  struct and §9.2's fetcher/wiring snippets standalone — valid Go, matches
  the real `serviceHandler`/`AuthBoot()` code. Empirically verified §9.1's
  typed-nil/`omitempty` claim by running `encoding/json.Marshal` on a map
  containing one nil-valued key — confirmed the key is NOT dropped, exactly
  as §9.1 states. Traced §9.2's `data != nil` guard and confirmed
  `res.ResourceData` is never allocated at all when the fetcher returns
  `(nil, nil)` (widget exists, no theme) — the envelope key is correctly
  omitted, not emitted as `{}` or `{"public_display_config": null}`.
  Confirmed §9.3's optional-chaining handles every combination without
  throwing. Confirmed §9.4's `oneOf` nested inside `properties.
  public_display_config` is valid OpenAPI 3.0 and correctly applies the
  `oneOf`-for-polymorphism rule to the discriminated entry, not the
  envelope itself.
- **Completeness/staleness audit angle: REQUEST CHANGES.** §9 itself was
  correct, but was applied as an appended section rather than in-place
  edits, leaving stale/uncorrected references in §3.1-§3.5, §4.1, §5, §6,
  and §8 that a top-to-bottom reader would hit BEFORE reaching §9's
  corrections. Specific findings, all fixed in this revision: (1) §3.1's
  own cross-reference pointed to the wrong section number ("§10" instead
  of "§9"); (2) the `ThemeConfig` struct's required SECURITY comment
  sample (§3.2, destined to ship verbatim into production Go source) still
  said "public_display_config field" instead of "resource_data envelope's
  public_display_config key"; (3) §3.3's failure-semantics prose said "set
  `public_display_config = nil`", which doesn't match §9.2's actual
  omit-the-key (not assign-nil) mechanism; (4) §3.4, §3.5, §4.1, §5, §6,
  and §8 all presented the pre-Revision-1 flat-field shape as current spec
  with zero inline forward marker. **Fixed** by adding explicit "[Superseded
  by Revision 1, §9.N — ...]" markers at the exact point of staleness in
  every one of these sections, correcting the §3.1 mispointer, correcting
  the SECURITY comment sample text, and reconciling §3.3's wording with
  §9.2's real mechanism — so a reader proceeding top-to-bottom is now
  redirected to §9 at each point where the pre-revision text would
  otherwise mislead, rather than only finding out via §9.5's appended delta
  list after already reading the stale version.

All required changes from Round A are incorporated above. Since staleness
markers are now threaded through every affected section (not just this
disposition), a fresh top-to-bottom read should no longer produce the
confusion the completeness angle found — proceeding to Round B to confirm
this and obtain the required 2nd consecutive APPROVE.

**Round B (2 parallel angles: fresh top-to-bottom marker-mechanism
verification, implementer-lens stale-content-risk check):**

- **Fresh top-to-bottom marker-mechanism verification: APPROVE.**
  Independently checked every superseded marker against its target §9
  subsection — all correct, no stray wrong-number references anywhere
  (the old §3.1 mispointer is confirmed fixed). Confirmed the `ThemeConfig`
  SECURITY comment sample now correctly says "envelope's ... key," not
  "field." Confirmed §3.3's wording now correctly describes "omit the key"
  matching §9.2's real mechanism. Confirmed a top-to-bottom reader now
  encounters a clear forward signal at every point of staleness, not just
  retroactively via §9.5. No new contradictions or fabricated citations
  found in a fresh full-document adversarial pass.
- **Implementer-lens stale-content-risk check: APPROVE.** Confirmed every
  superseded marker sits BEFORE the stale code block it warns about, at
  the start of the relevant paragraph — none buried mid-paragraph where a
  skimming implementer could miss it and copy-paste the wrong (superseded)
  snippet. Confirmed §9.5's delta list matches the now-marked §6/§8
  correctly, with no double-counting or missing deltas. One optional,
  non-blocking observation: §8's original omitempty test description
  didn't distinguish "envelope nil due to fetch failure" from "no fetcher
  registered at all" as two separate code paths worth separate test cases
  — **incorporated as a small improvement** (§8, updated above) even though
  the reviewer explicitly noted it doesn't block this round's closure.

**Round A and Round B are not both clean APPROVE from every angle (Round A's
completeness angle was REQUEST CHANGES) — Round B is the first fully clean
round. Proceeding to Round C to obtain the second consecutive clean APPROVE
required to re-close the loop after this revision.**

**Round C (2 parallel angles: final-signoff fresh trace-through of §8/§9.2,
holistic final judgment):**

- **Final-signoff fresh trace-through: REQUEST CHANGES.** Independently
  traced both of §8's (b)/(c) code paths against the real §9.2 wiring —
  both confirmed technically accurate. But found a genuine gap: §8's test
  list had NO case for the third, distinct code path that actually
  motivated the entire typed-nil fix in the first place — fetcher runs
  successfully, `WebchatV1WidgetGet` succeeds, but
  `ConvertWebhookMessage().ThemeConfig == nil` (widget exists, no custom
  theme configured), so the fetcher returns `(nil, nil)` and `data != nil`
  evaluates false. §9.5's own forward-reference had promised this case
  would be folded into (c), but the actual §8(c) text scoped itself only
  to "no registered fetcher," never mentioning this case. **Fixed** by
  adding an explicit (d) test case for exactly this scenario, and
  correcting §9.5's forward-reference to match. 5 fresh code-fact
  citations independently re-verified, all accurate — no other issues
  found.
- **Holistic final judgment: APPROVE.** Confirmed full-document coherence
  end to end, no remaining contradictions. Independently assessed the
  `resource_data` envelope decision itself (not just its documentation) —
  no hidden cost found versus the flat-field version; the `map[string]
  interface{}` wrapping doesn't lose meaningful type safety since the
  inner value was already an untyped `interface{}` before this revision
  too. Confirmed an implementer could extract the complete, correct
  technical decision (what `BootResponse` looks like, how `AuthBoot()`
  populates it) from §9 alone, without needing to read the now-superseded
  §3-§8 content or §10's history, in well under 2 minutes.

Round C's trace-through angle found one real gap (now fixed) — not a clean
APPROVE from both angles. Proceeding to Round D to obtain the second
consecutive clean APPROVE required to re-close the loop.

## 10. Round review disposition

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

### Round 2 (3 parallel angles: fresh-full re-verification, implementation-readiness, fresh adversarial pass)

- **Fresh-full re-verification angle: REQUEST CHANGES.** All Round 1 fixes
  independently re-verified as landed correctly (citations, ThemeConfig
  pointer-sharing claim, §4.1/§4.2 consistency, CI dependency accuracy, §8
  coverage). One new finding: the `_destroyed` guard referenced in §4.1's
  code snippet was never actually wired to a write site anywhere —
  `destroy()` didn't set it, making the guard permanently dead code. Fixed
  by adding explicit constructor-init and `destroy()`-sets-it instructions.
- **Implementation-readiness angle: REQUEST CHANGES.** Found three real
  ambiguities that would force an implementer to guess: (1) the
  `interface{}` + `omitempty` typed-nil trap — a nil `*ThemeConfig` boxed
  into the interface is a non-nil interface value, so `omitempty` would not
  drop it, contradicting the doc's own "nil/omitted" framing for the common
  "widget with no custom theme" case — fixed by adding an explicit
  typed-nil-normalization step to the fetcher (`if tc == nil { return nil,
  nil }`) and correcting the OpenAPI description's null/omitted framing;
  (2) the `_destroyed` flag gap (same finding as the re-verification angle
  above, confirmed independently); (3) `Warn`-vs-`Info` log-level ambiguity
  with no precedent in the file for `Warnf` — fixed by pinning `Infof` and
  matching the file's terse message-format convention. Confirmed
  independently: the fetcher-dispatch insertion point in `boot.go` and the
  `onBootResourceData` insertion point in `widget.js`'s constructor were
  already concrete and copy-pasteable — no changes needed there.
- **Fresh adversarial pass (new problems beyond Round 1's scope):
  REQUEST CHANGES.** Found two real completeness gaps: (1) §3.2's safety
  rationale was prose-only with no in-code enforcement artifact — fixed by
  specifying a required `SECURITY:` warning comment on the `ThemeConfig`
  struct definition in `widget.go`, to be added during the implementation
  phase (per this skill's design-to-implementation-handoff convention,
  code changes are not mixed into a documentation-only design-doc PR); (2)
  the design doc never addressed how an already-embedded customer
  `<script>` tag picks up the new widget-runtime JS bundle — fixed by
  adding §4.4, confirming via `nginx.conf`'s 5-minute cache and
  `package.json`'s `prebuild` step that no customer action is required.
  This angle also independently confirmed the `omitempty` behavior for
  `ai`/`ai_team` (no registered fetcher) causes zero behavior change for
  those existing consumers — no bug found there, contrary to what the
  review brief hinted might be found.

All required changes from Round 2 are incorporated above. Per this skill's
own design-to-implementation-handoff convention (design-doc PRs stay pure
prose so their diff reviews cleanly in isolation from code), the
`ThemeConfig` struct comment change is fully specified in §3.2 above for
the implementation phase to apply directly to `widget.go`, rather than
mixed into this documentation-only PR's diff. Proceeding to Round 3.

### Round 3 (3 parallel angles: fresh-full re-verification, holistic readability/product-correctness, test-suite feasibility)

- **Fresh-full re-verification angle: REQUEST CHANGES (minor).** All
  substantive Round 2 fixes independently re-verified as landed correctly
  in the doc body (typed-nil normalization, `_destroyed` flag consistency
  with real `widget.js`, log-level match to `boot.go`'s real `Infof`
  convention, §4.4's citation accuracy). One self-contradiction found: this
  §9 section's own Round 2 disposition previously said the `ThemeConfig`
  SECURITY comment was "landed in this revision," directly contradicting
  §3.2's correct "implementation-phase" framing — fixed by rewording this
  disposition (see the corrected paragraph above) to match.
- **Holistic readability/product-correctness angle: REQUEST CHANGES
  (minor).** Found: (1) several inline parentheticals citing "found in
  round N review" read as process narration rather than design rationale —
  fixed by stripping them from §3.2/§3.4/§4.1 (the technical content is
  unchanged, only the review-attribution framing was removed; this §9
  section remains the sole place review history is recorded); (2) §4.2 had
  treated "already-open session" and "closed-then-reopened, same page
  load" as equivalent-severity limitations, when the reopen case is a
  materially more common visitor interaction and was worth actively
  mitigating rather than silently accepting — fixed by adopting a
  `close()`-resets-`sessionId` mitigation (§4.2, §4.1's re-entrancy
  paragraph updated to match, §6 and §8 updated for the new `widget.js`
  change and test); (3) §7 updated to precisely scope the remaining
  out-of-scope item (live-push for the already-open case only, now that
  reopen is mitigated).
- **Test-suite feasibility angle: REQUEST CHANGES (minor).** Confirmed
  `bin-api-manager/pkg/servicehandler/boot_test.go` already exists with an
  extendable table-test shape using the exact mocking pattern needed
  (`gomock`, `WebchatV1WidgetGet` already mocked elsewhere in the package),
  and confirmed — contrary to an initial concern — that
  `square-admin/src/webchat-widget-runtime/__tests__/` already has
  extensive, actively-maintained Jest suites (`client.test.js`,
  `widget.test.js`, `render.test.js`) with directly reusable patterns for
  every proposed frontend test, including near-verbatim precedent for the
  destroy-no-op and close-before-settle-reopen cases. One real gap: §8(d)'s
  originally-proposed "reflection-based field-diff regression test" had no
  precedent anywhere in this codebase and would require inventing a new
  test pattern — fixed by replacing it with a code-review-plus-struct-
  comment enforcement approach instead (§8, updated above), consistent
  with how §3.2's rule is actually enforced.

All required changes from Round 3 are incorporated above. This design doc
has now been through 3 independent adversarial review rounds; Round 3
found only minor/polish issues with no remaining structural, security, or
correctness concerns. Ready for implementation-phase handoff per
`design-to-implementation-handoff.md`.

### Round 4 (2 parallel angles: re-verification + fresh look, whole-doc completeness/coherence gate — first of 2 consecutive APPROVE rounds required to close)

- **Re-verification + fresh-look angle: REQUEST CHANGES.** All Round 3
  fixes independently re-verified as landed correctly (§4.1/§4.2
  consistency, §9's self-contradiction genuinely fixed, round-attribution
  parentheticals confirmed stripped from §3.2/§3.4/§4.1). One new, real
  gap found: §4.2's `close()` mitigation ("resets `this.client.sessionId =
  null`") did not pin the ORDER relative to `close()`'s existing
  `this.client.end()` call. Since `close()` calls `end()` without
  awaiting it, and `end()` has its own synchronous early-return guard (`if
  (!this.sessionId) return`) gating the best-effort `POST
  /webchat_sessions/{id}/end` call, resetting `sessionId` to `null` BEFORE
  invoking `end()` would make that guard fire and silently skip the
  session-end RPC on every single close, for every visitor — a real
  regression, arguably worse than the staleness bug this mitigation was
  meant to fix. Fixed by explicitly pinning the reset to AFTER the `end()`
  call, with an inline code snippet and an explanation of why order
  matters, so an implementer cannot innocently pick the unsafe ordering.
- **Whole-doc completeness/coherence angle: APPROVE.** Independently
  re-derived, from scratch, that §1's problem statement is fully solved by
  §3-§5's design (confirmed live against `index.js`, `boot.go`,
  `client.js`, `widget.js` — the fix chain is complete and additive, no
  gaps). Confirmed §6/§7/§8 are mutually consistent with §3/§4 after all
  prior rounds' edits, and all code snippets are syntactically valid and
  consistent with each other (fetcher signature matches its invocation
  site). No structural, security, correctness, or cross-reference issues
  found. Noted the §9-only concentration of review-attribution language is
  intentional per the doc's own Round 3 disposition, not an inconsistency.

One required fix from this round is incorporated above (§4.2's ordering
fix). Since this round did not achieve a clean APPROVE from both angles,
per the 2-consecutive-APPROVE gate this is NOT yet closed — proceeding to
Round 5 to obtain the first of the required 2 consecutive full APPROVEs.

### Round 5 (2 parallel angles: ordering-fix re-verification + full adversarial pass, fresh skeptical full-doc review — restarts the consecutive-APPROVE counter)

- **Ordering-fix re-verification + fresh adversarial pass: APPROVE.**
  Independently re-traced the real `client.js` `end()` function and
  confirmed the Round 4 ordering fix's rationale was technically accurate.
  No new issues found in a fresh pass over the full document.
- **Fresh skeptical full-doc review, independent of prior rounds' framing:
  REQUEST CHANGES.** Found a genuinely serious issue Rounds 1-4 missed:
  the §4.2 `close()`-resets-`sessionId` mitigation (adopted in Round 3,
  ordering-fixed in Round 4) does correctly solve the theme-staleness
  problem, but does so by making `open()`'s existing `!sessionId` guard
  re-fire `start()` — and therefore re-create a webchat Session — on
  every single close/reopen cycle within a page load, not just once per
  visit as before. `bin-webchat-manager/pkg/sessionhandler/create.go:
  62-93` confirms every Session creation checks `Widget.SessionFlowID`
  and, if configured, executes a full Flow (AI Team calls, agent
  notifications, potentially billing-relevant actions) via
  `ConversationV1ConversationCreateAndExecuteFlow`. `POST
  /webchat_sessions` is not covered by the `/auth` group's rate limit, so
  the mitigation reintroduces an uncosted, unbounded flow-execution
  amplification path (a visitor idly toggling the chat bubble spams flow
  executions) in exchange for fixing a purely cosmetic staleness bug — a
  worse trade than the problem it solved. **Fixed by reverting the
  `close()`-resets-`sessionId` mitigation entirely**: §4.2 now explicitly
  documents this as a REJECTED alternative with the full cost rationale,
  restores both limitations (already-open session, closed-then-reopened
  session) as accepted for this ticket's scope, and names a live-push WS
  mechanism as the correct fix for both (since it updates an
  already-running client without creating a new Session or re-triggering
  `SessionFlowID`). §4.1's re-entrancy paragraph, §6's scope table, §7's
  out-of-scope list, and §8's verification plan are all reverted to match
  (no `close()` code change, no reopen-re-fires test — instead a
  no-re-fire-on-reopen test, as originally specified before Round 3).

This is exactly the value of running MULTIPLE independent review rounds
even after a mitigation has already passed two rounds of scrutiny (Round 3
adopted it, Round 4 only checked its ordering) — a genuinely fresh,
skeptical pass in Round 5 caught a real regression risk that neither
Round 3 nor Round 4's narrower briefs were positioned to find. Since this
round was not a clean APPROVE from both angles either, the
consecutive-APPROVE counter resets again — proceeding to Round 6.

### Round 6 (2 parallel angles: Round-5-revert re-verification + fresh pass, independent full-doc adversarial hunt for a new defect class)

- **Round-5-revert re-verification + fresh pass: APPROVE.** Independently
  re-confirmed the `SessionFlowID` trigger and the missing rate-limit
  coverage on `POST /webchat_sessions` are both real (traced
  `create.go`'s full function and `main.go`'s route registration). §4.2's
  revert is complete and consistent — no stray references anywhere to the
  removed `close()` mitigation. §4.1/§6/§7/§8 all confirmed consistent
  with the reverted §4.2. No new issues found in a fresh full-document
  pass from this angle.
- **Independent full-doc adversarial hunt, targeting undiscovered defect
  classes: REQUEST CHANGES.** Explicitly checked three fresh angles beyond
  anything covered in Rounds 1-5: (a) blast radius of `AuthBoot`/
  `BootResponse` — confirmed clean, exactly one call site and one
  construction site, no hidden fan-out to other code paths; (b)
  concurrency safety of the new fetcher-dispatch path — confirmed clean,
  `serviceHandler` is an immutable singleton and `resourceDisplayConfigFetchers`
  is a read-only map at request time; (c) whether §5's proposed OpenAPI
  schema for `public_display_config` is correctly typed — found a real
  gap: the snippet used a bare untyped `object` with a hand-written
  example, when `WebchatManagerWidgetThemeConfig` (`openapi.yaml:
  2340-2374`) already exists as a first-class named schema for exactly
  this shape and is already `$ref`'d elsewhere in the same file for the
  `Widget` resource's own `theme_config` field. The untyped blob both
  discarded real field-level typing for external SDK/docs consumers and
  conflicted with `bin-openapi-manager/CLAUDE.md`'s mandatory `oneOf`-for-
  polymorphism rule (this field is explicitly type-discriminated by
  `resource_type`). **Fixed** by changing §5's snippet to `oneOf: [$ref:
  WebchatManagerWidgetThemeConfig]` — the `oneOf` wrapper (rather than a
  bare `$ref`) is deliberately chosen to stay extensible for when a second
  resource type's fetcher is registered in the future, without a breaking
  spec change at that point.

Not a clean APPROVE from both angles — proceeding to Round 7.

### Round 7 (2 parallel angles: OpenAPI schema fix re-verification, holistic final-sign-off gate)

- **OpenAPI schema fix re-verification: APPROVE.** Independently confirmed
  `WebchatManagerWidgetThemeConfig` (`openapi.yaml:2340-2424`) has a
  correctly-typed OpenAPI property for all 14 fields on the real
  `ThemeConfig` Go struct (`widget.go:83-115`), with no drift in either
  direction. Confirmed `oneOf` with a single `$ref` member is syntactically
  valid OpenAPI 3.0 and the extensibility rationale is sound. No new issues
  found in a fresh full-document read.
- **Holistic final-sign-off gate, whole-document judgment: APPROVE.** No
  remaining internal contradictions found across any two sections after
  spot-checking 8 of §9's "required fix" claims against real code/doc
  state. Independently assessed the core design decision: extending
  `POST /auth/boot` with a best-effort, resource-type-scoped
  `public_display_config` field is the right solution — 6 rounds of
  patching found genuine implementation-detail defects (typed-nil traps,
  a dead teardown guard, a session-amplification regression, an untyped
  schema) but never invalidated the core architectural choice itself; each
  defect was a fixable specification gap, not evidence the approach is
  unsound. The final state's decision to accept a narrow, honestly
  documented staleness limitation rather than trade it for an uncosted
  billing-relevant amplification path (§4.2) is the correct call, with a
  live-push WS mechanism correctly deferred as a follow-up rather than
  gold-plating this ticket's scope.

This is the FIRST of the 2 required consecutive full APPROVEs. Proceeding
to Round 8 to obtain the second, closing round.

### Round 8 (2 parallel angles: independent fresh full-read APPROVE check, implementer-lens executability check — SECOND of 2 required consecutive full-APPROVE rounds)

- **Independent fresh full-read: APPROVE.** Verified 8 code-fact claims
  spanning §3.1/§3.2/§3.3/§4.1/§4.4/§5 against real files, all accurate.
  Confirmed the document reads as a coherent, complete, implementable
  design end to end, not a patchwork of disconnected fixes, and that §9's
  review history contains no claims contradicting §1-§8's current state.
- **Implementer-lens executability check: APPROVE.** Confirmed via `git
  log` that none of the referenced files
  (`bin-api-manager/pkg/servicehandler/boot.go`,
  `bin-webchat-manager/models/widget/widget.go`,
  `square-admin/src/webchat-widget-runtime/client.js`, `widget.js`) have
  been modified on this branch since it diverged from `main`, so every
  line-number citation accumulated across 8 review rounds is still
  accurate against the current worktree state. Walked through
  implementing every §3/§4 snippet mentally against the real current file
  contents and confirmed an implementer could write the diff today with
  zero clarifying questions — every place a prior round flagged ambiguity
  (typed-nil handling, log level, the `_destroyed` dead-code risk, the
  `close()`/`end()` ordering footgun) now has an explicit, unambiguous
  resolution with a literal code snippet in the doc.

**Round 7 and Round 8 are both clean APPROVEs from every angle, satisfying
the 2-consecutive-APPROVE gate. This design doc's review loop is CLOSED.**
Ready for implementation-phase handoff per
`design-to-implementation-handoff.md` — a fresh worktree/branch should be
created for the actual Go/JS implementation, separate from this design-doc
branch, per that skill's convention.
