# Webchat Widget Appearance Expansion: Design

Status: Draft

## 1. Scope (locked with CEO/CTO)

Expand `Widget.ThemeConfig` from 3 fields (`primary_color`, `logo_url`,
`position`) to 6 fields, adding:

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

Defaults (applied in `render.js`, matching the existing
`primary_color`/`position` fallback pattern — NOT baked into the Go
struct's zero value, consistent with current design):
- `secondary_color`: none — when unset, header/bubble text stays
  `#fff` (existing hardcoded default in `WIDGET_CSS`)
- `header_background_color`: falls back to `primary_color` (existing
  behavior — header currently uses `primary_color` for its
  background; this is now overridable independently)
- `header_text_color`: `#fff` (existing hardcoded default)
- `theme_mode`: `light`
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

## 5. Affected files

| File | Change |
|---|---|
| `bin-webchat-manager/models/widget/widget.go` | Add 6 fields + `ThemeMode` type to `ThemeConfig` |
| `bin-webchat-manager/models/widget/widget_test.go` | Add field assertions |
| `bin-openapi-manager/openapi/openapi.yaml` | Add 6 properties to `WebchatManagerWidgetThemeConfig`, add `WebchatManagerWidgetThemeMode` enum schema |
| `bin-api-manager/server/webchat_widgets.go` | Extend `convertWebchatThemeConfig()` to map 6 new fields |
| `bin-api-manager/server/webchat_widgets_test.go` | Add conversion test cases |
| `monorepo-javascript/square-admin/src/webchat-widget-runtime/render.js` | Extend `applyWidgetTheme()` to apply 6 new style properties + header title/subtitle text |
| `monorepo-javascript/square-admin/src/webchat-widget-runtime/widget.js` | `buildWidgetDom()`: add a `headerSubtitle` element (currently only `headerTitle` exists); `WIDGET_CSS`: theme_mode dark-mode base rules |
| `monorepo-javascript/square-admin/src/webchat-widget-runtime/__tests__/render.test.js` | Add test cases for 6 new fields |
| `monorepo-javascript/square-admin/src/views/webchat_widgets/{create,detail}.js` | Add form fields (Appearance tab: 3 new color pickers, theme_mode select, header_title/subtitle inputs) |
| `monorepo-javascript/square-admin/src/views/webchat_widgets/WidgetPreview.js` | Pass new `themeConfig` fields through (already passthrough — no logic change, just extend PropTypes) |
| `monorepo-javascript/square-admin/src/views/webchat_widgets/__tests__/{create,detail,WidgetPreview}.test.js` | Extend existing test coverage |

No Alembic migration (additive JSON column fields). No `bin-common-handler`
change (ThemeConfig is already passed as an opaque struct through the
RPC layer — no field-by-field parameter threading like `welcomeMessage`
had, because it was never split into positional args in the first place).

## 6. UX: Appearance tab (square-admin)

Current `create.js`/`detail.js` render Appearance fields inline,
unnamed as a distinct section. This design does NOT introduce a new
tab structure (avoiding a UI-restructure PR); the 6 new fields are
added to the existing appearance block using the same
`Label`+`Input`/`Select` pattern as `primary_color`/`logo_url`/`position`
today. If the form grows unwieldy this becomes a legitimate follow-up
(tab split), but is out of scope here per the "avoid overclaiming
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

Draft → pending Design Review loop (min 2, 2-consecutive-APPROVED).
