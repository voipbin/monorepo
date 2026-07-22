# Round 1 Review: webchat Session Referrer + Peer/Local design

Reviewer: Claude Code (independent verification against live source)
Target doc: `2026-07-22-webchat-session-referrer-peer-local-design.md` (round 1, resolves round 0 findings)
Prior review: `2026-07-22-webchat-session-referrer-peer-local-design-review-round0.md` (CHANGES_REQUESTED)

## 1. Did round 0's three requested changes actually land?

**§6.1 (`"web_session"` vs `"webchat_visitor"`) — RESOLVED, verified consistent... with one leftover exception (see §2 below).**
`TypeWebSession Type = "webchat_visitor"` is now stated explicitly in §4.1 with a full rationale (collision-avoidance vs. the three pre-existing `crmIneligiblePeerTypes` map entries). Confirmed against live source: `bin-common-handler/models/address/main.go` does not yet define `TypeWebSession` (expected — pre-implementation), and the three `crmIneligiblePeerTypes` maps still carry the bare string literal `"web_session"` unchanged:
- `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1265` — confirmed, exact line match.
- `bin-ai-manager/pkg/aicallhandler/tool.go:453` — confirmed, exact line match.
- `bin-contact-manager/pkg/contacthandler/interaction.go:66` — confirmed, exact line match.

All three remain distinct strings from the newly-chosen `"webchat_visitor"`, so the doc's non-collision claim holds for both literals.

**§6.3 (NOT NULL+backfill vs. nullable) — RESOLVED and now a single, implementable spec.**
§4.4's "Database" paragraph no longer describes two competing options (round 0's complaint). It commits to a single approach — `peer JSON NULL`, `local JSON NULL`, no backfill — with a concrete, verifiable technical justification: `id`/`widget_id` are `BINARY(16)` (confirmed in `sessions.sql`), so `HEX(id)` alone would not produce a canonical dashed UUID string parseable by `uuid.FromStringOrNil` (confirmed at `validate.go:67-71`). §5's file checklist (line 426) is consistent with this: `sessions.sql` gets `peer TEXT NULL, local TEXT NULL` with an explicit "per §4.4's round-1 decision" annotation — no stray NOT-NULL language remains anywhere else in the doc. ✅

**getorcreate.go:99 citation — REMOVED, confirmed.**
Grepped the full doc text for `getorcreate` — zero matches. Round 0's "minor, non-blocking" citation-looseness finding is fully addressed by deletion rather than by a corrected citation, which is an acceptable resolution for a non-blocking finding. ✅

## 2. New inconsistency introduced by the partial §4.1 edit (finding NOT present in round 0)

§4.2 ("Why Peer/Local now clears the earlier 'zero information' bar") still uses the OLD, now-rejected literal `web_session` twice as its illustrative example, directly contradicting §4.1's "Round 1 decision" stated immediately above it in the same document:

- Line 227: "What changes is that **Peer is now type-distinguishable from Local** (`web_session` vs `webchat`)..."
- Line 232: "...`web_session` -> \"Web visitor\" label, vs. today where `webchat`-typed Local/Peer are visually indistinguishable..."

This is a real, verifiable self-contradiction that did not exist in round 0 (round 0's §4.1 hadn't yet committed to any literal, so §4.2's use of `web_session` there was merely provisional/consistent-with-itself). The round-1 edit updated §4.1's decision but did not propagate the rename into §4.2's own illustrative text, which was written before the decision was finalized. Concretely, this doc now asserts in one place that the type value is `"webchat_visitor"` (§4.1, §4.4's code snippet at line 309 uses `commonaddress.TypeWebSession` correctly — the Go constant name is unaffected either way) and in another place (§4.2) uses the discarded string `"web_session"` as if it were the shipping value, in exactly the kind of prose a future engineer would grep or skim when building the "future cross-channel Peer/Local rendering component" §4.2 describes. This is precisely the "future reader confusion" risk §4.1 itself warns about, self-inflicted by the doc's own remaining leftover text.

This is cheap to fix (replace two literal-string mentions) but is a genuine, concrete defect introduced by this revision, not a stylistic nit — it's a factual inconsistency about the value of a schema constant, inside a document whose whole purpose is to be the implementable source of truth for that constant.

