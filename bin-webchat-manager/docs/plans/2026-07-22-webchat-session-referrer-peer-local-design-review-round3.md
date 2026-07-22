# Round 3 Review: webchat Session Referrer + Peer/Local design

Reviewer: Claude Code (independent verification against live source, fresh
top-to-bottom pass, deciding round for 2-consecutive-APPROVE close)
Target doc: `2026-07-22-webchat-session-referrer-peer-local-design.md`
Prior reviews:
- round0 — CHANGES_REQUESTED (§6.1/§6.3 open, `getorcreate.go:99` citation)
- round1 — CHANGES_REQUESTED (§4.2 lines 227/232 still said `web_session`)
- round2 — APPROVED (1/2 consecutive; explicitly noted it did NOT re-fetch
  `bin-conversation-manager`/monorepo-javascript because it believed those
  repos were "not present in this local worktree checkout")

## 0. Correcting round 2's stated limitation

Round 2 wrote: *"`webchat-widget-runtime/client.js` and the `167bebb7c46f`
contact-manager migration (both in repos not present in this local worktree
checkout) were previously verified directly in round 0's review and are
unchanged in this revision's text — not re-walked here."*

This is checked in this round and turns out to be **incorrect as a factual
premise, though the underlying conclusion (no defect) survives**:

- `bin-conversation-manager` **IS present** directly inside this worktree
  (`.worktrees/NOJIRA-webchat-session-referrer-peer-local/bin-conversation-manager/`,
  confirmed via `ls`) — round 2 could have opened it directly and did not.
- `bin-dbscheme-manager` (home of `167bebb7c46f`) is likewise present in the
  worktree.
- `monorepo-javascript` (home of `client.js`/`message_timeline.js`) is
  genuinely NOT inside this worktree, but exists as a sibling checkout at
  `/home/pchero/gitvoipbin/monorepo-javascript` and was accessible without
  any extra setup.

I opened all of these directly in this round (results in §1 below). The
substantive good news: every citation checked resolves correctly, so round
2's ultimate verdict was not undermined by the gap — but the gap was real,
and this round closes it with actual file reads rather than repeating the
same "not re-walked" caveat a third time.

## 1. Live verification of the citations round 2 could not re-fetch

