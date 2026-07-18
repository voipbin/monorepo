# Webchat Welcome Message: Flow Consolidation Design

Status: Draft

## 1. Problem statement

`bin-webchat-manager`'s `Widget` resource carries two independent
mechanisms that both fire "something" at the start of a webchat
session:

- `WelcomeMessage` (`string`, DB column `welcome_message`) — a static
  text field. On `POST /webchat_sessions`, `sessionhandler.Create()`
  copies it verbatim into the Session response's transient
  `welcome_message` field. It is never persisted as a real message
  and never appears in `bin-conversation-manager`'s conversation
  history.
- `SessionFlowID` (`uuid.UUID`, DB column `session_flow_id`) — fires
  once per Session at creation time via
  `ConversationV1ConversationCreateAndExecuteFlow`, which creates a
  real `Conversation` and executes an activeflow. A Flow can already
  send a message via the `message_send` (or equivalent conversation
  action) action.

Both exist to serve the same visitor-facing moment ("what does the
visitor see when the chat opens"), through two structurally different
paths. This violates the standing product principle: **state-field
duplication is disfavored where a Flow branch can express the same
behavior** (see CPO decision log, 2026-07). Concretely:

- `WelcomeMessage` bypasses the conversation/activeflow system
  entirely, so it never appears in conversation history — an
  unusual, undocumented exception every new engineer has to learn.
- `WelcomeMessage` is delivered synchronously in the `POST
  /webchat_sessions` response; anything richer (personalization,
  business-hours branching, agent availability) requires the
  customer to migrate to `SessionFlowID` anyway, so the static field
  is a dead end for any use case beyond the simplest one.
- Every codepath that reads/writes Widget's basic info
  (`create.go`, `db.go` UpdateBasicInfo, `listenhandler`,
  `bin-api-manager` server, OpenAPI spec, RST docs) threads
  `WelcomeMessage` as a parallel argument to `SessionFlowID`,
  doubling the surface area for a single UX concern.

## 2. Goals

1. Remove `Widget.WelcomeMessage` as a distinct mechanism; a
   customer who wants a welcome message expresses it as a
   `SessionFlowID` Flow with a single message-send action.
2. Preserve the existing Session-creation trigger contract for
   `SessionFlowID` (fires once, at session creation, owned by
   `bin-conversation-manager`) — this design changes nothing about
   that trigger path.
3. Keep the change backend-only and Go-monorepo-scoped. square-admin
   UI changes are explicitly deferred (see Non-goals).
4. Ship as a clean removal, no field-preserving compatibility shim.

## 3. Decisions locked (2026-07-18, CEO/CTO)

1. **Scope**: backend mechanism consolidation only
   (`bin-webchat-manager` + `bin-openapi-manager` +
   `bin-api-manager` codegen/docs). square-admin UI (`create.js`,
   `detail.js`, the Welcome Message text input) is explicitly
   deferred to a follow-up UI PR — this PR does not touch
   `monorepo-javascript`.
2. **Migration**: no backfill. `WelcomeMessage` is removed outright
   (column dropped, field removed from every layer). Existing
   customers who had a `welcome_message` set lose it silently and
   must reconfigure via a `SessionFlowID` Flow. No automatic Flow
   generation from the old text.
3. **UI dead-input acceptance**: known, accepted transitional state
   — square-admin's Welcome Message input box will silently stop
   persisting (PUT/POST body still includes `theme_config` etc. but
   `welcome_message` field is simply dropped/ignored server-side
   until the follow-up UI PR ships). Low user count at this stage
   makes this acceptable; the follow-up UI PR removes the input.

## 4. Non-goals

- square-admin UI changes (input removal, any Flow-based welcome
  message authoring UX). Follow-up PR, `monorepo-javascript`.
- Any backfill/migration tooling for existing `welcome_message`
  values.
- Appearance/ThemeConfig expansion (secondary_color,
  header_background_color, header_text_color, theme_mode,
  header_title) — tracked as a separate, subsequent design
  (`2026-07-18-webchat-widget-appearance-expansion-design.md`).
- Any WS/latency optimization for Flow-based welcome delivery. The
  existing `SessionFlowID` trigger path (conversation-manager
  `CreateAndExecuteFlow`) is reused unmodified.

## 5. Affected files

| File | Why |
|---|---|
| `bin-webchat-manager/models/widget/widget.go` | Remove `WelcomeMessage` field |
| `bin-webchat-manager/models/widget/field.go` | Remove `FieldWelcomeMessage` |
| `bin-webchat-manager/models/widget/webhook.go` | Remove `WelcomeMessage` from `WebhookMessage` + `ConvertWebhookMessage` |
| `bin-webchat-manager/models/widget/widget_test.go` | Drop welcome_message assertions |
| `bin-webchat-manager/models/session/session.go` | Remove transient `Session.WelcomeMessage` field |
| `bin-webchat-manager/models/session/webhook.go` | Remove `WelcomeMessage` from Session `WebhookMessage` |
| `bin-webchat-manager/pkg/widgethandler/main.go` | Remove `welcomeMessage` param from `Create`/`UpdateBasicInfo` interface signatures |
| `bin-webchat-manager/pkg/widgethandler/create.go` | Remove `welcomeMessage` param + field set |
| `bin-webchat-manager/pkg/widgethandler/db.go` | Remove `welcomeMessage` param from `UpdateBasicInfo` + `FieldWelcomeMessage` from update map |
| `bin-webchat-manager/pkg/widgethandler/create_test.go`, `db_test.go` | Drop welcome_message args/assertions |
| `bin-webchat-manager/pkg/dbhandler/widget_test.go` | Drop welcome_message from test fixtures |
| `bin-webchat-manager/pkg/listenhandler/v1_widgets.go` | Remove `req.WelcomeMessage` arg from `Create`/`UpdateBasicInfo` calls |
| `bin-webchat-manager/pkg/listenhandler/models/request/v1_widgets.go` | Remove `WelcomeMessage` from `V1DataWidgetsPost`/`V1DataWidgetsIDPut` |
| `bin-webchat-manager/pkg/sessionhandler/create.go` | Remove `res.WelcomeMessage = w.WelcomeMessage` line; comment update (Widget fetch now serves ONE purpose: SessionFlowID) |
| `bin-webchat-manager/pkg/sessionhandler/create_test.go` | Drop welcome_message assertions in all 3 test cases |
| `bin-webchat-manager/pkg/widgethandler/mock_main.go` | Regenerated (`go generate ./...` in `bin-webchat-manager`) |
| `bin-webchat-manager/scripts/database_scripts_test/widgets.sql` | Drop `welcome_message` column (test-DB fixture) |
| `bin-dbscheme-manager/bin-manager/main/versions/<new>.py` | New Alembic migration: `alter table webchat_widgets drop column welcome_message`. `down_revision` must chain off the CURRENT head at implementation time — re-verify with `alembic -c alembic.ini heads` immediately before running `alembic revision` (as of design time, head is `1a1f28d6842c`, but do not hardcode this; re-check for drift) |
| `bin-openapi-manager/openapi/openapi.yaml` | Remove `welcome_message` from `WebchatManagerWidget` and `WebchatManagerSession` schemas (incl. `required` arrays) |
| `bin-openapi-manager/openapi/paths/webchat_widgets/main.yaml` | Remove `welcome_message` from POST request body + required |
| `bin-openapi-manager/openapi/paths/webchat_widgets/id.yaml` | Remove `welcome_message` from PUT request body + required |
| `bin-openapi-manager/gens/models/gen.go` | Regenerated (`go generate ./...`) |
| `bin-common-handler/pkg/requesthandler/main.go` | Remove `welcomeMessage string` param from `WebchatV1WidgetCreate`/`WebchatV1WidgetUpdate` interface decls (~L1515, L1527) |
| `bin-common-handler/pkg/requesthandler/webchat_widget.go` | Remove `welcomeMessage` param + `WelcomeMessage: welcomeMessage` field set from both impl functions (L21/32, L101/111) — this is the actual RPC-client layer that `bin-api-manager/pkg/servicehandler` and `bin-webchat-manager`'s own callers depend on |
| `bin-common-handler/pkg/requesthandler/mock_main.go` | Regenerated (`go generate ./...` in `bin-common-handler`) |
| `bin-api-manager/pkg/servicehandler/main.go` | Remove `welcomeMessage string` param from `WebchatWidgetCreate`/`WebchatWidgetUpdate` interface decls (~L886, L897) |
| `bin-api-manager/pkg/servicehandler/webchat_widget.go` | Remove `welcomeMessage` param from both public functions and the `h.reqHandler.WebchatV1Widget*` call args (L122, 179, 196, 264) — this is the layer between `server/webchat_widgets.go` (HTTP) and `bin-common-handler` (RPC); it independently carries the same positional param and must be edited in the same PR |
| `bin-api-manager/pkg/servicehandler/mock_main.go` | Regenerated (`go generate ./...` in `bin-api-manager`) |
| `bin-api-manager/pkg/servicehandler/webchat_widget_test.go` | **Not cosmetic** — has 5 compile-breaking call sites (L161, 163, 211, 254, 304) passing a positional `"welcome"` string arg to `WebchatWidgetCreate`/`WebchatWidgetUpdate`/mocked `WebchatV1WidgetCreate` that must be removed to match the new shorter signature, in addition to the stale doc comment |
| `bin-api-manager/server/webchat_widgets.go` | Remove `req.WelcomeMessage` args (2 call sites: create + update) |
| `bin-api-manager/gens/openapi_server/gen.go`, `gens/openapi_redoc/*` | Regenerated |
| `bin-api-manager/docsdev/source/webchat_struct_session.rst` | Remove `welcome_message` field doc + example |
| `bin-api-manager/docsdev/source/webchat_struct_widget.rst` | Checked — this RST already has no `welcome_message` field (pre-existing drift vs. code); no edit needed here, noted explicitly so it is not mistaken for an omission |
| `bin-api-manager/docsdev/source/webchat_overview.rst` | Rewrite step 4 (no longer "receive welcome_message") |
| `bin-api-manager/docsdev/source/websocket_struct.rst` | Check/update any welcome_message example fields |
| `bin-api-manager/docsdev/build/` | Rebuilt Sphinx HTML, force-added |

## 6. Exact changes

### 6.1 `bin-webchat-manager/models/widget/widget.go`

Remove:
```go
WelcomeMessage string `json:"welcome_message,omitempty" db:"welcome_message"`
```

### 6.2 `bin-webchat-manager/models/session/session.go`

Remove the transient field (currently `db:"-"`):
```go
WelcomeMessage string `json:"welcome_message,omitempty" db:"-"`
```
and its doc comment block explaining the Widget-copy mechanism.

### 6.3 `bin-webchat-manager/pkg/sessionhandler/create.go`

Before:
```go
// The single WidgetGet call below serves TWO purposes: (1) read
// Widget.WelcomeMessage to attach to the response (§6), (2) read
// Widget.SessionFlowID to decide whether to trigger anything. ...
w, err := h.widgetHandler.Get(ctx, widgetID)
if err != nil {
    ...
    // welcome_message stays empty and SessionFlowID is simply skipped below.
    return res, nil
}

res.WelcomeMessage = w.WelcomeMessage

if w.SessionFlowID == uuid.Nil {
```

After: drop the `res.WelcomeMessage = w.WelcomeMessage` line and
rewrite the doc comment to say the WidgetGet call now serves the
single purpose of reading `SessionFlowID`.

### 6.4 Widget CRUD signature change (breaking, same-PR across ALL layers)

`welcomeMessage string` is a positional parameter threaded through
FOUR layers, not one — every layer must drop it in the same PR or
the monorepo fails to compile:

1. `bin-webchat-manager/pkg/widgethandler.WidgetHandler.Create` /
   `UpdateBasicInfo` (the service's own internal handler interface)
2. `bin-common-handler/pkg/requesthandler.RequestHandler.WebchatV1WidgetCreate`
   / `WebchatV1WidgetUpdate` (the shared RPC-client interface every
   OTHER service uses to call webchat-manager over RabbitMQ —
   `main.go` ~L1515/L1527 declares it, `webchat_widget.go` L21/32 and
   L101/111 implement it)
3. `bin-api-manager/pkg/servicehandler.ServiceHandler.WebchatWidgetCreate`
   / `WebchatWidgetUpdate` (the auth+RPC-delegation layer between
   HTTP and `bin-common-handler` — `main.go` ~L886/L897 interface,
   `webchat_widget.go` L122/179/196/264 impl + RPC call args)
4. `bin-api-manager/server/webchat_widgets.go` (the HTTP handler
   parsing the request body)

```go
// before (widgethandler, mirrored at requesthandler and servicehandler layers)
Create(ctx context.Context, customerID uuid.UUID, name string, welcomeMessage string, sessionFlowID uuid.UUID, messageFlowID uuid.UUID, sessionIdleTimeout int, themeConfig *widget.ThemeConfig) (*widget.Widget, error)

// after
Create(ctx context.Context, customerID uuid.UUID, name string, sessionFlowID uuid.UUID, messageFlowID uuid.UUID, sessionIdleTimeout int, themeConfig *widget.ThemeConfig) (*widget.Widget, error)
```
Same shape change applies to `UpdateBasicInfo`/`WebchatV1WidgetUpdate`/
`WebchatWidgetUpdate`. Every caller across all four layers drops the
corresponding `welcomeMessage`/`req.WelcomeMessage` argument,
including 5 compile-breaking call sites in
`bin-api-manager/pkg/servicehandler/webchat_widget_test.go`
(L161, 163, 211, 254, 304) that pass a positional `"welcome"` string.
Generated mocks (`bin-webchat-manager/pkg/widgethandler/mock_main.go`,
`bin-common-handler/pkg/requesthandler/mock_main.go`,
`bin-api-manager/pkg/servicehandler/mock_main.go`) are regenerated by
each service's own `go generate ./...` — no manual edits.

### 6.5 Alembic migration

Generate via `alembic revision -m "webchat_widgets_drop_welcome_message"`
in `bin-dbscheme-manager/bin-manager`, then fill in:

```python
def upgrade():
    op.execute("""
        alter table webchat_widgets
        drop column welcome_message;
    """)


def downgrade():
    op.execute("""
        alter table webchat_widgets
        add column welcome_message text after direct_hash;
    """)
```

Downgrade restores the column (empty) but cannot restore lost data —
this is a data-loss migration by design per the locked "no backfill"
decision (§3.2). Documented in the migration's docstring.

### 6.6 OpenAPI spec

Remove `welcome_message` from:
- `WebchatManagerWidget` schema `properties` + `required` array
  (`openapi.yaml` ~L2370, ~L2419)
- `WebchatManagerSession` schema `properties` (~L2466) — this is the
  transient session-response copy; removing `SessionFlowID` (or the
  Flow triggering it) still handles delivery, just not through this
  static field
- `webchat_widgets/main.yaml` POST request body `properties` +
  `required`
- `webchat_widgets/id.yaml` PUT request body `properties` +
  `required`

### 6.7 Wire-field checklist

Empirically verified against the current repo state (2026-07-18):

| Field | Current source | Action |
|---|---|---|
| `Widget.welcome_message` (DB col, `text`) | `bin-webchat-manager/scripts/database_scripts_test/widgets.sql:14`, live schema via `c9602a744cb3` migration | DROP via new migration |
| `WebchatManagerWidget.welcome_message` (OpenAPI, required) | `openapi.yaml:2370,2419` | REMOVE |
| `WebchatManagerSession.welcome_message` (OpenAPI, optional) | `openapi.yaml:2466` | REMOVE |
| `V1DataWidgetsPost.WelcomeMessage` / `V1DataWidgetsIDPut.WelcomeMessage` | `pkg/listenhandler/models/request/v1_widgets.go:16,31` | REMOVE |
| `gens/openapi_server/gen.go` WelcomeMessage (4 occurrences: PostWebchatWidgetsJSONBody, PutWebchatWidgetsIdJSONBody, WebchatManagerSession, WebchatManagerWidget) | generated | Regenerate, drop-out expected |

## 7. Copy/decision rationale

No customer-facing copy changes in this PR (backend-only, no UI).
RST doc rewrite for `webchat_overview.rst` step 4 changes "receive
the widget's welcome_message" to describe the `SessionFlowID`-based
flow trigger as the sole mechanism for greeting a visitor.

## 8. Verification plan

Build order matters: `bin-common-handler` is the shared RPC-client
library every other service vendors, so it must be edited AND
regenerated first, then `bin-webchat-manager` and
`bin-api-manager` (which both depend on it) after.

1. `cd bin-webchat-manager && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
2. `cd bin-common-handler && go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m` (regenerates `pkg/requesthandler/mock_main.go`)
3. `cd bin-openapi-manager && go generate ./...` (regenerate `gens/models/gen.go`)
4. `cd bin-api-manager && go generate ./... && go build ./...` (confirm `gens/openapi_server/gen.go` drops the field cleanly, and `pkg/servicehandler`, `server/webchat_widgets.go` compile once all call sites are edited)
5. `cd bin-api-manager && go mod tidy && go mod vendor && go test ./... && golangci-lint run -v --timeout 5m` (regenerates `pkg/servicehandler/mock_main.go`)
6. Grep verification: `grep -rn "welcome_message\|WelcomeMessage" bin-webchat-manager bin-openapi-manager bin-api-manager bin-common-handler` — zero hits expected outside `docsdev/` (updated separately) and the new migration's downgrade path.
7. Rebuild RST docs: `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`, `git add -f bin-api-manager/docsdev/build/`.
8. Alembic: re-verify head with `alembic -c alembic.ini heads` immediately before generating, generate migration file via `alembic revision`, hand-edit SQL, verify with `alembic -c alembic.ini heads` again (single head) — do NOT run `alembic upgrade` against any shared DB.
9. Add one test case (`sessionhandler/create_test.go`) explicitly asserting the accepted post-removal behavior: a Session created against a Widget with `SessionFlowID == uuid.Nil` succeeds with no welcome-message field/delivery of any kind, rather than relying solely on deleting the old assertions + a grep check. Makes the no-greeting outcome intentional-by-test, not just intentional-by-doc.

## 9. Rollout / risk

- **Data loss**: existing `welcome_message` values are dropped with
  no backfill (accepted, §3.2). Any customer relying on it loses the
  greeting until they configure a `SessionFlowID` Flow.
- **API breaking change**: `welcome_message` disappears from
  `POST/PUT /webchat_widgets` request/response and
  `POST /webchat_sessions` response. `bin-webchat-manager` has no
  external consumers yet beyond square-admin (merged 2026-07-16/17,
  early feature), so blast radius is limited to square-admin's
  currently-dead-after-this-PR input (§3.3, accepted).
- **UI dead input window**: square-admin's Welcome Message text box
  keeps rendering but silently no-ops until the follow-up UI PR.
  Accepted given the low current webchat-widget user count.
- **Downgrade is lossy**: the Alembic `downgrade()` restores the
  column but not prior data. Documented; matches project convention
  for other breaking-rename migrations in this service (see
  `1a1f28d6842c_webchat_widgets_session_message_flow_.py`).
- **Silent no-greeting for pre-`SessionFlowID` widgets**: `welcome_message`
  (migration `c9602a744cb3`) predates `session_flow_id` (migration
  `1a1f28d6842c`), so every widget created before `SessionFlowID`
  existed has no Flow configured. After this change, such a widget's
  visitor gets ZERO greeting on session start — `sessionhandler.Create()`
  silently returns when `SessionFlowID == uuid.Nil`, with no log level
  bump, no metric, no square-admin warning anywhere. This is the
  concrete runtime shape of the "existing customers lose it silently"
  decision in §3.2 — called out explicitly here so it is not only
  implied by the migration decision.
- **`DROP COLUMN` lock consideration**: `ALTER TABLE webchat_widgets
  DROP COLUMN welcome_message` is not covered by MySQL 8.0's instant-DDL
  algorithm (drop-column always requires a table rebuild) and can
  briefly hold a metadata/write lock under write load. Low risk given
  the service's low current row count (webchat-manager shipped
  2026-07-16), but worth this one-line note before running against a
  shared environment.

## 10. Open questions

None — all locked in §3.

## 11. Approval status

Design Review loop: Round 1 CHANGES_REQUESTED (missing 4-layer
signature-change coverage, fixed), Round 2 APPROVED (no findings),
Round 3 APPROVED (3 non-blocking informational findings, incorporated
into §8/§9 above). 2 consecutive APPROVED — loop closed. APPROVED,
ready for implementation.