## 3. §4.2 left unresolved — is that legitimate?

Yes. Unlike §6.1 (string literal — blocks writing the `const` and every caller/test that encodes it) and §6.3 (NOT NULL vs. nullable — blocks writing the actual migration DDL and the Go struct's zero-value story), §4.2's open question ("is type-based dispatch for a not-yet-built rendering component sufficient justification for the schema addition at all") does not block any concrete implementation step described in §4.4/§5. Every field, type, migration statement, and file-list entry is fully specified regardless of how that question is answered — an implementer can execute the entire checklist in §5 without needing an answer to §4.2 first. The doc's own framing (routing this to pchero as a product-scope call, not a technical unknown) is accurate: it is asking "should this ship at all," a scope/product decision that is pchero's to make, not "how should this be built," which is the reviewer's/author's job to nail down before implementation starts. Round 0's objection was specifically that §6.1/§6.3 were left open BUT self-identified by the author's own text as things needing resolution before implementation ("recommend resolving in round 1") — that dynamic does not apply to §4.2, which was never framed that way and isn't implementation-blocking. Leaving §4.2 open is legitimate and does not by itself justify CHANGES_REQUESTED.

## 4. Re-check of §1-§5, §7 for new contradictions from this revision

- §4.4's Go struct snippet (`Peer`/`Local`, no `omitempty`) still matches `kase.Case`'s actual precedent in `kase.go` (`Peer`/`Local` both without `omitempty`, "ALWAYS PRESENT in JSON output" comment) — re-confirmed against live source, unchanged from round 0. ✅
- `pkg/sessionhandler/create.go`'s current lines 78-79 (`self`/`peer` locals used only for `ConversationV1ConversationCreateAndExecuteFlow`) are still exactly as the doc describes and are correctly called out as unrelated/unchanged by this design (§4.4's "two independent uses" note) — re-confirmed against live source. ✅
- `normalize.go:50` / `validate.go:33` switch case lists are unchanged from round 0's verification and still require the doc's proposed additions — re-confirmed. ✅
- `sessions.sql` still has no `referrer`/`peer`/`local` columns today (pre-implementation state), consistent with the doc's premise. ✅
- No other stray `"web_session"`/NOT-NULL leftovers found elsewhere in the document body (§3, §5, §6) beyond the §4.2 instance flagged above — targeted grep of the full doc text for both strings confirms all remaining occurrences are either correctly contextual (e.g., §4.1's explanation of why `"web_session"` was rejected, §4.4's contrast with `contact_cases`'s NOT NULL precedent) or the two §4.2 leftovers already flagged.
- §4.3's scope boundary (Case/Interaction unchanged) is untouched by this revision and remains well-reasoned, as round 0 already found. No new issues.

## Verdict rationale

Round 1 genuinely resolves both implementation-blocking items from round 0 (§6.1's literal choice, §6.3's nullable-vs-NOT-NULL decision) with sound, source-grounded technical justification, and cleanly removes the shaky `getorcreate.go:99` citation rather than patching around it. §4.2 being left open is a legitimate, correctly-scoped product-judgment deferral, not a reviewer dodge, and is not grounds for CHANGES_REQUESTED on its own.

However, the revision itself introduces a new, concrete, verifiable defect: §4.2 still illustrates its own point using the literal `"web_session"` in two places, directly contradicting §4.1's Round 1 decision made earlier in the same document to use `"webchat_visitor"` instead. This is exactly the kind of leftover-text inconsistency §4.1 warns a future reader could stumble over, and it sits inside a section describing a not-yet-built consumer that a future engineer would plausibly build by reading §4.2's prose literally. It is a small, mechanical fix, but it is real and self-contradictory within the document as submitted, so the doc is not yet internally consistent enough to hand to an implementer as-is.

**Requested changes:**
1. §4.2 (lines 227 and 232): replace both remaining `web_session` literal-string mentions with `webchat_visitor`, matching §4.1's Round 1 decision. This is the only change requested for round 1 sign-off.

VERDICT: CHANGES_REQUESTED
