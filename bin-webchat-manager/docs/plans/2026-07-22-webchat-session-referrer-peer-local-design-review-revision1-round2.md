# Review: Revision 1 (§8), Round 2

Target: `2026-07-22-webchat-session-referrer-peer-local-design.md` §8
("string value reverts to `web_session`; three dead `"web_session"` map
entries deleted")

Scope: independent, full top-to-bottom re-read of §1-§8 (not scoped only
to the round-0/round-1 findings), per the brief's instruction to treat
this round as a fresh perspective rather than a continuation of the prior
two rounds' focus areas. All factual claims re-verified against live
source in this worktree, not taken from the document's self-report or
from the prior two review files.

## 1. `webchat_visitor` occurrence sweep (full re-classification)

Repo-wide grep of the document for `webchat_visitor` returns 18 hits
(one more than round 1's reported "16 total" — recount below; the
difference is accounted for, not a discrepancy, see note). Classified:

| Line(s) | Location | Classification |
|---|---|---|
| 151-158 | §4.1 SUPERSEDED marker + snippet | (a) legitimate review history, correctly marked — marker precedes the snippet, states current value lives at §8.3 |
| 161 | §4.1 code snippet itself | covered by the marker at 151-158 immediately above it |
| 164-208 | §4.1 body/rationale text | (a)/(b) — all within "through the end of §4.1" scope explicitly named by the marker at 152 |
| 213 | §4.1 forward-reference to a hypothetical future Case/Interaction unification | (a) — part of §4.1's rationale block, covered by the same marker |
| 228-236 | §4.2 SUPERSEDED marker | (a) marker itself |
| 246, 251 | §4.2 body text ("webchat_visitor vs webchat", "Web visitor" label) | (b) — covered by the §4.2 marker at 228-236, which explicitly names both mentions ("the type-pair comparison, the 'Web visitor' render-label example") |
| 463-464 | §6 item 1 "RESOLVED" line | (a) — immediately followed by its own SUPERSEDED marker (467-472) |
| 506, 522 | §8.1 rationale, discussing the original concern historically | (a) — explicitly framed as history within §8 itself, needs no marker (§8 IS the superseding section) |
| 611, 613 | §8.3's closing paragraph, explicitly stating "every other occurrence... is superseded by web_session per this revision" | (a) — this is the document's own summary sentence, correctly framed |

Every occurrence outside §8 itself is either (i) inside a section that
opens with an explicit `[SUPERSEDED BY §8 REVISION 1]` block naming that
exact mention, or (ii) §6 item 1's own inline superseded note. **No
occurrence is class (c) — none are unmarked.** This independently
confirms round 1's finding that the §4.2 gap it identified (lines
~229-241 in round 1's line numbering, now 228-256 in the current file
after the marker insertion shifted line numbers) has been closed, and
extends the check to the full document rather than just the three spots
round 0 originally named.

(Line-count note: round 1 reported "16 total hits" for a repo-wide grep;
this round's document-only grep returns 18. The difference is fully
explained by round 1 having grepped before the §4.2 marker insertion —
that edit added 2 new `webchat_visitor` mentions inside the marker text
itself, describing what it supersedes. Both prior review files
(round0.md, round1.md) also contain their own `webchat_visitor` prose
quoting the design doc, but those are separate files, not part of this
count.)

## 2. §8.4 file checklist completeness — re-verified against a fresh repo-wide search

Independent repo-wide grep (not scoped to the three services §8.2/§8.4
name) for the bare string `web_session` across all `.go` files in the
worktree returns 10 hits, in exactly these files:

1. `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1265` — map entry, listed in §8.4.
2. `bin-ai-manager/pkg/aicallhandler/tool.go:453` — map entry, listed in §8.4.
3. `bin-contact-manager/pkg/contacthandler/interaction.go:66` — map entry, listed in §8.4.
4. `bin-contact-manager/pkg/contacthandler/interaction_crm_eligibility_test.go:36` — listed in §8.4, exact text confirmed (`{"web_session is ineligible", commonaddress.Type("web_session"), false}`).
5. `bin-contact-manager/pkg/contacthandler/interaction_test.go:247,256,257` — listed in §8.4 (the round-0-added finding), exact text confirmed at the cited line range.
6. `bin-contact-manager/pkg/contacthandler/interaction_read.go:342` — a doc-comment, **not** a `commonaddress.Type` literal (`// Predicate: ... peer_type != web_session`).
7. `bin-contact-manager/pkg/dbhandler/interaction.go:387,423` — SQL predicate `AND i.peer_type != 'web_session'` plus its doc-comment.

Items 6-7 are the same "unrelated, pre-existing concept" round 0 flagged
in its "Other checks performed" section (citing the `docs/plans/2026-06-2[68]-*.md`
family) but did not trace into the actual `.go` implementation. I traced
it independently this round:

- `docs/plans/2026-06-26-add-contact-crm-interaction-timeline-design.md`
  §6 ("web_session handling") defines `web_session` as the **AIcall**
  session id for a web-direct AIcall messaging conversation
  (`reference_type=aicall`), stored as `peer_type`/`peer_target` — a
  plain string value, never routed through `commonaddress.Type` as an
  enum member. This is confirmed structurally distinct from
  `commonaddress.TypeWebSession` (this design's new address-type
  constant): the CRM timeline's `web_session` peer_type is populated
  from a **raw string reference id**, not from any `commonaddress.Address.Type`.
