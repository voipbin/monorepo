# Webchat Widget: Connecting/Typing Indicators + Shape/Font Options: Design

Status: APPROVED (3 rounds, rounds 2-3 consecutive 3/3 + 1/1 APPROVE — see §9-§10). Ready for implementation.

## 1. Scope (locked with CEO/CTO)

Fourth extension to `Widget.ThemeConfig` (currently 9 fields:
`primary_color`, `secondary_color`, `header_background_color`,
`header_text_color`, `logo_url`, `position`, `theme_mode`,
`header_title`, `header_subtitle`, added in
`2026-07-18-webchat-widget-appearance-expansion-design.md`). Adds 5 new
fields, taking the total to 14:

- `connecting_indicator_enabled` (bool, default `true`) — show a system
  message in the panel while the visitor's session is being created
  (between widget open and `POST /webchat_sessions` completing).
- `connecting_indicator_text` (string, default `"Connecting…"`, max 100
  chars) — the text shown while connecting.
- `typing_indicator_enabled` (bool, default `true`) — show the existing
  three-dot "waiting for response" animation after the visitor sends a
  message. Setting this to `false` suppresses it entirely.
- `border_radius` (enum: `sharp` | `rounded` | `pill`, default `rounded`)
  — corner rounding applied to the bubble, panel, message bubbles, and
  input field as a coordinated set.
- `font_size` (enum: `compact` | `default` | `large`, default `default`)
  — base font-size scale applied to header text and message text.

Explicitly OUT of scope (decided this round):
- **Typing indicator text/label customization.** pchero's explicit call:
  keep Option A (three bouncing dots only, no text label) as the ONLY
  supported typing-indicator style. `typing_indicator_enabled` is a pure
  on/off switch over the EXISTING dot animation — no new text field, no
  "Agent is typing…"/"Thinking…" style option. Do not add a
  `typing_indicator_text` field in this pass.
- **Font family / custom webfont URL loading.** Already excluded in the
  prior appearance-expansion design (§2 scope-discipline rule,
  `widget-theming-embed-snippet.md` §11) and re-confirmed this round —
  pchero: "폰트 패밀리 및 custom url 은 빼자. 위험해." `font_size` (a bounded
  enum) is a different, low-risk axis from `font_family` (arbitrary URL
  fetch, same risk class as `logo_url`/`bubble_icon`) and stays in scope.
- **Bubble icon / avatar_url custom icon URLs.** Already excluded
  (`widget-theming-embed-snippet.md` §11); not re-opened this round.
- **Live re-theming, animation toggles beyond typing on/off,
  unread-badge color.** Deferred as future polish, unchanged from the
  prior design's §1 exclusion list.

This mirrors the same scope-discipline pattern as the prior round: small,
bounded (enum or bool + short string), no new URL-fetch surface, no new
arbitrary-CSS door.

## 2. Current state

`bin-webchat-manager/models/widget/widget.go`:

```go
type ThemeConfig struct {
	PrimaryColor          string         `json:"primary_color,omitempty"`
	SecondaryColor        string         `json:"secondary_color,omitempty"`
	HeaderBackgroundColor string         `json:"header_background_color,omitempty"`
	HeaderTextColor       string         `json:"header_text_color,omitempty"`
	LogoURL               string         `json:"logo_url,omitempty"`
	Position              WidgetPosition `json:"position,omitempty"`
	ThemeMode             ThemeMode      `json:"theme_mode,omitempty"`
	HeaderTitle           string         `json:"header_title,omitempty"`
	HeaderSubtitle        string         `json:"header_subtitle,omitempty"`
}
```

Frontend (`monorepo-javascript/square-admin/src/webchat-widget-runtime/`):
- `widget.js` already implements a three-dot typing indicator
  (`appendTypingEl()` / `_typingEl` / `_clearTypingIndicator()`),
  unconditionally shown after every outbound send, cleared on reply,
  reconnect, or a 20s timeout. Added in the webchat WS reliability
  design doc (Phase 4, PR #362, 2026-07-18). There is currently NO
  on/off switch — it always fires.
- `widget.js`'s `open()` calls `await this.client.start()` (session
  bootstrap) with **no visual feedback at all** during the await. If
  session creation is slow (network latency, backend load), the panel
  sits empty with no indication anything is happening. This is a gap,
  not a regression — no prior design doc covers this moment.
