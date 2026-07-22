# Review: Revision 1 (§8), Round 3

Target: `2026-07-22-webchat-session-referrer-peer-local-design.md` §8
("string value reverts to `web_session`; three dead `"web_session"` map
entries deleted")

Scope: this is the **second-consecutive-APPROVE check** round (round 2 was
the first APPROVED verdict after two CHANGES_REQUESTED rounds). Per this
repo's two-consecutive-approvals protocol, this round is a full, independent,
adversarial re-read of §1-§8, not a diff against round 2's findings, and no
prior round's "Confirmed" claim (including round 2's own) is trusted without
being re-derived from live source.

## 1. Independent re-verification of round 2's central claim: is
   `interaction_read.go:342` / `dbhandler/interaction.go:387,423`'s
   `web_session` really unrelated to `commonaddress.Type`?

This is the specific item the task brief calls out for direct re-check, so it
was re-traced from scratch (not read from round 2's narrative) against live
source in this worktree:

- `bin-contact-manager/pkg/dbhandler/interaction.go:423` — the string appears
  only inside a raw SQL predicate: `AND i.peer_type != 'web_session'`, where
  `peer_type` is a **VARCHAR generated column** on `contact_interactions`
  (confirmed via `bin-dbscheme-manager/bin-manager/main/versions/b41d1b2317af_contact_interactions_peer_local_address_.py`:
  `peer_type VARCHAR(255) GENERATED ALWAYS AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.type'))) STORED NOT NULL`).
  This column is a plain string derived from JSON, not a Go
  `commonaddress.Type` enum member at the database layer — MySQL has no enum
  type binding to `commonaddress.Type` at all.
- `bin-contact-manager/models/interaction/interaction.go` (the Go struct
  backing this table) confirms `Peer commonaddress.Address` is the only
  typed field; `peer_type`/`peer_target` are DB-only generated columns with
  no corresponding Go struct field, so no Go code ever compares a
  `commonaddress.Type` constant against the literal `"web_session"` via this
  path — the comparison happens entirely in SQL text against a string column.
- Traced where `contact_interactions.peer_type = 'web_session'` values would
  actually originate: `docs/plans/2026-06-26-add-contact-crm-interaction-timeline-design.md`
  §6 ("web_session handling") and its interaction-source table (line ~234)
  define `web_session` as the **AIcall** grain key for a web-direct AIcall
  messaging conversation (`reference_type=aicall`) — populated as a raw
  string identifier, never through `commonaddress.Address.Type` construction.
  This is a completely independent concept from `commonaddress.TypeWebSession`
  (this design's new address-type enum member for `bin-webchat-manager`'s
  `Session.Peer`).
- Confirmed (independently, not reusing round 2's grep) that
  `bin-contact-manager/pkg/contacthandler/` today defines only
  `EventCallCreated`, `EventConversationMessageCreated`, and
  `EventCustomerDeleted` — no `EventAIcall*` handler exists yet. So no live
  code path currently writes `peer_type = 'web_session'` via any
  `commonaddress.Type` construction at all; the SQL predicate in
  `interaction_read.go`/`dbhandler/interaction.go` is a forward-looking
  guard against a data shape the AIcall-web projection design anticipates,
  written directly as a SQL string literal, structurally disconnected from
  `commonaddress.Type`.

**Conclusion, independently re-derived**: round 2's finding holds. This is a
different, pre-existing concept (CRM interaction-timeline's AIcall-session
`peer_type` column value) that happens to share the literal string
`"web_session"` by coincidence, not by shared type, shared column, or shared
code path with `commonaddress.TypeWebSession`. §8's scope boundary (deleting
`"web_session"` only from the three `commonaddress.Type`-keyed
`crmIneligiblePeerTypes`-family maps, and correctly not touching
`interaction_read.go`/`dbhandler/interaction.go`) is correct. No action
required on this design's part.

## 2. Independent re-verification of the three map-entry deletions and their test fallout

Re-read all five affected files directly (not relying on any prior round's
line-number citation):

- `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1265` —
  `"web_session": {}, // synthetic type; not in commonaddress.Type enum`
  inside `crmIneligiblePeerTypes` (lines 1257-1266), every other entry a real
  `commonaddress.Type` constant. Matches §8.3.
- `bin-ai-manager/pkg/aicallhandler/tool.go:453` — identical entry inside
  `caseCreateCRMIneligiblePeerTypes` (lines 445-454). Matches §8.3.
- `bin-contact-manager/pkg/contacthandler/interaction.go:66` — identical
  entry inside `crmIneligiblePeerTypes` (lines 58-67). Matches §8.3.
- `bin-contact-manager/pkg/contacthandler/interaction_crm_eligibility_test.go:36`
  — `{"web_session is ineligible", commonaddress.Type("web_session"), false}`,
  confirmed verbatim. This assertion inverts to `true` once the map entry is
  gone — §8.4 correctly flags it.
- `bin-contact-manager/pkg/contacthandler/interaction_test.go:247-266`
  (`Test_EventConversationMessageCreated`, "outgoing web_session message"
  case) — confirmed verbatim: `Source: commonaddress.Address{Type:
  "web_session", Target: "web_session:xyz"}`, `expectInteraction: nil`. No
  `mockDB.EXPECT().InteractionCreate(...)` is configured for this case
  (confirmed by reading the test harness's conditional mock-setup block,
  which only registers the expectation when `expectInteraction != nil`).
  Post-deletion, `isCRMEligiblePeer("web_session")` returns `true`, so
  `EventConversationMessageCreated` proceeds to call `h.db.InteractionCreate`
  against a mock with no such expectation — a gomock unexpected-call panic,
  exactly as §8.2/§8.4 describe. §8.4's ordering warning (fix this test
  BEFORE the map-entry deletion lands) is present and correctly sequenced.

