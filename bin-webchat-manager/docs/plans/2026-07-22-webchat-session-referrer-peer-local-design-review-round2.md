# Round 2 Review: webchat Session Referrer + Peer/Local design

Reviewer: Claude Code (independent verification against live source, fresh top-to-bottom pass)
Target doc: `2026-07-22-webchat-session-referrer-peer-local-design.md` (round 2, claims round 1's finding fixed)
Prior reviews:
- `2026-07-22-webchat-session-referrer-peer-local-design-review-round0.md` — CHANGES_REQUESTED (§6.1/§6.3 open, `getorcreate.go:99` citation)
- `2026-07-22-webchat-session-referrer-peer-local-design-review-round1.md` — CHANGES_REQUESTED (§4.2 lines 227/232 still said `web_session`, contradicting §4.1's `webchat_visitor` decision)

This round is an independent, whole-document pass (not a diff-only re-check of round 1's single finding), per the review-loop convention that Round 2 is the first round where APPROVED is possible, and per the request to re-verify the full doc rather than assume prior rounds' section-by-section coverage was exhaustive.

## 1. Did round 1's requested change land, and is it now consistent?

Full-document grep for `web_session` (with and without backticks) — 7 matches remain, all in §4.1 and §6, all *correctly* contextual (explaining why `"web_session"` was rejected in favor of `"webchat_visitor"`):

- L156, L159, L179, L184, L186, L203 (§4.1 rationale/collision-avoidance narrative)
- L445 (§6 "RESOLVED -- `webchat_visitor`, not `web_session`" summary)

§4.2 (the section round 1 flagged) now reads, at the two previously-broken lines:

- L227: "Peer is now type-distinguishable from Local (`webchat_visitor` vs `webchat`)"
- L232: "`webchat_visitor` -> \"Web visitor\" label"

Both now match §4.1's committed literal. **Round 1's finding is fixed, and no new `web_session` leftovers were introduced elsewhere in the document.** ✅

## 2. Full document re-read, §1–§7, for contradictions not caught by prior rounds

Read the entire document top to bottom independently (not scoped to §4):

- **§1 Problem / §2 Goals**: internally consistent; the "supersedes the CPO's earlier verbal rejection" framing in §1 is echoed correctly by §4.2's "zero information" discussion — no drift between the two.
- **§3 (`referrer`)**: unchanged from round 0/round 1 content; still describes the identical page_url-mirroring design, the `0d818afa1` XSS-fix precedent, and the `|| undefined` empty-string guard. No new claims introduced this round in this section, nothing to re-litigate.
- **§4.1**: `TypeWebSession Type = "webchat_visitor"` is the single, unambiguously stated literal throughout — confirmed live: `bin-common-handler/models/address/main.go` does not yet define `TypeWebSession` (still pre-implementation, expected), and the enum block only has `TypeNone` through `TypeWhatsApp`, no collision. ✅
- **§4.2**: now internally consistent with §4.1 (see §1 above). The "open question carried into round 1" framing is itself carried correctly into §6 (see §4 below).
- **§4.3**: scope boundary (Case/Interaction unchanged) re-read fully — consistent with §4.1's forward-looking note ("if Case/Interaction ever adopt `TypeWebSession`... the `crmIneligiblePeerTypes` maps' existing `web_session` entries would need re-evaluation") without contradiction.
- **§4.4**: Go struct snippet, DB nullable decision, and BINARY(16)/backfill rationale re-read fully; no internal contradiction with §4.1–§4.3.
- **§5 (file checklist)**: cross-checked against §3/§4's field lists — `referrer`, `peer`, `local` all appear consistently; `sessions.sql` line correctly annotated "per §4.4's round-1 decision" (nullable, no NOT NULL language anywhere).
- **§6**: see dedicated check below.

No new logical inconsistency found in this fresh full read.

## 3. Full re-verification of every code/line citation against current source (not the doc's self-report)

- `bin-webchat-manager/models/session/session.go` (current): confirmed `PageURL string \`json:"page_url,omitempty" db:"page_url"\`` exists, `Referrer`/`Peer`/`Local` do NOT exist yet — matches doc's pre-implementation premise. ✅
- `bin-webchat-manager/pkg/sessionhandler/create.go` (current): confirmed lines 78-79 are exactly
  ```go
  self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: widgetID.String()}
  peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: id.String()}
  ```
  matching §4.4's claim verbatim (both `TypeWebchat`, feeding only `ConversationV1ConversationCreateAndExecuteFlow`). The doc's warning not to conflate these with the new `Session.Peer`/`Session.Local` fields is well-founded — they are indeed a separate computation in the same function. ✅
- `bin-common-handler/models/address/main.go` (current): enum is `TypeNone, TypeAgent, TypeAI, TypeAITeam, TypeConference, TypeEmail, TypeExtension, TypeLine, TypeSIP, TypeTel, TypeWebchat, TypeWhatsApp` — no `TypeWebSession`, consistent with §4.1's premise (this is new). ✅
- `bin-common-handler/models/address/normalize.go:50`: confirmed `case TypeNone, TypeAgent, TypeAI, TypeAITeam, TypeConference, TypeExtension, TypeLine, TypeWebchat:` → identity passthrough. Doc's characterization ("opaque UUID" treatment, needs `TypeWebSession` added) is accurate. ✅
- `bin-common-handler/models/address/validate.go:33`: confirmed `case TypeAgent, TypeConference, TypeLine, TypeExtension, TypeWebchat:` → `validateUUID`. Doc's characterization and required addition both accurate. ✅
- `bin-contact-manager/models/kase/kase.go`: confirmed `Peer commonaddress.Address \`json:"peer" db:"peer,json"\`` (no omitempty) and `Local commonaddress.Address \`json:"local" db:"local,json"\`` with the comment "ALWAYS PRESENT in JSON output (no `omitempty`...)" at line 35 — matches §4.4's precedent claim exactly, including the field-ordering rationale (identity fields, then Peer/Local). ✅
- `bin-webchat-manager/models/widget/widget.go:162`: confirmed `const DefaultSessionIdleTimeout = 1800` (30 min) — matches §4.4's "short-lived, high-churn table" justification for skipping backfill. ✅
- `bin-webchat-manager/scripts/database_scripts_test/sessions.sql`: confirmed no `referrer`/`peer`/`local` columns exist yet (pre-implementation state), consistent with the doc's premise; `id`/`widget_id`/`customer_id`/`activeflow_id` are all `binary(16)`, supporting §4.4's BINARY(16)-to-UUID-string backfill-formatting argument. ✅
- `bin-dbscheme-manager/.../04b99363284c_webchat_sessions_add_column_page_url.py`: confirmed exists, matching the "mirrors 04b99363284c" claim for the referrer migration in §3.2. ✅
- `getorcreate.go:99` citation: re-confirmed absent from the full document text (grepped) — round 0's finding stays resolved. ✅

No hallucinated file paths, function names, or line numbers found in this independent re-check. `webchat-widget-runtime/client.js` and the `167bebb7c46f` contact-manager migration (both in repos not present in this local worktree checkout) were previously verified directly in round 0's review and are unchanged in this revision's text — not re-walked here since the doc made no new claims about them this round, but flagging for completeness that they were not independently re-fetched in round 2.

## 4. §6 open-question summary vs. §4.1/§4.2/§4.4 body text — consistency check

- §6.1 ("RESOLVED -- `webchat_visitor`, not `web_session`") accurately reflects §4.1's body, which states the same decision and rationale. ✅
- §6.2 ("§4.2 justification sufficiency: NOT further resolved -- standing caveat") accurately reflects §4.2's own final paragraph ("Open question carried into round 1 review... Flagging honestly rather than overselling it") — the summary neither overstates resolution nor drops the caveat. Consistent. ✅
- §6.3 ("RESOLVED -- nullable at the DB level") accurately reflects §4.4's "Database (Round 1 decision...)" paragraph, and §5's file-checklist line for `sessions.sql` is annotated consistently. ✅

No drift between §6's summary and the sections it purports to summarize.

## 5. Assessment of round 1's non-blocking judgment call (§4.2 left open)

Reaffirmed from round 1: §4.2's open question is a product-scope judgment call (does type-based dispatch alone justify the schema addition), explicitly routed to pchero, and does not block any concrete step in §4.4/§5's implementation checklist. This remains a legitimate deferral, not an authorship dodge.

## Verdict rationale

This round's sole substantive change — replacing the two leftover `web_session` literals in §4.2 with `webchat_visitor` — is verified done and complete; a full-document grep found no remaining occurrences of the string outside its correct, contextual (rejected-alternative) usages in §4.1/§6. A fresh, independent top-to-bottom read of §1–§7 found no new contradictions, and every code/line citation re-checked against current live source in this pass (session.go, create.go, address/main.go, normalize.go, validate.go, kase.go, widget.go, sessions.sql, the page_url migration) matches exactly, with no hallucinated paths or line numbers. §6's open-question summary is faithful to the body text it summarizes in all three sub-items.

I found no remaining defect. This is an implementable, internally consistent spec.

**Process note (not a review defect):** per this repo's design-review-loop convention, sign-off requires two consecutive APPROVED verdicts. This is the first APPROVED (round 2); one more independent APPROVED round (round 3) is still required before this design is considered fully approved for implementation.

VERDICT: APPROVED
