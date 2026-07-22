# Review: Revision 1 (§8), Round 1

Target: `2026-07-22-webchat-session-referrer-peer-local-design.md` §8
("string value reverts to `web_session`; three dead `"web_session"` map
entries deleted")

Scope: this round reviews ONLY §8 (Revision 1), specifically whether the
two round-0 findings were fully resolved. §1-§7 are checked only for new
contradictions/staleness introduced (or left unfixed) by the round-1 edit.

## 1. Finding 1 (round-0): missing `interaction_test.go` in dead-code audit — RESOLVED

Verified against live source
(`bin-contact-manager/pkg/contacthandler/interaction_test.go:247-266`):
the `Test_EventConversationMessageCreated` case "outgoing web_session
message - peer is synthetic web session type - projection skipped" exists
exactly as the document now describes it —

- `Source: commonaddress.Address{Type: "web_session", Target:
  "web_session:xyz"}` (lines 255-258, confirmed verbatim)
- `expectInteraction: nil` (line 265), and the test harness (lines
  288-293) only sets `mockDB.EXPECT().InteractionCreate(...)` when
  `expectInteraction != nil` — so this case indeed has NO
  `InteractionCreate` mock expectation configured.

§8.2's narrative ("this test... asserts NO Interaction row gets created
... this test must be updated/removed alongside the map deletion... or it
fails with a gomock panic") and §8.4's bin-contact-manager checklist entry
(which reproduces the same test name, line range, and gomock-panic
mechanism) both match the actual code precisely. §8.2's "no production
caller" conclusion and the "test must be updated" requirement are stated
together without contradiction — the doc is explicit that deleting the map
entries changes zero PRODUCTION runtime behavior but does require updating
two test files that exercise the entries directly; those are two distinct
claims (production-path safety vs. test-fixture staleness) and they don't
conflict.

Also re-verified the three map-entry deletion targets §8.3/§8.4 cite are
still accurate against live source:

- `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1265` — confirmed present, `"web_session": {}, // synthetic type; not in commonaddress.Type enum`.
- `bin-ai-manager/pkg/aicallhandler/tool.go:453` — confirmed present, identical entry.
- `bin-contact-manager/pkg/contacthandler/interaction.go:66` — confirmed present, identical entry.

Finding 1 is fully closed.

## 2. Finding 2 (round-0): missing inline "superseded" markers — PARTIALLY RESOLVED, one spot still missing

Round 0 named **three** specific spots needing an inline forward-pointer
to §8: "§4.1's snippet, §4.2's `webchat_visitor` discussion, §6 item 1's
'RESOLVED' line."

Re-checked this revision:

- **§4.1's snippet (line ~151-158)**: FIXED. A `[SUPERSEDED BY §8 REVISION
  1]` block now sits immediately before the code snippet, explicitly
  stating the shipped value is `"web_session"` and pointing to §8.3.
  Correctly placed — a reader stopping at §4.1 now hits the marker before
  the stale literal.
- **§6 item 1 (line ~453-462)**: FIXED. The "RESOLVED... `webchat_visitor`"
  line now has an appended `[SUPERSEDED BY §8 REVISION 1]:` note stating
  the round-1 resolution was itself reverted, with a pointer to §8.1-§8.3.
- **§4.2 (lines 217-252, specifically the `webchat_visitor` vs `webchat`
  type-pair discussion and the "Web visitor" render-label example at
  lines 229-241)**: **NOT FIXED.** Repo-wide grep for `SUPERSEDED` in this
  document returns exactly two hits (line 151 and line 457) — §4.2 has
  none. The text at lines 229 (`Peer.Target: TypeWebSession, ...`
  comment), 236 (`webchat_visitor` vs `webchat`), and 241 (`webchat_visitor
  -> "Web visitor" label`) still reads as current, unqualified spec, with
  no inline pointer to §8's reversion. This is the exact spot round 0
  called out by name and it was skipped in this revision's edit.

This is not a cosmetic miss. §4.2's surrounding prose is argumentative
("this design changes the PEER role's type to the new `TypeWebSession`...
`webchat_visitor` vs `webchat`... which matters for exactly one concrete
consumer") — i.e., it reads as the document's live reasoning for why the
literal was chosen, not as a code snippet an implementer would obviously
recognize as historical. A reader who reads §4.1 (now correctly flagged)
and skips ahead, or who jumps directly to §4.2 because they're looking for
"why does Peer use a different type than Local," will still hit two
unmarked assertions using the reverted-from string with no signal to check
§8. Given the round-0 review explicitly enumerated three spots and this
revision fixed two of three, this reads as an incomplete pass over the
round-0 finding rather than a newly-introduced regression — but it leaves
the same staleness-risk class of defect the skill's own prior-incident
lesson is about (implementer copies/reads a stale literal without hitting
a forward-pointer).

**Action required:** add the same style of inline marker at §4.2's
`webchat_visitor`/`webchat` discussion (lines ~229-241), e.g. immediately
before or after the `Peer commonaddress.Address ... {Type: TypeWebSession,
...}` snippet block, noting the type constant is unchanged but its
underlying string reverted per §8.3.

## 3. Other staleness sweep (§1-§7, full re-grep)

Repo-wide grep of the document for `webchat_visitor` (16 total hits) and
cross-checked each remaining occurrence outside §4.1/§4.2:

- Lines 453, 489, 496, 512, 601, 603 all sit inside §6 item 1 (now marked)
  or §8 itself (§8.1/§8.3, which explicitly frame `webchat_visitor` as
  historical/superseded prose by design — no issue).
- No occurrences found in §4.3, §4.4, or §5 — those sections reference
  only the symbolic constant `commonaddress.TypeWebSession` /
  `TypeWebchat`, never the literal string value, so they were never stale
  to begin with and need no marker.

So the residual gap is confined to §4.2 as identified in §2 above.

## 4. §8.5's review-loop framing

§8.5 states Revision 1 needs its own fresh 2-consecutive-APPROVE loop,
independent of the original round 0-3 closure. Consistent with this
repo's post-closure-revision convention; no issue.

## Verdict rationale

One concrete, fixable defect remains: round 0 named three spots needing an
inline "superseded by §8" pointer; this revision fixed two (§4.1, §6 item
1) but left the third (§4.2's `webchat_visitor`/`webchat` discussion,
lines ~229-241) unmarked. Finding 1 (the `interaction_test.go` gomock-panic
gap) is fully and correctly resolved — narrative, line numbers, and
mechanism all verified against live source. §8.2's "no production
caller"/"test needs updating" framing is logically sound, not
contradictory. The three `crmIneligiblePeerTypes` deletion targets in
§8.3/§8.4 are re-confirmed accurate.

Because the outstanding gap is a partial (not full) fix of an
already-identified finding, and it is a small, mechanical, one-paragraph
addition rather than a new substantive question, this is close to
approvable — but per this doc's own explicit round-0 request for three
specific edits, shipping with one of three not applied is not yet a
complete, implementer-ready state: an implementer reading §4.2 top-to-
bottom (a very plausible path, since it's the section explaining WHY Peer
uses a different type than Local) still has no signal to distrust the
`webchat_visitor` string there.

VERDICT: CHANGES_REQUESTED