Checked for any other file referencing the bare string `web_session` in Go
source across the whole worktree (independent repo-wide grep, not scoped to
any one service): exactly the same seven hits round 2 found —
the three map entries, the two contact-manager test files, and the two
unrelated SQL-comment/predicate sites in `interaction_read.go`/
`dbhandler/interaction.go` addressed in §1 above. No eighth hit exists. §8.4's
checklist is complete; no missing file.

## 3. Full top-to-bottom re-read, §1-§6 and §8 (treating no prior round's verdict as trustworthy)

- **§1-§3** (`referrer` feature): no `webchat_visitor`/`web_session`
  literal appears in these sections at all — confirmed by the full-document
  grep in §4 below. Nothing for Revision 1 to touch here, and nothing stale
  found.
- **§4.1**: `[SUPERSEDED BY §8 REVISION 1]` marker (lines 151-158) sits
  immediately before the `"webchat_visitor"` code snippet (line 161),
  correctly pointing to §8.3. Verified the marker precedes the snippet in
  reading order (not after), so a top-to-bottom reader hits the warning
  before the stale literal.
- **§4.2**: `[SUPERSEDED BY §8 REVISION 1]` marker (lines 228-236) sits
  immediately before the `Peer`/`Local` field-comment snippet (lines
  238-241) and explicitly names both remaining `webchat_visitor` mentions
  in the section ("the type-pair comparison, the 'Web visitor'
  render-label example") — confirmed those two mentions (lines 246, 251)
  are the only ones after the marker, and both are the ones the marker
  names. No unmarked residue.
- **§4.3, §4.4, §5**: independently re-confirmed (grep, §4 below) — zero
  `webchat_visitor`/`web_session` literal occurrences in these sections.
  They reference only the symbolic constant `commonaddress.TypeWebSession`,
  which is unchanged by the revert (§8.3 explicitly notes the Go symbol
  name is unchanged, only its underlying string value reverts). No
  staleness possible here.
- **§6 item 1**: "RESOLVED (at round 1) -- `webchat_visitor`..." immediately
  followed by its own `[SUPERSEDED BY §8 REVISION 1]` note (lines 467-472)
  pointing to §8.1-§8.3. §6 items 2 and 3 do not mention the type-string
  value at all and need no marker.
- **§8.1-§8.5**: internally consistent — §8.1 states the rationale, §8.2
  the dead-code audit (verified accurate above), §8.3 the updated spec
  (verified accurate), §8.4 the file-checklist addendum (verified complete
  above), §8.5 correctly frames this revision as requiring its own fresh
  2-consecutive-APPROVE loop rather than inheriting the original closure.

## 4. Fresh full-document grep for both literals (independent recount, not trusting round 2's count)

- `webchat_visitor`: 18 hits total. Classified every one independently:
  151, 153, 155, 161, 165, 193, 204, 207, 213 (§4.1, inside or immediately
  following the marker's stated scope "through the end of §4.1"); 229, 234,
  246, 251 (§4.2, inside/covered by its marker); 463, 499, 506, 522 (§6
  item 1 / §8.1, all in already-marked or explicitly-historical prose); 611,
  613 (§8.3's own closing summary sentence, explicitly framed as historical).
  Every occurrence is either inside a `[SUPERSEDED BY §8 REVISION 1]`-marked
  block or inside §8 itself (which IS the superseding section and needs no
  marker for its own prose). Zero unmarked occurrences. This independently
  reproduces round 2's count and classification exactly — no discrepancy
  found on a third independent pass.
- `web_session` (the literal string, doc-wide): appears throughout §8 as the
  current authoritative value (expected, no issue) and inside §4.1/§4.2's
  marker text itself (also expected — the markers explicitly state the
  reverted-to value for context). No occurrence outside §8 or the markers
  presents `"web_session"` as anything other than the superseding/current
  value, so there is no inverse staleness risk (i.e., no spot incorrectly
  implies `"webchat_visitor"` is still live).

## 5. Composition check: does §8 combine cleanly with §1-§6 as one system?

- §8.3 explicitly states "§5's file checklist and §4.4's Go/DB field
  definitions are otherwise unaffected (same fields, same nullable-at-DB-level
  decision, same RPC-threading files)" — verified true: §4.4's DB migration
  (`ALTER TABLE webchat_sessions ADD COLUMN peer JSON NULL, local JSON NULL`)
  and §4.4's `NormalizeTarget`/`ValidateTarget` switch-exhaustiveness
  requirement reference only the symbolic `commonaddress.TypeWebSession`
  constant, never the literal string — so neither needs any edit for the
  string-value revert. No migration conflict, no field-tag conflict.
- §8.4's cross-service test-fix additions (bin-flow-manager,
  bin-ai-manager, bin-contact-manager) are non-overlapping with §5's
  original `bin-webchat-manager`/`bin-common-handler`/`bin-api-manager`
  file list — §8 adds new files to the checklist rather than
  contradicting any §5 entry.
- No shared migration, shared mock, or shared test file is touched by both
  the original (§1-§6) feature build-out and the Revision 1 (§8) string
  revert + dead-code deletion — the two change-sets are cleanly separable
  in the implementation checklist, consistent with §8.3's own claim.

## Verdict rationale

This round performed a full independent re-derivation (not a review of round
2's narrative) of every substantive claim in §8, plus the specific
re-verification the task brief asked for (`interaction_read.go`/
`dbhandler/interaction.go`'s `web_session` really is a structurally distinct,
non-`commonaddress.Type` concept — confirmed by reading the generated-column
migration DDL and the Go struct directly, not just repeating the prior
rounds' prose). Findings:

1. The task brief's specific re-check target — whether `web_session` in
   `interaction_read.go:342`/`dbhandler/interaction.go:387,423` is unrelated
   to `commonaddress.Type` — is confirmed true, traced this round through
   the actual generated-column DDL (`peer_type VARCHAR(255) GENERATED ALWAYS
   AS (JSON_UNQUOTE(JSON_EXTRACT(peer, '$.type'))) STORED`) and the AIcall-web
   projection design doc, not just re-asserted from a prior round's summary.
2. All three `crmIneligiblePeerTypes`-family map deletions and their two
   test-fallout files are re-confirmed accurate, complete, and correctly
   sequenced (test fix before map-entry deletion).
3. A fresh, independent 18-occurrence classification of `webchat_visitor`
   across the whole document finds zero unmarked residue — every mention is
   inside a `[SUPERSEDED BY §8 REVISION 1]`-marked block or inside §8 itself.
4. §1-§6 show no new contradiction or staleness introduced by §8; §4.3/§4.4/§5
   reference only the symbolic constant and needed no edit, correctly.
5. §8 composes cleanly with the rest of the document — no shared
   migration/test/mock conflict between the original feature build-out and
   the Revision 1 revert.

No defects found. This is the **second consecutive APPROVED verdict**
(round 2 was the first). Per this repo's two-consecutive-APPROVE closure
convention, **Revision 1 (§8) is now closed** — no further review round is
required for §8 unless it is edited again.

VERDICT: APPROVED