- Confirmed no `EventAIcall*` handler exists in
  `bin-contact-manager/pkg/contacthandler/` today (only
  `EventCallCreated`, `EventConversationMessageCreated`,
  `EventCustomerDeleted` are defined) — so despite the design doc
  describing an AIcall-web projection path, no live code path currently
  writes `peer_type = 'web_session'` into `contact_interactions` via
  `commonaddress.Type` at all; `interaction_read.go:342` /
  `dbhandler/interaction.go:387,423`'s `'web_session'` predicate is a
  defensive/forward-looking guard against a projection source that is
  either not yet implemented or implemented via a path this search did
  not need to resolve, since it is a **plain SQL string literal**
  comparison, not a `commonaddress.Type` enum reference, and is
  therefore untouched by this revision's `TypeWebSession` constant
  regardless of implementation status.
- This confirms §8's implicit scope boundary is correct: it deletes
  `"web_session"` **only** from the three `commonaddress.Type`-keyed
  `crmIneligiblePeerTypes`-family maps (a synthetic, non-enum string used
  as a map key against `commonaddress.Address.Type`), and correctly does
  NOT touch `interaction_read.go`/`dbhandler/interaction.go`'s SQL-level
  `peer_type` string comparison, which is a different concept (CRM
  interaction-timeline's own `peer_type` column value for AIcall-web
  interactions) that happens to share the same literal string by
  coincidence, not by shared type or shared code path. No action is
  needed on this design's part regarding items 6-7.

**No new "web_session" test file exists anywhere else in the monorepo**
beyond what §8.4 already lists. §8.4's checklist is complete.

## 3. Implementer-executability check

Read §4.4, §5, §8.3, §8.4 together as an implementer would, top to
bottom:

- §4.1's snippet (superseded) + §8.3's snippet (authoritative,
  `TypeWebSession Type = "web_session"`) — no conflict once the marker is
  followed; §8.3 is unambiguous about which value ships.
- §4.4's Go/DB field definitions (`Peer`/`Local` fields, nullable-DB
  columns, `NormalizeTarget`/`ValidateTarget` switch-exhaustiveness
  requirement) reference only the symbolic constant
  `commonaddress.TypeWebSession`, never the literal string — confirmed
  by the earlier grep (no `webchat_visitor`/`web_session` literal
  appears in §4.3, §4.4, or §5), so §4.4 needed no revision and has none
  — consistent with §8.3's own claim ("§5's file checklist and §4.4's
  Go/DB field definitions are otherwise unaffected").
- §5 (original file checklist) + §8.4 (revision addendum) together give
  a complete, non-overlapping file list: §5 covers the
  referrer/Peer/Local feature build-out, §8.4 covers only the
  string-value revert + dead-map-entry deletions. No file appears in
  both with conflicting instructions.
- §8.3's ordering warning (the `interaction_test.go` case "must be
  removed or rewritten... BEFORE the map-entry deletion lands, or `go
  test ./...` fails immediately") is present and correctly sequenced
  relative to §8.4's checklist ordering.
- Header line 3 correctly forward-points to §8 for the current decision.

An implementer reading top-to-bottom, or jumping directly to any of
§4.1/§4.2/§6/§8, now has a consistent, unambiguous path to the correct
current spec. No remaining contradiction blocks implementation start.

## 4. Re-verification of round-0/round-1's other factual claims (spot-checked, not re-derived from scratch)

- Three map-entry line numbers (`actionhandle.go:1265`, `tool.go:453`,
  `interaction.go:66`) — reconfirmed present and unchanged in this
  round's independent read of each file (§2 above).
- `interaction_crm_eligibility_test.go:36` and
  `interaction_test.go:247-266` — reconfirmed verbatim present and
  unchanged.
- `event_webchat.go`'s webchat Conversation self/peer both using
  `TypeWebchat` was not independently re-verified this round (out of
  scope — no `webchat_visitor`/`web_session` claim in §4.3 depends on it
  changing, and neither round-0 nor round-1 found any issue there).

## Verdict rationale

This round's independent, full-document re-read (§1-§8) plus a
repo-wide re-search scoped wider than either prior round (including
files neither round-0 nor round-1 examined, `interaction_read.go` and
`dbhandler/interaction.go`) found:

1. **No unmarked `webchat_visitor` occurrences remain.** The §4.2 gap
   round 1 identified is fixed, and this round's independent
   classification of all 18 occurrences confirms no other spot was
   missed.
2. **§8.4's file checklist is complete** against a fresh, broader
   repo-wide grep; the two additional `web_session`-string hits found
   outside §8.4's list (`interaction_read.go`, `dbhandler/interaction.go`)
   are a structurally distinct concept (CRM timeline's AIcall-session
   `peer_type` column, a plain string, never a `commonaddress.Type` enum
   member) correctly out of this revision's scope — traced to source and
   confirmed, not merely asserted.
3. **The document is implementer-executable as written**: an
   implementer following §4.4 + §5 + §8.3 + §8.4 top-to-bottom lands on
   a single, consistent spec with no contradictory literal values and no
   missing file in the deletion checklist.

No new defects found. Both round-0 and round-1's findings are correctly
and fully resolved, and this independent pass — deliberately covering
ground the first two rounds did not (the CRM-timeline `web_session`
cross-check, the full 18-occurrence classification rather than the
originally-named 3 spots) — surfaces nothing further requiring changes.

Per §8.5, this is the first APPROVE-eligible round after two
CHANGES_REQUESTED rounds; per the repo's 2-consecutive-APPROVE closure
convention, one more round (revision1-round3) is still required after
this APPROVE before Revision 1 can close.

VERDICT: APPROVED
