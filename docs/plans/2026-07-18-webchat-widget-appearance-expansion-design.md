# Webchat Widget Appearance Expansion: Design

Status: Draft

## 1. Scope (locked with CEO/CTO)

Expand `Widget.ThemeConfig` from 3 fields (`primary_color`, `logo_url`,
`position`) to 9 fields (adding 6), adding:

- `secondary_color` (hex `#RRGGBB`) — accent/text-contrast color
- `header_background_color` (hex `#RRGGBB`) — widget header bar background
- `header_text_color` (hex `#RRGGBB`) — widget header bar text color
- `theme_mode` (enum: `light` | `dark` | `auto`, default `light`)
- `header_title` (string, default `"Chat with us"`) — widget header text
- `header_subtitle` (string, optional) — header subtext

Explicitly OUT of scope (rejected as high-risk/low-value per CPO
recommendation, confirmed by CEO/CTO):
- Arbitrary `avatar_url` / `bubble_icon` custom-icon URLs (XSS/resource-
  loading risk beyond `logo_url`, which already exists and is precedented)
- Free-form `width`/`height` panel sizing (responsive-layout risk)
- Custom `font_family` URL loading (same class of risk as icon URLs)
- Open/close animation toggles, unread-badge color (low value, pure
  polish, can be a follow-up if requested)

This mirrors the precedent set by `logo_url`: only pre-validated,
enum-or-hex-constrained values are accepted; no raw URLs beyond the
already-shipped `logo_url` field are added in this pass.

## 2. Current state

`bin-webchat-manager/models/widget/widget.go`:

```go
type ThemeConfig struct {
	PrimaryColor string         `json:"primary_color,omitempty"`
	LogoURL      string         `json:"logo_url,omitempty"`
	Position     WidgetPosition `json:"position,omitempty"`
}
```

Stored as a single JSON column (`db:"theme_config,json"`) — no Alembic
migration required for new fields (unlike the welcome_message removal,
this is additive to an existing JSON blob column, not a new SQL column).

Consumers:
- `bin-webchat-manager`: struct definition, DB read/write (JSON
  marshal/unmarshal, no per-field SQL)
- `bin-openapi-manager`: `WebchatManagerWidgetThemeConfig` schema
  (openapi.yaml L2327-2344), referenced from `WebchatManagerWidget`
  and both POST/PUT request bodies
- `bin-api-manager`: passthrough only — `server/webchat_widgets.go`'s
  `convertWebchatThemeConfig()` maps the generated OpenAPI type to the
  `wcwidget.ThemeConfig` Go struct; `servicehandler` passes it through
  unmodified (no validation logic here today — validation of hex color
  format / logo URL safety happens ONLY on the frontend and is NOT
  re-validated server-side, a pre-existing gap noted in §5 below)
- `monorepo-javascript/square-admin`:
  - `webchat-widget-runtime/render.js`: `applyWidgetTheme()` — the
    ONLY place that actually applies theme values to DOM styles.
    Shared by the real embeddable widget (`widget.js`) and the
    admin's live preview (`WidgetPreview.js`) — single source of
    truth for the resulting visual output (design doc precedent: G2).
  - `webchat-widget-runtime/widget.js`: `buildWidgetDom()` /
    `WIDGET_CSS` — static inline CSS baseline that `render.js`
    overrides per-instance.
  - `views/webchat_widgets/{create,detail}.js`: admin form (refs +
    state), submits `theme_config` in the POST/PUT body.
  - `views/webchat_widgets/WidgetPreview.js`: live preview, calls
    `applyWidgetTheme()` directly with in-progress (unsaved) form
    state.

## 3. New ThemeConfig fields

```go
// ThemeConfig holds cosmetic, customer-editable widget appearance
// settings. All fields are optional; a nil ThemeConfig or empty field
// falls back to the platform default.
type ThemeConfig struct {
	PrimaryColor           string         `json:"primary_color,omitempty"`
	SecondaryColor         string         `json:"secondary_color,omitempty"`
	HeaderBackgroundColor  string         `json:"header_background_color,omitempty"`
	HeaderTextColor        string         `json:"header_text_color,omitempty"`
	LogoURL                string         `json:"logo_url,omitempty"`
	Position               WidgetPosition `json:"position,omitempty"`
	ThemeMode              ThemeMode      `json:"theme_mode,omitempty"`
	HeaderTitle            string         `json:"header_title,omitempty"`
	HeaderSubtitle         string         `json:"header_subtitle,omitempty"`
}

// ThemeMode controls light/dark/auto rendering of the widget panel.
type ThemeMode string

const (
	ThemeModeLight ThemeMode = "light" // default
	ThemeModeDark  ThemeMode = "dark"
	ThemeModeAuto  ThemeMode = "auto" // follows prefers-color-scheme
)
```

