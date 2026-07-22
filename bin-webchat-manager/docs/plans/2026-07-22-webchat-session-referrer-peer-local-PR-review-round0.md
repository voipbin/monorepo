# PR review — webchat session referrer + Peer/Local (web_session type) — Round 0

Reviewed against the approved design doc (§8 is the final spec):
`bin-webchat-manager/docs/plans/2026-07-22-webchat-session-referrer-peer-local-design.md`

Scope: 9 Go-monorepo commits + 1 JS-monorepo commit (7 doc-only commits excluded
from code review, spot-checked for content only).

Go commits reviewed (`git show <hash>` read in full):
- `b1395b2ba` bin-flow-manager — dead `"web_session"` map entry removal
- `5b7dca225` bin-ai-manager — dead `"web_session"` map entry removal
- `48e9f7fa7` bin-contact-manager — dead map entry + 2 test removals
- `5a9f23b26` bin-common-handler — `TypeWebSession` const, normalize/validate switches, `WebchatV1SessionCreate` referrer param
- `13ba46b60` bin-webchat-manager — `Session.Referrer/Peer/Local`, `sessionhandler.Create()`
- `51af257e7` bin-openapi-manager — OpenAPI schema + oapi-codegen tool directive fix
- `f0852361b` bin-dbscheme-manager — 2 Alembic migrations
- `d4cae2865` bin-api-manager — `validateReferrer`, RST docs

JS commit reviewed:
- `22704ae2` square-admin — client.js referrer capture, message_timeline.js rename + "Came from" display

---

## 1. Design-doc conformance (§3/§4/§8)

**Matches.** Every field/type/behavior in the diffs traces to a specific
design-doc clause with no unexplained additions:

- `TypeWebSession Type = "web_session"` — confirmed in
  `bin-common-handler/models/address/main.go` (commit `5a9f23b26`), exactly
  matching §8.3's updated spec (the post-revision value, NOT the interim
  `"webchat_visitor"` from §4.1's original round-3 decision).
- `Peer{Type: TypeWebSession, Target: id}` / `Local{Type: TypeWebchat, Target:
  widgetID}` — confirmed in `bin-webchat-manager/pkg/sessionhandler/create.go`
  (commit `13ba46b60`), matching §4's spec verbatim including field ordering
  (referrer/peer/local block) and the "no omitempty" requirement.
