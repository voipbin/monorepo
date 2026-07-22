# Round 0 Review: webchat Session Referrer + Peer/Local design

Reviewer: Claude Code (independent verification against live source)
Target doc: `2026-07-22-webchat-session-referrer-peer-local-design.md`

## 1. Factual accuracy against current source (관점 1)

Verified directly against the worktree/main-repo source (not the doc's self-report):

- `session.go` currently has `PageURL string \`json:"page_url,omitempty" db:"page_url"\`` — matches §3.2's premise that `page_url` is already shipped. ✅
- `create.go` lines 78-79 today are exactly:
  ```go
  self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()}
  peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: id.String()}
  ```
  matching §4.4's claim precisely (both `TypeWebchat`, used only for the `ConversationV1ConversationCreateAndExecuteFlow` call). ✅
- `bin-common-handler/models/address/main.go`'s `Type` enum: `TypeNone, TypeAgent, TypeAI, TypeAITeam, TypeConference, TypeEmail, TypeExtension, TypeLine, TypeSIP, TypeTel, TypeWebchat, TypeWhatsApp`. No `TypeWebSession` yet — consistent with §4.1's premise. ✅
- `normalize.go:50` switch is `case TypeNone, TypeAgent, TypeAI, TypeAITeam, TypeConference, TypeExtension, TypeLine, TypeWebchat:` — the doc quotes this as `case TypeNone, TypeAgent, ..., TypeWebchat:` (elided) which is accurate in substance but **the doc's cited line number "normalize.go:50" is correct**, and the case list is correctly characterized as "opaque UUID" treatment. ✅
- `validate.go:33` switch: `case TypeAgent, TypeConference, TypeLine, TypeExtension, TypeWebchat:` → `validateUUID`. Doc quotes this exactly, line number correct. ✅ Note: `TypeSIP`/`TypeEmail`/`TypeTel`/`TypeWhatsApp` go through their own cases; `TypeNone` returns nil directly in `validate.go` (not grouped with the UUID case, differs slightly from `normalize.go` where `TypeNone` IS grouped with the opaque-UUID case) — a minor structural asymmetry between the two switches that the doc doesn't call out, but it doesn't affect the doc's correctness claim (adding `TypeWebSession` to both listed cases is still exactly right).
- `sessions.sql` test schema: confirmed `page_url TEXT` already present, matching §3.2. `referrer`/`peer`/`local` are NOT yet present, consistent with this being pre-implementation. ✅
- `message_timeline.js`: `truncatePageURL`/`isSafePageURL` exist today exactly as named, with the round-1 XSS-fix comment already in place (`isSafePageURL` enforces http/https-only). The commit `0d818afa1` ("`NOJIRA-webchat-session-referrer-page-url`... Add http/https scheme allowlist to validatePageURL") is a **real commit** in `bin-webchat-manager`'s Go-side validator, confirmed via `git show`. §3's claim that this fix must not be re-introduced/missed is accurate and well-grounded. ✅
- `webchat-widget-runtime/client.js`: confirmed `page_url: (typeof window !== 'undefined' && window.location?.href) || undefined` inside `_doStart()`, matching the doc's characterization of the existing pattern that `referrer` should mirror. ✅
- `04b99363284c_webchat_sessions_add_column_page_url.py`: confirmed `ALTER TABLE webchat_sessions ADD COLUMN page_url VARCHAR(2048) NULL` — matches the doc's "mirrors 04b99363284c exactly" claim for the proposed referrer migration. ✅
- `167bebb7c46f` (Case Peer/Local migration): confirmed the three-step nullable→backfill→NOT NULL pattern and the `open_peer_uk` generated-column dependency chain exactly as the doc describes. ✅
- `kase.Case.Peer`/`.Local`: confirmed `db:"peer,json"`/`db:"local,json"`, no `omitempty` on Local, comment "ALWAYS PRESENT in JSON output" — matches doc's characterization. ✅
- `casehandler/getorcreate.go`: confirmed no `NormalizeTarget`/`ValidateTarget` call in the visible `GetOrCreate`/`insertWithRetry` path (Peer/Local are passed through as given by the caller) — the doc's claim that Session's new Peer/Local likewise wouldn't need a `NormalizeTarget` call is directionally consistent, though I could not confirm the doc's specific citation "`getorcreate.go:99`'s cited pattern" resolves to anything meaningful in the CURRENT file (line 99 in the present file is inside `GetOrCreate`'s peer-lock logic, not a `NormalizeTarget` call) — **minor citation looseness**, not a substantive error, since the surrounding claim (no normalize call needed) still holds by inspection.
- `crmIneligiblePeerTypes` locations: confirmed at `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1257-1266` (doc cites :1265 for the `"web_session"` line — actual line is 1265, exact match ✅), `bin-ai-manager/pkg/aicallhandler/tool.go:453` (actual: line 453 `"web_session": {},` — exact match ✅), `bin-contact-manager/pkg/contacthandler/interaction.go:66` (actual: line 66 — exact match ✅). All three are unexported map literals with the `"web_session"` string key, each individually commented `// synthetic type; not in commonaddress.Type enum`. **The doc's file/line citations for this section are all correct, non-hallucinated.**
- Webchat conversations do flow through `TypeWebchat`: confirmed at `bin-conversation-manager/pkg/conversationhandler/event.go:31` (`case conversation.TypeWebchat:`) and `pkg/messagehandler/send.go:39`/`191`, plus `event_webchat.go:88-89` where `self`/`peer` addresses for the webchat-originated Conversation are literally `commonaddress.Address{Type: commonaddress.TypeWebchat, ...}` for BOTH self and peer — the string `"web_session"` genuinely never appears anywhere in this conversation-tagging path. ✅

**Verdict on relevo 1: no hallucinated file paths, line numbers, or code quotes found.** This is an unusually well-grounded design doc — every checked citation resolved to real, matching code.

## 2. §4.1 "web_session" collision analysis (관점 2)

The claim that the three `crmIneligiblePeerTypes` maps are non-interacting with a new `commonaddress.TypeWebSession = "web_session"` enum member is **correct and verifiable**:

- The three maps are `map[commonaddress.Type]struct{}` keyed by string literals, including the bare literal `"web_session"` (not a symbolic reference to any enum member, since no such member exists today).
- These maps are consulted ONLY via `isCRMEligiblePeer(peerType commonaddress.Type)`, which is called (per each file) against the peer type attached to **Call**/**Conversation-message**-derived Cases, i.e. `deriveEndpointsForCase`'s output from `source`/`dest` addresses that `bin-call-manager`/`bin-conversation-manager` populate. For webchat, the peer/self addresses supplied to `CreateAndExecuteFlow`/`event_webchat.go` are hard-coded to `TypeWebchat`, never `"web_session"` in the current codebase.
- Therefore: introducing `commonaddress.TypeWebSession = "web_session"` as a REAL enum value does not change `isCRMEligiblePeer`'s behavior for any TODAY-existing call site, because none of them currently produce a peer address whose `Type` field is the string `"web_session"`. The doc's claim holds.

However — and this is the one substantive gap I'd flag — the doc's own §4.2/§4.4 do NOT propose ever setting `Interaction.Peer.Type` or `Case.Peer.Type` to `TypeWebSession`. If a *future* engineer, motivated by this design's own §4.3 "if a future need... to ALSO use TypeWebSession emerges" note, wires `TypeWebSession` into `bin-flow-manager`'s Call/Conversation peer derivation (e.g., because Session.Peer now legitimately carries that type and someone forwards it into a Case/Interaction Peer without re-reading §4.3's scope boundary), `isCRMEligiblePeer` would then classify it as **CRM-ineligible** (since `"web_session"` is already blacklisted in the map) — which may or may not be the intended behavior for a real webchat-visitor peer at that point. This is a forward risk, not a defect in the present design's own claims, and the doc already flags a related open question (§6.1). Worth an explicit one-line note that "if Case/Interaction ever adopt TypeWebSession, the three ineligibility maps' existing `\"web_session\"` entry will need re-evaluation" — currently implicit, not stated.

## 3. §4.4 NOT NULL / backfill / BINARY(16) tradeoff (관점 3)

- The `id`/`widget_id` columns are confirmed `binary(16)` in the current `sessions.sql`/migration files (`id binary(16)`, `widget_id binary(16)`). The doc's point that `HEX(id)` alone would NOT produce a canonical UUID string (it would produce a 32-char hex blob without dashes, e.g. `550e8400e29b41d4a716446655440000` instead of `550e8400-e29b-41d4-a716-446655440000`) is **technically correct** — MySQL's `HEX()` on a `BINARY(16)` UUID gives the raw hex with no dash formatting, so a naive `HEX(id)` backfill would silently produce Target values that don't match the format `uuid.FromStringOrNil` (used in `validateUUID`, confirmed in `validate.go:67-71`) can parse. If `ValidateTarget`/`validateUUID` is ever run against a backfilled row's Peer.Target, an undashed hex string would fail `uuid.FromStringOrNil` (returns `uuid.Nil` for non-canonical input) and trip `"invalid uuid format"`. This is a real, correctly-identified risk, not overengineering.
- The doc's fallback recommendation (nullable-at-DB-level, matching Case's own precedent where `Local` is nullable while JSON-required at the app layer) is consistent with the ACTUAL `167bebb7c46f` migration, which keeps `local` (but not `peer`) permanently nullable — confirmed. This is a legitimate, precedent-following simplification.
- One thing worth double-checking in round 1: the doc's asserted `DefaultSessionIdleTimeout` (1800s / 30 min) is accurate (`widget.go:162`, confirmed), so the "short-lived, high-churn" framing used to justify skipping a UUID-formatting backfill is grounded in a real number, not an assumption.

Conclusion: §4.4's technical analysis is sound and not overengineered; the BINARY(16)-formatting concern is real.

## 4. normalize.go/validate.go switch necessity (관점 4)

Confirmed both switches are literally exhaustive over `commonaddress.Type` (both have a `default:` branch returning `ErrUnknownType`/an "unknown address type" error) — so YES, adding `TypeWebSession` to both is strictly required or `NormalizeTarget`/`ValidateTarget` will error for the new type the moment any caller invokes them with it, exactly as the doc states. Current case lists (re-confirmed above) match the doc's quotes verbatim, including the correct line numbers.

## 5. §3 referrer / page_url symmetry vs. actual XSS-fixed implementation (관점 5)

- The referrer design in §3 mirrors the *current, XSS-fixed* `page_url` implementation faithfully: same 2048-char cap, same http/https-scheme-only validation pattern (`validatePageURL` exists and is XSS-hardened per commit `0d818afa1`, confirmed), and §3.3's proposed rename of `truncatePageURL`/`isSafePageURL` → `truncateURL`/`isSafeURL` is a reasonable no-behavior-change refactor since both helpers are string-only concerns with no page_url-specific logic inside them (confirmed by reading the current implementation — `isSafePageURL` only checks `startsWith('http://'...)`, nothing page_url-specific).
- The `document.referrer` `|| undefined` empty-string-guard logic (§3.1) is correct JS behavior: `document.referrer` is `""` (not `undefined`) when there's no referrer, and the guard converts that to `undefined` so `JSON.stringify` omits the key — consistent with how `page_url` is already handled in `client.js` (`window.location?.href) || undefined`).

No inconsistency found between §3's design and the actual shipped page_url implementation.

## 6. §4.3 scope boundary + §6 open questions as review material (관점 6, 7)

§4.3's decision to exclude Case/Interaction from this round is well-reasoned (cites the real generated-column dependency chain, the real broader blast radius across bin-flow-manager/bin-ai-manager/bin-conversation-manager, and pchero's own standing CPO principle against consistency-only schema changes) and appropriately left as a documented, explicit non-goal rather than silently dropped. This is acceptable as-is; no changes requested on §4.3 itself.

**However, §6 is the deciding issue for this round's verdict.** The doc explicitly lists three open questions and, in two of the three (§6.1 the `"web_session"` string-literal choice, §6.3 NOT NULL-vs-nullable), the body text ALREADY states a recommendation ("Recommend resolving this explicitly in round 1" / "Recommendation to simplify, flagged for round 1 discussion"). This is a document that self-identifies as incomplete — by its own words it is asking the round-1 reviewer to make design decisions that are properly the author's to make BEFORE the design is submitted for implementation sign-off, not decisions to be resolved live during code review.

This matters concretely:
- §6.3 in particular is not a stylistic open question — it changes the DB migration's actual DDL (nullable `peer`/`local` vs. `NOT NULL` + backfill), the Go struct's zero-value semantics story, and whether a backfill script is written at all. An implementer cannot start on the DB/migration/model layer without this being settled.
- §6.1 changes the literal string constant shipped in the enum (`"web_session"` vs. e.g. `"webchat_visitor"`), which is also implementation-blocking, not cosmetic, once any client/API/test/documentation encodes the chosen string.
- §6.2 is genuinely closer to "product judgment call, may reasonably ship as documented, non-blocking" — but the other two are blocking.

## Verdict rationale

The document is exceptionally well-verified against the actual codebase — I found zero hallucinated file paths, function names, line numbers, or misdescribed code behavior across every citation checked, which is uncommon and worth acknowledging. The technical analysis in §4.1 (collision check) and §4.4 (BINARY(16)/backfill) is correct and appropriately cautious rather than hand-wavy.

But a design doc that leaves two implementation-blocking questions open, with the author's own text recommending they be "resolved explicitly in round 1" rather than deciding them itself, is not yet ready for APPROVED. Approving it as-is would either (a) implicitly force this reviewer to make the author's design decisions unilaterally mid-review, which conflates review with authorship, or (b) let an implementer start work against an ambiguous contract (unclear whether `peer`/`local` are NOT NULL or nullable; unclear what string literal ships). Round 0 is exactly the right stage to send this back for the author to pick a position on §6.1 and §6.3 (§6.2 can reasonably stay open/non-blocking, or be resolved the same round) and resubmit as round 1.

**Requested changes:**
1. §6.1: resolve the `"web_session"` string-literal choice — either keep it (with the non-interaction proof already written up in §4.1 as the justification) or pick an alternative literal. Do not leave this to the reviewer.
2. §6.3: resolve NOT NULL+backfill vs. nullable-at-DB-level for `peer`/`local`, and update §4.4's "Database" paragraph and §5's migration-file checklist to reflect the single chosen approach (right now §4.4 describes BOTH the NOT NULL+backfill path and the nullable alternative in the same paragraph, which is not an implementable spec).
3. (Minor, non-blocking) §4.4's citation "`getorcreate.go:99`'s cited pattern" does not resolve to a `NormalizeTarget` call at that line in the current file — tighten or drop this specific line citation in the next revision so it doesn't mislead an implementer who checks it.

VERDICT: CHANGES_REQUESTED