Defaults and dark-mode precedence (applied entirely in `render.js`'s
`applyWidgetTheme()` — NOT via CSS class rules, and NOT baked into the
Go struct's zero value):

**Critical architectural constraint** (found in round-3 design
review): `render.js`'s `applyWidgetTheme()` sets `els.header.style.background`
(and bubble/sendButton) as an inline style, UNCONDITIONALLY, on every
call — see current code, `render.js:51-53`. Inline styles always win
the CSS cascade over stylesheet class rules with no exception. This
means dark mode CANNOT be implemented as passive CSS class rules for
any property `applyWidgetTheme()` already sets inline (header/bubble/
send-button background, header text color) — a `.vb-theme-dark` CSS
class rule for `header background` would be silently shadowed and
never render. Therefore:

- `applyWidgetTheme()` itself MUST become `theme_mode`-aware: when
  computing the effective header background/text/secondary color, it
  must branch on `themeConfig.theme_mode` and compute the correct
  dark-vs-light default INLINE, not delegate default-selection to a
  stylesheet. CSS classes (`vb-theme-dark` on the root) remain valid
  ONLY for properties `applyWidgetTheme()` does NOT set inline today
  (e.g. panel/messages background, message bubble colors) — those may
  still use the class-based approach originally sketched in §5's
  `widget.js` row.
- Precedence rule (previously unspecified, now locked): an explicit
  `header_background_color`/`header_text_color`/`secondary_color`
  ALWAYS wins over the `theme_mode` default, regardless of
  light/dark/auto. `theme_mode` only selects which DEFAULT applies
  when the corresponding color field is unset.
- Effective default resolution order per color field, evaluated in
  `applyWidgetTheme()`:
  1. Explicit `theme_config.<field>` value, if set → use as-is.
  2. Else resolve from `theme_mode`: `light` → existing light
     defaults (below); `dark` → dark-mode defaults (below); `auto` →
     evaluate `window.matchMedia('(prefers-color-scheme: dark)')` at
     render time, same as light/dark branch. This always executes in
     a real, non-sandboxed `window` context: the real widget runs
     directly on the host page, and `WidgetPreview.js` runs
     `applyWidgetTheme()` in the ADMIN'S OWN parent React
     component/window (via `buildWidgetDom(document)` against the
     real global `document`) — only the resulting serialized HTML
     string with inline styles already baked in is injected into the
     sandboxed `srcdoc` iframe afterward. `matchMedia` is therefore
     never actually evaluated inside a sandboxed iframe in either
     consumption path.
  3. `secondary_color`: light default `#fff` / dark default `#1f2937`
     (existing hardcoded default in `WIDGET_CSS` for light; dark is
     new)
  4. `header_background_color`: light default = resolved
     `primary_color` (existing behavior); dark default = a fixed dark
     surface color (e.g. `#111827`), independent of `primary_color`,
     since forcing a bright `primary_color` as a dark-mode header
     background would defeat the point of dark mode
  5. `header_text_color`: light default `#fff`; dark default `#f9fafb`
- Accepted, non-blocking limitation: `theme_mode=auto` is resolved
  ONCE, at `applyWidgetTheme()` call time (`widget.js`'s constructor
  calls it once on mount, no `matchMedia(...).addEventListener('change', ...)`
  wired up). If the visitor's OS theme changes while the widget is
  already mounted, it will not live-re-theme without a page reload.
  Same severity tier as the header-contrast gap noted in §4 — deferred
  as a future polish item, not required for this design to ship.
- `theme_mode`: default `light`
- `header_title`: `"Chat with us"` (existing hardcoded default,
  currently NOT configurable — `widget.js` L183 hardcodes this
  string; this design makes it configurable)
- `header_subtitle`: none (no subtitle row rendered)

## 4. Validation (new, addresses a pre-existing gap)

Per §2, hex-color format is currently validated ONLY on the frontend
(HTML `<input type="color">` or manual regex in the form), never on
the backend. This design does NOT expand validation scope beyond
what already exists for `primary_color` — for consistency, the same
"no backend validation, frontend-only" posture applies to the three
new color fields. `theme_mode` IS validated backend-side (enum,
mirroring `WidgetPosition`'s existing enum-validation precedent) since
it's a closed enum, not free text.

`header_title`/`header_subtitle`: plain strings, rendered via
`textContent` only in `render.js` (never `innerHTML`), so no
additional XSS surface — same textContent-only guarantee that already
covers all other message/text rendering in this codebase (design doc
§5/§9.5 precedent cited in `widget.js`'s own comments). Length capped
at 100 chars (title) / 200 chars (subtitle) — enforced with a
`maxlength` HTML attribute on the frontend form inputs only, NOT
backend-enforced (matches the existing precedent: `Widget.Name` has no
backend length validation either).

`header_background_color`/`header_text_color` contrast: no validation
anywhere (frontend or backend) prevents an admin from picking colors
that make header text unreadable (e.g. white-on-white). This is a
cosmetic UX gap, not a security concern (per round-3 review: CSSOM
style assignment silently ignores malformed values, no injection
vector). Recommended lightweight mitigation, non-blocking for this
design: the frontend form shows a live contrast warning (e.g. compute
relative luminance client-side and flag low-contrast pairs) — deferred
to implementation as a nice-to-have, not a hard requirement.

## 5. Affected files

| File | Change |
|---|---|
| `bin-webchat-manager/models/widget/widget.go` | Add 6 fields + `ThemeMode` type to `ThemeConfig` |
| `bin-webchat-manager/models/widget/widget_test.go` | Add field assertions |
| `bin-openapi-manager/openapi/openapi.yaml` | Add 6 properties to `WebchatManagerWidgetThemeConfig`, add `WebchatManagerWidgetThemeMode` enum schema |
| `bin-api-manager/server/webchat_widgets.go` | Extend `convertWebchatThemeConfig()` to map 6 new fields |
| `bin-api-manager/server/webchat_widgets_test.go` | Create conversion test cases (file does not exist yet under `server/`) |
| `monorepo-javascript/square-admin/src/webchat-widget-runtime/render.js` | Extend `applyWidgetTheme()` to apply 6 new style properties + header title/subtitle text. **Must become `theme_mode`-aware** for header background/text/secondary-color default resolution (see §3's precedence rule) — it computes and sets these INLINE per the resolution order in §3, it does not delegate to a CSS class for any property it already sets inline today |
| `monorepo-javascript/square-admin/src/webchat-widget-runtime/widget.js` | `buildWidgetDom()`: add a `headerSubtitle` element (currently only `headerTitle` exists); `WIDGET_CSS`: `.vb-theme-dark` class rules ONLY for properties `applyWidgetTheme()` does not set inline (panel/messages background, message bubble colors) — NOT for header/bubble/send-button background or header text color, which `render.js` must set inline per §3; L183's hardcoded `headerTitle.textContent = 'Chat with us'` becomes the fallback default applied by `render.js`, not a permanent hardcode in this file |
| `monorepo-javascript/square-admin/src/webchat-widget-runtime/__tests__/render.test.js` | Add test cases for 6 new fields. **Requires a `window.matchMedia` mock in test setup** — jsdom does not implement `matchMedia` natively, and the new `theme_mode=auto` branch will throw `TypeError` in Jest without one (standard jsdom workaround: add a mock in `square-admin/src/setupTests.js` or per-test) |
| `monorepo-javascript/square-admin/src/views/webchat_widgets/{create,detail}.js` | Add 6 new `useState` hooks (mirroring the existing `primaryColor`/`logoUrl` pattern), 6 new form fields (Appearance block: 3 color pickers, theme_mode select, header_title/subtitle inputs), and include the new keys in the `themeConfig` object literal passed to `WidgetPreview` and in the PUT/POST request body |
| `monorepo-javascript/square-admin/src/views/webchat_widgets/WidgetPreview.js` | **Real logic change, not just PropTypes**: `WidgetPreview.js:42` unconditionally overwrites `dom.headerTitle.textContent = 'Chat with us'` AFTER `applyWidgetTheme()` runs (line 38) — this line must be removed/changed to let `render.js`'s `header_title` handling (with its own default fallback) take effect, otherwise the live preview will never reflect a custom `header_title`. Also extend `PropTypes.shape` (L95-99) and the `useMemo` dependency array (L81) to include all 6 new fields |
| `monorepo-javascript/square-admin/src/views/webchat_widgets/__tests__/{create,detail,WidgetPreview}.test.js` | Extend existing test coverage |
| `bin-api-manager/docsdev/source/webchat_struct_widget.rst` | **Required, not conditional**: add the 6 new `theme_config` fields to the struct doc (lines 22-26 code sample, ~L40 field list) and rewrite/remove line 79's now-stale "only the three fields above are configurable" sentence |

No Alembic migration (additive JSON column fields — confirmed: the
`theme_config` column is a plain `json` type with no
`GENERATED ALWAYS AS` virtual/generated column indexing into it
anywhere in the Alembic history). No `bin-common-handler` change
(ThemeConfig is already passed as an opaque `*widget.ThemeConfig`
pointer through the RPC layer at every hop — `widgethandler` →
`bin-common-handler/requesthandler` → `bin-api-manager/servicehandler`
— never split into positional args like `welcomeMessage` had been).

**Forward-compat note**: `WebchatManagerWidgetThemeConfig` has no
`additionalProperties: false`, so if square-admin ships new
`theme_config` keys ahead of the Go backend/OpenAPI spec adding them
(deploy-ordering skew across the two independently-deployed repos),
the extra JSON keys are silently dropped by Go's JSON unmarshal into
the (older) generated struct — no request rejection, no data loss for
the fields the backend does know about. Safe either deploy order.

## 6. UX: Appearance tab (square-admin)

Current `create.js`/`detail.js` render Appearance fields inline as a
single `Card` (verified: today's block is exactly 3 fields /
`primary_color` + `logo_url` + `position`, `create.js:182-231`, ~2
visual rows). This design does NOT introduce a new tab structure
(avoiding a UI-restructure PR) — but per round-3 review, adding 6 more
fields to that same `Card` triples the field count to 9 and will read
as an undifferentiated wall of labeled rows without at least visual
sub-grouping. Committed mitigation (does not violate the "no new tab"
scope boundary): the Appearance `Card` gets 3 mini sub-headings within
its existing single-Card layout — "Colors" (primary/secondary/header
background/header text, 4 fields), "Header text" (title/subtitle, 2
fields), "Layout" (logo, position, theme_mode, 3 fields) — plain
`<h4>`-style section labels, no new `Card`/`Tabs` component, no new
navigation. If this still proves unwieldy in practice, a full tab
split remains a legitimate follow-up per the "avoid overclaiming
scope" principle from `design-first-with-review-loops`.

Live preview (`WidgetPreview.js`) already re-renders on every
`themeConfig`-related `useMemo` dependency change — extending its
`PropTypes.shape` dependency array to include the 6 new fields is a
one-line change per component, not a new mechanism.

## 7. Verification plan

Build order: `bin-webchat-manager` (Go, standalone — no
`bin-common-handler`/`bin-api-manager` signature changes needed since
ThemeConfig is opaque-passthrough) → `bin-openapi-manager` →
`bin-api-manager` → `monorepo-javascript/square-admin` (separate repo,
separate CI).

1. `cd bin-webchat-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
2. `cd bin-openapi-manager && go generate ./...`
3. `cd bin-api-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
4. `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build` (if RST docs reference ThemeConfig fields — check `webchat_struct_widget.rst`)
5. `cd monorepo-javascript/square-admin && npm test -- --watchAll=false && npm run lint && npm run build`

## 8. Approval status

Design Review loop: Round 1 CHANGES_REQUESTED (5 issues, fixed), Round
2 APPROVED (no findings), Round 3 CHANGES_REQUESTED (dark-mode
CSS-class/inline-style architectural collision, fixed by making
`applyWidgetTheme()` `theme_mode`-aware), Round 4 APPROVED (traced
the fix concretely, confirmed genuine; 2 non-blocking gaps
documented), Round 5 APPROVED (final independent re-verification, 1
cosmetic wording nit fixed) — loop closed at 2 consecutive APPROVED
(rounds 4-5). Ready for implementation.