- Three dead-map-entry deletions (`b1395b2ba`, `5b7dca225`, `48e9f7fa7`) match
  §8.3/§8.4 exactly, including the two extra test deletions in
  bin-contact-manager that §8.4 called out by name
  (`Test_isCRMEligiblePeer`'s "web_session is ineligible" case,
  `Test_EventConversationMessageCreated`'s outgoing-web_session case).
- No scope creep found: no unrelated refactors, no unrequested fields, no
  drive-by renames outside what commit messages declare.

## 2. `TypeWebSession` string value — verified in actual `.go` source, not docs

```
grep -rn "TypeWebSession" bin-common-handler/models/address/main.go
  TypeWebSession Type = "web_session" // target is webchat-manager's Session.ID (the visitor's continuity token)
```

Repo-wide `grep -rn "webchat_visitor"` across all tracked `.go`/non-doc files
in the worktree returns **zero hits** — the interim string exists only inside
the design-doc-history markdown files (round0-3 + revision1 round0-3), which
is expected and correct (historical record, not live code). No stray
`"webchat_visitor"` literal survived the revision. ✅

## 3. Dead-code removal completeness + test verification

All three removals are single-line, surgical, and leave the enclosing map
otherwise untouched (`git show` diffs confirmed above — no collateral
whitespace/reformatting). Ran the actual test suites, not just read them:

```
bin-flow-manager:    go test ./pkg/activeflowhandler/...  → ok
bin-ai-manager:      go test ./pkg/aicallhandler/...      → ok
bin-contact-manager: go test ./pkg/contacthandler/...     → ok
```

`48e9f7fa7`'s test deletions are correctly scoped — `Test_isCRMEligiblePeer`'s
now-invalid case and `Test_EventConversationMessageCreated`'s
now-would-panic case (unconditional `InteractionCreate` mock expectation
would otherwise fire because `isCRMEligiblePeer("web_session")` now returns
`true`) are both removed, not merely commented out. Confirmed by full `go
test ./...` passing with no lingering skipped/panicking cases.

## 4. §4.3 scope boundary — self/peer locals for `ConversationV1ConversationCreateAndExecuteFlow`

**Verified directly in `bin-webchat-manager/pkg/sessionhandler/create.go`:**

```go
// line 50-51 (new Session.Peer/Local — commit 13ba46b60)
Peer:     commonaddress.Address{Type: commonaddress.TypeWebSession, Target: id.String()},
Local:    commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()},

// line 81-82 (existing self/peer feeding ConversationV1ConversationCreateAndExecuteFlow)
self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()}
peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: id.String()}
```

Both `self` and `peer` at lines 81-82 remain `TypeWebchat`/`TypeWebchat`,
exactly as required by §4.3 — these were NOT accidentally changed to
`TypeWebSession`. The commit message itself flags this explicitly
("self/peer locals feeding ConversationV1ConversationCreateAndExecuteFlow
left unchanged (still TypeWebchat/TypeWebchat, out of scope)"), and the code
confirms the claim. ✅ This is the single highest-risk line in the whole
change set (easy to "clean up" by mistake) and it was handled correctly.

## 5. `validateReferrer` vs `validatePageURL` — byte-for-byte logic comparison

```go
func validatePageURL(pageURL string) error {
	if len(pageURL) > 2048 { ... "page_url exceeds maximum length..." }
	if pageURL == "" { return nil }
	if !strings.HasPrefix(pageURL, "http://") && !strings.HasPrefix(pageURL, "https://") { ... }
	return nil
}

func validateReferrer(referrer string) error {
	if len(referrer) > 2048 { ... "referrer exceeds maximum length..." }
	if referrer == "" { return nil }
	if !strings.HasPrefix(referrer, "http://") && !strings.HasPrefix(referrer, "https://") { ... }
	return nil
}
```

Identical structure: 2048-char cap, empty-is-valid short-circuit, http/https
scheme allowlist via `strings.HasPrefix`. Test table in
`webchat_session_test.go` (`Test_validateReferrer`) mirrors
`Test_validatePageURL`'s cases 1:1 (empty/normal/exactly-2048/2049/js-scheme/
data-scheme/ftp-scheme/http/https), all asserting `errors.Is(err,
serviceerrors.ErrInvalidArgument)`. Ran `go test ./pkg/servicehandler/...` →
ok. ✅

## 6. RPC/data-flow chain — traced end to end

```
square-admin client.js (_doStart)
  → POST /v1/webchat_sessions {..., referrer: document.referrer || undefined}
bin-api-manager server/webchat_sessions.go (PostWebchatSessions)
  → req.Referrer *string → referrer := "" ; if req.Referrer != nil { referrer = *req.Referrer }
bin-api-manager servicehandler/webchat_session.go (WebchatSessionCreate)
  → validateReferrer(referrer) → h.reqHandler.WebchatV1SessionCreate(ctx, ownerCustomerID, widgetID, pageURL, referrer)
bin-common-handler requesthandler/webchat_session.go (WebchatV1SessionCreate)
  → wcrequest.V1DataSessionsPost{..., Referrer: referrer} → RabbitMQ RPC POST /v1/sessions
bin-webchat-manager listenhandler/v1_sessions.go (processV1SessionsPost)
  → h.sessionHandler.Create(ctx, req.CustomerID, req.WidgetID, req.PageURL, req.Referrer)
bin-webchat-manager sessionhandler/create.go (Create)
  → session.Session{..., Referrer: referrer, Peer: {...}, Local: {...}} → h.db.SessionCreate(ctx, s)
bin-webchat-manager dbhandler/session.go (SessionCreate)
  → commondatabasehandler.PrepareFields(s) (reflection over `db:"..."` tags) → INSERT
```

No hop drops the value — confirmed by reading every intermediate signature
change in the diffs (`main.go` interface signatures, `mock_main.go`
regenerated call args, `V1DataSessionsPost` struct field) and by the new
`Test_Create_ReferrerPeerLocal` test in `bin-webchat-manager`, which asserts
`mockDB.EXPECT().SessionCreate(...)` is called with the referrer value
threaded all the way from `h.Create(ctx, customerID, widgetID, "", referrer)`
into the persisted `Session` struct.

## 7. Peer/Local nullable DB column vs non-omitempty Go struct — no contradiction

`Session.Peer`/`Session.Local` have `json:"peer"`/`json:"local"` (no
`omitempty`) and `db:"peer,json"`/`db:"local,json"`. The DB columns are
`JSON NULL` (migration `80ddd8772905`). This is consistent, not
contradictory: nullability is a DB-column property covering **pre-existing
rows** (created before this migration, where the column legitimately has no
value); the Go struct's non-omitempty guarantee only applies to rows created
**by the current code path**, which always computes both fields
unconditionally in `sessionhandler.Create()`. The migration file's own doc
comment states this rationale explicitly and it matches §4.4's round-1
decision. No backfill was attempted or needed, and none was claimed. ✅

## 8. Mock regeneration completeness

Checked every mock touched by an interface-signature change:
- `bin-common-handler/pkg/requesthandler/mock_main.go` — `WebchatV1SessionCreate` mock signature updated (5 args incl. `referrer`), both `Call` and `Recorder` methods. ✅
- `bin-webchat-manager/pkg/sessionhandler/mock_main.go` — `Create` mock signature updated (5 args incl. `referrer`). ✅
- `bin-api-manager/pkg/servicehandler/mock_main.go` — `WebchatSessionCreate` mock signature updated (5 args incl. `referrer`). ✅

No stale mock found — grepped for old 4-arg call sites of any of these three
methods across the whole worktree; none exist outside the updated files.
`go build ./...` succeeds in all three services (a stale mock would fail to
compile against the new interface).

## 9. OpenAPI contract — additive, non-breaking

`referrer`/`peer`/`local` are all added as optional (`*string`/`*CommonAddress`,
`omitempty` in the generated Go types, no `required:` entry added in the YAML
for `referrer` on `PostWebchatSessions`). Existing consumers unaffected. The
`51af257e7` commit's `oapi-codegen` tool-directive fix
(`go run -mod=mod ... oapi-codegen` → `go tool ... oapi-codegen`) is an
unrelated but harmless drive-by fix bundled into the same commit — mildly
against "one logical change per commit" but low-risk and clearly labeled in
the commit message ("fix oapi-codegen tool directive"). Not blocking.

## 10. JS side — referrer capture, rendering, and rename regression check

- `client.js` `_doStart()`: `referrer: (typeof document !== 'undefined' &&
  document.referrer) || undefined` — correctly omits the key entirely (not
  empty string) when absent, matching `page_url`'s existing pattern. Verified
  by the "omits referrer... when empty" and "...when document is undefined"
  tests, both passing.
- `message_timeline.js`: new "Came from" line reads `session.referrer`,
  gated through the renamed `isSafeURL`/`truncateURL` (previously
  `isSafePageURL`/`truncatePageURL`), reusing the exact same http(s)-only
  guard and 60-char truncation logic as the pre-existing "Started from"
  line. The rename is a pure identifier rename with call-site updates only —
  no logic change — confirmed by diffing the function bodies (unchanged
  except naming) and by the "Started from" tests still passing unmodified in
  behavior (only one assertion changed, from `getByRole('link')` singular to
  a name-scoped query, because there are now two links on the page).
- Ran the actual test suite (`react-scripts test`, not raw `npx jest` — CRA's
  Babel config is required, `npx jest` fails with a preset-parse error):
  **41/41 tests passed** across both `message_timeline.test.js` and
  `client.test.js`, including 4 new referrer-specific cases (present,
  truncated, absent, javascript-scheme-rejected) and 3 new client-side
  capture cases.

## 11. Commit hygiene

- No AI attribution/co-authorship lines in any of the 8 Go commits or the 1 JS commit (`git show -s --format='%b'` checked for all).
- All 8 Go commits + the JS commit authored by `Sungtae Kim <pchero21@gmail.com>`.
- Commit titles match branch name (`NOJIRA-webchat-session-referrer-peer-local`) for the Go commits; the JS commit title (`Add webchat session referrer capture and display`) does not literally match the JS branch name string but the JS repo's CLAUDE.md format requires only a descriptive title + `project-name:` bullets, which this commit follows correctly (`square-admin:` prefix on all three bullets).

## 12. Code quality

- Error handling: `validateReferrer` follows the exact existing
  `serviceerrors.ErrInvalidArgument`-wrapping convention; `sessionhandler.Create()`
  propagates `h.db.SessionCreate` errors unchanged (no swallowed errors).
- Nil checks: `server/webchat_sessions.go`'s `req.Referrer != nil` guard
  before dereferencing the OpenAPI-generated pointer field is correct and
  matches the existing `req.PageUrl` pattern immediately above it.
- Conventions: `commonaddress` import aliasing, `db:"...,json"` tag usage,
  `Field*` constants, and mock regeneration all follow pre-existing patterns
  in the same files with no stylistic drift.
- Migration hygiene: single head after both new migrations chained correctly
  (`04b99363284c` → `ffa2b1c5d1e6` → `80ddd8772905`), confirmed no branching
  via `grep` across `down_revision`/`revision` pairs.

## 13. Actual test execution (not just read)

```
bin-common-handler:  go test ./models/address/... ./pkg/requesthandler/...     → ok
bin-webchat-manager: go test ./...                                             → ok (incl. real sqlite round-trip in dbhandler)
bin-flow-manager:    go test ./pkg/activeflowhandler/...                       → ok
bin-ai-manager:      go test ./pkg/aicallhandler/...                           → ok
bin-contact-manager: go test ./pkg/contacthandler/...                          → ok
bin-api-manager:     go build ./... ; go test ./pkg/servicehandler/...         → ok
bin-openapi-manager: go build ./...                                            → ok
bin-dbscheme-manager: py_compile on both new migration files                   → ok
bin-webchat-manager: golangci-lint run ./pkg/sessionhandler/... ./models/session/...  → 0 issues
square-admin (JS):   react-scripts test (message_timeline + client)            → 41/41 passed
```

---

## Finding: RST doc prose for `peer`/`local` is factually wrong (non-blocking, but must be fixed before final merge)

`bin-api-manager/docsdev/source/webchat_struct_session.rst` (added in
commit `d4cae2865`) describes the new fields as:

```rst
* ``peer`` (:ref:`Address <common-struct-address>`): The visitor's address as observed by the server (remote IP/port) at session-creation time.
* ``local`` (:ref:`Address <common-struct-address>`): The server-side address (local IP/port) that accepted the session-creation request.
```

This is **not what the field actually contains**. Per the design doc
(§4, §4.2) and the verified code (`sessionhandler/create.go`):

- `Peer` = `{Type: "web_session", Target: <this session's own ID>}` — a
  logical channel-address record (mirroring `kase.Case.Peer`/
  `interaction.Interaction.Peer`'s existing pattern), NOT a network
  remote-IP/port. VoIPBin does not capture visitor IP/port for webchat
  sessions anywhere in this diff.
- `Local` = `{Type: "webchat", Target: <Widget.ID>}` — the widget-channel
  identifier, NOT a server-side local IP/port.

The OpenAPI YAML description for the same fields (added in the SAME
diff, `51af257e7`) gets this right:

```yaml
peer:
  description: "The visitor's own address for this session. type is always \"web_session\", target is this session's own id."
local:
  description: "The widget-channel address this session belongs to. type is always \"webchat\", target is the session's widget_id."
```

So the two pieces of documentation added in this same feature — the
OpenAPI spec description and the RST customer-facing doc — directly
contradict each other about what these fields mean. This looks like
boilerplate copy-pasted from a generic "Address struct" IP/port
description used elsewhere in the docs (e.g. SIP/network-address
contexts) without being adapted for webchat's session/widget-ID
semantics. This is customer-facing documentation
(`docs.voipbin.net`) — a developer integrating against this API reading
the RST page would be actively misled about what to expect in
`peer`/`local`, and would not learn that `peer.type` is always
`"web_session"` / `local.type` is always `"webchat"`, which the OpenAPI
description states but the RST prose omits and contradicts.

**Requested fix:** rewrite the two RST bullet lines to match the OpenAPI
schema's description (or something equivalent, in the RST doc's own
voice), e.g.:

```rst
* ``peer`` (:ref:`Address <common-struct-address>`): The visitor's own address for this session. ``type`` is always ``web_session``, ``target`` is this session's own ``id``.
* ``local`` (:ref:`Address <common-struct-address>`): The widget-channel address this session belongs to. ``type`` is always ``webchat``, ``target`` is the session's ``widget_id``.
```

This is a documentation-only fix (no code/schema/test change needed) and
does not require touching any Go/JS source or regenerating any
mock/OpenAPI artifact — only the `.rst` source and rebuilt `build/html`
+ `build/doctrees` outputs already tracked in this same commit.

---

## Verdict rationale

Every functional code path — Go and JS — was verified against the design
doc and against actual re-executed tests (`go test`, `golangci-lint`,
`react-scripts test`), not just read. The §4.3 scope-boundary risk (the
single easiest place to introduce a real bug in this change set) was
checked directly in source and is correctly handled. The `web_session`
string-value revert and dead-code deletions are complete and verified with
zero stray literals. RPC/data-flow chain integrity, mock regeneration,
migration chaining, and OpenAPI contract additivity are all confirmed
correct.

The one finding — the RST doc's factually wrong `peer`/`local` prose — is a
real, customer-facing documentation defect, not a code-correctness or
test-coverage gap, and is trivially fixable (RST + rebuilt HTML only, no
code changes). Per this review loop's own rules (minimum 3 rounds, 2
consecutive APPROVE required to close), this alone is not something I'm
willing to let slide silently into a closed loop, but it also does not
rise to a level that should block THIS round from being a substantively
clean pass — it should be logged and fixed in the next round.

**VERDICT: CHANGES_REQUESTED**

Required for round 1: fix the two `peer`/`local` RST prose lines in
`bin-api-manager/docsdev/source/webchat_struct_session.rst` to match the
OpenAPI schema's (correct) description, rebuild `docsdev/build/`, and amend
commit `d4cae2865` (or add a follow-up commit) with the corrected doc.
Everything else in this PR is approved as-is.