- `render.js`'s `applyWidgetTheme()` is the single pure-DOM-styling
  function consuming `themeConfig` and setting inline styles / CSS
  classes on the widget's DOM elements. Border-radius and font-size are
  new axes not yet touched by this function.

## 3. New field behavior

### 3.1 `connecting_indicator_enabled` / `connecting_indicator_text`

- Trigger: `WebchatWidget.open()`, for the duration between the panel
  becoming visible and `client.start()` resolving (success or error).
- **Scope: fires until the session is created; safe across replays.**
  `client.start()` is already gated on `if (!this.client.sessionId)`
  (existing code, `widget.js:507`), and `client.js`'s `sessionId` is
  never cleared by `end()`/`close()` once a session is successfully
  created — confirmed by grep, the ONLY assignment to `sessionId` in
  the runtime is its initial `null` at construction. A visitor who
  closes and reopens the panel AFTER a successful `client.start()`
  therefore never re-invokes it, so the connecting indicator does not
  re-fire on a normal reopen. This is NOT an absolute "fires exactly
  once" guarantee, however: if the visitor closes/reopens WHILE the
  first `client.start()` attempt is still pending, or if that attempt
  REJECTS (per `client.js`'s cached-promise-clears-on-reject behavior,
  a rejected `start()` allows a later call to issue a genuinely new
  session-creation POST), the indicator can legitimately show again on
  a retry. This is intentional and safe — every fire/teardown cycle is
  independently correct per the element-lifecycle spec below, so a
  possible re-fire on retry is not a bug, just not a hard "once ever"
  bound. This design does NOT change the underlying session-continuity
  lifecycle (out of scope — resetting `sessionId` on `end()` for
  UNRELATED reasons, e.g. deliberately starting a fresh server-side
  session on every reopen, is a separate behavior change with its own
  blast radius on session-continuity semantics this pass does not
  intend to revisit).
- Rendering: reuses the EXISTING system-message pattern already used
  for `"Reconnecting…"` (`appendMessageEl(doc, messages, { text,
  direction: 'outbound' })`) — no new DOM/CSS component, just a new
  call site. This keeps the implementation mechanical (see
  `widget-theming-embed-snippet.md` §9's DOM-API-not-innerHTML rule:
  `appendMessageEl` already uses `textContent`-only assignment, so the
  customer-supplied `connecting_indicator_text` string is safe to reuse
  as-is with no new sanitization work).
- **Element lifecycle (element created inside `open()`, tracked on the
  instance, torn down defensively):** the element reference MUST be
  stored on a new instance field `this._connectingEl` (mirroring the
  existing `_typingEl`/`_reconnectingEl` pattern), not a local
  variable inside `open()`. Two teardown paths are required, not just
  the happy-path "removed when `client.start()` settles":
  1. **Normal settle path:** when `client.start()` resolves (success
     or error), remove `this._connectingEl` and null it out — but ONLY
     if `this.isOpen` is still `true` at that point. If the visitor
     called `close()` while `client.start()` was still pending,
     `close()` (see below) has already removed and nulled the element;
     the late-resolving settle handler must be a no-op in that case
     (check `this._connectingEl` for `null` before touching it, same
     idempotency guard `_clearTypingIndicator()` already uses).
  2. **Early-close path:** `close()` and `destroy()` must both clear
     `this._connectingEl` (remove from DOM if present, null the
     reference) unconditionally, the same way they already clear
     `_typingEl` via `_clearTypingIndicator()`. This guarantees a
     visitor who closes the panel mid-connect never leaves a stale
     "Connecting…" bubble sitting in a closed panel, and a subsequent
     `open()` call (on the rare path where `sessionId` was somehow
     cleared/a new instance is created) never operates on a dangling
     reference from a previous cycle.
- `connecting_indicator_enabled: false` skips rendering the element
  entirely — no timing/animation change, purely a presence toggle.
- Default text `"Connecting…"` matches the existing `"Reconnecting…"`
  wording convention (ellipsis character, present participle).
- Validation: `connecting_indicator_text` max 100 chars, matching
  `header_title`'s existing length cap precedent.
- **Multi-tab note:** each browser tab runs a fully independent
  `WebchatWidget` instance with its own `client`/`sessionId`/DOM — two
  tabs each showing their own "Connecting…" independently is expected
  and not a race; VoIPBin webchat sessions are tab-scoped, not
  visitor-scoped (pre-existing characteristic, unrelated to this
  design).

