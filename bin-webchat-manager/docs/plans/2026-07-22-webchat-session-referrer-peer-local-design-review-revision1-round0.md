# Review: Revision 1 (§8), Round 0

Target: `2026-07-22-webchat-session-referrer-peer-local-design.md` §8
("string value reverts to `web_session`; three dead `"web_session"` map
entries deleted")

Scope: this round reviews ONLY §8 (Revision 1) per the
post-closure-revision-and-superseded-markers procedure. §1-§7 are treated
as prior-audit-trail content, but checked for new contradictions
introduced by §8's addition (per the brief).

## 1. §8.2's "dead code" claim — re-verified against live source, and it is INCOMPLETE

Repo-wide grep for the literal `"web_session"` string (Go source, not the
`docs/plans/*.md` prose that also uses it descriptively for an unrelated
AIcall concept) turns up **five** hits, not the three map entries + "their
own test files" that §8.2 claims:

1. `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1265` — map entry (as claimed)
2. `bin-ai-manager/pkg/aicallhandler/tool.go:453` — map entry (as claimed)
3. `bin-contact-manager/pkg/contacthandler/interaction.go:66` — map entry (as claimed)
4. `bin-contact-manager/pkg/contacthandler/interaction_crm_eligibility_test.go:36` — `Test_isCRMEligiblePeer` case (the map's own dedicated test, correctly identified in §8.4)
5. **`bin-contact-manager/pkg/contacthandler/interaction_test.go:247-266`** — **NOT mentioned anywhere in §8.2 or §8.4**

Item 5 is a genuine gap, not a nitpick. It is a test case inside
`Test_EventConversationMessageCreated` (not `interaction_crm_eligibility_test.go`,
which §8.2's "their own test files" phrasing implicitly limits the dead-code
claim to):

```go
{
    name: "outgoing web_session message - peer is synthetic web session type - projection skipped",
    message: &convmsg.WebhookMessage{
        ...
        Source: commonaddress.Address{
            Type:   "web_session",
            Target: "web_session:xyz",
        },
        Destination: commonaddress.Address{Type: commonaddress.TypeAI, Target: ""},
    },
    expectInteraction: nil,
},
```

This test exercises `EventConversationMessageCreated` → `deriveEndpoints` →
`isCRMEligiblePeer(peer.Type)` end-to-end, asserting that when the peer
type is `"web_session"`, `isCRMEligiblePeer` returns `false` and
`EventConversationMessageCreated` returns early **without** calling
`h.db.InteractionCreate` (the test's `expectInteraction: nil` branch
deliberately skips setting `mockDB.EXPECT().InteractionCreate(...)`, at
lines 288-293 of the same file).

**After §8.3's deletion lands**, `crmIneligiblePeerTypes` no longer
contains `"web_session"`, so `isCRMEligiblePeer("web_session")` returns
`true`. `EventConversationMessageCreated` will then proceed past the
eligibility check and call `h.db.InteractionCreate` on a mock that was
never told to expect that call — this test will fail (gomock panics on an
unexpected call) exactly like the acknowledged `interaction_crm_eligibility_test.go:36`
case, but §8.2/§8.4 never identify it, so an implementer following §8.4's
checklist literally would hit a **surprise test failure** not flagged by
the design doc, and would have to independently rediscover this test the
way this review just did.

This directly falls under the skill's "grep for every occurrence" rule
(`post-closure-revision-and-superseded-markers.md` item 3/5): §8.2's dead-code
audit re-verified the three map entries and the ONE test file that directly
tests the map (`interaction_crm_eligibility_test.go`), but did not
grep broadly enough to catch a second, independent test file
(`interaction_test.go`) that also hardcodes the literal and depends on the
current (soon-to-be-deleted) filtering behavior.

**Action required:** §8.2 and §8.4's bin-contact-manager checklist must be
updated to also list `interaction_test.go` (the
`Test_EventConversationMessageCreated` "outgoing web_session message" case,
lines 247-266) as a file requiring an update/removal, with the same
"deletion changes zero runtime behavior *today*, but this stored test
expectation will start failing" caveat already applied correctly to
`interaction_crm_eligibility_test.go:36`.

## 2. §8.3's deletion plan — line numbers verified correct

Checked against live source:

- `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1265` — confirmed: `"web_session": {}, // synthetic type; not in commonaddress.Type enum`, inside `crmIneligiblePeerTypes` (lines 1257-1266), all other entries are real `commonaddress.Type` constants. Matches §8.3 exactly.
- `bin-ai-manager/pkg/aicallhandler/tool.go:453` — confirmed: identical entry inside `caseCreateCRMIneligiblePeerTypes` (lines 445-454). Matches §8.3 exactly.
- `bin-contact-manager/pkg/contacthandler/interaction.go:66` — confirmed: identical entry inside `crmIneligiblePeerTypes` (lines 58-67). Matches §8.3 exactly.

All three line citations are accurate as of current source. §8.3's claim
that "the surrounding rationale... is unaffected and still accurate for
the remaining entries" also checks out — none of the doc-comments (e.g.
`interaction.go:34-57`) reference the `"web_session"` entry specifically.

## 3. §8.4's file checklist — INCOMPLETE (see §1 above); test-existence claims otherwise verified

- `bin-flow-manager/pkg/activeflowhandler/actionhandle_case_create_test.go`
  → `Test_isCRMEligiblePeer` (line 398) does **not** assert on `"web_session"`
  at all (only `tel`/`agent`/`sip` cases). §8.4's conditional phrasing
  ("if `Test_isCRMEligiblePeer` asserts on the now-removed entry, update
  it") is correctly hedged — no action needed here, consistent with what
  the doc says.
- `bin-ai-manager/pkg/aicallhandler/tool_case_create_test.go` →
  `Test_isCRMEligiblePeer` (line 58) likewise has no `"web_session"` case
  (`tel`/`email`/`agent`/`conference`/`sip` only). Same correct hedge, no
  action needed. Confirmed no other `_test.go` file in either
  `bin-flow-manager` or `bin-ai-manager` references the literal
  `"web_session"` (repo-wide grep, zero hits in both trees).
- `bin-contact-manager/pkg/contacthandler/interaction.go` — confirmed dead
  entry at line 66 as stated.
- `bin-contact-manager/pkg/contacthandler/interaction_crm_eligibility_test.go:36`
  — confirmed verbatim: `{"web_session is ineligible", commonaddress.Type("web_session"), false}`,
  and confirmed this assertion breaks post-deletion (isCRMEligiblePeer
  would return `true`, not the asserted `false`). §8.4's description of
  this file is accurate.
- **Missing**: `interaction_test.go` (see §1). This is the one concrete
  defect in this revision's file checklist.

## 4. §8.3's "future risk" callout — legitimate, correctly scoped, not overblown

The concern (a future Call/Conversation-derived peer legitimately carrying
`Type: TypeWebSession` would silently become CRM-eligible once the
blacklist entries are gone, whereas today's dead string literal would have
coincidentally excluded it) is a real, non-obvious second-order effect of
turning a bare string into a live enum value while simultaneously removing
the entries that would have blocked it. It is appropriately hedged ("very
unlikely... but is recorded here for completeness") rather than presented
as a blocking risk, and it correctly avoids scope creep — it explicitly
does not propose adding a new `TypeWebSession` blacklist entry now (which
would reintroduce exactly the ambiguity pchero is asking to remove). This
is right-sized: informative for a future implementer, not an argument
against the revision. No changes needed here.

## 5. §1-§7 vs. §8 — new contradictions introduced by adding §8?

- Doc header (line 3) correctly points forward: "see Revision 1 below for
  a post-closure change to §4.1's type-string decision." Good.
- §8's own header, §8.1, and §8.3 all explicitly state they supersede
  §4.1's text and that §4.1's body is left as audit trail. Good — follows
  the skill's "append, don't rewrite in place" rule.
- **However**, per the skill's explicit completeness-audit lesson
  (`post-closure-revision-and-superseded-markers.md` item 3: "six separate
  downstream sections... all still presenting the pre-revision shape as
  current spec, with ZERO inline forward-pointer at the point of
  staleness"), this document has the same shape of gap:
  - §4.1's code snippet (line 152, `TypeWebSession Type = "webchat_visitor"`)
    has no inline note pointing to §8.3's superseding value. A reader who
    stops at §4.1 (very plausible — it's the section literally titled "New
    `commonaddress.Type`") will read the wrong literal with no signal to
    keep going.
  - §4.2 (lines 227-237) discusses the `webchat_visitor`/`webchat` type
    pair and a "Web visitor" render-label example as if still current,
    with no forward pointer.
  - **§6, item 1** (lines 444-448) is the sharpest case: it says
    "**RESOLVED** — `\"webchat_visitor\"`, not `\"web_session\"`" — stated
    as a settled, resolved fact, sitting structurally BEFORE §8 in reading
    order, with no marker that this resolution was itself later reverted.
    A top-to-bottom reader reaches an explicit "RESOLVED" claim that is no
    longer true before ever reaching §8.
  - None of these three spots carry so much as a one-line "(superseded by
    §8 — value reverted to `web_session`)" pointer; only the document's
    top-of-file status line and §8's own header mention the revision at
    all.
- This matters more than a typical staleness nit because §4.1's snippet
  is exactly the kind of thing an implementer might copy-paste (the doc's
  own convention elsewhere, e.g. §3's code blocks, is written to be
  copied close to verbatim into source) — shipping the reverted-from
  string into new code is the "ships into a real source file" severity
  class the skill explicitly ranks highest (item 4).

**Action required:** add short inline "(superseded by §8, revert to
`"web_session"`)" markers at the three spots above (§4.1's snippet, §4.2's
`webchat_visitor` discussion, §6 item 1's "RESOLVED" line) — mid-section,
at the point of staleness, not only in a delta list. This is a real,
fixable completeness gap, not a nitpick given the skill's own prior
documented incident of exactly this failure mode.

## 6. Other checks performed

- `bin-conversation-manager/pkg/conversationhandler/event_webchat.go:88-89`
  — confirmed both `self` and `peer` still use `commonaddress.TypeWebchat`
  exclusively for webchat messages. §4.1/§8.2's claim that webchat never
  produces `"web_session"`-typed addresses today holds.
- `bin-common-handler/models/address/main.go` — current `Type` enum has no
  `TypeWebSession` member yet (this design has not been implemented; only
  the doc exists). Confirms this revision is purely a spec-stage change,
  consistent with the doc's own framing.
- No other `bin-*` service outside the three named ones references the
  bare string `"web_session"` as an address-type value (repo-wide grep);
  the `docs/plans/2026-06-2[68]-*.md` hits are an unrelated, pre-existing
  concept (`peer_type='web_session'` for un-attributed web AIcall sessions
  in the CRM interaction-timeline design) that does not intersect with
  `commonaddress.Type` at all and is correctly out of this revision's
  scope.

## Verdict rationale

Two concrete, fixable defects in this revision:

1. §8.2's dead-code confirmation and §8.4's file checklist both miss
   `bin-contact-manager/pkg/contacthandler/interaction_test.go`'s
   `Test_EventConversationMessageCreated` "outgoing web_session message"
   case, which will start failing once the map entry is deleted — an
   implementer following §8.4 literally hits an unflagged test failure.
2. §4.1/§4.2/§6 item 1 present the pre-revision `"webchat_visitor"`
   decision as current/resolved with no inline pointer to §8, including
   one spot (§4.1's code snippet) that risks being copy-pasted verbatim
   into source with the wrong (reverted-from) string value.

Both are the same class of gap this repo's own skill documentation
(`post-closure-revision-and-superseded-markers.md`) explicitly calls out
as the "single most valuable, non-obvious finding" reviewers should hunt
for in exactly this kind of revision — a completeness/staleness audit,
not a correctness-of-the-new-snippet check. §8.3's actual deletion plan
(line numbers, rationale) is accurate, and the future-risk callout is
sound, but the revision is not yet complete.

VERDICT: CHANGES_REQUESTED