**`bin-conversation-manager` (§4.1's "confirmed by tracing the actual
webchat data flow" claim):**
- `pkg/conversationhandler/event.go:31` — confirmed literally
  `case conversation.TypeWebchat:` dispatching to `h.eventWebchat`. ✅
- `pkg/messagehandler/send.go:39` — confirmed `case conversation.TypeWebchat:
  return h.sendWebchat(ctx, cv, text)`. ✅ (doc's second citation,
  `send.go:191`, is inside `Create(...)`'s `ReferenceType: message.ReferenceTypeWebchat`
  call — a related but not identical line to what §4.1's prose implies;
  this is a loose/approximate citation, not a wrong one — the surrounding
  claim that webchat messages are `ReferenceTypeWebchat`-tagged, never
  `"web_session"`, holds regardless.)
- `pkg/conversationhandler/event_webchat.go:88-89` — confirmed **exact**
  match:
  ```go
  self := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: wm.WidgetID.String()}
  peer := commonaddress.Address{Type: commonaddress.TypeWebchat, Target: wm.SessionID.String()}
  ```
  Both self AND peer are `TypeWebchat` — exactly as §4.1 claims. The
  string `"web_session"` genuinely does not appear anywhere in this file.
  ✅ **This is the single most load-bearing citation in the whole
  document** (it's the evidence for "introducing `TypeWebSession` does not
  interact with any code that exists today"), and it checks out precisely.

**`bin-dbscheme-manager/.../167bebb7c46f_contact_cases_peer_local_address_json.py`:**
Opened in full. Confirmed exactly as the doc (§4.3, §4.4) and round 0
describe: `peer` is added `JSON NULL`, backfilled via
`JSON_OBJECT('type', peer_type, 'target', peer_target)`, then tightened to
`NOT NULL`; `local` stays permanently nullable; `open_peer_uk`/
`uq_case_open_peer` are explicitly dropped-and-recreated around the
peer_type/peer_target column swap because `open_peer_uk` is itself a STORED
generated column referencing them (errno 3108 dependency, exactly as cited).
This is the real precedent the doc leans on for both "why NOT NULL+backfill
is the harder path" (§4.4) and "why unifying Case/Interaction now would be
higher-risk" (§4.3). Both framings are accurate. ✅

**`monorepo-javascript` (sibling repo, not in this worktree, checked
directly at `/home/pchero/gitvoipbin/monorepo-javascript`):**
- `square-admin/src/webchat-widget-runtime/client.js`: confirmed `_doStart()`
  posts `page_url: (typeof window !== 'undefined' && window.location?.href)
  || undefined` with no `referrer` field yet — matches doc's pre-implementation
  premise and the exact code shape §3.1 says to mirror. ✅
- `square-admin/src/views/webchat_widgets/message_timeline.js`: confirmed
  `truncatePageURL`/`isSafePageURL` exist exactly as named, with
  `isSafePageURL = (url) => typeof url === 'string' && (url.startsWith('http://')
  || url.startsWith('https://'))` — a page_url-only-in-name, content-generic
  helper, supporting §3.3's rename claim (no page_url-specific logic inside
  either function to strand). ✅
- **One real, if minor, gap round 2 (and rounds 0-1) also missed:** the
  doc's §3.1/§5 file paths are written as `webchat-widget-runtime/client.js`
  and `src/views/webchat_widgets/message_timeline.js` — omitting the
  `square-admin/` prefix that the actual path requires
  (`square-admin/src/webchat-widget-runtime/client.js`,
  `square-admin/src/views/webchat_widgets/message_timeline.js`). This is
  consistent within the doc itself (§3.1's header does the same
  abbreviation, so it's not self-contradictory), and any engineer working
  in this monorepo's `square-admin/` directory would not actually be misled
  in practice (relative paths from inside `square-admin/` are correct) —
  but taken completely literally against the monorepo root, the path is
  incomplete. Flagging as a **NICE-TO-HAVE**, not a blocker: every prior
  round (0/1/2) also let this pass, and it does not change which file gets
  edited by anyone actually opening `square-admin/`.

**Conclusion of this section:** round 2's underlying verdict was not wrong
in substance — every claim about these three repos checks out against real
code, including the single most important one (`event_webchat.go:88-89`).
But round 2's report of *why* it didn't check them ("not present in this
local worktree checkout") was itself an unverified, and in
`bin-conversation-manager`'s/`bin-dbscheme-manager`'s case incorrect,
claim. Noting this so the review-loop's own audit trail is accurate, not to
relitigate the verdict.

## 2. Full document re-read, §1–§7, independently

Read the entire doc top to bottom without assuming prior rounds' section
coverage was exhaustive.

- **§1 Problem**: consistent framing; the "supersedes the CPO's earlier
  verbal rejection" claim correctly sets up §4.2's "zero information" reuse
  later. No drift.
- **§2 Goals/Non-goals**: all three non-goals (Case/Interaction untouched,
  no UTM parsing, no backfill) are each individually elaborated later
  (§4.3, referenced non-goal for UTM has no further elaboration needed
  since it's simply "not doing X", and §4.4/§6.3 for the no-backfill
  decision) — no orphaned promise.
- **§3 (`referrer`)**: internally consistent, matches live `page_url`
  precedent exactly (verified independently above and via re-reading
  `client.js`/`message_timeline.js`/`validatePageURL`'s XSS-fix precedent).
  §3.3's rename claim (`truncatePageURL`→`truncateURL`, `isSafePageURL`→
  `isSafeURL`) is honest about being a "pure rename, no behavior change" —
  confirmed true by reading both functions' actual bodies (string-only
  logic, nothing page_url-specific baked in).
- **§4.1**: `TypeWebSession Type = "webchat_visitor"` stated once,
  unambiguously, with the collision analysis grounded in real, re-verified
  citations (all three `crmIneligiblePeerTypes` maps, all three line
  numbers exact-match live source: `actionhandle.go:1265`, `tool.go:453`,
  `interaction.go:66`).
- **§4.2**: no lingering `web_session` literal (grepped fresh — see §3
  below), internally consistent with §4.1's committed value in both
  illustrative uses.
- **§4.3**: scope boundary reasoning re-checked against the real
  `167bebb7c46f` migration file (see §1 above) — the claimed blast radius
  (bin-flow-manager/bin-ai-manager/bin-conversation-manager touching
  webchat Peer/Local construction) is accurate; `event_webchat.go`,
  `send.go`, and `event.go` are genuinely the call sites that would need
  to change if Case/Interaction ever adopted `TypeWebSession`.
- **§4.4**: Go struct snippet, `NormalizeTarget`/`ValidateTarget` switch
  additions, and the nullable-DB decision all re-verified directly against
  live `normalize.go:50`, `validate.go:33`, `kase.go`, `sessions.sql`,
  `widget.go:162` — all exact matches (see §3 below for the line-by-line
  log). The distinction drawn between the NEW `Session.Peer`/`Session.Local`
  fields and the EXISTING unrelated `self`/`peer` locals in
  `create.go:78-79` (used only for the Flow-trigger RPC) is accurate and,
  re-reading `create.go` fresh in this round, genuinely easy to conflate if
  an implementer isn't paying attention — the doc's explicit "must not
  deduplicate" warning is warranted, not paranoid.
- **§5 (file checklist)**: cross-checked field-by-field against §3/§4 —
  `referrer`, `peer`, `local` appear consistently across every layer
  (session.go, field.go, webhook.go, request DTO, requesthandler,
  api-manager servicehandler, openapi, dbscheme, sql test schema,
  square-admin). No field named in §3/§4 is missing from §5, and no §5
  entry names a field/type not defined in §3/§4. See §4 below for the
  detailed cross-check.
- **§6**: all three round-0-era open questions' RESOLVED/NOT-RESOLVED
  status accurately reflects the current §4.1/§4.2/§4.4 body text — no
  drift (re-confirmed, same as round 2 found).

No new logical inconsistency found in this fresh full read.

## 3. Fresh full-document grep sweep for stray literals

- `web_session` (with/without backticks): 7 occurrences, all in §4.1's
  rationale narrative and §6.1's resolution summary, all correctly
  contextual (explaining what was rejected and why). Zero occurrences
  outside §4.1/§6. ✅
- `NOT NULL` occurrences: only in §4.4's contrast with `contact_cases`'s
  precedent (`peer JSON NOT NULL` is `contact_cases`'s pattern, explicitly
  described as the REJECTED alternative for this design) and in the
  `167bebb7c46f` migration's own real DDL as quoted for context. No stray
  "Session.peer/local is NOT NULL" claim anywhere — the doc consistently
  says nullable throughout §4.4, §5, §6.3. ✅
- `getorcreate.go:99`: zero matches (round 0's finding stays resolved). ✅

## 4. §5 checklist vs. §4's final decisions — line-by-line consistency

| §4 decision | §5 checklist reflects it? |
|---|---|
| `TypeWebSession = "webchat_visitor"` (§4.1) | `models/address/main.go` listed under bin-common-handler ✅ |
| `normalize.go`/`validate.go` switch additions (§4.4) | both explicitly listed ✅ |
| `Peer`/`Local` NOT `omitempty` in Go (§4.4) | `session.go` listed; §5's own annotation for `sessions.sql` explicitly says "nullable per §4.4's round-1 decision" — correctly distinguishes the DB-level nullability from the Go-level non-omitempty contract, not conflating the two ✅ |
| `peer JSON NULL`, `local JSON NULL`, no backfill (§4.4) | dbscheme-manager migration file listed as a single new file (`_webchat_sessions_add_columns_peer_local.py`) — consistent with "single, unconditional step with no backfill" (no separate backfill-script file listed, correctly) ✅ |
| `referrer` mirrors `page_url` exactly, own migration (§3.2) | listed as a SEPARATE migration file from peer/local (`_webchat_sessions_add_column_referrer.py`) — correct, since §3 and §4 are independent features landing as independent ALTERs ✅ |
| Rename `truncatePageURL`→`truncateURL` etc. (§3.3) | `message_timeline.js` entry explicitly names the rename ✅ |
| `WebchatV1SessionCreate` gains `referrer string` (§3.2) | `bin-common-handler/pkg/requesthandler/main.go` + `webchat_session.go` both listed ✅ |

No mismatch found between §5's checklist and §4/§3's final decisions.

## 5. Implementer-executability check

Re-reading §3 and §4 as an implementer would (not as a reviewer looking for
prior findings): every field name, Go type, JSON tag, db tag, migration
DDL, and validation rule is stated explicitly and singly (no "or"/either-or
language remains anywhere in §3/§4/§5 — confirmed by the grep sweep in §3
above). The two genuinely open items (§4.2's product-judgment call, and the
minor `square-admin/` path-prefix omission noted in §1) are each either
explicitly routed to a human decision-maker (§4.2, correctly non-blocking
per round 1's reasoning, re-affirmed here) or too small to block coding
(the path prefix — an implementer opening the actual `square-admin/`
directory tree will find the named files regardless).

The document is a complete, implementable spec: an engineer could start
writing the migration, the Go struct changes, the switch-statement
additions, and the JS renames directly from §3–§5 with no further design
decisions required.

## Verdict rationale

This round did the one thing the prior APPROVE (round 2) explicitly flagged
as unfinished — opening `bin-conversation-manager`'s
`event.go`/`send.go`/`event_webchat.go` and monorepo-javascript's
`client.js`/`message_timeline.js` directly, rather than relying on round
0's older verification. All of it checks out, including the single most
load-bearing citation in the document (`event_webchat.go:88-89`'s exact
`TypeWebchat`-for-both-self-and-peer code, which is the entire factual
basis for §4.1's "no collision with today's code" claim). Round 2's stated
reason for not re-fetching those files turned out to be factually wrong for
two of the three repos (they ARE in this worktree), which is worth
correcting in the audit trail even though it didn't change the outcome.

A full independent top-to-bottom read of §1–§7 found no new contradiction,
no stray leftover literal, and a clean cross-check of §5's checklist
against every decision in §3/§4. The one new (very minor) finding — §3.1/§5
citing `webchat-widget-runtime/client.js`/`src/views/webchat_widgets/
message_timeline.js` without the `square-admin/` prefix the real path
requires — does not block implementation and is not the kind of defect
that has driven CHANGES_REQUESTED in any prior round (blocking spec
ambiguity, factual contradiction, hallucinated citation). It's noted as a
NICE-TO-HAVE for the author to tighten in a future doc, not grounds to
extend the review loop again.

This is an implementable, internally consistent, fully-verified spec, and
this round independently confirms the parts of the codebase round 2 was
unable (or, per §0/§1 above, mistakenly believed itself unable) to check.

VERDICT: APPROVED

**Process note:** per this repo's design-review-loop convention, this is
the SECOND consecutive APPROVED (round 2, round 3). The 2-consecutive-
APPROVE bar is met — the review loop is closed. This design is approved for
implementation.