### 3.2 `typing_indicator_enabled`

- Pure on/off gate around the EXISTING `appendTypingEl()` call site in
  `_handleSend()`. No change to the dot animation itself, no new text.
- `typing_indicator_enabled: false`: `_handleSend()` skips the
  `appendTypingEl()`/`_typingTimeoutId` block entirely. All other
  `_handleSend()` behavior (optimistic append, `sendMessage()` call)
  unchanged.
- Default `true` preserves current (pre-this-design) behavior exactly
  — this field is additive/opt-out, not a behavior change for existing
  widgets that don't set it.
- **Accepted limitation, not implemented this pass:** `themeConfig` is
  read once at `WebchatWidget` construction time and there is no
  live-theme-refetch mechanism in the runtime, so
  `typing_indicator_enabled` cannot actually flip mid-session under
  the current architecture (matches §1's "live re-theming... deferred"
  exclusion) — a config edit by the admin only takes effect on the
  visitor's NEXT fresh page load, not the current session. Documented
  here explicitly so this is not mistaken for an oversight: no code is
  needed to handle a retroactive disable clearing an already-visible
  indicator, because that scenario cannot occur given the current
  theme-loading model.

### 3.3 `border_radius`

- Enum, three values. Applied as a coordinated multiplier/preset over
  the panel's existing corner-radius CSS (currently hardcoded px values
  in `widget.js`'s injected `<style>` block — bubble, panel, message
  bubbles, input field, send button).
- `sharp` = 0 (or near-0, e.g. 2px to avoid a jarring pixel-perfect
  square on the bubble), `rounded` = current hardcoded values
  (unchanged default), `pill` = fully rounded (bubble/send-button
  `border-radius: 50%`, panel/input capsule-shaped where geometrically
  sane — the panel itself stays a large rounded rect, not literally
  pill-shaped, since a chat PANEL cannot be a pill without clipping
  content; "pill" applies fully to circular elements (bubble, send
  button) and as max-rounded to rectangular ones (panel, input, message
  bubbles)).
- Implementation: `render.js` gets a `BORDER_RADIUS_PRESETS` map (enum
  value -> a small object of per-element px values), consumed by
  `applyWidgetTheme()` the same way `DEFAULT_PRIMARY_COLOR` is today.
  **Concrete CSS strategy (resolves the "geometrically sane" hand-wave
  above):** use a large fixed px value (`border-radius: 999px`) for the
  `pill` preset on ALL elements, not percent-based math. CSS
  `border-radius` is well-defined to clamp automatically at 50% of an
  element's shorter dimension — a 999px radius on a circular
  bubble/send-button renders as a perfect circle, and the same 999px
  value on a rectangular panel/input/message-bubble renders as a fully
  rounded (stadium-shaped) rectangle, with no clipping or unpredictable
  overflow at any panel size. This is the same technique already
  common for "pill button" CSS elsewhere; no per-element special-casing
  or percent calculation is needed. `sharp` = `2px` (all elements,
  avoids a literally-0px jarring corner on the floating bubble while
  reading as "sharp" against the `rounded` default), `rounded` =
  current hardcoded per-element values (unchanged default).

### 3.4 `font_size`

- Enum, three values, applied as a CSS custom property or direct
  inline font-size on the header title/subtitle and message text
  elements. `compact` = -2px from current hardcoded values, `default`
  = current hardcoded values (unchanged), `large` = +2px.
- Rationale for a coarse 3-step enum over a free numeric field: matches
  the existing `theme_mode`/`position`/`border_radius` enum precedent
  (bounded, predictable, no layout-breaking edge values a customer
  could input, e.g. `font_size: 200px` blowing out the panel).

## 4. API / schema changes

`bin-webchat-manager/models/widget/widget.go`:

