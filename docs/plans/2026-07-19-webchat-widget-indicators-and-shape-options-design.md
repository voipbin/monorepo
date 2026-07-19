# Webchat Widget: Connecting/Typing Indicators + Shape/Font Options: Design

Status: Draft

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
- Rendering: reuses the EXISTING system-message pattern already used
  for `"Reconnecting…"` (`appendMessageEl(doc, messages, { text,
  direction: 'outbound' })`) — no new DOM/CSS component, just a new
  call site. This keeps the implementation mechanical (see
  `widget-theming-embed-snippet.md` §9's DOM-API-not-innerHTML rule:
  `appendMessageEl` already uses `textContent`-only assignment, so the
  customer-supplied `connecting_indicator_text` string is safe to reuse
  as-is with no new sanitization work).
- Removal: the element is removed the moment `client.start()` settles
  (success -> proceed to render; error -> existing
  `console.error('WebchatWidget: failed to start session:', err)` path
  unchanged, indicator still clears so the visitor is not left staring
  at a stale "Connecting…" forever).
- `connecting_indicator_enabled: false` skips rendering the element
  entirely — no timing/animation change, purely a presence toggle.
- Default text `"Connecting…"` matches the existing `"Reconnecting…"`
  wording convention (ellipsis character, present participle).
- Validation: `connecting_indicator_text` max 100 chars, matching
  `header_title`'s existing length cap precedent.

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
  value -> a small object of per-element px/percent values), consumed
  by `applyWidgetTheme()` the same way `DEFAULT_PRIMARY_COLOR` is today.

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
- `WidgetPreview.js`: extend to reflect border_radius/font_size (and,
  where feasible without over-engineering the preview, a static
  "Connecting…" preview state) — matches the existing "live preview
  reflects uncommitted edits" pattern (design doc G3, prior round).

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
  indicator test suite (`__tests__/widget.test.js`).