```go
type ThemeConfig struct {
	PrimaryColor          string         `json:"primary_color,omitempty"`
	SecondaryColor        string         `json:"secondary_color,omitempty"`
	HeaderBackgroundColor string         `json:"header_background_color,omitempty"`
	HeaderTextColor       string         `json:"header_text_color,omitempty"`
	LogoURL               string         `json:"logo_url,omitempty"`
	Position              WidgetPosition `json:"position,omitempty"`
	ThemeMode             ThemeMode      `json:"theme_mode,omitempty"`
	HeaderTitle           string         `json:"header_title,omitempty"`
	HeaderSubtitle        string         `json:"header_subtitle,omitempty"`

	// NEW
	ConnectingIndicatorEnabled *bool        `json:"connecting_indicator_enabled,omitempty"`
	ConnectingIndicatorText    string       `json:"connecting_indicator_text,omitempty"`
	TypingIndicatorEnabled     *bool        `json:"typing_indicator_enabled,omitempty"`
	BorderRadius               BorderRadius `json:"border_radius,omitempty"`
	FontSize                   FontSize     `json:"font_size,omitempty"`
}

type BorderRadius string

const (
	BorderRadiusSharp   BorderRadius = "sharp"
	BorderRadiusRounded BorderRadius = "rounded" // default
	BorderRadiusPill    BorderRadius = "pill"
)

type FontSize string

const (
	FontSizeCompact FontSize = "compact"
	FontSizeDefault FontSize = "default" // default
	FontSizeLarge   FontSize = "large"
)
```

**`*bool` pointer, not plain `bool`, for the two `_enabled` fields.**
This is the one field-type decision in this pass that departs from the
prior round's all-string/enum shape, and it needs explicit reasoning:
a plain `bool` cannot distinguish "customer explicitly set `false`"
from "customer never touched this field" (both marshal/unmarshal as
the zero value). Since the DEFAULT for both indicators is `true`
(preserve existing always-on behavior for widgets that predate this
field), a plain `bool` would make "field omitted" indistinguishable
from "customer explicitly disabled it" only if the zero value were the
default — but here the default is `true`, so omitted-vs-false MUST be
distinguishable, or every existing widget's un-set field would
round-trip as `false` (disabled) the moment it passes through any
code path that re-serializes the whole `ThemeConfig` (e.g. an update
request that reads-modifies-writes the struct). `*bool` with
`omitempty` correctly serializes "unset" as an absent JSON key (falls
back to `true` at render time) versus an explicit `false` key.

Widget frontend consumption resolves: `enabled == nil || *enabled ==
true` -> show; `enabled != nil && *enabled == false` -> hide.

No Alembic migration needed — `theme_config` remains a single opaque
JSON column, same precedent as the prior three ThemeConfig expansion
rounds.

**Forward-compat / deploy-order note:** this change is purely additive
JSON (5 new optional keys on an existing opaque `theme_config` column,
no renames/removals). Matching the deploy-order precedent from the
prior three ThemeConfig rounds, either PR (backend schema/validation
in `monorepo`, or frontend consumption/appearance-form in
`monorepo-javascript`) is safe to land first: an old frontend talking
to a new backend simply never sends the 5 new keys (backend defaults
apply); a new frontend talking to an old backend has its new keys
silently accepted-and-ignored by the backend's existing "unknown JSON
keys in an opaque column" tolerance (no strict-schema rejection at
that layer). No coordinated/simultaneous-deploy requirement.

`bin-openapi-manager` + `bin-api-manager`: mirror the 5 new fields in
the `ThemeConfig` schema/conversion code, same mechanical pattern as
the prior round (schema field + Go struct field + validation, no new
endpoint). `bin-api-manager` handler-boundary validation:
- `connecting_indicator_text`: max 100 chars (matches `header_title`).
- `border_radius`, `font_size`: enum membership check, reject unknown
  values (matches `theme_mode`/`position` precedent).
- `connecting_indicator_enabled`, `typing_indicator_enabled`: standard
  bool JSON validation (no custom check needed).

`webchat_struct_widget.rst`: add the 5 new fields to the `theme_config`
struct table and JSON example, matching the existing per-field
documentation format (name, type, default, behavior).

## 5. Frontend changes (monorepo-javascript)

`square-admin/src/webchat-widget-runtime/widget.js`:
- `open()`: wrap the `client.start()` await with connecting-indicator
  show/hide, gated on `this.themeConfig?.connecting_indicator_enabled
  !== false` (defaults to shown).
- `_handleSend()`: gate the existing `appendTypingEl()` block on
  `this.themeConfig?.typing_indicator_enabled !== false`.

`square-admin/src/webchat-widget-runtime/render.js`:
- `applyWidgetTheme()`: add `BORDER_RADIUS_PRESETS` and `FONT_SIZE_PRESETS`
  maps, apply to the relevant DOM elements alongside the existing
  color/theme-mode logic. Same function, no new exported entry point.

`square-admin/src/views/webchat_widgets/{create.js,detail.js}`:
- Appearance card gains: two new toggle switches (Connecting
  indicator, Typing indicator) with the connecting toggle revealing a
  conditional text input when enabled (mirrors existing conditional-
  field patterns already in this form, e.g. how `message_flow_id`
  fields conditionally appear); two new select dropdowns
  (`border_radius`, `font_size`) alongside the existing `position`/
  `theme_mode` selects.
- **Explicit tri-state form contract for the two new bool toggles
  (resolves the round-1 review's highest-severity finding — a naive
  controlled checkbox would silently convert "field never set" into an
  explicit `false` on save, permanently disabling an existing widget's
  default-on indicator the next time an admin edits any unrelated
  field):**
  - **`create.js`** (new widget, no prior `theme_config`): no ambiguity
    exists yet. Default the toggle's displayed state to checked
    (`true`), matching the field's platform default. On submit,
    include the key in `body.theme_config` ONLY if the admin actively
    toggled it to `false` (i.e. away from the default) — mirrors the
    existing pattern in this file where color/text fields are only
    added to `body.theme_config` when non-empty/non-default (see
    current `create.js:112-127`). An untouched toggle contributes
    nothing to the request body, so the field is correctly absent
    server-side (falls back to the Go struct default at read time).
  - **`detail.js`** (editing an existing widget): on load, read the
    RAW fetched value for each field —
    `widget.theme_config?.connecting_indicator_enabled` and
    `...typing_indicator_enabled` — which is one of `true`, `false`,
    or `undefined` (server omits the key entirely when never set, per
    §4's `*bool`/`omitempty` contract). Store this raw value alongside
    a separate `touched` flag per toggle (starts `false`, flips to
    `true` only inside the toggle's own `onChange` handler — never set
    by any other field's edit). Render the toggle's CHECKED state as
    `rawValue !== false` (so `undefined` displays as checked, matching
    the `true` default). On submit, include the key in
    `body.theme_config` if EITHER `touched === true` (admin explicitly
    interacted with this specific toggle this session) OR
    `rawValue !== undefined` (the widget already had an explicit value
    persisted before this edit, so resubmitting the same resolved
    value is a safe no-op, not a new false-write). An untouched toggle
    on a widget that never had the field set contributes nothing to
    the request body — editing `header_title` alone can never silently
    flip `connecting_indicator_enabled` to `false` under this contract.
  - This tri-state handling is a NEW pattern relative to the prior
    round's string/enum fields (those have no ambiguous "unset" vs
    "empty string" collision the way a bool has "unset" vs "false") —
    call this out explicitly during implementation review as the one
    genuinely new form-state class in this pass.
- `WidgetPreview.js`: extend `applyWidgetTheme()`'s consumption of
  `themeConfig` to cover `border_radius`/`font_size` (automatic once
  `render.js` is updated, since `renderPreviewHtml` already forwards
  the whole `themeConfig` object). **Required companion change:** the
  `useMemo` recompute at `WidgetPreview.js:83-99` is keyed off an
  explicit, hand-enumerated dependency array (not the whole
  `themeConfig` object) — `themeConfig?.border_radius` and
  `themeConfig?.font_size` MUST be added to that array, or the live
  preview will not visually update when an admin changes either field
  (silent staleness, same failure class the file's own existing
  comment at lines 44-48 already warns about for `header_title`). The
  "static Connecting… preview state" idea from the initial draft of
  this design is DROPPED from scope — `WidgetPreview.js` already does
  not simulate the reconnecting-indicator state either, and adding a
  static connecting-state mockup would need its own element lifecycle
  spec disproportionate to its preview value; the live preview
  continues to show steady-state styling only, unchanged from today.

## 6. Out-of-scope confirmation checklist (per §1)

- [x] No `typing_indicator_text` / custom typing label field.
- [x] No `font_family` / webfont URL field.
- [x] No `bubble_icon` / `avatar_url` field.
- [x] No live re-theming, no animation-style toggles beyond the single
      typing on/off switch, no unread-badge color field.

## 7. Testing

- `bin-webchat-manager`: table-driven unit tests for `ThemeConfig`
  marshal/unmarshal round-trip including the `*bool` omitted-vs-false
  distinction (the one behavior in this pass most likely to regress
  silently).
- `bin-api-manager`: handler validation tests for the two new enums
  (reject unknown `border_radius`/`font_size` values) and the
  `connecting_indicator_text` length cap, mirroring existing
  `header_title` validation tests.
- `square-admin` (Jest): widget runtime tests for
  connecting-indicator show/hide around `client.start()`, and
  typing-indicator suppression when `typing_indicator_enabled: false`
  — both as new test cases alongside the existing Phase 4 typing-
  indicator test suite (`__tests__/widget.test.js`). **Additional
  required cases per §3.1/§5's revised lifecycle spec:** (a) close()
  called while client.start() is still pending correctly removes
  `_connectingEl` and the LATE resolution of that same start() promise
  is a no-op (no re-append, no error); (b) `create.js`/`detail.js` form
  tests asserting the tri-state contract directly — an untouched
  toggle on an existing widget with `connecting_indicator_enabled`
  previously absent produces a submit body with the key still absent
  (not `false`); a widget that previously had the field explicitly set
  to `false` and is saved again with no toggle interaction still sends
  `false` (not silently dropped back to default-true).

## 8. Round 1 review disposition

Three independent review angles ran against the initial draft
(API/schema-contract: APPROVE; frontend/backend contract-parity +
XSS: REQUEST CHANGES; negative-path/failure-mode: REQUEST CHANGES).
All BLOCKING findings addressed in this revision:

- Connecting-indicator element lifecycle now explicitly tracked on
  `this._connectingEl`, with defined teardown on both the normal
  settle path AND the early-close path (§3.1).
- Connecting-indicator scope narrowed to "fires once per widget
  instance, first `open()` only" with explicit reasoning for why
  `sessionId`-reset (which would enable per-open re-firing) is kept
  out of scope (§3.1).
- Frontend tri-state bool form contract fully specified for both
  `create.js` (default-checked, omit-unless-toggled-away-from-default)
  and `detail.js` (raw-value + touched-flag, omit-unless-touched-or-
  previously-explicit) (§5).
- `WidgetPreview.js`'s `useMemo` dependency-array gap called out as a
  required companion change, not left implicit (§5).
- `border_radius: pill` given a concrete CSS strategy (`999px`,
  browser-clamped) instead of hand-wavy "geometrically sane" language
  (§3.3).
- Typing-indicator mid-session-toggle question resolved as a genuine
  accepted limitation (no live theme refetch exists) rather than an
  unaddressed gap (§3.2).
- Multi-tab behavior confirmed safe (independent per-tab instances,
  no shared state) and documented as one sentence per reviewer
  suggestion (§3.1).
- "Static Connecting… preview state" dropped from `WidgetPreview.js`
  scope — reviewer correctly noted it was underspecified relative to
  its value; the live preview stays steady-state-only, matching
  existing behavior for the reconnecting indicator.

## 9. Round 2 review disposition

Round 2 re-ran all three review angles against the round-1 fixes:
API/schema-contract APPROVE, frontend/backend contract-parity + XSS
APPROVE, negative-path/failure-mode APPROVE — 3/3, all findings this
round were VERIFIED-FIXED or non-blocking. One cosmetic wording fix
applied: §3.1's "eliminated entirely" language was overstated (a retry
after a rejected/pending `client.start()` can legitimately re-show the
indicator; this is safe, not a bug, just not a hard "fires once ever"
bound) — reworded to "fires until the session is created; safe across
replays." No other changes. Remaining implementation nits noted for
the implementer, not requiring further design-doc iteration: prefer
`useRef` over `useState` for the frontend `touched` flag (avoids an
unnecessary re-render on toggle; either is correct given this
codebase's routing/remount pattern).

## 10. Round 3 review disposition

Round 3 rotated to a fresh angle not covered in rounds 1-2:
implementation-readiness / scope-discipline / cross-repo-sequencing.
Verdict: APPROVE. Findings: §4's Go struct diff and §5's frontend
detail were both confirmed sufficiently concrete for an implementer to
proceed without further clarification; no scope-exclusion (§1) was
quietly reintroduced; §6's out-of-scope checklist remains accurate
given all round-1/2 additions. One non-blocking suggestion applied:
§4 gained an explicit forward-compat/deploy-order note (additive-JSON
change, either repo's PR safe to land first, no coordinated-deploy
requirement) mirroring the equivalent note in the prior appearance-
expansion round's design doc.

Design review loop closed: 3 rounds total, rounds 2 and 3 both
unanimous APPROVE (2 consecutive), satisfying the platform's minimum-3-
round / 2-consecutive-APPROVE closing rule. Status: APPROVED, ready for
implementation.
