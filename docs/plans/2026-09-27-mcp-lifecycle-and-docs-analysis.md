# Analysis: MCP server lifecycle correctness + square-admin parity (v29, PR A/C only)

Date: 2026-09-27
Author: CPO (Hermes)
Status: v29, Analysis Review Loop round 21 pending
Round 5: fact-check **APPROVED**; adversarial CHANGES_REQUESTED (7 findings → D15-D19).
Round 6: fact-check **APPROVED**; adversarial CHANGES_REQUESTED (3 blocking → D20).
Round 7: fact-check **APPROVED** (3 consecutive); adversarial CHANGES_REQUESTED
(1 blocking, PR A ↔ PR C boundary → D21 + §5 item 11).
Round 8: **BOTH tracks CHANGES_REQUESTED.** Fact-check refuted v10's "the PUT path
performs no pre-write `AIGet`"; adversarial found the **fourth instance** of the
recurring failure mode in §5 item 5 (D22).
Round 9: **BOTH tracks CHANGES_REQUESTED.** Adversarial found the **fifth instance**,
inside the v12 material written to close the fourth (D23): a type flip omitting
`mcp_server_ids` bypassed the gate and produced a permanently frozen Insight AI.
Fact-check found a missed third transition site and a FALSE api-validator citation
(`test_ai_lifecycle.py` is skipped at `:4`).
**v14 — 대표님 REVERSED the underlying policy: "인사이트도 mcp 사용가능하도록 해."**
Insight AIs may use MCP servers. No AI-type gate is added anywhere, which **voids
D13, D22 and D23 outright** (retained in §3.12b/§3.12e as recorded instances of the
failure mode, marked VOIDED and out of scope). Five rounds of gate design collapse into
one docs sentence. D18 is re-scoped: it no longer rides on a signature rewrite.
v9: 대표님 resolved §5 item 11 and §5 item 6. v10: §5 item 5 and §5 item 9. v12: §5 item 5(i)/(ii), now withdrawn.
Round 10: **both tracks CHANGES_REQUESTED.** Fact-check confirmed the v15 external
claims (PyPI/GitHub, D24) but found the D16 justification false. Adversarial found the
**SIXTH instance (D25)**: this document's own §5 item 11 exemption, implemented as it mandated,
disables the only ownership check on the whitelist → consume path.

Round 11: **both tracks CHANGES_REQUESTED.** Fact-check confirmed the D25 fix is sound
and complete (single write site, no first-write hole) but caught two overstated claims.
Adversarial found the **SEVENTH instance (D26)**: the ownership assertion D25 motivated
was prescribed at dispatch, which §3.1 of this same document proves is dead code, while
the live leak path at resolution got none. Also raised **D27** (pre-deploy deleted rows
keep credentials) as a genuinely OPEN decision.

Round 12: **both tracks CHANGES_REQUESTED.** Fact-check found no false code claim in the
v17/v18 material (the D27 column list is complete and correctly excludes the two metadata
columns) but caught a corrupted paragraph and four stale references. Adversarial found the
**EIGHTH instance (D28)**: D26 fixed the ownership half of D12 and never looked at the
member half, which has the identical shape — diagnosed, risk-rowed, and never given an
enforcement point, with its only promised test on the dead dispatch path.

**v16 — the five unconfirmed "Recommend …" items (§5 items 1, 3, 4, 7, 8) are now
closed** by 대표님 ("a로 가자" for the MCP link, recommendations accepted for the rest).
Investigating §5 item 3 surfaced **D24**: the docs advertise a stale third-party fork while
`uvx voipbin-mcp` installs the official repo — a published-falsehood defect the v14
draft had wrongly deferred to 대표님 as unjudgeable instead of simply checking.
**Readiness is still NOT claimed in this header.** Every §5 item is now decided, but
v10, v12 and v15 each made that claim and each was refuted within one round; a round must
return clean on both tracks before the design doc starts.
Scope: **PR A and PR C only.** Phase 2 LLM tool exposure (formerly PR B) is split
out to `2026-09-27-mcp-phase2-tool-exposure-analysis.md` and is NOT in scope here.

Supersedes (all retained for the failure-mode record):
- `…-mcp-server-customer-gap-analysis-v1-superseded.md` (§2.4 factually wrong)
- `…-mcp-server-revocation-and-docs-analysis-v2-superseded.md` (D1 severity overstated)
- `…-mcp-lifecycle-and-phase2-analysis-v3-superseded.md` (PR B scope understated)
- `…-mcp-lifecycle-and-phase2-analysis-v4-superseded.md` (PR B still unconverged; four citation defects)

Round 13: **both tracks CHANGES_REQUESTED, and both reviewers independently found the same
NINTH instance** — v19's D28 ORDER fix is a literal no-op, because
`resolveTeamMemberForSend` returns only `error` and `refreshMcpToolMap` reads the whitelist
from its `a` parameter. v19 also withdrew a correct §7 dispatch assertion on a false
impossibility claim. Both corrected in v21 via `resolveActiveAIIDFromAIcall`, the resolver
this package already uses at `listen_trigger.go:155`. Fact-check found no false code claim
in the v19/v20 material other than the fix itself, and confirmed Q17's split withdrawal is
clean.

v20: 대표님 declined round 12's PR A split proposal — "하나로 가자." One PR per repository
stands; Q17's "split out if it grows" hedge is withdrawn.

Round 14: **both tracks CHANGES_REQUESTED, TENTH instance, again in the newest edit.** v21's
replacement fix asserted "ordering is irrelevant" while `resolveActiveAIIDFromAIcall` has no
start-member fallback — so for a stale `CurrentMemberID` it resolves `uuid.Nil`, the refresh
is skipped, and the stale map survives, i.e. v21 regressed the degraded case while fixing the
normal one. v22 requires BOTH the order (repair first) and the resolved AI, plus an explicit
fallback contract, and corrects the residual paragraph (the risk is under-availability, not
acceptance) and adds the second unrefreshed path (reused contact_case).

Round 15: **both tracks CHANGES_REQUESTED, ELEVENTH instance, fourth consecutive one inside
D28.** v22 mandated "never fail the call" and cited `resolveTeamMemberAI` as already
providing it; that helper skips its fallback loop for 2 of 3 error modes (member present but
`aiHandler.Get` fails, team fetch fails), so at dispatch v22 would have converted today's
silent start-member degrade into `fillFailed` for every MCP tool call on that team. v22's two
offered mechanisms were also mutually exclusive (one needs a forbidden third RPC or a
signature change; the other mutates a resolver with six callers plus a duplicate copy). v23
replaces both with ONE narrow method, states that the reorder is optional hygiene rather than
the protection, and requires the AI-fetch-failure assertion v22 omitted.

Round 16: **both tracks CHANGES_REQUESTED — the first round to review CODE, and it found two
BLOCKING defects in it (fail-open dispatch gate, duplicated failing fetch) plus stale citations
the commit's own line shift created.** See §3.5c's v25 note. The decisive lesson: four
consecutive rounds of prose review had certified a prescription that, once compiled, contained
a fail-open security gate. **Reviewing an argument about code is not reviewing code.**

Round 17: **both tracks CHANGES_REQUESTED.** The code reviewer enumerated 480 reachable shapes
and proved the dispatch fallback added in v25 could recover in NONE of them — dead code that
doubled an RPC on the LLM's critical path — and found the new retry guard keyed on member id
while the fetch it deduplicates is keyed on AIID. Both are the round-15 duplicate-fetch defect
surviving its own fix. The fact-checker found 4 blocking citation classes (a `start.go` anchor
repeated 7 times pointing at a closing brace, pre-fix `mcp_tool.go` dispatch numbering in 12
places) and **a commit hash cited 3 times that does not exist in the repository**. See §3.5c v26.

Round 18: **both tracks CHANGES_REQUESTED.** The code reviewer independently re-derived the
480-shape subsumption argument (confirming the round-17 deletion was safe) and then found that
the resolver's own doc comment still described the WITHDRAWN unconditional-fallback contract and
told callers to "keep their existing behaviour rather than failing the call" — the exact opposite
of the fail-closed dispatch the same commit had just implemented. It also proved with two
mutations that **the reuse half of D28 had ZERO behavioural coverage**: discarding the resolved
AI, or skipping the refresh entirely, left the whole package green, because the only test on that
path built the handler without MCP mocks so `refreshMcpToolMap` returned at its nil guard before
reading the AI. The fact-checker found §5 item 1(c) still prescribing the deleted fallback, 7
surviving pre-fix `mcp_tool.go` anchors, three stale subtest counts, and §3.5c's heading still
announcing a live BLOCKER. See §3.5c v27.

Round 20: **both tracks CHANGES_REQUESTED, and this time the anti-drift mechanism itself was the
defect.** The code was confirmed correct by ten independent mutations, all caught. Every blocking
finding was in prose or in the new script. (1) The commit that mechanised the citation sweep
expanded a comment by five lines and shifted `refreshMcpToolMap` from `:372` to `:375`, stale in
ten places including §5 item 1(b) — the sixth consecutive instance of the same class, in the same
file, by the same mechanism. (2) The script could not see it: a range check passes any anchor that
stays inside the file, and a reviewer proved it by making a probe document cite `start.go:1` for an
arbitrary claim and still exiting 0. (3) The script's commit-hash check was logically inverted: it
reported tokens git resolved to a non-commit but silently skipped tokens git could not find at all,
so the round-17 blocker (a cited commit that had never existed) would pass today. (4) The
"fails twice over" test comment named a second failure mechanism that gomock's `Fatalf` makes
unreachable. Closed by giving the script symbol anchors, a corrected commit check, continuation and
comma citation forms, and `--extra-root` so sibling-repo citations are verified instead of
suppressed; the first run of the anchored gate immediately caught two of my own fresh errors.
See §3.5c v29.

Round 19: **both tracks CHANGES_REQUESTED, but neither blocking finding survived verification
intact.** The code reviewer reported that removing the reuse branch's team gate left the package
green ("NOT CAUGHT"). Re-running that mutation shows it FAILS
`Test_ServiceStart_serviceStartReferenceTypeConversation/normal`, so the claim is refuted — but the
reviewer's underlying observation was correct and worse than the headline: every reuse fixture left
`AssistanceType` at the zero value, so no reuse sub-case exercised a plain AI aicall at all, and the
gate's own comment claimed protection from "the reuse tests" that did not exist. Two further test
comments misdescribed the fixture (claiming a start-member AI whitelisting "a different server"
when it had no `McpServerIDs` at all) and the failure mechanism (an unexpected call that could not
occur against a nil slice). The fact-checker's mechanical sweep — 353 citations, zero out-of-range,
all 8 commit hashes valid — found that this commit's comment expansion had again shifted
`mcp_tool.go`, leaving five anchors on a bare `return res`, one of them inside the §5 item 1
prescription. See §3.5c v28.

## 0. Why this document was split

Rounds 1-4 of the review loop all returned CHANGES_REQUESTED. The pattern changed
at round 4: rounds 1-3 found factual errors in the lifecycle analysis, round 4
found only **PR B scope discoveries** (a fourth AIcall path missing the tool map,
a whitelist-reintroduction hazard, an asymmetric sanitizer in the team path) while
confirming every lifecycle claim as correct.

CEO decision (2026-09-27): split the analysis. Verbatim: "권고대로 진행해" in
response to the recommendation that PR A/C analysis is converged and should not
wait on PR B's, since PR A fixes live security defects and is PR B's prerequisite
anyway.

This document therefore covers only the converged half. PR B keeps iterating
separately.

## 1. CEO decisions locked 2026-09-27

| # | Question | Decision |
|---|---|---|
| Q1 | Implement the documented "MCP tools presented to the LLM" behavior, or correct the docs? | **Implement** (belongs to PR B, out of scope here). "§5 item 1. 구현하자." |
| Q2 | How much of the lifecycle scope lands? | **All of it.** "§5 item 2. 전부." |
| Q3 | Zero stored credentials on delete? | **Yes.** "Q3. 지우자. 그게 맞지." |
| Q4 | Dead `mcp_tools_list_cache_ttl_seconds` flag: implement or remove? | **Remove the flag, no cache.** |
| Q5 | PR B as one PR or split? | **One PR.** "하나로 가자." (PR B doc) |
| Q6 | Split the analysis so A/C can proceed without waiting on B? | **Yes.** "권고대로 진행해" |
| Q7 | D21: how to handle an AI whose whitelist already holds a deleted id? | **Skip the DELETED-row rejection for already-stored ids; existence and ownership still run on every id (D25); reject newly added ones.** "skip-not-reject 방식으로 하자." + "이후에 AI 업데이트를 할 때 ... 이때는 reject 를 해야하지 않을까?" |
| Q8 | Add a prune loop that strips deleted ids from every affected AI? | **No.** "비쌀려나?" → no transaction on delete, un-indexed JSON column, one `ai_updated` webhook per affected AI, and skip-not-reject already resolves D21 alone |
| Q9 | D13: add an AI-type parameter to `ValidateMcpServerIDs`? | **REVERSED in v14 — No.** "인사이트도 mcp 사용가능하도록 해." Insight AIs may use MCP servers. No type gate anywhere; D13/D22/D23 are voided. PR A owes one docs sentence stating the behavior |
| Q10 | D17: reject `auth_type: "oauth"` on POST/PUT? | **Yes, reject.** "좋아, 네 제안대로 가자" Gate the write handlers, not `IsValid()`; reject the transition INTO oauth, not the value |
| Q11 | ~~Where does the Insight AI-type gate live?~~ | **MOOT (v14).** No gate exists. The v12 answer (both write and consume path) is withdrawn |
| Q12 | ~~Does PR C clear `mcpServerIds` on the Insight flip?~~ | **MOOT (v14).** No gate, so nothing to clear and nothing to hide. `mcpServerIds` survives a type flip and the MCP card stays visible for every type |
| Q13 | Does the `RowsAffected` check apply to every `McpServerUpdate` caller, token rotation included? | **Yes, uniformly** (v15). A rotation against a soft-deleted row is a bug signal; swallowing it hides it. Design doc must confirm the failure degrades gracefully |
| Q14 | Does `McpServerDelete` gain the same predicate (D16)? | **Yes**, but the handler keeps **200 on repeat DELETE** and stops re-publishing `EventTypeDeleted`. 200 is required because DELETE is idempotent and GET still returns the soft-deleted row; v15's "the validator fixture would fail on 404" reason was FALSE (`cleanup_report.py:86` tolerates 404) |
| Q15 | Naming-collision rename scope, and the third-party MCP link | **Heading only**; and **replace** `nrjchnd/voipbin-mcp` with `github.com/voipbin/mcp`, remove the fork link entirely ("a로 가자"). `uvx voipbin-mcp` already installs the official repo per PyPI, so the published link contradicts the published command |
| Q16 | Add api-validator MCP coverage now? | **Defer until ETC-18 is fixed.** The validator leaks ~4 MCP servers per run into production (100 measured); more coverage before the fix accelerates it |
| Q17 | PR A docs scope for D14, and is PR A split? | **NOT split — one PR per repository** (대표님 확정: "하나로 가자"). Every PR A item, including the three webhook event types and the `ai_struct_ai.rst:69` contradiction, ships in ONE `monorepo` PR. Round 12's A1/A2 (logic vs docs) split proposal is **declined**; the earlier "split out if it grows" wording is withdrawn |
| Q18 | Does the already-stored-id exemption cover the ownership check too? | **NO — D25 (§3.5a), SIXTH instance.** The exemption applies to the `tm_delete` predicate ONLY. `ValidateMcpServerIDs` gains a `storedIDs` parameter so existence and ownership run on every id; v12-v15's "no signature change" claim created a cross-tenant hole and is retracted |
| Q19 | Where does the D12 ownership assertion live? | **BOTH resolution and dispatch** — D26 (§3.5b), SEVENTH instance. Dispatch-only was dead code (§3.1); resolution is where the credential is decrypted and sent. `resolveTools` already holds `a.CustomerID`, so cost is zero |
| Q20 | Do pre-deploy soft-deleted rows get their credentials zeroed? | **Yes — one-off Alembic data migration** ("a 로 가자"), all seven columns where `tm_delete IS NOT NULL`, authored in PR A and applied by 대표님, downgrade an explicit no-op. The affected-row count is unmeasured and remains a design-doc precondition (it changes the PR body's impact claim, not the decision) |
| Q21 | How is D12's team-member half enforced? | **IMPLEMENTED, not prescribed** (`3fca779e2`, `dd2fb0b39`): `resolveActiveAIForMcp` (`helpers.go:194`) resolves the CURRENT member and degrades to the start member on ALL failure modes; used at `start.go:375` (team-gated — applying it unconditionally there breaks three existing tests) and `mcp_tool.go:151` (all types, with a fail-closed `default`). Shared resolver untouched. The reorder of `resolveTeamMemberForSend` was deliberately NOT done. v19/v21/v22/v23 prescriptions ALL withdrawn — D28 (§3.5c) |

## 2. Issue statement (PR A/C scope)

**PR A — lifecycle correctness + docs (`voipbin/monorepo`):**
- **D1** Soft-delete does not revoke tool access. A deleted server keeps
  `status = "active"`, every read path ignores `tm_delete`, so the AI tool path
  keeps making credentialed outbound `ListTools` calls to it forever.
- **D2** `McpServerUpdate` has no `tm_delete` predicate and never clears it, and
  `status` is PUT-settable, so **a deleted server is mutable and can be flipped
  back to `active` with a new URL and a new secret.**
- **D3** OAuth start/complete reach a deleted server; `Complete` has *no*
  ownership/existence re-check by explicit design comment, so tokens get written
  onto a `tm_delete`-populated row.
- **D4** `McpServerDelete` retains `SecretCiphertext`,
  `AccessTokenCiphertext`, `RefreshTokenCiphertext` indefinitely.
- **D5** Published docs give a 404 endpoint path (`/mcp_servers` vs `/mcpservers`)
  plus two broken `:ref:` targets.
- **D24** `ai_overview.rst:685` advertises a third-party fork
  (`github.com/nrjchnd/voipbin-mcp`, ~17 months stale, no tests, no CI) as "A
  recommended open-source implementation", while the documented command `uvx
  voipbin-mcp` installs `github.com/voipbin/mcp` per PyPI. **The published link and the
  published command point at different code.** Same class as D5. Resolved in §5 item 3:
  replace the link, label it official, remove the fork.
- **D6** `mcp_tools_list_cache_ttl_seconds` is a dead flag; the design doc
  justifies a *separate* design decision by citing this nonexistent cache.
- **D12** `toolHandleMcpCall` never asserts `server.CustomerID == c.CustomerID`,
  and on team AIcalls it validates the whitelist against the **start** member while
  the session runs the **current** member.
- ~~**D13**~~ **VOIDED in v14.** `ValidateMcpServerIDs` takes no AI type. 대표님
  confirmed this is intended ("인사이트도 mcp 사용가능하도록 해"): Insight AIs may use
  MCP servers, so the asymmetry with `tool_names` is deliberate, not a defect. PR A adds
  no type gate; it owes only one docs sentence stating the behavior (see §5 item 5).
- **D14** The three mcpserver webhook event types are undocumented, and
  `ai.McpServerIDs` carries `,omitempty` while the RST says it defaults to `[]`.
- **D15** `McpServerUpdate` discards `RowsAffected`, so **the `tm_delete` predicate
  D2 requires would turn a rejected PUT into a silent 200** with unchanged fields
  and a spurious `EventTypeUpdated`. Gating alone is not enough; the function must
  report "no row matched".
- **D16** `McpServerDelete` has no `tm_delete` predicate and ignores `RowsAffected`,
  so **delete is not idempotent**: a second DELETE re-stamps `tm_delete` with a
  fresh timestamp and re-publishes `EventTypeDeleted`.
- **D17** `auth_type: "oauth"` can be set directly via POST/PUT with a
  customer-supplied secret, which `ai_struct_mcpserver.rst:35` says is impossible
  and which no code enforces.
- **D18** `ValidateMcpServerIDs` runs **after** the AI write has already committed,
  so a rejected whitelist leaves an orphaned/mutated AI behind and still returns 400.
- **D19** `ai_struct_mcpserver.rst` documents a `9999-01-01` timestamp sentinel this
  resource does not use (it uses `null`), plus a bogus "(enum string)" label on
  `oauth_vendor`.
- **D20** `mcpserverhandler.Update` **short-circuits to `h.Get` before touching the
  DB** when every field is omitted, so a `PUT {}` on a deleted server returns 200
  with the full row — a path NO dbhandler-level gate can reach.
- **D21** An AI holding a deleted MCP server id must stay saveable. Write-time
  validation **skips ids already in the stored whitelist and rejects only newly added
  ones** (§5 item 11, 대표님 확정). Without the skip, every save of that AI 400s forever,
  for any unrelated edit; without the reject, attaching a deleted server fails
  silently. This is a PR A ↔ PR C cross-boundary defect that neither PR sees alone.
- **D25** The D21 exemption above, as v12-v15 mandated it (filter the incoming id list,
  "no signature change needed"), also exempts already-stored ids from the **customer
  ownership** check — existence and ownership share one loop
  (`mcpserver_validation.go:36-56`). A foreign id, once stored, then reaches resolution
  and is used with its owner's decrypted credential. **SIXTH instance**; §3.5a. Fixed by
  a `storedIDs` parameter so the exemption is predicate-scoped.
- **D26** The D12 ownership assertion was prescribed at dispatch
  (`mcp_tool.go:160-176`) only — the one path §3.1 proves is **unreachable in
  production** — while the live leak runs through resolution (`:72-88` → `ListTools` →
  `client.go:73`/`:80` decrypt → outbound). **SEVENTH instance**; §3.5b. Fixed by
  asserting at both sites; `resolveTools` already holds `a.CustomerID`.
- **D27** Credential zeroing is forward-only: rows soft-deleted BEFORE the deploy keep
  their ciphertext, and re-deleting them matches zero rows and reports 200 while zeroing
  nothing. **RESOLVED (v18): a one-off Alembic data migration** nulls all seven credential
  columns where `tm_delete IS NOT NULL` (§5 item 2), authored in PR A and applied by
  대표님, with an explicit no-op downgrade.
- **D28** — **IMPLEMENTED in this branch (commits `3fca779e2`, `dd2fb0b39`), not a
  prescription.** The team half of D12: the MCP paths resolved the START member's AI, so after
  a mid-conversation member switch a team aicall built its tool map from, and authorised tool
  calls against, the wrong member's whitelist. Fixed by `resolveActiveAIForMcp`
  (`helpers.go:194`), a dedicated resolver that returns the CURRENT member's AI and degrades to
  the start member on every failure mode, used at `start.go:375` (team-gated) and
  `mcp_tool.go:151` (all types). The shared `resolveActiveAIIDFromAIcall` was deliberately NOT
  modified — six callers plus a twin in `messagehandler/event.go:45`. Four prescriptions were
  withdrawn before the code was written (§3.5c); the code then exposed two more errors that
  four rounds of prose review had not (see §3.5c's v24/v25 notes).
- ~~**D22**~~ **VOIDED in v14** by the §5 item 5 reversal — no AI-type gate exists, so a gate
  that under-enforces and freezes AIs cannot occur. Retained in §3.12b as the fourth
  recorded instance of the recurring failure mode. Not PR A or PR C scope.
- ~~**D23**~~ **VOIDED in v14** by the §5 item 5 reversal — no gate to bypass, no row to
  freeze; its three open decisions are withdrawn. Retained in §3.12e as the fifth
  recorded instance, created by the fix for the fourth. Not PR A or PR C scope.

**PR C — square-admin parity (`voipbin/monorepo-javascript`):**
- **D8** `mcp_server_ids` is settable only on the AI detail page, though the
  backend accepts it on `POST /ais` too.

Deferred to the PR B document: D7 (tools never advertised to the LLM), D9
(`inputSchema` tag), D10 (`AllowedToolNames` interaction), D11 (name validation).

## 3. Validity re-check (code-verified; unchanged and re-confirmed in round 4)

### 3.1 D1: revocation does not revoke. CONFIRMED.

| Step | File:line | Finding |
|---|---|---|
| Soft delete | `bin-ai-manager/pkg/dbhandler/mcpserver.go:163-182` | sets ONLY `tm_update` + `tm_delete`; `status` untouched, stays `active` |
| DB read | same `:53-59` | `Where(sq.Eq{"id"})` only, no `tm_delete` predicate |
| DB read entry | same `:84-86` | `McpServerGet` → `mcpserverGetFromDB` |
| Handler read | `pkg/mcpserverhandler/handler.go:93-107` | `Get` maps `ErrNotFound`→404, no `TMDelete` check |
| Resolution | `pkg/aicallhandler/mcp_tool.go:72-88` | gates on `err != nil` and `Status != StatusActive` only |
| Dispatch | `pkg/aicallhandler/mcp_tool.go:151-168` | whitelist + `Status` only |
| Transport | `pkg/mcptoolhandler/client.go:190-196` (`ListTools`), `:213-219` (`CallTool`) | both call `db.McpServerGet` directly; check NEITHER `Status` NOR `TMDelete` |
| Whitelist validation | `pkg/aihandler/mcpserver_validation.go:37` | validates via `db.McpServerGet`; a deleted server passes |
| Grep proof | `TMDelete\|tm_delete` over `pkg/aicallhandler/mcp_tool.go`, `pkg/mcpserverhandler/`, `pkg/mcptoolhandler/` | **zero matches**, even including tests |
| Prune on delete | `McpServerIDs` in `pkg/` (excl. tests/mocks) | 24 hits, none prunes a deleted id from `ai.mcp_server_ids` |

**Live exposure today.** On every AIcall/insight session start that calls
`resolveTools`, `ListTools` (`mcp_tool.go:84`) fires against every whitelisted
server including deleted ones, decrypting the stored secret and issuing an
authenticated outbound HTTP request to the customer's URL. That is the real,
ongoing defect in PR A's scope.

`CallTool` dispatch is currently **unreachable in production** because the merged
tool list `resolveTools` returns is discarded by every caller (`start.go:1184 resolveTools`,
`insight_session.go:225 resolveTools`, `mcp_tool.go:293 resolveTools` all use `_,`), and the lists actually
sent to models are built elsewhere with no MCP awareness. That is D7's subject
matter and belongs to the PR B document; it is stated here only to keep PR A's
severity claim honest and to explain why PR A must precede PR B.

Contradicted UI promise:
`square-admin/src/views/mcpservers/mcpservers_detail.js:496` — "Any AI still
whitelisting it will stop being able to call it." False today.

### 3.2 D2: a deleted server is mutable and resurrectable. CONFIRMED.

- `pkg/dbhandler/mcpserver.go:146-149`: `sq.Update(mcpserverTable).SetMap(...)
  .Where(sq.Eq{"id": id.Bytes()})`. No `tm_delete` predicate. It sets `tm_update`
  and **never clears `tm_delete`**. `RowsAffected` is ignored (`:154-156`) — see
  §5 item 2.
- `pkg/mcpserverhandler/handler.go:168-169`: `status` is PUT-settable.
- Net: `PUT /mcpservers/{id}` on a deleted server succeeds, can set
  `status = active`, and can replace `url` and the secret.

This retracts any claim that "`tm_delete` is terminal": on the write paths it is
not, which is why gating reads alone leaves a resurrection path.

### 3.3 D3: OAuth reaches a deleted server; Complete has no re-check. CONFIRMED.

- `pkg/mcpoauthhandler/start.go:37` `h.db.McpServerGet(ctx, *mcpServerID)` — the
  reconnect ownership check, which ignores `tm_delete`, so an OAuth reconnect can
  be **started** against a deleted server, creating a `mcpoauthstate` row.
- `pkg/mcpoauthhandler/complete.go:112-133` then calls `McpServerUpdate` on that
  id. The comment at `:113-115` states verbatim that ownership "was already
  verified in Start … so this is a plain update, not a second ownership check."
  **No existence or deletion re-check at all**, so a server deleted *between*
  Start and Complete (window = `stateTTL`) receives fresh
  `access_token`/`refresh_token` ciphertext on a deleted row. Gating `Start` alone
  does not close this.
- `pkg/mcpoauthhandler/access_token.go:37-112` `GetValidAccessToken` receives an
  already-fetched `*mcpserver.McpServer` from `pkg/mcptoolhandler/client.go:97`
  and persists rotated tokens via `McpServerUpdate:108`. It has NO independent
  read, so a transport gate at `client.go:191/214` also closes the refresh path —
  no separate gate needed. Until then, `ListTools` against a deleted OAuth server
  keeps refreshing and re-persisting vendor tokens on it.
- **Verified clean, do NOT over-gate:** the public unauthenticated
  `GET /mcpservers/oauth/callback`
  (`bin-api-manager/cmd/api-manager/main.go:310-321`) calls `CallbackExists`
  (`pkg/mcpoauthhandler/callback.go:17-31`), which touches only
  `McpOAuthStateGet` and never `McpServer`. No deleted-server exposure.

### 3.4 D4: credentials survive delete. CONFIRMED, and schema-safe to zero.

`McpServerDelete` (`pkg/dbhandler/mcpserver.go:163-182`) writes only `tm_update`
and `tm_delete`. `SecretCiphertext`, `SecretNonce`, `AccessTokenCiphertext`,
`AccessTokenNonce`, `RefreshTokenCiphertext`, `RefreshTokenNonce`, `KeyVersion`
all remain indefinitely.

**Verified no schema obstacle:** `bin-dbscheme-manager` migration
`9b0ad37e0360_ai_mcp_servers_create_table.py:35-37` declares `secret_ciphertext`
(blob), `secret_nonce` (binary(12)), `key_version` (smallint) with **no NOT NULL**,
and `62c10f986f07_ai_mcp_servers_add_oauth_columns.py:30-34` declares all six OAuth
columns explicitly nullable. Zeroing is safe.

### 3.5 D12: no ownership check; team validates the wrong member. CONFIRMED.

`pkg/aicallhandler/mcp_tool.go:127-178` has exactly three gates:
`lookupMcpToolRef` (`:137`), whitelist membership (`:160`), and
`server.Status != StatusActive` (`:172`). **There is no
`server.CustomerID == c.CustomerID` assertion anywhere in the dispatch path.** The
only ownership check **on the AI-whitelist → consume path** is write-time
`ValidateMcpServerIDs`
(`pkg/aihandler/mcpserver_validation.go:50`, `srv.CustomerID != customerID`) — a
single-layer defense.

The whitelist check also validates the wrong AI on team calls. `mcp_tool.go:151`
calls `h.resolveAI(...)`, whose team branch (`pkg/aicallhandler/start.go:75`)
resolves **`t.StartMemberID`**:

```go
a, memberID, err := h.resolveTeamMemberAI(ctx, t, t.StartMemberID)
```

while the running session tracks `CurrentMemberID`
(`bin-pipecat-manager/pkg/pipecatcallhandler/run.go:150-163`,
`resolveAIFromAIcall` `:218-224`). On a team call that has transitioned members,
the whitelist check runs against the START member's `McpServerIDs`. Two failure
directions: a tool the current member legitimately whitelists is refused, and a
tool only the start member whitelists is ACCEPTED for the current member.

**Corrected blast radius (round 4).** The same mismatch affects
`start.go:1184 resolveTools` and `mcp_tool.go:293 resolveTools` (`refreshMcpToolMap`, called from
`start.go:375` with the `a` resolved at `start.go:187`). It does **NOT** affect
`insight_session.go:225 resolveTools`: `writeInsightSessionMetadata` is reached only from
`insight_session.go:143` inside `runInsightSessionRefresh`, which returns early at
`:68-72` when `AssistanceType != AssistanceTypeAI`, and its `a` comes from
`h.aiHandler.Get(...)` at `:104`, not from `resolveAI`'s team branch. Teams never
reach `:225`.

### 3.5a D25: this document's own §5 item 11 exemption disables the whitelist path's only ownership check. CONFIRMED. BLOCKER.

**SIXTH instance of the recurring failure mode**, and the first one that is a security
regression rather than a usability one. Unlike D22/D23 it was not introduced by a
product decision — this document introduced it, then asserted it was free.

`ValidateMcpServerIDs` (`pkg/aihandler/mcpserver_validation.go:35-60`) runs two checks
in a single loop over the incoming ids:

```go
for _, id := range ids {
    srv, err := h.db.McpServerGet(ctx, id)          // :37  existence
    ...
    if srv == nil || srv.CustomerID != customerID { // :50  OWNERSHIP
```

§5 item 11 grants already-stored ids an exemption so that a soft-deleted-but-held id cannot
freeze the AI (D21). v12-v15 additionally claimed the exemption needs **"zero extra DB
reads and no signature change"**, explicitly retracting an earlier draft that asked for
a parameter. But with no signature change the function cannot see the stored list, so
the only way to implement the exemption is to **call it with the newly-added subset
only** — which exempts already-stored ids from the ownership check in the same stroke.

Why that is exploitable. §3.5 establishes this line is the **only** ownership check on the
AI-whitelist → consume path (the REST surface and the OAuth reconnect path have their
own, per `servicehandler/mcpserver.go:121` and `mcpoauthhandler/start.go:48`). Verified again in v16: neither consume-path site compares
`CustomerID` — `resolveTools:72-88` checks `Get` success and `Status` only, and
`toolHandleMcpCall:160-176` checks whitelist membership, `Get` success, and `Status`
only. §5 item 1 as v16 wrote it prescribed an ownership assertion at **dispatch** (`:160-176`), not at
resolution. So a foreign `mcp_server_id`, once stored, reaches `resolveTools:84`
→ `ListTools` → `client.go:190-196`, whose `buildAuthHeader` decrypts the stored
credential at `client.go:73` (bearer) / `:80` (api_key) and attaches it to an outbound
POST at `:125`/`:136-138` — **another customer's decrypted secret on the wire**. Storing it the first time is what the ownership check exists to
prevent, and the exemption removes exactly that.

This is the document's own closing lesson (the retrospective lessons, "an exemption written for one
predicate does not cover another") applied backwards: an exemption written for the
`tm_delete` predicate was allowed to cover existence and ownership as well.

**Resolution (v16), decided in §5 item 11:** `ValidateMcpServerIDs` gains a `storedIDs`
parameter; existence and ownership run for every id, and only the `tm_delete` rejection
consults `storedIDs`. One signature change, still zero extra DB reads. §7 gains the
assertion that **an already-stored id owned by another customer is still rejected** —
absent that test, the regression is silent.

### 3.5b D26: the prescribed ownership assertion sits on the one path that never runs. CONFIRMED. BLOCKER.

**SEVENTH instance**, and the second in a row created by a fix-up edit rather than by the
original analysis. D25 (v16) correctly diagnosed that the harm is at **resolution**. The
gate-placement decision it was supposed to inform (§5 item 1) still said **dispatch**,
because that line was written in v5 for D12 and was never revisited when D25 moved the
diagnosis.

Two facts, both re-verified in v17, make dispatch the wrong and resolution the right
place:

1. **Dispatch does not execute in production.** §3.1 already establishes this
   for a different purpose: every caller of `resolveTools` discards the merged tool list
   — `start.go:1184 resolveTools`, `insight_session.go:225 resolveTools`, `mcp_tool.go:293 resolveTools` all bind `_,` — so no
   MCP tool name is ever advertised to the LLM and `tool.go:142 toolHandleMcpCall` never dispatches one.
   Placing the only ownership assertion there makes it **dead code until PR B lands**.
2. **Resolution is where the credential is used.** `resolveTools:84` calls `ListTools`,
   which reaches `client.go:190-196` (takes `serverID` only, no customer context, no
   `tm_delete` filter) and then `buildAuthHeader`, which decrypts the stored secret at
   `client.go:73` (bearer) / `:80` (api_key) and attaches it to an outbound POST to the
   other customer's URL at `:125`/`:136-138`.

So the v16 document ended in this state: the write-time hole closed (D25), the harm
correctly located at resolution, and the runtime assertion placed on a dead path. Net
runtime defense against a foreign id that is already stored: **none**.

**Resolution (v17), folded into §5 item 1:** assert ownership at **both** sites.
`resolveTools` already holds `a.CustomerID` because `ai.AI` embeds `identity.Identity`
(`models/ai/main.go:46`, `identity.go:9`), and its loop already fetches `server` at
`:73`. Resolution skips (best-effort contract `:44-51`); dispatch refuses with the
existing error string. Zero new DB reads, zero signature changes, zero new
customer-facing strings. §7 gains a resolution-side counterpart to the existing
dispatch-side D12 assertion.

**Note on the correct scope of the "only ownership check" claim.** v16 wrote that
`mcpserver_validation.go:50` is "the only ownership check in the system". Round 11
refuted this: `mcpoauthhandler/start.go:48` guards reconnect (and this document says so
itself in §3.3), `complete.go:68` guards the state row, and
`bin-api-manager/pkg/servicehandler/mcpserver.go:121`/`:158`/`:181` gate the REST
surface. The accurate and still-sufficient claim is that it is **the only ownership check
on the AI-whitelist → consume path**. Corrected wherever it appeared.

### 3.5c D28: D12's team half had no enforcement point and its only promised test was on the dead path. **CLOSED IN CODE (`3fca779e2`, `dd2fb0b39`, `7dced7a90`, `90d2e2f44`, `66c00676e`) — see the v24 through v29 notes at the end of this section; the NEWEST is v29.**

> **Reading order.** Everything between here and the v24 note is the ORIGINAL DIAGNOSIS,
> written in the present tense while the defect was live, and kept as the record of how four
> prose prescriptions went wrong. It does NOT describe the current code. For what the code
> does now, and for what an implementer must not undo, read the v29 note (the newest) and §5 item 1.

**EIGHTH instance**, and the third in a row created by a fix-up edit. D26 corrected the
ownership half of D12 from dispatch to resolution. It did not look at the **member**
half of the same defect, which has the identical shape.

**The live defect.** On a team AIcall, `resolveAI` resolves the **start** member
(`start.go:75`, `h.resolveTeamMemberAI(ctx, t, t.StartMemberID)`). On the reuse branch,
`start.go:375` called `refreshMcpToolMap(ctx, res, a)` with that start-member `a` (it now passes the resolved `mcpAI`), which
reaches `resolveTools` (`mcp_tool.go:293 resolveTools`) and fires `ListTools` at `:84` — a credentialed
outbound request to the **start** member's MCP servers, not the running member's.

**And the mechanism to fix it already exists, two lines later.** `start.go:383-387`
calls `resolveTeamMemberForSend(ctx, res)`, which resolves `c.CurrentMemberID`
(`send.go:166`) and overrides the engine model in memory. So the reuse branch already
knows how to get the current member's AI — it just does the MCP refresh **before** that
call instead of after. This is an ordering defect with a ready-made helper, not a new
subsystem.

**What the document did with it.** §3.5 diagnosed it correctly, §6 carries a High risk row
pointing at §3.5, and §7 promises a test that "the team path validates the CURRENT
member's whitelist" — but:

- v5-v16's §5 item 1 gate-placement list named **no** member-resolution prescription at all; v17+ carries the D28 block.
- The promised §7 test targets whitelist validation, which lives at `mcp_tool.go:160`
  — **dispatch**, which §3.1 proves is dead code. As written the assertion is
  unimplementable: `toolHandleMcpCall` calls `resolveAI`, which cannot see
  `c.CurrentMemberID`.

So the diagnosis was right, the risk row was right, and the enforcement point was never
named — for the second time in two rounds, on the two halves of one defect.

**~~Resolution (v19): fix the ORDER, not the gate.~~ WITHDRAWN in v21 — the hoist is a
literal no-op.** Both round-13 reviewers found this independently, and they are right.

**Why the v19 prescription cannot work.** `refreshMcpToolMap(ctx, existing *aicall.AIcall,
a *ai.AI)` (`mcp_tool.go:288`) reads the whitelist from its **`a` parameter** only
(`:293` `h.resolveTools(ctx, a)`); `existing` is used solely for `existing.ID` on the
re-read and write (`:289`, `:300`). And `resolveTeamMemberForSend(ctx, c *aicall.AIcall)`
(`send.go:154`) returns **only `error`**: it resolves the current member's AI into a local
at `:166`, then keeps just the engine model (`:173` `c.AIEngineModel = a.EngineModel`) and,
on fallback, `c.CurrentMemberID` (`:181`). **It never returns or rebinds `a`.** So moving
the call earlier leaves `refreshMcpToolMap` receiving the same start-member AI (bound at
`start.go:187` from `resolveAI` → `:75` `t.StartMemberID`). Reordering is *safe* and
*completely ineffective* — after the v19 change the §7 assertion would still fail.

This is the **NINTH instance**, and the fourth in a row created by the edit that closed the
previous one. §3.5c had `refreshMcpToolMap(ctx, res, a)` quoted on screen, parameter list
visible, and still prescribed ordering.

**The v19 fallback clause was also wrong as written.** "Pass the current-member `a` that
helper already resolves" is not free: the helper discards it, so producing it requires a
signature change — precisely what the v19 text claimed was unnecessary.

**~~Resolution (v21): use the resolver this package already owns, at BOTH sites.~~
AMENDED in v22 — the helper has no fallback, so placement and the failure contract are
load-bearing.** Round 14 found this; it is the **TENTH instance**, the fifth consecutive one
introduced by the edit that closed the previous one.

**What v21 got wrong.** `resolveActiveAIIDFromAIcall` (`helpers.go:21`) returns `uuid.Nil`
with only a `Warnf` when `ac.CurrentMemberID` is not in `t.Members` (`:35`). It has **no
start-member fallback**. `resolveTeamMemberAI` (`start.go:42-52`) **does** have one, and
`resolveTeamMemberForSend` (`start.go:384`) is the only thing that repairs a stale
`CurrentMemberID`, persisting the repair (`send.go:176-183`). v21 placed its call in the reuse
branch above that repair (at the line the refresh now occupies), **before it**, and then asserted "Ordering is irrelevant." For a stale
member that yields `uuid.Nil` → `aiHandler.Get` fails → the refresh is skipped (its error is
only `Warnf`'d) → **the stale map survives, which is the exact state D28 exists to
remove.** Today's code, by passing the start-member AI, at least refreshes with the correct
fallback target. So v21 regressed the degraded case while fixing the normal one, and
"ordering is irrelevant" was written with `resolveTeamMemberForSend` quoted three lines away.

**~~Resolution (v22): repair first, then resolve once, and name the failure contract.~~
AMENDED in v23 — the named mechanism does not implement the mandated contract.** Round 15
found this; **ELEVENTH instance**, and the fourth consecutive one inside D28.

**What v22 got wrong.** v22 mandated "fall back to the start member, never fail the call"
and cited `resolveTeamMemberAI` as already defining it. It does not.
`resolveTeamMemberAI` (`start.go:30-54`) reaches its fallback loop (`:43-51`) for exactly
**one** condition: the requested member is absent from `t.Members`. It returns an error
without ever reaching the fallback when:

1. the member IS present but `aiHandler.Get(m.AIID)` fails (`:36`) — transient RPC error, or
   a deleted AI row;
2. `teamHandler.Get` fails upstream (`send.go:160-163`);
3. neither the member nor the start member is in the team (`:53`).

So v22's contract held for one of three error modes. At dispatch that is a **regression we
introduce**: today `resolveAI` fetches the START member's AI (`start.go:69-79`) and never
reads `CurrentMemberID`, so a broken current-member AI cannot affect MCP dispatch at all.
Under v22 it becomes `fillFailed("could not retrieve AI configuration")` (`mcp_tool.go:151`)
— **every MCP tool call for that team dies.** v22 forbade exactly this outcome in the same
paragraph that prescribed it.

**And v22's two offered mechanisms were mutually unsatisfiable.** "Reuse
`resolveTeamMemberAI`" needs a `*team.Team`, which only `resolveTeamMemberForSend` holds and
discards (it returns `error`), so it costs either a signature change or a third
`teamHandler.Get` — which v22's own clause (d) forbids. "Give the helper the same fallback"
touches `resolveActiveAIIDFromAIcall`, which has **six other callers**
(`send.go:78`, `:106`, `tool.go:87`, `tool_insight.go:1430`, `listen_trigger.go:155`,
`start.go:396`) driving message attribution, **plus a duplicate copy in
`messagehandler/event.go:45`** that would silently diverge. v22 called that "one place, one
behaviour" without enumerating any of it.

**The reorder is NOT load-bearing.** Stated plainly, because v22 presented a cosmetic and a
load-bearing mechanism as co-required. `resolveTeamMemberForSend`'s caller only `Warnf`s
(`start.go:384`), so when the repair fails `CurrentMemberID` is still stale when the refresh
runs; and when the resolution itself carries the fallback, it yields the same AI whether it
runs before or after the repair. **The fallback-bearing resolution does all the work.**

**ROUND 20 (v29): the mechanism written to stop citation drift could not see citation drift, and
the commit that added it drifted six anchors.**

1. **A range check does not close this class.** `check_doc_citations.py` verified only that a cited
   line number existed in the file. Every one of the six recurrences was an anchor that stayed in
   range and slid onto the wrong construct, so the gate reported OK on live drift in §5 item 1(b) —
   the one line an implementer opens first. Closed by SYMBOL ANCHORS: a citation may carry the
   identifier its line must contain (`` `start.go:375 refreshMcpToolMap` ``), and a mismatch fails
   the run. The summary now prints how many citations are unanchored, so the unverifiable remainder
   is a visible number instead of an invisible assumption; `--strict-anchors` fails on any.
2. **The commit-hash check was inverted.** It reported tokens `git cat-file -t` resolved to a
   non-commit but silently skipped tokens git could not resolve at all (exit 128) — exactly the
   round-17 blocker, a hash cited three times that had never existed. Now a nonexistent token fails,
   with an allowlist for the five Alembic revision ids, which are 12-hex but not git objects.
3. **Cross-repo citations were suppressed, not checked.** `EXTERNAL_HINTS` skipped every
   square-admin and api-validator path. Adding `--extra-root` and resolving them for real
   immediately exposed a `:514-528` continuation bound to the wrong file. Two more forms the regex
   never extracted (`client.go:190,213` and the `62c10f986f07…:30,:31,:33,:34` column list, the most
   load-bearing citation in PR A) are now parsed.
4. **`refreshMcpToolMap` moved `:372`→`:375`** because round 19's own comment fix added five lines.
   Ten sites corrected, plus `:382-386`→`:383-387`, `:388-391`→`:392-395`, the
   `resolveActiveAIIDFromAIcall` caller →`:396`, and `:600`→`:603`. The anchored gate caught two
   fresh errors of mine on its first run (`validateMcpServerIDs` is exported, and
   `McpServerUpdate` is at `:134`, not `:146`).
5. **The "fails twice over" test comment was half false.** gomock's `Fatalf` Goexits inside
   `resolveTools`, so the persist-step callback never runs on that mutation. Verified by executing
   it and grepping the output for the callback's message: absent. Rewritten to say what each
   mechanism actually catches.

Lesson recorded: *a verification mechanism must be tested against the defect it exists to catch,
with a deliberately broken input, before it is trusted. Both of this script's original checks
passed their target defects.*

**Earlier note (v28), retained as the record of the fifth implementation round.**

**ROUND 19 (v28): the coverage added in v27 ran, but it ran on the wrong assistance type — every
reuse fixture was untyped, so the gate that keeps D28 fixed was still unobservable on the reuse
path, and its comment claimed protection that did not exist.**

1. **A refuted blocking claim, with a real defect underneath it.** The reviewer reported that
   deleting the reuse branch's team gate left the package green. Re-running that mutation
   (`if res.AssistanceType == aicall.AssistanceTypeTeam` → `if true`) FAILS
   `Test_ServiceStart_serviceStartReferenceTypeConversation/normal`, so the headline is wrong. But
   the supporting observation was right: all four reuse fixtures left `AssistanceType` at the zero
   value, which reaches the resolver's fail-closed `default` and returns nil before any RPC. No
   reuse sub-case had ever exercised `AssistanceTypeAI`, so the gate was covered only by an
   unrelated service-level test. Fixed by typing the `reuse: alive previous pipecat` fixture
   `AssistanceTypeAI`, wiring its MCP mocks, and giving its AI a whitelisted server: the mutation
   now fails there too, on the unexpected second fetch.
2. **The gate's comment asserted test protection that did not exist** ("breaks the reuse tests"),
   the round-18 failure class repeating verbatim one round later in a line the same commit wrote.
   It now names the two tests that actually fail, both verified by mutation.
3. **Two test comments misdescribed the fixture and the failure mechanism.** One claimed the
   start-member AI "whitelists a different server" when that fixture set no `McpServerIDs` at all,
   contradicting a correct comment 33 lines below it; the other claimed gomock would reject an
   unexpected call, impossible against a nil slice. Rather than weaken the prose, the fixture was
   strengthened to match it: the start-member AI now really does whitelist a different, unexpected
   server, so both mechanisms (unexpected fetch and empty tool map) are genuinely live.
4. Citations: this commit's comment expansion shifted `mcp_tool.go` again, leaving five anchors on
   a bare `return res` including one inside §5 item 1. Re-derived mechanically: `:157`→`:160`,
   `:157-174`→`:160-176`, `:245`→`:248`, `start.go:1180`→`:1184`. Two stale subtest totals fixed
   (the v25 note restored to its historically correct 15; §8 corrected to 18).

Lesson recorded: *a fixture's zero value is a behaviour, not a blank. An untyped enum silently
routes the case into the fail-closed default, so the branch you meant to test never runs.*

Lesson recorded: *never write "the tests cover this" in a comment without running the mutation that
proves it. Two consecutive rounds shipped that exact false claim, in the very commit that fixed the
previous one.*

Mechanism now in place, and now actually load-bearing:

    python3 docs/plans/scripts/check_doc_citations.py docs/plans/<doc>.md \
        --extra-root ~/gitvoipbin/monorepo-javascript \
        --extra-root ~/gitvoipbin/monorepo-monitoring

It resolves every citation (including `:N,M` and bare `:N` continuations), verifies symbol-anchored
ones against the landed line, validates every commit-like token, and warns on landings on blank
lines and lone braces. Run it before committing ANY revision. When you touch an anchor, give it a
symbol anchor rather than a bare line number: bare numbers are the form that drifted six times.

**Earlier note (v27), retained as the record of the fourth implementation round.**

**ROUND 18 (v27): the fix was correct but its own doc comment still published the contract it
had just withdrawn, and the reuse half of D28 had no behavioural coverage at all.**

1. **The resolver's doc comment described the withdrawn v22 contract.** It claimed the fallback
   was "unconditional" and fired "on ANY failure", and closed by telling callers that nil means
   they "must keep their existing behaviour rather than failing the call" — the exact fallback
   that `7dced7a90` had just deleted as provably dead. A comment is the next implementer's
   specification; this one instructed them to re-introduce the defect. Rewritten to state the
   real contract: the retry fires only where it can change the outcome, and nil means fail
   closed because the resolver already subsumes `resolveAI`. The same falsehood was in the test
   file header and is fixed there too.
2. **Two mutations proved the reuse path was untested.** Discarding the resolved AI (which
   re-creates D28 exactly: the tool map rebuilt from the start member's whitelist after a member
   switch), and skipping the refresh when the resolver returns nil, both left the entire package
   green. Cause: the only test touching that path constructed the handler without
   `mcpServerHandler`/`mcptoolHandler`, so `refreshMcpToolMap` returned at its nil guard before
   ever reading the AI, and the `.Times(2)` expectation pinned only an RPC count. **Six rounds of
   review, including three that ran mutations, had accepted this as coverage.** Fixed by wiring
   the MCP mocks behind a per-case `wireMcpHandlers` flag (the other sub-cases keep them nil and
   are unaffected), asserting that the persisted tool map contains the CURRENT member's server,
   and adding a degraded sub-case (team unfetchable) that pins "resolver returns nil, refresh
   still runs with the start-member AI". All three mutations now fail.
3. Non-blocking, also fixed: the dispatch refusal logged at Error with no payload while its
   neighbouring gates warn (now Warn); the subsumption comment overstated the retry; the
   duplicate-`teamHandler.Get` TODO still said two fetches when the reuse path now makes three.

Lesson recorded: *a mock left nil is not a neutral default, it is a silent early return. Before
trusting a test as coverage, mutate the behaviour it claims to pin and watch it fail.*

**Earlier note (v26), retained as the record of the third implementation round.**

**ROUND 17 (v26): the dispatch fallback I added in v25 was DEAD CODE that doubled an RPC, and
the retry guard was keyed on the wrong identifier. Both found by executing the code, not by
reading it.**

1. **The `resolveAI` fallback at the dispatch site could never recover anything.** v25 justified
   it as "never more failure-prone than before". But `resolveActiveAIForMcp` strictly subsumes
   `resolveAI`: for `AssistanceTypeAI` both issue the same single fetch; for a team the resolver
   tries the current member AND then the start member that `resolveAI` alone would have tried;
   for any other type both refuse. The reviewer enumerated **480 reachable shapes and found 0**
   in which the resolver returned nil and `resolveAI` then succeeded. So the fallback only ever
   doubled an `aiHandler.Get`/`teamHandler.Get` on the LLM's critical path before refusing
   anyway — including for plain AI aicalls, which D28 was never about. **Removed; nil now fails
   closed directly.** This is the round-15 duplicate-fetch defect reappearing one stack frame
   higher: the v25 fix moved the duplication outward instead of removing it.
2. **`teamHasMember` compared member ids, but the fetch it deduplicates is keyed on AIID.**
   `teamhandler` validates member-id uniqueness and a non-nil AIID (`validation.go:32-34` (member-id uniqueness) and `:56-57` (non-nil AIID));
   it does **not** require AIIDs to be distinct, so two members may legally share one AI. For
   such a roster the guard saw "different members", did not fire, and re-issued the
   byte-identical `aiHandler.Get` — exactly the defect the guard was added to remove. Replaced
   with `teamMemberAIID`, which compares the AIIDs actually about to be fetched.
3. **Two committed `.Times(2)` expectations pinned the duplication as intended behaviour.** Their
   comments described the control flow accurately, which is what made them dangerous: they read
   as justification while asserting a doubled RPC that provably could not change any outcome.
   **This is the mechanism by which the round-15 duplicate survived four rounds of review.** Both
   corrected to exactly-once. Lesson recorded: *a test that asserts a redundant call makes the
   redundancy permanent; before writing `.Times(2)`, prove the second call can change an
   outcome.*

Coverage is now 13 unit subtests plus 5 dispatch subtests, including the three shapes round 17
found unpinned (shared AIID, empty roster, absent start member). Every fix above was
mutation-verified: reverting the guard to member-id keying fails the shared-AIID case, and
re-adding the fallback fails two dispatch cases.

**Earlier note (v25), retained as the record of the second implementation round.**

**CLOSED IN CODE (v25). Round 16 reviewed the COMMITTED CODE rather than this prose and found
two BLOCKING defects in it, both of the same species as the four withdrawn prescriptions.**

1. **The first implementation made the dispatch gate FAIL OPEN.** It branched on
   `if AssistanceType != AssistanceTypeTeam` and treated everything else as an AI id. Before
   the change, `resolveAI`'s `default:` arm (`start.go:81`) refused an unrecognised type; after
   it, a non-nil AI came back, the `resolveAI` fallback was never entered, and the whitelist
   was evaluated against whatever row `AssistanceID` happened to hit. The reviewer proved
   dispatch with a probe. **A security gate was converted from fail-closed to fail-open by a
   commit whose stated purpose was tightening it.** Fixed with an explicit `switch` mirroring
   `resolveAI` and `resolveActiveAIIDFromAIcall`, both of which already had the `default:` arm
   this function was missing. The empty `AssistanceType` occurs in the package's own fixtures,
   so this was not hypothetical.
2. **The retry guard tested the wrong predicate.** It skipped the start-member retry only when
   `CurrentMemberID == StartMemberID`, but `resolveTeamMemberAI`'s INNER fallback loop
   (`start.go:42-52`) had already tried the start member whenever the current member was absent
   from the roster. When that inner fetch was what failed, the guard did not fire and the outer
   retry re-issued a byte-identical `aiHandler.Get` — a doubled RPC and doubled warning on the
   LLM's critical path, under exactly the timeout conditions where retry amplification hurts.
   The guard's own comment stated the intent correctly and the code did not implement it. Fixed
   by gating on roster membership (`teamHasMember`).

**Both were found only by executing the code.** Neither was visible in four rounds of prose
review, and the second was introduced by the very edit that fixed the case v22 had missed.
That is the eleventh and twelfth instance of this defect's failure pattern, and the first two
to be caught by a compiler and a test runner rather than by an argument.

Coverage after the fix: 15 subtests (10 unit + 5 dispatch), each verified to FAIL against the
un-fixed code by mutation. The reorder of `resolveTeamMemberForSend` above the refresh was
**deliberately NOT performed** — round 15 established it is hygiene, not protection, and the
protection is now unconditional inside the resolver.

**Earlier note (v24), retained as the record of the first implementation round.** 대표님 authorised implementing D28 ahead of the remaining
analysis, because three successive prescriptions for it were refuted by code the document had
only reasoned about. The implementation is committed in this branch and the compiler and test
suite adjudicated it. Two things the prose had NOT anticipated surfaced immediately:

1. **v23 said "use it at both MCP sites" without qualifying the non-team case.** Implemented
   literally, it added an `aiHandler.Get` for plain AI aicalls, where the bound `a` already IS
   the governing AI — three existing tests failed on the unexpected RPC (two reuse cases plus the service-level conversation test). Now gated on
   `AssistanceType == AssistanceTypeTeam`.
2. **The "start member is already the current member" case needed an explicit skip**, or the
   fallback retries the identical failing fetch. Not mentioned anywhere in v19-v23.

`Test_aicallHandler_resolveActiveAIForMcp` pins all thirteen cases (the happy path resolving the CURRENT member; each
degraded mode; the already-start-member skip; the no-team RPC for plain AI; the unknown and
empty assistance types; the composed degraded mode; a nil CurrentMemberID; a shared AIID; an
empty roster; an absent start member). `go build ./...`, `go vet`, and `go test ./...` are clean across
bin-ai-manager. The prescription below is retained as the record of what was implemented.

**Resolution (v23): one mechanism — a local fallback-bearing resolution at the two MCP
sites, and no change to the shared helper.**

(a) **Add one narrow method, `resolveActiveAIForMcp(ctx, c *aicall.AIcall) *ai.AI`**, whose
    single job is: return the current member's AI; on ANY failure (member absent, AI fetch
    error, team fetch error) fall back to the START member's AI; if even that fails, return
    `nil`. It is the only place the three error modes are handled, and it does not touch
    `resolveActiveAIIDFromAIcall` or its six callers or the `messagehandler` twin.
(b) **Reuse branch (`start.go:375`)**: pass its result to `refreshMcpToolMap` instead of the
    start-member `a`. On `nil`, keep today's behaviour — pass the start-member `a` already in
    hand, so the map is still refreshed. **Never skip the refresh**; a skipped refresh is how
    v21 preserved the stale map.
(c) **Dispatch (`mcp_tool.go:151`)**: replace `resolveAI` with it. On `nil`, **preserve
    today's outcome** by falling through to the existing start-member `resolveAI` result
    rather than `fillFailed`. Dispatch must not become more failure-prone than before our fix.
(d) **The reorder is OPTIONAL, not required.** Moving the team repair above the refresh is
    harmless and desirable (it persists `CurrentMemberID` via `send.go:176-183` and keeps
    `start.go:396` consistent, and nothing between `:364` and `:397` reads `AIEngineModel`),
    but it is **not** what protects the degraded case. Recorded as hygiene so no implementer
    believes ordering is the safety mechanism.
(e) **One team RPC.** `resolveActiveAIForMcp` performs the single `teamHandler.Get` the two
    MCP sites need; the pre-existing duplicate flagged by the TODO at `start.go:392-395` is
    **out of scope** for PR A and stays as-is.

**Why the start-member fallback is correct at dispatch, not a cross-member leak.** When
`CurrentMemberID` cannot be resolved, the platform's own defined state for that session IS
the start member — `send.go:176-183` persists exactly that repair and the engine model
already follows it. Customer-ownership (D26) and the deleted/status gates still run, so the
executed server belongs to the same customer. The choice is between the start member's tools
and no tools at all, and today's behaviour is already the former.

**§7 additions (v23): assert BOTH degraded modes, at BOTH sites** — member-absent AND
current-member-AI-fetch-failure. v22 asserted only the first, which is why its regression
would have shipped green.

**Residual gaps (unchanged from v22, restated).** Post-fix, dispatch resolves the active
member and `mcpServerIDIsWhitelisted` (call site `mcp_tool.go:160`, func `:248`) **refuses** a
server the active member does not whitelist, so the residual is **under-availability**, not
exposure: the active member's own newly whitelisted tools are missing until the next Start.
Two paths never refresh the map — the send path (`refreshMcpToolMap` has exactly one caller,
`start.go:375`) and a reused contact_case AIcall (`start.go:603` →
`refreshInsightSessionIfIdle`, which returns "kept" for a team). Every prescribed gate
re-reads the row, so a deleted or foreign server stays inert whichever member's map is
loaded. **PR B precondition: refresh the map on member switch, covering both paths.**

### 3.6 D13: `mcp_server_ids` is not AI-type validated. VOIDED in v14, retained for history.

**STATUS: VOIDED, not fixed.** 대표님 확정 (v14): "인사이트도 mcp 사용가능하도록 해."
Insight AIs MAY use MCP servers, so the absence of an AI-type check is **intended
behavior, not a defect**. Nothing in this section is a PR A work item. The findings are
kept only because §3.12b/§3.12e and the §8 retrospective reference them.


```go
// pkg/aihandler/mcpserver_validation.go:35 — no AI type parameter
func (h *aiHandler) ValidateMcpServerIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) error

// models/ai/tool_validation.go:72 — takes the type
func ValidateToolNames(t Type, toolNames []tool.ToolName) error
```

`ValidateMcpServerIDs` checks only existence and customer ownership. Both call
sites (`pkg/listenhandler/v1_ais.go:118`, `:273`) pass only `CustomerID`. Meanwhile
`tool_names` IS type-validated against `AllowedToolNames`
(`models/ai/tool_validation.go:77`), and the Insight catalog
(`AllInsightToolNames`, `models/tool/main.go:90`) is deliberately narrower than
Normal's.

So an Insight AI may whitelist arbitrary MCP servers while being denied most
built-in tools. Whether that is intended is a **policy decision** (§5 item 5). It is a
write-time gate, so if the answer is "not unconditionally," it belongs in PR A next
to the other gates. If the answer is "MCP is type-agnostic by design," PR A should
record that explicitly so the asymmetry stops looking like an oversight.

### 3.7 D14: undocumented webhook events + an omitempty/docs contradiction. CONFIRMED.

- `bin-ai-manager/models/mcpserver/event.go:5-7` declares
  `mcp_server_created`, `mcp_server_updated`, `mcp_server_deleted`. Grep across
  `bin-api-manager/docsdev/source/` and `bin-openapi-manager/openapi/` returns
  **zero** occurrences of all three. They are published
  (`mcpserverhandler/handler.go:242` for deleted) but undocumented.
  Silver lining for §5 item 2: no documented contract breaks when `has_secret` flips.
- `bin-ai-manager/models/ai/webhook.go:47`
  `McpServerIDs []uuid.UUID \`json:"mcp_server_ids,omitempty"\`` — so an
  explicitly cleared whitelist is **absent** from the AI webhook payload, while
  `ai_struct_ai.rst:69` says it "Defaults to `[]`". Same defect class as
  `oauth_vendor` in §3.10.

### 3.7a D15: gating `McpServerUpdate` silently succeeds. CONFIRMED. BLOCKER for §5 item 1.

```go
// pkg/dbhandler/mcpserver.go:154-156 — result discarded
if _, err := h.db.ExecContext(ctx, query, args...); err != nil {
	return fmt.Errorf("McpServerUpdate: could not execute. err: %v", err)
}
return nil
```

Adding `tm_delete IS NULL` to the `Where` makes the statement match zero rows and
return `nil`. `mcpserverHandler.Update` then continues:

```go
// pkg/mcpserverhandler/handler.go:206-217
if err := h.db.McpServerUpdate(ctx, id, fields); err != nil { ... }
res, err := h.db.McpServerGet(ctx, id)      // does NOT filter tm_delete
...
h.notifyHandler.PublishWebhookEvent(ctx, res.CustomerID, mcpserver.EventTypeUpdated, res)
return res, nil
```

So a PUT against a deleted server would return **200 with unchanged fields and a
spurious `EventTypeUpdated` webhook**. Today's visible resurrection becomes a
silent success, which is arguably worse: the customer believes the update applied.

The same mechanic defeats the OAuth `Complete` gate, since `complete.go:126` writes
through `McpServerUpdate`.

**Therefore §7's promised tests ("rejected by `McpServerUpdate`", "rejected by
OAuth `Complete`") are unachievable by gating alone.** PR A must additionally make
`McpServerUpdate` inspect `RowsAffected` and return `ErrNotFound` on zero. That is a
behavior change affecting **every** existing caller, including
`mcpoauthhandler/access_token.go:101-108`'s token rotation — see §6's new risk row.

This is the same mechanic §5 item 2 already established for credential zeroing; v5
failed to carry it into §5 item 1.

### 3.7b D16: delete is not idempotent. CONFIRMED.

`McpServerDelete` (`pkg/dbhandler/mcpserver.go:163-182`) has no `tm_delete`
predicate and also ignores `RowsAffected`, so a second DELETE **re-stamps
`tm_delete` with a fresh timestamp** and `handler.Delete:242` re-publishes
`EventTypeDeleted`.

This is unavoidable for PR A because §5 item 2 mandates zeroing credentials **in this
exact statement**, so PR A is rewriting it regardless. The decision must preserve a 200 on repeat
DELETE even if the timestamp stops moving — **because DELETE is idempotent by contract
and GET keeps returning the soft-deleted row**, not because of the test suite. (v15
cited the validator fixture as the reason; `cleanup_report.py:86` tolerates 404, so that
reason was false. See §5 item 1.) Resolved: add the predicate, swallow the zero-row error
at `handler.go:226` so the response stays 200, stop re-publishing the webhook.

### 3.7c D17: `auth_type: "oauth"` is settable directly, contradicting the docs. CONFIRMED.

`ai_struct_mcpserver.rst:35` states `oauth` "can never be set directly via
POST/PUT." Nothing enforces that:
- `AuthType.IsValid()` (`models/mcpserver/main.go:23-30`) **accepts**
  `AuthTypeOAuth`.
- `mcpserverhandler.Create:48-50` and `Update:154-156` check only `IsValid()`.
- `bin-api-manager/server/mcpservers.go:91-94` (POST) and `:185-189` (PUT) cast the
  raw string straight through.
- **There is no OpenAPI request-validator middleware.** `grep -rn
  'OapiRequestValidator\|openapi3filter\|ValidateRequest' bin-api-manager`
  (excluding `vendor/`) → **zero hits**. The chain at
  `cmd/api-manager/main.go:366-378` is RequestID / RateLimit / Authenticate /
  EnforceAccountStatus / DirectResourceScope only. So the generated enum
  (`gens/openapi_server/gen.go:3550-3566`) and the spec enums
  (`paths/mcpservers/main.yaml:57`, `id.yaml:74`) are **decorative**.

Result: `POST /mcpservers {"auth_type":"oauth","secret":"x"}` succeeds, stores a
customer-supplied secret on an `oauth` row, and reports `has_secret: true`, while
`buildAuthHeader`'s oauth branch (`mcptoolhandler/client.go:90-101`) ignores
`SecretCiphertext` and fails on an empty `AccessTokenCiphertext`
(`access_token.go:38`). A silently broken server the customer cannot diagnose.

Same defect class as D2 (a missing write-time gate), on the same docs sentence D5
edits. **Decision needed (§5 item 9): add the gate, or correct the sentence.**

### 3.7d D18: whitelist validation runs after the AI write commits. CONFIRMED.

```go
// pkg/listenhandler/v1_ais.go:92-128 (POST) — Create FIRST
tmp, err := h.aiHandler.Create(ctx, req.CustomerID, ... req.ToolNames, ...)
if err != nil { ... }

if req.McpServerIDs != nil {
	if err := h.aiHandler.ValidateMcpServerIDs(ctx, tmp.CustomerID, *req.McpServerIDs); err != nil {
		return errorResponse(err), nil        // AI already exists
	}
	tmp, err = h.aiHandler.UpdateMcpServerIDs(ctx, tmp.ID, *req.McpServerIDs)
```

`mcp_server_ids` is not even a `Create` parameter; it is applied by a **second**
call. So a rejected whitelist returns 400 while the AI **has already been
created** (POST) or mutated (PUT `:247-283`), leaving an orphan. Note `tool_names`
IS a `Create` parameter and IS validated inside the write.

**v14:** the earlier argument here — that D13's type gate rewrites this signature
anyway, making the reordering nearly free — is withdrawn with D13 (§5 item 5). D18 now
stands alone: the orphan path is still real, but PR A must justify the reordering on
its own merits. §5 item 11's diff logic touches the same two blocks, which is the remaining
reason to do it here.

### 3.7e D19: the RST documents a timestamp sentinel this resource does not use. CONFIRMED.

`ai_struct_mcpserver.rst:53` asserts that `tm_delete` = `9999-01-01
00:00:00.000000` means "not deleted," and the example block at `:71-72` prints that
value for **both** `tm_update` and `tm_delete`. The API cannot produce that output:
- `models/mcpserver/main.go:98` is `TMDelete *time.Time` (pointer, so `null` on the
  wire).
- `ai_mcp_servers` was created 2026-09-11
  (`9b0ad37e0360_ai_mcp_servers_create_table.py:22-42`), **after**
  `071504ef41d0_timestamp_sentinel_to_null.py` retired the sentinel, and the table
  is absent from that migration's `TABLE_COLUMNS` (`:26-71`).
- api-validator confirms the real contract: `test_mcpservers_lifecycle.py:266,274`
  assert `tm_delete is not None` rather than the sentinel comparison older
  resources use (`test_ai_lifecycle.py:66`).
- Correct precedent in the same docs tree: `talk_struct_talk.rst:37`,
  `customer_struct_customer.rst:67`.

Also in the same file: `:37` labels `oauth_vendor` "(enum string)", but the model
field is a plain `string` (`models/mcpserver/main.go:82`) populated from the
runtime-configurable vendor catalog (`internal/config/main.go:151-154`) with no
`IsValid`. And neither the Status table (`:79-86`) nor the delete note states what
DELETE does to tool access — which becomes the headline customer-visible change once
PR A lands, so it belongs in this file.

All of the above sit in the same note block and example that D5 already edits, so
PR A fixes them in the same pass.

### 3.7f D20: the empty-PUT path bypasses every DB-level gate. CONFIRMED. BLOCKER for §5 item 1.

`mcpserverhandler.Update` builds a field map from nil-able pointers and then:

```go
// pkg/mcpserverhandler/handler.go:194-204
if len(fields) == 0 {
	// A PUT with every field omitted … is a client no-op, not a server error.
	// … Reuses Get's existing ErrNotFound -> cerrors.NotFound mapping …
	return h.Get(ctx, id)
}

if err := h.db.McpServerUpdate(ctx, id, fields); err != nil { ... }
```

A `PUT /mcpservers/{id}` with an empty or all-null body yields `len(fields) == 0` —
every field is a nil pointer end to end
(`bin-api-manager/server/mcpservers.go:179-191` →
`requesthandler/ai_mcpservers.go:122-130` → `listenhandler/v1_mcpservers.go:187-197`)
— so it **never calls `db.McpServerUpdate`** and returns 200 with the deleted row's
full body. An existing test pins this: `pkg/mcpserverhandler/handler_partial_update_test.go:115-148`
(`Test_Update_AllFieldsOmitted_IsANoOp`) asserts `db.McpServerUpdate` is not called.

**Consequence: §5 item 1's mandated fix does not close PUT-against-a-deleted-server.**
The `tm_delete IS NULL` predicate plus the D15 `RowsAffected` check both live below
a branch this request never enters, and §5 item 1's own forbid-clause rules out the only
other interception point (`dbhandler.McpServerGet`). After PR A lands exactly as
v6 wrote it, `PUT {}` on a deleted server still answers 200 with the row.

Fix: an explicit existence gate inside `mcpserverhandler.Update` **ahead of** the
`len(fields) == 0` branch — and, per §3.7g, ahead of `ValidateURL` too.

This is the same failure shape as D15 one layer higher: **a prescribed fix that does
not reach the path it was meant to close.**

### 3.7g D20 corollary: validation precedes existence, so the error contract is inconsistent.

`ValidateURL` runs at the very top of `Update` (`handler.go:146-150`), before any
field assembly and far before the DB write:

```go
// pkg/mcpserverhandler/handler.go:146-150
if url != nil {
	if err := ValidateURL(*url); err != nil {
		return nil, cerrors.InvalidArgument(..., "INVALID_MCP_SERVER_URL", ...)
	}
}
```

`status.IsValid()` (`:151-153`) and `authType.IsValid()` (`:154-156`) follow, also
before existence is known. So on a **deleted** server: a PUT with a malformed URL
returns **400 `INVALID_MCP_SERVER_URL`** while the same PUT with a valid URL would
return 404 once gated. That is both inconsistent and a weak existence oracle (a
caller can probe which ids exist by varying only the URL's validity).

The gate must precede all three validations.

**Error code choice (must be decided, §5 item 10).** Both candidates terminate at 404,
so there is no 500 anywhere — verified plumbing:
- bare `dbhandler.ErrNotFound` → `listenhandler/main.go:194-195` →
  `requesthandler.ErrNotFound` → `server/error_translate.go:83-84` → **404
  `RESOURCE_NOT_FOUND`**
- typed `cerrors.NotFound` → `listenhandler/main.go:184-189` → typed passthrough
  (`error_translate.go:52-56`) → **404 `MCP_SERVER_NOT_FOUND`**

`handler.Get:97-101` already emits `MCP_SERVER_NOT_FOUND` for a nonexistent id, so
using the same code for "deleted" keeps the two indistinguishable — which is the
correct privacy posture, but it must be a deliberate choice, not an accident of
which error type the implementer happens to return.

### 3.7h What CANNOT be gated: `mcpserverhandler.Get` has four consumers pulling opposite ways.

§5 item 1 forbids gating `dbhandler.McpServerGet` and explains why. The **handler-level**
`Get` (`handler.go:93-107`) is a different function and equally ungateable, for a
different reason — its four non-test consumers have contradictory requirements:

| Consumer | Requirement |
|---|---|
| `pkg/listenhandler/v1_mcpservers.go:142` | customer GET — **must keep returning 200** for a deleted row (§4 non-goals, validator-tested) |
| `pkg/aicallhandler/mcp_tool.go:73` (resolution) | must **fail closed** |
| `pkg/aicallhandler/mcp_tool.go:160` (dispatch) | must **fail closed** |
| `pkg/mcpserverhandler/handler.go:203` | D20's empty-PUT short-circuit — must fail closed |

So the defense-in-depth gates §5 item 1 lists must be written **in `mcp_tool.go`
itself**, not in the shared `Get`. The design doc has to say this outright: an
implementer who "helpfully" gates `Get` breaks the GET-after-DELETE 200 contract,
which is §6's top risk row one layer up.

### 3.8 D5: published docs give a 404 path. CONFIRMED.

`bin-api-manager/docsdev/source/ai_struct_mcpserver.rst`:
- `:29` `POST /mcp_servers`, `GET /mcp_servers` — wrong
- `:45` `POST /mcp_servers`, `PUT /mcp_servers/{id}` — wrong
- `:49` `PUT /mcp_servers/{id}` — wrong
- `:35`, `:100` correctly `/mcpservers/oauth/...` — internally inconsistent

Those three are the only `mcp_servers` hits in the whole RST tree. Authoritative:
`bin-openapi-manager/openapi/openapi.yaml:8569-8578`, five routes, all
`/mcpservers`.

Two broken references in the same file:
- `:29` `:ref:` `mcp_server_ids <ai-struct-ai-tool_names>` points at the
  `tool_names` anchor (`ai_struct_ai.rst:340`); no `mcp_server_ids` anchor exists.
- `:100` self-references `oauth_vendor <mcpserver-struct-mcpserver-mcpserver>`,
  the containing section rather than a field anchor.

### 3.9 D6: `mcp_tools_list_cache_ttl_seconds` is a dead flag. CONFIRMED.

Every reference, repo-wide (excluding `vendor/`):

```
internal/config/main.go:84    struct field declaration
internal/config/main.go:149   f.Int(..., 60, "Redis cache TTL (seconds) for an MCP server's tools/list result")
internal/config/main.go:199   env mapping MCP_TOOLS_LIST_CACHE_TTL_SECONDS
internal/config/main.go:276   viper.GetInt(...)
```

Four hits, all inside config. **Zero consumers.** Contrast the sibling that works:
`McpToolCallTimeoutSeconds` at `config/main.go:85`, `:277`, and
**`cmd/ai-manager/main.go:156`** where `NewMcpToolHandler(db,
cfg.McpSecretEncryptionKeys, cfg.McpToolCallTimeoutSeconds, mcpOAuthHandler)`
consumes it. `NewMcpToolHandler` (`pkg/mcptoolhandler/main.go:74`) has no
cache-TTL parameter. `grep -rni 'mcp' pkg/cachehandler/` = zero matches.
`pkg/mcptoolhandler/client.go:190-207` `ListTools` goes straight to DB + HTTP.

The design doc asserts the cache exists in three places:
`docs/plans/2026-09-11-mcp-tool-integration-design.md:466-467`, `:705` (config
table), and most damagingly **`:750-754`**, which justifies a *different* decision
("a second cache layer over the `McpServer` row itself is not needed") by citing
this nonexistent cache as existing mitigation.

**Decision: remove the flag and correct the design doc. Do not implement a cache.**
(i) No measured signal — one call per session start, no threshold alarm, so a cache
layer now is speculative infrastructure. (ii) A cache actively conflicts with
revocation — it would serve a deleted server's tools for up to the TTL, requiring
invalidation design on top of the gates.

### 3.10 Wire contract (for doc accuracy)

Source of truth: `bin-ai-manager/models/mcpserver/webhook.go:14-45`, NOT the
internal struct. 13 fields: `id`, `customer_id` (embedded
`commonidentity.Identity`), `name`, `detail`, `url`, `status`, `auth_type`,
`api_key_header`, `oauth_vendor`, `has_secret`, `tm_create`, `tm_update`,
`tm_delete`. Never document `SecretCiphertext`, `SecretNonce`, `KeyVersion`.

`auth_type` deliberately omits `,omitempty` (`:32`) because `AuthTypeNone` is the
empty string and is meaningful, so it is ALWAYS on the wire. `oauth_vendor` DOES
carry `,omitempty` (`:38`), so it is ABSENT, not empty, when unset —
`ai_struct_mcpserver.rst:37` says "empty otherwise" (imprecise) and the example at
`:60-73` omits the field entirely.

Enums (`models/mcpserver/main.go:13-45`): `auth_type` = `""` | `bearer` |
`api_key` | `oauth`; `status` = `active` | `disabled`. OAuth vendors wired:
GitHub, Linear only (`internal/config/main.go:151-154`).

### 3.11 What the docs ALREADY have (v1's false claim, retained as a guard)

`grep -ric mcp *.rst` → `ai_overview.rst:2`, `ai.rst:1`, `ai_struct_ai.rst:2`,
`ai_struct_mcpserver.rst:28`. Four files. `ai_struct_mcpserver.rst` is 101 lines on
`origin/main`, in the `ai.rst:15` toctree, reachable from `index.rst:83`, built to
`docsdev/build/html/ai_struct_mcpserver.html`. The CLAUDE.md `*_struct_*.rst`
obligation IS satisfied. Genuinely missing: narrative coverage (no
`mcpserver_overview.rst`, no `mcpserver_tutorial.rst`; `rag` has both).

### 3.12 D8: square-admin gap is frontend-only. CONFIRMED.

Backend accepts `mcp_server_ids` on BOTH create and update, four layers:
`openapi/paths/ais/main.yaml:93-102`; `gens/models/gen.go`
`PostAisJSONBody.McpServerIds` `:10609` and `PutAisIdJSONBody` `:10667`;
`bin-api-manager/server/ais.go:74-80` (POST) and `:293-299` (PUT);
`bin-ai-manager/pkg/listenhandler/v1_ais.go:117-128` (POST) and `:272-283` (PUT).
**No backend or OpenAPI change needed for D8.**

Frontend hits exist ONLY in `square-admin/src/views/ais/ais_detail.js` (`:103`
state, `:242` hydrate, `:374` list fetch, `:382` toggle, `:421` PUT body,
`:552-553` dirty check, `:1239-1276` UI) plus its test. Zero hits
(case-insensitive) in: `views/ais/ais_create.js` (438 lines; body `:114`, POST
`:154`), `views/ais/AIEngineFields.js` (659 lines; the SHARED component both pages
use), `views/teamgraph/sidebar.js` (1769 lines; AI create body `:514-534`, AI edit
body `:753-782`), `src/types/api.ts` (`tool_names?` at `:503`).

`square-admin/CLAUDE.md:161-181` Field Sync Points names three required sites; MCP
satisfies one. `src/types/api.ts` is a de-facto FOURTH site (the checklist's step 1
says "Add type to src/types/api.ts") but is absent from the table — the table needs
a row.

**Deleted servers are never rendered.** `bin-api-manager/pkg/servicehandler/mcpserver.go:67-70`
hardcodes `filters{"deleted":"false"}` and `ais_detail.js:374` fetches through that
path. **RETRACTED in v8:** v5-v7 called the residual UI behavior "a PR C display
decision, not a data problem." That was true of today's code and is **false once PR
A's whitelist gate lands** — see §3.12a (D21).

### 3.12a D21: PR A's whitelist gate freezes any AI holding a deleted id. CONFIRMED. BLOCKER.

Three prescriptions in this document, each defensible alone, combine into a
customer-facing dead end. Traced forward through square-admin:

1. **The stale id is still there.** §5 item 6 decides NOT to prune
   `ai.mcp_server_ids` on delete. `ais_detail.js:242` hydrates `mcpServerIds` from
   the GET response, which still contains the deleted id.
2. **The user cannot see or remove it.** The picker at `ais_detail.js:1252-1271`
   iterates `mcpServersList.map(...)` and derives `selected` from
   `mcpServerIds.includes(server.id)`. Since the list comes through the
   `deleted:"false"` filter (`servicehandler/mcpserver.go:67-70`), the deleted server
   is **absent from the iteration entirely** — no row, no checkbox, no
   `toggleMcpServer` affordance. The id is invisible and unremovable from the UI.
3. **Every save re-submits it.** `ais_detail.js:406-422` builds the PUT body with
   `mcp_server_ids: mcpServerIds` **unconditionally** (verified: the field sits in
   the literal alongside `tool_names`, with no conditional wrapper).
4. **PR A then rejects it.** A non-nil array reaches `server/ais.go:294` →
   `v1_ais.go:272-276` always calls the now-gated `ValidateMcpServerIDs` →
   `INVALID_MCP_SERVER_ID` → **400 on every save of that AI, forever**, including
   edits to completely unrelated fields (name, prompt, engine, TTS).
5. **The user gets no hint why.** `ais_detail.js:551-553` computes dirtiness by
   comparing old vs new whitelist, so the dead id never marks the whitelist section
   dirty. The failure surfaces while editing something else, as the generic
   "Could not update the AI configuration" at `:445`.
6. **D18 compounds it.** `aiHandler.Update` at `v1_ais.go:247` has **already
   committed** before validation runs, so the unrelated edits persist while the user
   is told the save failed. If §5 item 5's recommendation (move validation ahead of the
   write) is adopted, nothing persists and the AI is simply frozen.

The same trap propagates to `teamgraph/sidebar.js:753-782` the moment PR C adds the
field there, and `sidebar.js` has no `mcpservers` fetch of its own (§4).

**This is the third instance of this document's recurring failure mode** (after D15
and D20): a fix traced only to its own layer, never forward to the surface that
consumes it. Here the two layers are in **different repositories**, which is why
five rounds of review missed it.

**Resolution is a decision, not an implementation detail — RESOLVED in §5 item 11:**
skip the DELETED-row rejection for ids already in the stored whitelist (existence and ownership still run — D25), reject ids newly added. The diff is free
because `tmp.McpServerIDs` at `v1_ais.go:272` is still the unmodified stored list
(`aiHandler.Update` at `:247` does not take `McpServerIDs`). With that, none of the
six steps above fires: step 4 no longer rejects, so steps 5 and 6 never trigger.

square-admin reads `has_secret` only from GET
responses (`mcpservers_list.js:86`, `mcpservers_detail.js:266`) and does not consume
mcpserver webhooks, and it already handles `auth_type: 'oauth'`
(`mcpservers_detail.js:38`), so §5 item 2's `has_secret` flip breaks nothing.

### 3.12b D22: VOIDED in v14 — §5 item 5's Insight gate enforces nothing where it matters and freezes AIs elsewhere

**STATUS: VOIDED, not fixed.** 대표님 reversed the underlying policy in v14 ("인사이트도
mcp 사용가능하도록 해") — Insight AIs may use MCP servers, so no AI-type gate is added
and this defect cannot occur. The analysis below is retained verbatim because it is the
fourth recorded instance of this document's recurring failure mode and its reasoning is
what led to the reversal: the gate could not be placed anywhere that both enforced the
policy and left customers able to save. **Nothing here is PR A or PR C scope.**

**This is the FOURTH instance of this document's recurring failure mode** (after D15,
D20, D21), and it was NOT the instance the author self-flagged. §5 item 5 prescribes a
write-time AI-type gate. Traced forward in both directions, the write-time layer is
the one layer that cannot deliver the policy.

**(a) It does not enforce the policy for existing rows.** The consume path has NO type
check. `grep -n 'Type' pkg/aicallhandler/mcp_tool.go` returns exactly one line,
`:144` (`resolveAI`), and never `a.Type`. Resolution iterates the whitelist blind to
type:

```go
// pkg/aicallhandler/mcp_tool.go:72
for _, serverID := range a.McpServerIDs {
	server, err := h.mcpServerHandler.Get(ctx, serverID)
```

Insight sessions reach this through `insight_session.go:104` → `:143`. Combined with
**§5 item 6's decision never to prune**, any Insight AI that already stores
`mcp_server_ids` — or any Normal AI later flipped to Insight — keeps resolving and
`ListTools`-calling those servers forever. §7's promised test ("`TypeInsight` cannot
whitelist an MCP server") would pass while the policy is unenforced in production.

**(b) It reintroduces D21 verbatim, and §5 item 11's skip does NOT cover it.** §5 item 11 exempts
already-stored ids from the **deleted-server** predicate only. The type predicate is
different, so the exemption does not apply. In square-admin:

```javascript
// ais_detail.js:738-743 — switching to insight clears TOOLS but not MCP servers
onValueChange={(value) => {
  setAiType(value);
  if (value === 'insight') {
    setSelectedTools([]);
  }
}}
```

`mcpServerIds` is never cleared, `:421` puts it unconditionally, and the MCP Servers
card at `:1238` is rendered with **no `aiType` condition** — unlike
`AIEngineFields.js:301`/`:320`, which do gate tool sets on `aiType === 'insight'`. So a
customer flipping an existing AI to Insight sends `type: "insight"` plus a non-empty
`mcp_server_ids` and receives a hard 400 behind the generic "Could not update the AI
configuration" (`:445`). The dead end is identical to D21 and lands in the repo PR A
does not touch.

Resolution: **none needed — VOIDED in v14.** 대표님 reversed the policy, so no gate is
built and this defect cannot occur.

### 3.12c §5 item 9's transition gate needs a read that does not exist, and its frontend safety is accidental

**The pre-image is missing.** `mcpserverhandler.Update` (`handler.go:138-192`) builds
its field map purely from incoming pointers; the only `McpServerGet` is the
**post-write** read-back at `:210`. So "validate the transition, not the value"
(§5 item 9) requires a pre-write row read that does not exist today. **It is the same read
D20's existence gate needs** (§5 item 1) and the same one §3.7g's precedence fix needs.
The design doc must state that **one** pre-write `McpServerGet` serves all three —
otherwise three implementers add three reads, or one adds none and silently falls back
to a value gate, reproducing the freeze §5 item 9 explicitly warns about.

**The frontend is compatible, but by accident.** §3.12a's earlier claim that
square-admin "already handles `auth_type: 'oauth'`" is true and materially incomplete.
`mcpservers_detail.js:156` re-submits `auth_type: buildAuthTypeWireValue(authType)`
**unconditionally** on every save, and `:38`/`:91` hydrate `'oauth'`, so an OAuth row's
PUT always carries `auth_type: "oauth"`. A value gate would 400 every edit of a
Connected server. The transition gate survives only because `AUTH_TYPE_OPTIONS`
(`:31-35`) omits `oauth`, so the form cannot move a row INTO oauth; and `:359-366`
renders oauth as a read-only badge, so it cannot move a row OUT of oauth either. **Both
are load-bearing UI facts that PR A's tests do not pin.** If either changes, the gate
breaks. Record them, and assert the transition semantics in a backend test rather than
relying on the form's option list.

### 3.12d §5 item 11 implementation constraints (non-blocking, but must be written down)

- **The exemption is a predicate-scoped condition INSIDE the validator, not a filtered
  input list (D25, §3.5a).** `ValidateMcpServerIDs` receives the full incoming list plus
  `storedIDs`; existence (`mcpserver_validation.go:37-48`) and ownership (`:50-56`) run
  on every id, and only the new `tm_delete` rejection consults `storedIDs`. Passing a
  pre-filtered subset — what v12-v15 mandated — would exempt ownership too.
- **The diff is SET membership, not sequence comparison.** `mcpServerIds` is
  order-dependent in the frontend (`ais_detail.js:382-386` appends on toggle; `:553`
  sorts only for the dirty check, never for submission). A `slices.Equal`-style
  comparison would misclassify a reordered list as "newly added." Build a
  `map[uuid.UUID]struct{}` of the stored list and test membership. Duplicates in the
  incoming list and the explicit empty-slice clear (pinned by
  `ais_detail.test.js:690-700`) both fall out correctly from set membership; no extra
  decision is needed.
- **POST has no diff code path at all.** `mcp_server_ids` is not a `Create` parameter
  (`chatbot.go:20-39`), so the POST rule is simply "reject all deleted ids." An
  implementer writing one shared helper must not have it read a stored list that does
  not exist.
- **Both internal oauth writes are exempt, not one.** §5 item 9 cites `complete.go:136-158`
  (new row). The **reconnect branch also writes it**: `complete.go:112-133` sets
  `FieldAuthType: AuthTypeOAuth` at `:117` via `McpServerUpdate` at `:126`. Neither
  passes through `mcpserverhandler`, so the decision is unchanged, but §7 must assert
  both survive the gate.

### 3.12e D23: VOIDED in v14 — the Insight gate is unreachable by the request that creates the violating row

**STATUS: VOIDED, not fixed.** Same reversal as D22: there is no AI-type gate, so there
is nothing to bypass and nothing to freeze. Its three open decisions (flip semantics,
empty-list exemption, on-load cleanup) are **withdrawn**. Retained verbatim as the fifth
recorded instance of the recurring failure mode — it was created by the fix for the
fourth, which is the strongest evidence in this document that the gate was the wrong
shape. **Nothing here is PR A or PR C scope.**

**FIFTH instance of this document's recurring failure mode** (after D15, D20, D21,
D22), found in the v12 material that was written to close the fourth. v12 was verified
for *reachability of the gate points*; it was not verified for *reachability of the
gate by the offending request*.

**(a) A type flip that OMITS the whitelist bypasses the gate entirely.** Both call
sites of `ValidateMcpServerIDs` sit inside `if req.McpServerIDs != nil`
(`v1_ais.go:117` and `:272`) — §5 item 11's own table codifies "field omitted → no
validation". So:

```
PUT /ais/{id}  {"type": "insight"}     // mcp_server_ids omitted
```

flips the type (`chatbot.go:145-149` resolves and writes it), never enters the
`req.McpServerIDs != nil` block, and leaves the stored non-empty whitelist untouched
(§5 item 6 never prunes; `aiHandler.Update`'s parameter list excludes `McpServerIDs`).
**Result: an Insight AI holding a non-empty whitelist, created AFTER the deploy,
through the public API, with nothing rejected.** §5 item 5(iii)'s production count of zero
does not cover this — it is not a migration artifact, it is a new row.

**(b) That row is then permanently frozen, and §5 item 5(ii)'s cleanup does not reach it.**
`ais_detail.js:197` hydrates `aiType = 'insight'` and `:242` hydrates
`mcpServerIds = [X]` on load. §5 item 5(ii)'s clear lives in `onValueChange`
(`:738-743`), which fires only when the user **changes** the select — **not on load of
an already-Insight AI.** Meanwhile §5 item 5(ii)'s other half hides the MCP card for Insight,
so there is no picker and no `toggleMcpServer` to remove `[X]`. `:421` still submits
`mcp_server_ids` unconditionally (pinned by `__tests__/ais_detail.test.js:690-726`), so
every subsequent save is a non-empty whitelist on an Insight AI → 400 → generic "Could
not update the AI configuration" (`:445`). Save is not dirty-gated (`:831`, `:840`
disable only on `isSaving`/`isDeleting`), so any unrelated edit hits it. **This is D21's
freeze reconstructed by the fix for D22.**

**(c) The gate's predicate is unspecified, and the obvious reading 400s every Insight
AI save.** `ais_detail.js:421` sends `mcp_server_ids` on every save; for an Insight AI
that value is `[]`. §7's wording ("`TypeInsight` cannot whitelist an MCP server") and
§5 item 5's "denied outright" read as "type is Insight → reject". Implemented literally,
`ValidateMcpServerIDs(…, TypeInsight, [])` rejects and **every save of every Insight
AI 400s**, including AIs that never had MCP data, and including the POST from
`ais_create.js` once PR C adds the field. **The empty list must be explicitly
exempt:** reject only a NON-EMPTY whitelist for Insight.

**(d) A third Insight-transition site exists.** §5 item 5(ii) names two;
`ais_create.js:92` (`setAiType(template.type || 'normal')` in `handleTemplateSelect`)
is a third, reachable for `type: 'insight'` via `prompt_templates.js:659`
(`insight_case_assistant`, `:653`) and re-triggerable after the form is filled by the
"Change Template" button at `:186`. It clears tools but not MCP ids.

**~~Open decisions this creates~~ — ALL THREE WITHDRAWN in v14.** No AI-type gate exists,
so none of the following is a question, a work item, or a §7 assertion. Retained verbatim
only as evidence of how much prescribed work the reversal removed. **Do not implement
anything below.**
1. ~~**What happens on a type flip to Insight while a whitelist is stored?**~~ Options:
   (a) reject the flip while a non-empty whitelist is stored, telling the customer to
   clear it first; (b) auto-clear the whitelist server-side as part of the flip (one
   write, no webhook storm, unlike §5 item 6's rejected bulk loop); (c) validate on the flip
   even when the field is omitted, which requires reading the stored list. Recommend
   **(b)** — it is the only option that cannot freeze a row, and the flip is already a
   deliberate destructive act on the tool set.
2. ~~**Confirm the empty-list exemption** in both the gate and §7's assertions.~~ WITHDRAWN.
3. ~~**PR C: clear `mcpServerIds` on load when the fetched AI is already Insight**, not
   only in `onValueChange`, and add the third transition site (`ais_create.js:92`).~~ WITHDRAWN.

### 3.13a Verified clean in round 6 — recorded so PR A does not over-scope

| Area | Finding |
|---|---|
| SSRF / URL validation | `ValidateURL` IS applied on both write paths: `mcpserverhandler/handler.go:38` (Create) and `:146-150` (Update, under `if url != nil`). Literal private/loopback/link-local addresses are rejected (`ssrf.go:31-57`, `rejectDisallowedIP:63-77`), and the DNS-rebinding case is closed at dial time by `controlRejectDisallowedAddr` (`ssrf.go:117-133`) via the shared guarded client (`mcptoolhandler/client.go:140-144`). **Nothing for PR A to add** beyond §5 item 10's precedence fix |
| Key rotation | **No rotation or re-encryption job exists anywhere in the repo.** Rotation is config-side and decrypt-by-row-version (`mcpserverhandler/secret.go:123-130`, `NewSecretCrypto:79-89`); nothing iterates rows, so zeroed rows would be encountered by no job. Credential zeroing is safe on this axis |
| Caller-set completeness | Full non-test, non-mock enumeration. `db.McpServerUpdate`: exactly 3 callers (`mcpserverhandler/handler.go:206`, `mcpoauthhandler/access_token.go:108`, `mcpoauthhandler/complete.go:126`) — all named in §5 item 1/§6. `db.McpServerDelete`: exactly 1 (`handler.go:226`) — named. `db.McpServerGet`: 10; the three not named in this analysis (`handler.go:82`, `complete.go:129`, `complete.go:161`) are post-write read-backs of a row the same function just wrote, harmless once `:126` is gated. **No caller of consequence is unmentioned** |
| Concurrency | No transaction or row lock on any mcpserver path — `McpServerDelete` is a bare UPDATE, unlike `dbhandler/ai.go:243`+`:294` and `dbhandler/aipromptproposal.go:227`+`:253` which use `BeginTx` + `FOR UPDATE`. For delete-vs-tool-call, the fail-closed re-read per call (`client.go:191`, `:214`) is **sufficient**: the residual window is at most one already-dispatched outbound request. **No transaction warranted.** State this bound in the PR body, since `mcpservers_detail.js:496` promises immediacy |
| Tool-path error surface | `toolHandleMcpCall` converts every failure into a generic `fillFailed(...)` tool result (`mcp_tool.go:137-174`), so gating never leaks a status code to a customer through the AI path |

### 3.14 Other surfaces

| Surface | Finding |
|---|---|
| api-validator, `~/gitvoipbin/monorepo-monitoring/api-validator/tests/scenarios/` (NOT `monorepo/monitoring`, which holds only CLAUDE.md + Grafana dashboards) | `test_mcpservers_lifecycle.py` (12 tests) + `test_mcpservers_oauth.py` (9) = **21 tests**, no parametrize; CRUD/OAuth coverage is substantial. The gap is specific: `grep -rn mcp_server_ids api-validator/` → one hit, a docstring at `test_mcpservers_lifecycle.py:36`. **Zero assertions** on `mcp_server_ids` via `POST /ais` or `PUT /ais/{id}`, **no OAuth-against-deleted-server test**, and **no test asserting `has_secret` after DELETE** (`:244-274` asserts only 200 + `tm_delete is not None`; `has_secret` only on create/update at `:71`, `:95`, `:202`, `:224`) — so credential zeroing does not break the current suite |
| `square-main/public/skill.md`, `llms.txt` | `skill.md:414` / `llms.txt:49` enumerate AI creation fields incl. `tool_names`, no `mcp_server_ids`. Separately `skill.md:774-795` / `llms.txt:11` document "MCP Server" meaning `uvx voipbin-mcp` (VoIPBin AS an MCP server) — same terminology collision as `ai_overview.rst:681`, on a second customer-visible surface |
| `ai_overview.rst:677-685` | links `github.com/nrjchnd/voipbin-mcp`, a third-party fork, while `skill.md:794` links the official `github.com/voipbin/mcp` |
| `openapi.yaml:2239-2244` | the `mcp_server_ids` description omits the "8 hex chars" detail, says nothing about best-effort skip, and promises LLM merging that does not happen (D7, PR B). Redoc is customer-facing, so this is an editable surface — but the merging text must not be "corrected" here, because PR B will make it true |
| `square-talk`, `square-meet`, `square-dev`, `square-admin_new` | zero MCP / AI-tool-config references (`square-admin_new` has no `src/`). Genuinely clear |

## 4. Proceed decision and PR plan

### PROCEED with PR A and PR C.

**PR A — `voipbin/monorepo` — MCP server lifecycle correctness + docs.**
D1, D2, D3, D4, D5, D6, D12, D14, **D15, D16, D17, D18, D19, D20, D21, D24, D25, D26, D27, D28**.
D27 adds the only schema artifact in PR A: one Alembic data migration, authored here and
applied by 대표님.
(D13, D22, D23 are VOIDED by the v14 policy reversal — Insight AIs may use MCP servers — and
are NOT work items.) One
logical unit: "a deleted or
foreign MCP server must be inert everywhere, write paths must reject rather than
silently succeed, and the docs/config must stop describing things that are not true."

**PR C — `voipbin/monorepo-javascript` — square-admin parity.**
D8 + `types/api.ts` + the CLAUDE.md Field Sync Points table row + re-verification
of the `mcpservers_detail.js:496` delete-dialog copy (PR A makes it true).
**Per the Field Sync Points rule this covers FOUR form bodies, not one:**
`ais_create.js:114-129`, and `teamgraph/sidebar.js:514-528` (AI-create) and
AI-edit `:753-768` (both verified to carry `tool_names` and omit
`mcp_server_ids`), plus `ais_detail.js`'s existing implementation left intact.
`sidebar.js` has no `mcpservers` fetch of its own, so the
`ProviderGet('mcpservers?page_size=100')` call at `ais_detail.js:373-377` (a bare
`useEffect`, token-scoped, no extra customer context needed) must be reused there.
**The Insight-transition cleanup prescribed in v12 is WITHDRAWN (v14).** With no
AI-type gate, `mcpServerIds` must SURVIVE a type flip and the MCP Servers card
(`ais_detail.js:1238`) must stay visible for every AI type — including Insight, since
Insight AIs may now use MCP servers. PR C must not add `aiType === 'insight'`
conditions around MCP state or the MCP card.
**Independent of A again; may proceed in parallel.**

**PR B — deferred to its own analysis.** PR A remains its hard prerequisite: PR B
is what makes `CallTool` reachable, so landing it before PR A's gates would turn
today's `ListTools`-only exposure into live "deleted server's tools are callable"
exposure.

**Possible fourth PR** in `monorepo-monitoring` for api-validator coverage (§5 item 4).

### Why this is not overengineering

D1..D4, D12 are correctness/security fixes on a shipped feature with code-verified
failure modes. D5/D6/D14 remove published falsehoods and dead configuration. D8
completes an already-shipped capability. The one thing explicitly NOT built is the
Redis cache (§3.9), because there is no measured signal for it.

### Non-goals

- Everything in the PR B document (D7, D9, D10, D11).
- **Tool-level (per-tool) whitelisting.** `resolveTools:91-105` adds every tool a
  server returns, no per-tool filter. Deferred: no demand signal, and
  `mcp_server_ids []uuid.UUID` extends additively. **The deferral is not
  consequence-free:** server-level whitelisting grants all present AND FUTURE tools
  that server exposes, i.e. a third party can silently expand its own scope.
  Mitigation is one explicit sentence in the docs, not building the feature.
- Changing the soft-delete READ contract (GET after DELETE returns 200) —
  intentional and validator-tested.
- The Redis tools/list cache (§3.9).
- Additional OAuth vendors beyond GitHub/Linear.
- In-app help content (`src/views/help/content/navHelpManifest.js:39`, explicit
  `helpTopicId: null`).
- Narrative docs (`mcpserver_overview` / `_tutorial`) — desirable but not required
  to fix a live 404; decide at PR A design time whether they fit without bloating
  review.
- Resolving the `uvx voipbin-mcp` naming collision platform-wide (§5 item 3): the
  `ai_overview.rst:681` heading is fixed, but `skill.md` and `llms.txt` are out of
  scope.
- api-validator MCP coverage (§5 item 4) — deferred behind Jira ETC-18, its own PR.

## 5. Decisions to lock before the PR A design doc

1. **Gate placement.** Mandatory: transport (`client.go:190,213`), whitelist
   validation (`mcpserver_validation.go:35 ValidateMcpServerIDs`), `McpServerUpdate`
   (`dbhandler/mcpserver.go:146-149`), OAuth `start.go:37` AND `complete.go:112`
   independently, plus the D12 ownership assertion at **BOTH `mcp_tool.go:72-88`
   (resolution) AND `:160-176` (dispatch)**.
   **CORRECTED in v17 (D26, SEVENTH instance).** v5-v16 put the ownership assertion at
   dispatch only. That is the one place it cannot help: §3.1 establishes
   dispatch is **unreachable in production** until PR B advertises MCP tools to the LLM
   (verified again in v17 — `start.go:1184 resolveTools`, `insight_session.go:225 resolveTools` and
   `mcp_tool.go:293 resolveTools` all discard the merged tool list with `_,`, so `tool.go:142 toolHandleMcpCall` never
   fires on an MCP name), while §3.5a shows the customer-visible harm happens at
   **resolution**: `resolveTools:84` → `ListTools` → `client.go:190-196` →
   `buildAuthHeader:73`/`:80` decrypts the stored credential → authenticated outbound
   request to the other customer's URL. A dispatch-only assertion therefore executes
   nowhere while the live leak path stays unguarded — the same shape as D15/D20/D22/D25,
   and a direct violation of this document's own lesson that a policy needs an
   enforcement point on the path that actually runs.
   **Cost: zero.** `resolveTools(ctx, a *ai.AI)` (`mcp_tool.go:52`) already holds
   `a.CustomerID` — `ai.AI` embeds `identity.Identity`
   (`models/ai/main.go:46`, `bin-common-handler/models/identity/identity.go:9`) — and
   the loop at `:73` already fetches `server`. One comparison, no new DB read, no
   signature change. Resolution **skips** the foreign server (best-effort contract,
   `:44-51`); dispatch **refuses** it with the existing
   `"mcp tool is no longer available"` string.
   **ALSO MANDATORY (D28, §3.5c — the member half of D12): add ONE narrow
   fallback-bearing resolution and use it at both MCP sites.**
   (a) Add `resolveActiveAIForMcp(ctx, c *aicall.AIcall) *ai.AI`: current member's AI, else
   the START member's AI on ANY failure (member absent, AI fetch error, team fetch error),
   else `nil`. **Do NOT modify `resolveActiveAIIDFromAIcall`** — it has six other callers
   (`send.go:78`, `:106`, `tool.go:87`, `tool_insight.go:1430`, `listen_trigger.go:155`,
   `start.go:396`) plus a duplicate in `messagehandler/event.go:45`.
   (b) Reuse branch `start.go:375`: pass its result to `refreshMcpToolMap` instead of the
   start-member `a`, **but ONLY when `res.AssistanceType == aicall.AssistanceTypeTeam`** — for a
   plain AI aicall `a` already IS the governing AI, and resolving again both adds an RPC and
   breaks three existing tests (verified: `Test_startReferenceTypeConversation/reuse:_alive…`,
   `…/reuse:_dead…`, `Test_ServiceStart_serviceStartReferenceTypeConversation/normal`). On
   `nil`, keep `a` — **never skip the refresh** (skipping is how v21 preserved the stale map).
   (c) Dispatch `mcp_tool.go:151`: replace `resolveAI` with it, and on `nil` **fail closed**
   (`fillFailed`). **Do NOT add a `resolveAI` fallback behind it** — the resolver subsumes
   `resolveAI` on every path (the same single fetch for `AssistanceTypeAI`; current-then-start for
   a team, where `resolveAI` tries only the start member; refusal for any other type), so such a
   fallback can never recover and only doubles an RPC on the LLM's critical path. Round 17
   enumerated 480 reachable shapes and found zero recoveries; the fallback v25 added here was
   deleted in `7dced7a90`. Dispatch is still not more failure-prone than before the fix: every
   shape in which the old code succeeded is one in which the resolver also succeeds.
   (d) **DECLINED — do not perform.** Moving the team repair (`start.go:383-387`) above the
   refresh was considered and rejected: its caller only `Warnf`s (`:382`), so a failed repair
   leaves `CurrentMemberID` stale regardless and the reorder protects nothing. The fallback
   contract in (a) is the entire mechanism. The committed code deliberately does not do this.
   **WITHDRAWN: v19's hoist-alone (no-op), v21's resolve-alone ("ordering is irrelevant",
   defeated by the missing fallback), and v22's "`resolveTeamMemberAI` already defines
   never-fail" — that helper skips its fallback loop when the member is present but
   `aiHandler.Get` fails (`start.go:36`) or the team fetch fails (`send.go:160-163`).**
   Defense in depth (deleted/status only): resolution (`mcp_tool.go:72-88`) and
   dispatch (`:160-176`).
   **Explicitly forbidden:** gating inside `dbhandler.McpServerGet` —
   `mcpServerHandler.Delete` calls it at `handler.go:230` AFTER `McpServerDelete`
   to return the row and publish `EventTypeDeleted`, and `Update` reads back at
   `:210`; gating there breaks Delete's own success path and the validator's 200
   contract. Confirm, and confirm whether a deleted server is skipped silently
   (matching the existing non-active pattern) or logged at WARN.
   **RESOLVED sub-decision (D15): gating `McpServerUpdate` is necessary but NOT
   sufficient.** Because it discards `RowsAffected`
   (`dbhandler/mcpserver.go:154-156`), a predicate alone yields a silent 200 plus a
   spurious `EventTypeUpdated` (§3.7a). PR A must make `McpServerUpdate` inspect
   `RowsAffected` and return `ErrNotFound` on zero.
   **RESOLVED (v15): apply it uniformly to every caller, token rotation included.**
   `access_token.go:101-108` currently swallows "no row matched"; after this change it
   surfaces an error. That is the desired behavior — rotating a token against a
   soft-deleted server is a bug signal, and swallowing it means never learning about
   it. **Design-doc obligation:** confirm a rotation failure degrades gracefully (the
   tool call fails, per `client.go:97-100`'s existing error path) and does not abort
   the surrounding call.
   **RESOLVED (v15, D16): `McpServerDelete` gains the same predicate**, while the
   handler keeps answering **200 on a repeat DELETE** (idempotent) and stops
   re-publishing `EventTypeDeleted`.
   **Justification CORRECTED in v16.** v15 claimed the 200 was "load-bearing" because
   the validator cleanup fixture would fail on a 404, citing
   `test_mcpservers_lifecycle.py:249-252`. **Both halves were false.** That citation is
   a *docstring*, in a test that says it does **not** use the fixture; and
   `cleanup_report.py:86` declares `ok_codes=(200, 204, 404)`, so `delete_one` counts a
   404 as success (`:96`). **Narrowed in v17:** v16 wrote "no test in the suite issues a
   repeat DELETE" — false, and asserted without the grep. Round 11 found **13 tests**
   across other resources that inline-DELETE an id they also registered with a cleanup
   fixture, so teardown re-DELETEs it (e.g. `test_rag_lifecycle.py:20`/`:53`). The true
   and sufficient claim is that **no *mcpserver* test issues a repeat DELETE**
   (`test_mcpservers_lifecycle.py` deletes at `:263` and `:281` only, and its
   fixture-using tests never delete inline). This
   is the same error class as the `test_ai_lifecycle.py:4` mis-citation (round 9): a
   test artifact quoted as a live constraint without being read.
   **The real justification is API semantics, not the test suite.** DELETE is idempotent
   by contract, `/mcpservers/{id}` GET still returns a soft-deleted row
   (`dbhandler/mcpserver.go:53-59` applies no `tm_delete` filter, asserted deliberately
   by `test_delete_mcpserver_soft_delete_then_get_still_200`), so answering 404 only on
   the *second* DELETE while GET keeps answering 200 would be incoherent.
   **Mechanism the design doc must specify (v16):** the new zero-row error from
   `McpServerDelete` is **swallowed at the handler** (`handler.go:226`), which then falls
   through to the existing read-back at `:230` and returns 200 while skipping the
   duplicate `EventTypeDeleted`. Without stating this, the naive implementation
   propagates the error and yields 404, contradicting this very decision.
   **MANDATORY ADDITION (D20): an existence gate inside `mcpserverhandler.Update`,
   placed ahead of BOTH the `len(fields)==0` short-circuit (`handler.go:194-204`)
   AND the `ValidateURL`/`status.IsValid`/`authType.IsValid` block
   (`handler.go:146-156`).** Without it, `PUT {}` on a deleted server still returns
   200 with the row (§3.7f) and a malformed-URL PUT returns 400 instead of 404
   (§3.7g). The existing test `handler_partial_update_test.go:115-148` pins the
   no-op behavior and will need a deleted-row counterpart.
   **ALSO EXPLICITLY FORBIDDEN (D20 corollary): gating `mcpserverhandler.Get`
   (`handler.go:93-107`).** Its four consumers have contradictory requirements
   (§3.7h): `v1_mcpservers.go:142` must stay 200, while `mcp_tool.go:73`/`:160` and
   `handler.go:203` must fail closed. The defense-in-depth gates therefore live in
   `mcp_tool.go` itself.
2. **Credential zeroing mechanics. RESOLVED (v7): the same statement as the delete
   timestamps. RESOLVED (v18): pre-deploy rows are backfilled by a one-off Alembic data
   migration** (대표님 확정: "a 로 가자").

   **(i) Why the same statement.** A separate `dbhandler.McpServerUpdate` after
   `McpServerDelete` would be **silently no-op'd** by the `tm_delete IS NULL` predicate
   §5 item 1 mandates, because `McpServerUpdate` ignores `RowsAffected`
   (`dbhandler/mcpserver.go:154-156`). Confirmed acceptable side effect: `has_secret`
   flips to false on the `EventTypeDeleted` payload (`mcpserverhandler/handler.go:242`)
   and on the GET-after-DELETE response; no validator test asserts it (§3.14), no
   square-admin consumer breaks (§3.12), no documented contract exists (§3.7), and it is
   truthful.

   **(ii) `key_version` is zeroed alongside the ciphertext columns**, matching the
   explicit secret-clear precedent already in-file, which sets `FieldKeyVersion = 0` next
   to the nil ciphertext/nonce (`mcpserverhandler/handler.go:178-182`). **Use `0`, not
   `NULL`, in both the delete statement and the migration** — functionally equivalent
   after `ScanRow`, but one convention must be picked and the in-file precedent is `0`.

   **(iii) The pre-deploy backfill.** §5 item 2 as written only zeroes credentials going **forward**, inside the `McpServerDelete`
   statement. D4 (`:97-98`, §3.4) states the defect as ciphertext surviving *indefinitely*
   on already-deleted rows — exactly what the forward fix cannot reach. Worse, the obvious
   remediation fails silently: `McpServerDelete` gains `tm_delete IS NULL` (§5 item 1), so
   re-deleting an already-deleted server matches **zero rows**, the handler swallows that
   error by design (the repeat-DELETE 200 rule), and the caller is told **200 success**
   while nothing was zeroed. A forward-only fix would leave D4 half-closed while the PR
   claimed to close it.

   **Scope of the migration.** All seven credential columns, verified against the schema:
   `secret_ciphertext`, `secret_nonce` (`9b0ad37e0360…:35-37`),
   `access_token_ciphertext`, `access_token_nonce`, `refresh_token_ciphertext`,
   `refresh_token_nonce` (`62c10f986f07…:30,:31,:33,:34`), plus `key_version`. All are
   nullable. Note `access_token_expires_at` (`:32`) and `oauth_vendor` (`:29`) sit in the
   same DDL block but are **metadata, not credentials**, and are deliberately excluded.
   Ciphertext/nonce columns are set `NULL`; `key_version` is set `0` per (ii). Predicate: `tm_delete IS NOT NULL`. `op.execute`
   data migrations have precedent in this tree (e.g.
   `1ebd3fdcea8d_call_outbound_configs_add_default_.py`), so no new pattern is introduced.

   **Hard constraint on the design doc.** The migration file is **authored** in PR A;
   `alembic upgrade` is **never run by this workstream** — 대표님 applies it. The
   downgrade must be a no-op with a comment saying so: zeroed ciphertext is
   unrecoverable, and a downgrade that silently "succeeds" while restoring nothing would
   be a worse lie than the bug.

   **Unmeasured precondition, stated rather than guessed.** How many rows have
   `tm_delete IS NOT NULL` with non-empty ciphertext is **not obtainable from the public
   API**, and this document has NOT measured it. The production MCP inventory is known to
   be entirely `api-validator-mcp-*` test rows (100 measured 2026-09-27, zero
   customer-owned), so the blast radius is *probably* test data only — but that is an
   inference, not a measurement, and this document has already been burned three times by
   inference presented as fact. The count is therefore a **design-doc precondition**: one
   DB query by 대표님 before the migration lands. It does not change the decision (the
   migration is correct at any count), only the PR body's claim about impact.

   **(iv) Migration-vs-deploy ordering: either order is safe, and the PR body must say so.**
   If the migration is applied BEFORE the code deploy, a pre-gate `ListTools` against a
   soft-deleted OAuth row hits `Decrypt` with a zeroed `key_version` and fails with
   "key version 0 is not configured" (`mcpserverhandler/secret.go:126-129`) — a clean error on
   a best-effort path, skipped like any other. If AFTER, the gates already make the row inert
   and the migration only cleans residue. **No ordering constraint is imposed on 대표님;**
   stating this prevents an implementer from inventing one.
3. **Naming collision** (`uvx voipbin-mcp` = VoIPBin-as-MCP-server vs
   customer-registered MCP server). **RESOLVED (v15).**
   - **Rename scope: the `ai_overview.rst:681` heading only.** Not `skill.md` or
     `llms.txt` — PR A is already large and a platform-wide rename is its own task
     (kept as a §4 non-goal).
   - **The third-party link is REPLACED, not kept. 대표님 확정: "a로 가자."**
     `ai_overview.rst:685` currently advertises
     `https://github.com/nrjchnd/voipbin-mcp` as "A recommended open-source
     implementation". **This is a documentation defect of the same class as D5, not a
     preference**, and the earlier draft of this document wrongly deferred it to 대표님
     as unjudgeable. Measured 2026-09-27:

     | | `voipbin/mcp` | `nrjchnd/voipbin-mcp` (currently linked) |
     |---|---|---|
     | Owner | **VoIPBin org** | third-party individual |
     | Last push | **2026-04-06** | 2025-05-05 (~17 months stale) |
     | Tests / CI | **both present** | neither |
     | Packaging | **`pyproject.toml` + `uv.lock`** | `requirements.txt` |

     Decisive: the PyPI package `voipbin-mcp` (v0.1.1) declares
     `Repository: https://github.com/voipbin/mcp`. **So `uvx voipbin-mcp` installs the
     official repo, NOT the linked fork** — the docs point somewhere the documented
     command does not go. PR A replaces the link with `github.com/voipbin/mcp`, changes
     "A recommended open-source implementation" to **official**, states `uvx
     voipbin-mcp` as the run command, and **removes the fork link entirely** rather
     than listing it as a community alternative, so exactly one answer is published.
4. **api-validator coverage** (third repo `monorepo-monitoring`, therefore a fourth
   PR): `mcp_server_ids` on `POST /ais`, revocation behavior after DELETE, OAuth
   start/complete against a deleted server, `has_secret` false after DELETE.
   **RESOLVED (v15): DEFER — not in PR A, and not until ETC-18 is fixed.** Not a
   priority judgement: the validator is currently **leaking test resources into the
   production account** (100 `api-validator-mcp-*` servers measured 2026-09-27, ~4 per
   run, Jira **ETC-18**). Adding MCP coverage before the leak is fixed accelerates the
   accumulation. Sequence: ETC-18 → then this coverage as its own PR.
5. **D13 policy: are MCP servers permitted on `TypeInsight` AIs?**
   (v10 confirmed "add the AI-type gate" — "§5 item 5. 추가하도록 하자." — then)
   **REVERSED in v14 — 대표님 확정: "인사이트도 mcp 사용가능하도록 해."** Insight AIs
   MAY use MCP servers. **No AI-type gate is added anywhere**, so:
   - `ValidateMcpServerIDs` does NOT gain a `Type` parameter, and neither call site
     changes signature.
   - **No consume-path type gate.** `mcp_tool.go:73` and `:160` keep only D1's
     `tm_delete`/status gates.
   - **No PR C Insight-transition cleanup and no card hiding.** `mcpServerIds` survives
     a type flip, and the MCP Servers card stays visible for every AI type.

   **Three defects are VOIDED by this decision, not fixed:**
   - **D13** (Insight AIs may whitelist arbitrary MCP servers) — no longer a defect;
     it is the intended behavior.
   - **D22** (§3.12b: a write-time-only gate enforces nothing and freezes flipped AIs)
     — moot; there is no gate.
   - **D23** (§3.12e: a whitelist-omitting type flip bypasses the gate and permanently
     freezes the row) — moot; there is no gate. The three open decisions D23 raised
     (flip semantics, empty-list exemption, on-load cleanup) are all withdrawn.

   **What the decision means, stated honestly.** `AllowedToolNames(TypeInsight)`
   returns `tool.AllInsightToolNames`, and `models/tool/main.go:74-88` requires every
   entry to have "no side effects outside the session's own message/expression
   surface". An arbitrary customer MCP server carries no such guarantee, so MCP tools
   on an Insight AI are NOT bound by the Insight tool contract. That asymmetry is now
   deliberate: **the built-in tool set stays curated; MCP servers are the customer's
   own trust decision on their own AI, whatever its type.** PR A must NOT silently
   reintroduce the restriction, and §7 must not assert it.

   **PR A owes one docs sentence** (D14/§4 docs scope): state plainly that
   `mcp_server_ids` applies to every AI type including Insight, and that MCP tools are
   outside the curated Insight tool set. Without it, a reader comparing
   `AllInsightToolNames`' comment against observed behavior will reasonably file this
   as a bug.

   **Deferred, not decided against:** per-server or per-tool capability limits (already
   a §4 non-goal). If Insight-specific restriction is ever wanted, that is the honest
   mechanism — not the AI-type gate this decision reverses.

   **Coupled decision (D18): does validation move AHEAD of the AI write?** Today
   `ValidateMcpServerIDs` runs after `aiHandler.Create`/`Update` has committed
   (`v1_ais.go:92-128`, `:247-283`), so a rejected whitelist leaves an orphaned or
   already-mutated AI and still returns 400. **Re-scoped in v14:** the §5 item 5 reversal
   removes the signature rewrite that made this "nearly free", so D18 now stands on its
   own merits. It is still worth doing — an orphaned AI on a rejected whitelist is a
   real defect — but it is now an independent change, and the design doc may legitimately
   defer it. Recommend still fixing it in PR A, since D21/§5 item 11's diff logic touches the
   same two blocks anyway.

   **Where the validation must move, if D18 is taken.** `aihandler/chatbot.go:135-139`
   pre-fetches unconditionally ("Pre-fetch unconditionally so all three branches can
   detect changes", `preUpdateAI, errGet := h.db.AIGet(ctx, id)`) before any `AIUpdate`,
   so reordering costs no extra read **provided validation moves INTO
   `aiHandler.Update`**, where `preUpdateAI.McpServerIDs` is in hand. Merely swapping
   the two statements in `v1_ais.go` does not work: that frame has no row read, so
   §5 item 11's diff would lose the stored whitelist it compares against and force a second
   `AIGet`. D18 also destroys `tmp.CustomerID` (passed today at `v1_ais.go:118`,
   `:273`), so the POST path must use the request's customer id and the PUT path must
   take it from `preUpdateAI`.

   **v14 note on a now-moot hazard.** v11-v13 warned at length that an omitted `type`
   decodes to `TypeNone` (`listenhandler/models/request/ais.go:17`, `:56`) and would be
   denied by a deny-by-default `default:` branch, 400ing every type-omitting PUT. **That
   hazard disappears with the §5 item 5 reversal** — no type is passed to
   `ValidateMcpServerIDs` at all. Retained here only so a future reader who revisits
   AI-type gating knows the trap exists: resolution (`TypeNone` → stored →
   `TypeNormal`) happens at `chatbot.go:145-149`, i.e. after the `AIGet`, so any
   type-keyed validation must live downstream of that point.
   **Also corrected in v13 and still worth recording:** v11/v12 cited
   `api-validator/tests/scenarios/test_ai_lifecycle.py:38-48`/`:96-106` as live proof of
   type-omitting traffic. The citation was FALSE — the module is skipped at
   `test_ai_lifecycle.py:4`. **No api-validator test breaks under any gate in this
   document**: `test_mcpservers_lifecycle.py` and `test_mcpservers_oauth.py` never set
   `auth_type: "oauth"` and never touch `mcp_server_ids`.

9. **D17: is `auth_type: "oauth"` allowed on POST/PUT?** **RESOLVED — 대표님 확정:
   reject it (option a).** ("좋아, 네 제안대로 가자")

   The code already declares this the correct behavior and only the implementation
   failed to follow: `models/mcpserver/main.go:17-19` says "never set directly via
   POST/PUT with a customer-supplied secret," and the OpenAPI spec enumerates only
   `["", "bearer", "api_key"]` (`paths/mcpservers/main.yaml:57`, `id.yaml:74`). But
   `validAuthTypes` at `:23-25` includes `AuthTypeOAuth: true`, the write paths check
   only `IsValid()` (`handler.go:48-50`, `:154-156`), and there is no OpenAPI
   request-validator middleware in `bin-api-manager` to fall back on (§3.7c). Option
   (b) — relaxing the docs to match — would deliberately make an already-correct
   document wrong.

   **What the defect produces today:** the row saves with 200 and lists as `active`,
   but the oauth branch resolves its credential through
   `oauthHandler.GetValidAccessToken` (`client.go:98-101`), which fails on an empty
   access token. The customer sees a healthy-looking server whose tools never fire,
   with nothing on the API surface explaining why. **Same class as §5 item 11's
   silent-attach:** a write that succeeds and does nothing.

   **Implementation constraints for the design doc:**
   - Gate `mcpserverhandler.Create`/`Update`, **not** `AuthType.IsValid()` — the
     OAuth completion path legitimately writes `oauth` internally
     (`mcpoauthhandler/complete.go:136-158`) and must keep passing model validation.
   - Follow §5 item 11's shape: **validate the transition, not the value.** A server that
     legitimately completed OAuth must stay editable — a PUT changing only `name`
     that re-submits `auth_type: "oauth"` unchanged must NOT 400. Reject only when the
     request *moves* a row into `oauth`. Without this the gate reproduces D21's freeze
     on OAuth-backed servers.
   - Decide whether `Create` must also reject a `secret` when `auth_type` is `oauth`
     (the secret would be stored and then ignored by `client.go:90-101`).

   **Migration risk: none measured.** All 100 active production MCP servers are
   api-validator leftovers with `auth_type` `bearer` (75) or `""` (25) — **zero rows
   carry a customer-set `oauth`**, so no existing row is frozen by this gate. (Source:
   full cursor walk of `GET /v1.0/mcpservers` on `api.voipbin.net`, 2026-09-27; the
   leftover accumulation itself is tracked separately as ETC-18.)

   **Scope note:** the unenforced assertion appears TWICE in the same file, `:35`
   and again in the Auth Type table at `:100` ("Never set directly by the customer"),
   plus as a code comment at `models/mcpserver/main.go:17-19`. All three now become
   true statements and must be verified together rather than edited.
10. **D20/D15 error contract: which 404 do gated writes return?** Both candidates
   terminate at 404, so there is no 500 risk (§3.7g verified the plumbing), but the
   customer-visible code differs: bare `dbhandler.ErrNotFound` → `RESOURCE_NOT_FOUND`,
   typed `cerrors.NotFound` → `MCP_SERVER_NOT_FOUND`. Recommend `MCP_SERVER_NOT_FOUND`
   for consistency with `handler.Get:97-101`, which already returns it for a
   nonexistent id — keeping "nonexistent" and "deleted" indistinguishable is the right
   privacy posture, but it must be deliberate. Also confirm the gate precedes
   `ValidateURL` (`handler.go:146-150`) so a deleted row never answers 400.
   **v8 correction:** the recommended code is NOT what the D15 path currently
   produces. `handler.go:206-208` wraps with a plain `errors.Wrapf`, so a bare
   `dbhandler.ErrNotFound` from a `RowsAffected`-gated `McpServerUpdate` resolves via
   `listenhandler/main.go:194-195` to `RESOURCE_NOT_FOUND`. PR A must translate it at
   `handler.go:206` (mirroring `Get:96-101`), or the gate path and the TOCTOU race
   path return two different codes for the same condition.
6. **Does PR A prune stale ids from `ai.mcp_server_ids` on delete?** **RESOLVED: no.**
   §5 item 11's skip-not-reject removes the reason pruning existed (D21's frozen AI), and the
   loop's own costs (no transaction on `McpServerDelete`, un-indexed JSON column, one
   `ai_updated` webhook per affected AI) are real. See §5 item 11's "Rejected alternatives."
11. **D21 (RESOLVED — 대표님 확정): what happens when an AI's whitelist already
   contains a deleted id?** **Decision: skip what is already there, reject what is
   newly added.** ("skip-not-reject 방식으로 하자." + "이후에 AI 업데이트를 할 때
   ... 이때는 reject 를 해야하지 않을까?")

   | Path | Behavior | Rationale |
   |---|---|---|
   | POST `/ais` | reject ALL deleted ids | no prior list exists, so every id is new |
   | PUT with field omitted (`req.McpServerIDs == nil`) | no validation | current behavior, unchanged |
   | PUT, id already in the stored whitelist | **skip the DELETED predicate ONLY** — existence and ownership still enforced | already-held; rejecting a deleted-but-held id freezes the AI (D21). Ownership is NOT exempt (D25) |
   | PUT, id newly added to the list | **reject** (`INVALID_MCP_SERVER_ID`) | attaching a deleted server is a user error and must not fail silently |

   **The exemption is per-PREDICATE, not per-id. CORRECTED in v16 — see D25 (§3.5a).**
   `ValidateMcpServerIDs` (`mcpserver_validation.go:35 ValidateMcpServerIDs`) runs **two** checks in one
   loop: existence (`:37-48`) and **customer ownership** (`:50-56`). v12-v15 said the
   diff needs "zero extra DB reads and no signature change", which forces the diff to
   be computed at the call sites and the function to be invoked **with the new subset
   only** — silently exempting already-stored ids from the ownership check too. Per
   §3.5 that line is *the only ownership check on the AI-whitelist → consume path*, and neither
   consume-path gate (`mcp_tool.go:72-88`, `:160-176`) compares `CustomerID`. So the
   exemption as written opens cross-tenant use of another customer's MCP server,
   including its decrypted secret. **This document asserted the opposite and was
   wrong: the earlier draft that called for an extra parameter was RIGHT, and the v12
   "correction" that removed it introduced the hole.**

   **Resolution (v16): pass the stored list in; keep ONE loop.**
   `ValidateMcpServerIDs` gains a `storedIDs []uuid.UUID` parameter (nil at
   `v1_ais.go:118`, `tmp.McpServerIDs` at `:273`, which is the pre-modification
   whitelist because `Update`'s argument list at `:249-266` excludes `McpServerIDs`).
   Inside the loop: existence and ownership run for **every** id; the `tm_delete`
   rejection is skipped when the id is in `storedIDs`. Rejected alternative: two calls
   (full list for existence/ownership, subset for the deleted predicate) — that issues
   a second `McpServerGet` per already-stored id for no benefit. **Still zero extra DB
   reads; one signature change.**

   **Why the diff is mandatory, not an optimization.** `ais_detail.js:406-440` always
   puts `mcp_server_ids` in the PUT body, so `req.McpServerIDs` is non-nil on every
   square-admin save. Without the diff, square-admin traffic must pass the gate
   unconditionally, which is exactly the D21 freeze. The "field omitted" branch only
   ever applies to direct API callers.

   **Why plain skip-everything was rejected.** It would silently accept "attach this
   deleted server to my AI now." The consume-time gate keeps that inert, so there is no
   security incident, but the customer believes the server is connected and never
   learns why its tools never fire — a silent failure. Rejecting only the newly added
   ids reports the error at the moment it is made.

   **Pointer semantics already support this** and are deliberate:
   `server/ais.go:68-74` documents that nil means "field omitted, leave the whitelist
   untouched" while a non-nil pointer to an empty slice means "explicitly clear it."
   PR A must not collapse that distinction.

   **Rejected alternatives.** (b) Prune stale ids on MCP delete: `McpServerDelete`
   (`dbhandler/mcpserver.go:163+`) is a bare UPDATE with no transaction, so a
   mid-loop failure leaves dead ids behind and the AI still frozen; `mcp_server_ids`
   is an un-indexed JSON column (`d8e342656cf0…:22`) with no reverse lookup, so the
   loop must read every AI of the customer; and each rewrite fires
   `ai.EventTypeUpdated` (`mcpserver_validation.go:86`), so deleting one MCP server
   emits a webhook per affected AI — the customer is told N AIs changed when they
   deleted a server. It also cannot clean rows that already exist without a
   migration. Since skip-not-reject fully resolves D21 on its own, the loop adds
   webhook noise, partial-failure handling, and a migration while solving nothing
   that remains. (c) PR C rendering removable "unavailable" entries: not needed for
   correctness; the stale id disappears naturally the next time the customer saves
   that AI, since the form submits only currently-valid ids.

   Applies equally to `teamgraph/sidebar.js:753-782` once PR C adds the field (§4).
7. **Does `square-admin/CLAUDE.md` Field Sync Points gain `src/types/api.ts` as a
   fourth row?** **RESOLVED (v15): yes, in PR C.** One table row; `types/api.ts` is
   exactly the site the existing three-row table lets contributors forget.
8. **D14 scope in PR A:** document the three webhook event types now, or only fix
   the `ai_struct_ai.rst:69` "Defaults to `[]`" vs `,omitempty` contradiction?
   **RESOLVED (v15/v20): the contradiction is MANDATORY in PR A, and so is the
   event-type documentation — PR A is NOT split.** 대표님 확정 (v20): "하나로 가자."
   Round 12's adversarial review proposed splitting PR A into A1 (Go logic + migration)
   and A2 (docs/config, no Go changes), on the grounds that 20 defects across four
   services is a lot to review at once. **Declined.** The standing rule is one PR per
   repository, and the earlier "split out if it stays reviewable" hedge is withdrawn:
   every PR A item lands in a single `monorepo` PR. PR C remains separate only because it
   is a different repository, which is a structural necessity rather than a split.
   The contradiction is a published falsehood customers read today, so it ships with
   PR A regardless. The three event types are net-new prose whose volume is only known
   once the other doc fixes are written; the design doc makes the call, and splitting
   is a scope decision, not a new question.

## 6. Risk table

| Risk | Severity | Mitigation |
|---|---|---|
| Gating inside `dbhandler.McpServerGet` breaks `Delete`/`Update` read-back and the validator's GET-after-DELETE 200 contract | **High** | §5 item 1 forbids it explicitly; run the full 21-test api-validator mcpserver suite |
| Credential zeroing as a separate UPDATE is silently no-op'd by the new `tm_delete` predicate (`RowsAffected` ignored at `dbhandler/mcpserver.go:154-156`) | **High** | §5 item 2 resolved: same statement; test that ciphertext columns are actually null after delete |
| Credential zeroing is irreversible; a mis-fire destroys a live server's secret | **High** | Zero only on the delete path, never on update; unit test that update paths never clear ciphertext columns |
| No customer-ownership assertion in dispatch; team path resolves the START member's AI before refreshing the MCP tool map, so a reused team AIcall calls the wrong member's MCP servers | **High** | D12 (§3.5) prescribed; **D28 (§3.5c) CLOSED IN CODE** (`3fca779e2`, `dd2fb0b39`, `7dced7a90`, `90d2e2f44`) — `resolveActiveAIForMcp` (`helpers.go:194`) resolves the CURRENT member and degrades to the start member on every failure mode, at `start.go:375` (team-gated) and `mcp_tool.go:151` (fail-closed `default` for unknown types). 18 subtests, incl. dispatch team cases verified to FAIL against the pre-fix code. Shared resolver untouched; the `resolveTeamMemberForSend` reorder deliberately NOT done. v19/v21/v22/v23 prescriptions all withdrawn. Residual, stated: TWO paths never refresh the map (send, and reused contact_case via `start.go:603`) — **under-availability**, not exposure (every gate re-reads the row), deferred to PR B |
| A stored FOREIGN id freezes the AI for every unrelated edit | Medium | **D25's ownership predicate reconstructs D21's dead end one predicate over.** `storedIDs` exempts only the deleted-row rejection, so a stored foreign id makes every PUT 400, and the picker is customer-scoped (`servicehandler/mcpserver.go:67-70`) so the id is invisible and unremovable in the UI while `ais_detail.js:421` re-submits it unconditionally. **Accepted, not fixed:** today's code validates ownership on every write, so a stored foreign id cannot be created going forward, and the production walk found **zero** AIs holding any `mcp_server_ids`, so no such legacy row is known to exist. The decision stands (ownership must not be exempt); what PR A owes is this sentence in the PR body, so the state is documented rather than discovered by a stuck customer. If one is ever reported, the remedy is a targeted data fix, not an exemption |
| A deleted server can be resurrected to `active` with a new URL + secret | **High** | D2 |
| Published docs give a 404 endpoint path | **High** (live) | D5 |
| Published docs recommend an unmaintained third-party fork as the way to reach the platform's MCP surface | **High** (live) | D24 (§2; fix in §5 item 3) — same class as D5: the fix is documentation-only (§5 item 3 = remove the fork link, keep `voipbin/mcp` as the sole official client). No code risk; the risk of NOT fixing it is that customers install an abandoned client |
| OAuth Complete has no re-check (TOCTOU between Start and Complete) | Medium | D3; gate both independently |
| PR A "corrects" the `openapi.yaml`/`ai_struct_ai.rst` LLM-merging text that PR B will make true | Medium | Leave D7's text alone in PR A; only fix the `,omitempty`/"Defaults to `[]`" contradiction (§5 item 8) |
| Dead config flag + design doc justifying another decision by citing a nonexistent cache | Medium | D6; correct `2026-09-11-…:466-467`, `:705`, `:750-754` |
| Insight AIs can whitelist arbitrary MCP servers while denied most built-ins | n/a | **NOT A RISK (v14).** 대표님 confirmed this is intended. MCP tools are outside the curated Insight tool set by design; PR A documents the asymmetry rather than gating it (§5 item 5) |
| ~~An Insight AI created by a type flip becomes permanently un-saveable~~ | n/a | **VOIDED (v14).** D23 depended on the AI-type gate, which no longer exists |
| §5 item 9's transition gate has no pre-write row to compare against, and survives square-admin only because `AUTH_TYPE_OPTIONS` omits oauth and `:359-366` renders it read-only | Medium | §3.12c — one pre-write `McpServerGet` must serve D20, §3.7g and §5 item 9 together; assert transition semantics in a backend test, not via the form's option list |
| `mcp_server_ids` grants all present and FUTURE tools of a server | Medium | One explicit docs sentence (§4 non-goals) |
| A foreign customer's MCP server id, once stored, is used at resolution with its decrypted credential on an outbound request | **High** | D25 (§3.5a) + D26 (§3.5b) — the exemption must be predicate-scoped (`storedIDs` param), AND ownership must be asserted at resolution, not only at the dead dispatch path. Both need §7 assertions or the regression is silent |
| Credentials on rows soft-deleted before the deploy are never zeroed, and re-deleting reports 200 while zeroing nothing | **Medium** | D27 — **RESOLVED**, §5 item 2: one-off Alembic data migration over all seven credential columns. Residual: the affected-row count is unmeasured, so the PR body must not claim an impact figure until 대표님 runs the count |
| An AI whose whitelist holds a deleted id becomes un-saveable for ANY edit (400 on every PUT, no UI affordance to clear it, no dirty-state hint) | **High** | D21 (§3.12a) — **RESOLVED by §5 item 11**: skip the deleted-row rejection for already-stored ids (existence + ownership still enforced, D25), reject only newly added ones. Both directions must be unit-tested (§7) |
| Gating OAuth `Complete` discards a freshly-minted vendor grant: `complete.go:84` deletes the state row and `:88` completes the token exchange BEFORE `:126`, so VoIPBin holds a live GitHub/Linear token and drops it unrevoked | Medium | Preferable to writing onto a deleted row, but the orphaned vendor-side grant must be acknowledged in the PR body (and revocation considered) |
| `PUT {}` on a deleted server returns 200 with the row because `mcpserverhandler.Update` short-circuits to `h.Get` before any DB write; NO dbhandler gate reaches it | **High** | D20 (§3.7f). Gate inside `mcpserverhandler.Update` ahead of the `len(fields)==0` branch AND ahead of `ValidateURL` |
| An implementer gates `mcpserverhandler.Get` to fix the above and breaks the GET-after-DELETE 200 contract | **High** | §3.7h names its four contradictory consumers; §5 item 1 forbids it explicitly; defense-in-depth gates go in `mcp_tool.go` |
| Validation precedes existence, so a deleted row answers 400 `INVALID_MCP_SERVER_URL` on a malformed URL — inconsistent, and a weak existence oracle | Medium | D20 corollary (§3.7g) / §5 item 10 |
| Gating `McpServerUpdate` yields a SILENT 200 + spurious `EventTypeUpdated` instead of a rejection, because `RowsAffected` is discarded (`dbhandler/mcpserver.go:154-156`); the same defeats the OAuth Complete gate | **High** | D15 (§3.7a). PR A must return `ErrNotFound` on zero rows; §7's rejection tests are otherwise unachievable |
| Making `McpServerUpdate` honor `RowsAffected` changes behavior for EVERY existing caller, notably `access_token.go:101-108` token rotation | **High** | §5 item 1; enumerate all callers and add a test per caller for the new error path |
| `auth_type: "oauth"` settable via POST/PUT with a secret the oauth path ignores, producing an undiagnosable broken server; no OpenAPI validator middleware exists to catch it | **High** | D17 (§3.7c) / §5 item 9 |
| A rejected `mcp_server_ids` leaves an orphaned (POST) or already-mutated (PUT) AI behind and still returns 400 | Medium | D18 (§3.7d); standalone ordering fix — move validation inside `aiHandler.Update` to reuse `chatbot.go:135-139`'s unconditional pre-fetch. (v15 said "alongside D13's signature change"; D13 is VOIDED, so D18 carries no dependency) |
| Repeat DELETE re-stamps `tm_delete` and re-publishes `EventTypeDeleted`; PR A is rewriting that exact statement for credential zeroing | Medium | D16 (§3.7b); keep repeat DELETE at 200 because DELETE is idempotent by contract and GET still returns the soft-deleted row (v15's "the validator fixture requires it" reason was FALSE — `cleanup_report.py:86` tolerates 404) |
| Docs assert a `9999-01-01` sentinel and an "(enum string)" type this resource does not have, and say nothing about what DELETE does to tool access | Medium | D19 (§3.7e); same note block D5 edits |
| Coverage for `mcp_server_ids` on `POST /ais` is Go-unit-only, no end-to-end | Medium | §5 item 4 (deferred behind ETC-18) |
| `GetValidAccessToken`'s error surface changes once `McpServerUpdate` is gated | Low | Desired behavior; needs an explicit test (§7) |
| PR A "breaks" a customer whose AI depends on a deleted-but-working server | Low | No customer AI can be *calling* those tools today (§3.1); breakage is limited to stopping the unwanted `ListTools` round trip. Note it in the PR body |
| Field Sync Points drift recurs | Low | Update all four sites + amend the table in PR C |
| Teamgraph AI panels are inline, not the shared component | Low | Accept the duplication to match the file's pattern; do not refactor mid-task |
| PR C lands in the dead nav/layout generation | Low | No nav change needed; `/resources/mcpservers/*` routes already live (`routes.js:333-336`, `_nav.js:239`). Changes are inside existing AI forms only |

## 7. Verification plan

**PR A (`monorepo`):** `go mod tidy && go mod vendor && go generate ./... && go
test ./... && golangci-lint run -v --timeout 5m` in `bin-ai-manager`. New unit
tests proving a soft-deleted server is: skipped in resolution, refused in dispatch,
refused in transport, rejected by `ValidateMcpServerIDs` **when newly added and
skipped when already stored** (§5 item 11 — assert BOTH directions: a PUT adding a
deleted id 400s, and a PUT that merely re-submits an already-stored deleted id
succeeds). **New in v16 (D25), MANDATORY:** a test that a PUT re-submitting an
**already-stored id owned by a DIFFERENT customer is still rejected** — the
`storedIDs` exemption must cover the `tm_delete` predicate only, never existence or
ownership, and without this assertion the cross-tenant regression is silent. Also
rejected by
`McpServerUpdate`, rejected by OAuth `Start`, rejected by OAuth `Complete`; that
delete zeroes all secret/token ciphertext in the SAME statement while update paths
never do; **that the D27 data migration is authored but NOT applied by this workstream**
(host: the migration file itself plus `alembic history` — assert by inspection that its
predicate is `tm_delete IS NOT NULL`, that it covers all seven credential columns and
excludes `oauth_vendor`/`access_token_expires_at`, that `key_version` is set to `0` per
§5 item 2(ii), and that `downgrade()` is an explicit commented no-op, since zeroed
ciphertext is unrecoverable and a downgrade that appears to succeed while restoring nothing
would be worse than the bug. `alembic upgrade` is 대표님's step, not this PR's); that **resolution SKIPS and dispatch REFUSES** a server owned by another customer (D12/D26 — both sites, since dispatch alone is dead code per §3.1); that **a reused team AIcall whose `CurrentMemberID`
differs from `StartMemberID` builds its `mcp_tool_map` from the CURRENT member's
whitelist** (D28, resolution side — host: `startReferenceTypeConversation` /
`refreshMcpToolMap`, and `start_test.go:462` already exercises this path); **[COVERED — `helpers_test.go` `Test_aicallHandler_resolveActiveAIForMcp`, 13 subtests;
`mcp_tool_test.go` `Test_toolHandleMcpCall_team`, 5 subtests]** that **BOTH degraded modes at
BOTH sites resolve via the START member and do not fail the call: (i) `CurrentMemberID` absent
from the team, and (ii) the current member IS present but its `aiHandler.Get` fails** — plus
three cases the prose never predicted and the code review found: an assistance type outside
{ai, team} must be REFUSED rather than resolved as an AI (the first implementation failed
OPEN here), the composed mode (member absent AND the start-member fetch fails) must not
re-issue the identical failing fetch, and `CurrentMemberID == uuid.Nil` must resolve the start
member once. All three verified to fail against the un-fixed code by mutation (D28 fallback contract;
hosts: `refreshMcpToolMap` via `startReferenceTypeConversation`, and `toolHandleMcpCall`).
Mode (ii) is mandatory: v22 asserted only (i), which is why its dispatch regression would
have shipped green); **and that
dispatch rejects a tool belonging to a server only the START member whitelists** (D28,
dispatch side — host: `toolHandleMcpCall`. REINSTATED in v21: v19 withdrew this as
unimplementable, which was true of `resolveAI` and false of the package, since
`resolveActiveAIIDFromAIcall` sees `c.CurrentMemberID` and `:127` holds the aicall. It is a
unit test, not E2E, because dispatch is dead code until PR B per §3.1); and `GetValidAccessToken`'s new
error when rotating against a row deleted mid-flight. **§5 item 5 (D13) REVERSED in v14 — assert the OPPOSITE:** a test that an
Insight AI CAN hold and use `mcp_server_ids` (`ValidateMcpServerIDs` accepts it, and
`resolveTools` returns that server's tools for a `TypeInsight` AI). This is a
regression guard in the literal sense: it fails if anyone reintroduces the type gate
this decision removes. No `TypeInsight`-denial test, no unknown-`Type` test, and no
consume-path type-gate test — v12's three prescribed assertions are withdrawn.
**§5 item 9 (D17) now confirmed:** that
POST/PUT setting `auth_type: "oauth"` is rejected; that a PUT re-submitting an
unchanged `auth_type: "oauth"` on an OAuth-completed row still succeeds (transition
gate, not a value gate — a value gate 400s every save from
`mcpservers_detail.js:156`); and that BOTH internal oauth writers still succeed —
`complete.go:136-158` (new row) and the reconnect branch `complete.go:112-133` (`:117`
via `McpServerUpdate` at `:126`).
**New in v6:** that a PUT against a deleted server returns an
error rather than a silent 200 with no `EventTypeUpdated` published (D15 — assert
BOTH the status and the absence of the webhook); that every other
`McpServerUpdate` caller still behaves correctly under the new `RowsAffected`
check, `access_token.go:101-108` included; that a repeat DELETE does not move
`tm_delete` or re-publish `EventTypeDeleted` while still answering 200 (D16); that
`POST`/`PUT` with `auth_type: "oauth"` is rejected (D17, §5 item 9 resolved);
and that a rejected `mcp_server_ids` leaves no AI behind on POST (D18). **New in v7
(D20):** that `PUT {}` (every field omitted) against a deleted server is rejected
rather than returning 200 with the row — the existing
`handler_partial_update_test.go:115-148` no-op test needs a deleted-row counterpart;
that a malformed-URL PUT on a deleted row returns the gate's 404, not 400
`INVALID_MCP_SERVER_URL`; that a **repeat DELETE still answers 200** and publishes
`EventTypeDeleted` exactly once (§5 item 1 — the handler swallows the new zero-row
error and falls through to the read-back, so this asserts the mechanism, not just the
status code); and that `mcpserverhandler.Get` is NOT gated, i.e.
`v1_mcpservers.go:142`'s customer GET still answers 200 while `mcp_tool.go:73`/`:160`
fail closed. **New in v8 (D21), per §5 item 11:** a test that an
existing AI holding a deleted MCP server id can still be saved when editing an
unrelated field (name/prompt/engine) — this is the regression that freezes customer
AIs, and it spans both repos, so PR C must exercise it in a real browser against a
seeded deleted id. A docs assertion for **D24**: no published page links
`nrjchnd/voipbin-mcp`, and the MCP Server section names `github.com/voipbin/mcp` with
the `uvx voipbin-mcp` command (grep-able, same shape as the `mcp_servers` and
`9999-01-01` greps). Clean Sphinx
rebuild; `grep -n 'mcp_servers' docsdev/source/*.rst` returns nothing **and
`grep -n '9999-01-01' docsdev/source/ai_struct_mcpserver.rst` returns nothing**
(D19). Full 21-test api-validator mcpserver suite green (GET-after-DELETE 200 must
still hold, and a repeat DELETE must still answer 200 (API idempotency, not the cleanup fixture — `cleanup_report.py:86` tolerates 404 either way)).

**PR C (`monorepo-javascript`):** baseline-vs-branch test comparison per CLAUDE.md
test gate; `npm run build`; production build served and verified in a real browser
(reviews and RTL do not catch unmount races — 대표님 has repeatedly found real bugs
this way after green reviews). Verified no existing test breaks: `ais_create.test.js:157`
is a blanket `ProviderGet.mockResolvedValue`, `:320/:380/:398/:431` use
`expect.objectContaining`, `:446` uses `stringMatching(/rags/)`; `ais_detail.test.js:690-726`
asserts only the PUT body. All four form bodies (§4) must be exercised.

## 8. Retrospective (twenty rounds)

**v1** asserted "exactly ONE grep hit" and "no `mcpserver*.rst` files exist." Both
false. Root cause: a case-sensitive `grep mcp` that missed uppercase `MCP`, written
up as evidence without re-running it.

**v2** fixed that but overstated D1 ("still resolved and dispatched … indefinitely")
without checking whether the resolved list is ever used, and missed D2, D3, D4, D6,
D7. Root cause: treating `resolveTools`' existence as proof of its effect.

**v3** understated PR B by an order of magnitude: missed the `inputSchema` tag bug,
the `AllowedToolNames` filter, unvalidated tool names, and the missing ownership
check; and invented a type-conversion problem that does not exist. Root cause:
having traced `resolveTools` to its callers, I stopped, never tracing forward to the
python consumer or backward to the MCP wire format.

**v4** still mis-cited the team registration mechanism (`tools.py:113-116`, which
the team path never calls — the real site is `team_flow.py:63-70`, a file absent
from v4 entirely), missed a FOURTH AIcall path that never writes the tool map
(`startAIcallByRealtime:1087`), cited the wrong `GetByNames` (ai-manager has its own
aiType-blind one), and presented a dead code path (`StreamingSend` has no callers)
as a live advertising surface. Root cause: assuming symmetry — that two paths doing
"the same thing" do it the same way.

All four are the same shape: **a claim about behavior inferred from the existence or
apparent symmetry of code, instead of traced end-to-end.**

**v5 (this document's first split version)** passed the fact-check reviewer with no
errors found, but the adversarial reviewer found a NEW failure mode: **prescribing a
fix without verifying the fix produces the promised observable.** v5 mandated a
`tm_delete` predicate on `McpServerUpdate` and promised a test asserting rejection —
without checking that `McpServerUpdate` discards `RowsAffected`, which makes the
predicate a silent no-op returning `nil`. v5 had already established that exact
mechanic one section earlier, for credential zeroing, and failed to carry it across.
v5 also under-scoped PR C (its scope line listed one form body while its own §3.12
and §6 named three) and missed four defects in the very file it was editing
(`ai_struct_mcpserver.rst`: the sentinel note, the example block, the `oauth_vendor`
label, the unenforced oauth sentence).

**v6** fixed all of that and was approved on facts a second time, but the adversarial
reviewer found the **same failure mode one layer higher**: v6 prescribed a
`McpServerUpdate` gate plus a `RowsAffected` check, and `PUT {}` never reaches
either, because `mcpserverhandler.Update` short-circuits to `h.Get` before the DB
write (D20). v6 had even quoted the surrounding function twice without noticing the
branch above the line it quoted. It also missed that validation runs before
existence (so a deleted row answers 400), and that the handler-level `Get` is as
ungateable as the dbhandler one, for a different reason.

**v7** was approved on facts a third time, and the adversarial reviewer found the
**third instance of the same shape — this time spanning two repositories.** v7's own
prescriptions (gate `ValidateMcpServerIDs` on `tm_delete`, and don't prune stale ids)
combine with square-admin's unconditional `mcp_server_ids` PUT and its
`deleted=false`-filtered picker to make any affected AI un-saveable forever (D21).
v7 had even documented the picker behavior and called it "a display decision, not a
data problem" — a judgement that was correct for the code as it stands and wrong for
the code v7 was prescribing. Five rounds missed it because the cause and the effect
live in different repositories, and each repo's section was reviewed against its own
code.

**v10** resolved §5 item 5 and §5 item 9 and declared the document ready for the design doc. Round 8
refuted both halves of that. The fact-check track found that v10 had **fabricated a
constraint**: it asserted the PUT path performs no pre-write `AIGet` when
`chatbot.go:134-138` does exactly that, unconditionally, with a comment saying so. The
adversarial track found the **fourth instance** of the recurring failure mode inside
the newly resolved §5 item 5 itself — and worse, §5 item 5's write-time gate is the one layer that
can neither enforce the Insight policy (the consume path has no type check) nor avoid
freezing customers (square-admin clears tools but not MCP servers on the Insight
flip). The author had diagnosed this exact shape three times, written two lessons
about it, self-flagged a possible fourth instance — and then created a fifth in the
very decision meant to close the loop.

Lessons now in effect:
- **A doc comment is a specification the next implementer will follow.** When a contract is
  withdrawn, grep for the prose that published it — including comments in the code and test
  headers — not just the prescription section. v27's blocking defect was a comment that told
  callers to undo the commit it was attached to.
- **A nil collaborator is a silent early return, not a neutral default.** `refreshMcpToolMap`
  returns immediately unless both MCP handlers are set, so the reuse-path test exercised none of
  the logic it appeared to cover. Wire the collaborators a test claims to assert on, and prove it
  by mutating the behaviour and watching the test fail.
- **Before writing `.Times(2)`, prove the second call can change an outcome.** Two committed
  expectations pinned a doubled RPC as intended behaviour, with comments that described the
  control flow correctly and therefore read as justification. A test asserting a redundant call
  makes the redundancy permanent and hides it from every later reviewer.
- **A fallback is only a fallback if it can succeed where the primary failed.** The v25 dispatch
  fallback was justified as defence in depth and was provably unreachable-as-recovery in all 480
  shapes, because the primary already subsumed it. **State the shape in which the fallback wins;
  if you cannot, delete it.**
- **Key a deduplication guard on the identity of the operation, not the identity of the caller.**
  The guard compared member ids while the RPC was keyed on AIID, and two members may share one
  AI. Check the uniqueness rules that actually exist (`teamhandler/validation.go`) instead of the
  ones you assume.
- **Cite commit hashes only after `git cat-file -t` succeeds.** A hash invented in a summary was
  carried into the decision table and the risk table, the two places an implementer starts.
- **Re-derive every line citation after any commit that shifts lines.** Three consecutive rounds
  fixed anchors and the next commit broke them again. Anchors are mechanically derivable from the
  tree; derive them, never retype them.
- **A prescription is not verified until it compiles and its tests run.** D28 accumulated four
  withdrawn prescriptions across four rounds of document review; the fifth was implemented and
  the compiler plus the existing suite exposed two more errors within minutes, one of them a
  fail-open security gate. **For any change to a gate or resolver, write the code and run the
  tests before the design doc claims the behaviour.**
- **When adding a type dispatcher, copy the `default:` arm from the function you are replacing.**
  The fail-open defect was a `!=` where the function it replaced had a `switch` with an
  explicit refusal. `resolveAI` and `resolveActiveAIIDFromAIcall` both had it; the new one did
  not, and nothing in the diff made that visible.
- **Before adding a retry, read the callee's own fallback.** The duplicated fetch existed
  because the outer retry did not know the inner loop had already attempted the same thing.
  **Ask what the callee already tried, not just what you asked it for.**
- **Mutation-test every assertion a fix depends on.** Each of the 18 D28 subtests was checked
  to FAIL against the un-fixed code. Two earlier `Times(2)` edits were confirmed load-bearing
  rather than masking, which prose review could not have established.
- **A fallback is not "a fallback" until you enumerate the error modes it does NOT cover.**
  v22 cited `resolveTeamMemberAI` as defining "never fail the call" after reading only its
  member-absent branch; two other paths return before the fallback loop. **Read every `return`
  in a function before citing its behaviour as a contract.**
- **When a prescription offers two mechanisms, check they are not mutually exclusive.** v22's
  option A needed an RPC its own clause (d) forbade, and option B mutated a helper with six
  callers and a duplicate copy the document never mentioned. **Offering alternatives hides the
  cost of each; name ONE mechanism and its price.**
- **Never let a safety fix make an existing path MORE failure-prone.** v22's dispatch change
  would have killed every MCP tool call for a team whose current-member AI was briefly
  unfetchable — a strictly worse outcome than the bug. **For each site a fix touches, write
  down today's behaviour on each error path and confirm the fix is no worse.**
- **Say which mechanism is load-bearing.** v22 presented a cosmetic reorder and a
  load-bearing fallback as co-required, so an implementer could have shipped the decorative
  half. **When prescribing two changes, state explicitly which one provides the guarantee.**
- **A helper's error return is part of the prescription.** v21 adopted an existing resolver
  without reading its failure branch: it returns `uuid.Nil` and only warns, with no
  start-member fallback, while the function it replaced had one. **Before prescribing "use
  helper X", read X's error paths and state what happens on each — a fix that is correct on
  the happy path and worse on the degraded path is still a regression.**
- **"Ordering is irrelevant" and "no signature change needed" are the same species of claim,
  and both have now been wrong.** Each asserts that surrounding code does not matter. Three
  instances (D25, D28-v19, D28-v21) came from exactly these two sentences. **When tempted to
  write either, name the neighbouring statement you checked.**
- **Six consecutive rounds found the defect inside the fix for the previous one.** The
  inventory bullet is part of the fix: v21 corrected §3.5c and left §2's D28 bullet still
  prescribing the withdrawn no-op, which is the first thing a design-doc author reads.
  **When a resolution changes, grep the defect id and update every occurrence, then re-read
  the summary layers (inventory, decision table, risk table, verification plan) as a set.**
- **Verify that the prescribed edit produces the observable you promise to test.** v19
  prescribed reordering two statements to change which AI a function receives, with that
  function's parameter list (`refreshMcpToolMap(ctx, res, a)`) quoted three lines above.
  Reordering cannot rebind an argument. **Before writing "fix the order", read the callee's
  signature and name the variable that changes value.**
- **"Function X cannot do this" is not "the codebase cannot do this."** v19 withdrew a
  fourteen-version §7 assertion because `resolveAI` cannot see `CurrentMemberID`, without
  grepping for another resolver. `resolveActiveAIIDFromAIcall` existed, and its own comment
  said it is what every other such site uses. **When an assertion looks unimplementable,
  grep for the capability before deleting the test — deleting a correct test is worse than
  leaving it failing.**
- **Six consecutive rounds found an instance in the edit that closed the previous one.**
  The fix-up edit is now the highest-risk region of any revision, because it is written
  under the impression that the hard thinking is already done. **Review your own newest
  edit first and hardest, against code, not against the paragraph you just wrote.**
- **When a defect has two halves, fixing one half does not inspect the other.** D12 was
  "no ownership check" AND "team validates the wrong member". D26 corrected the ownership
  half's enforcement point and left the member half exactly as broken, in the same
  section, one round later (D28). **Enumerate a defect's independent claims before
  declaring any of them fixed**, and re-check the §7 assertion for each half separately —
  D28 was visible the whole time in a §7 line that could not be implemented.
- **An unimplementable test assertion is a design defect, not a wording problem.** §7
  promised "the team path validates the CURRENT member's whitelist" for fourteen versions.
  Nobody asked *at which call site*, and the answer was "none that can see
  `CurrentMemberID`". **When writing a §7 line, name the function that will host the
  assertion**; if you cannot, the prescription behind it does not exist yet.
- **"Only the owner can judge this" is a claim that itself needs evidence.** v14
  deferred the MCP-link question to 대표님 as unjudgeable without opening either
  repository. Five minutes of checking (commit dates, tests, CI, and the PyPI package's
  declared repo) showed the documented link contradicts the documented command — an
  objective defect, D24. Escalate a question only after establishing there is no fact
  that settles it; otherwise escalation is just unfinished work.
- **Check the artifact a doc tells users to run, not only the text.** The collision here
  was invisible in prose and obvious from PyPI metadata.
- **When a gate needs five rounds of design, question the policy, not the gate.**
  D13 → D22 → D23 was three escalating attempts to place one AI-type gate; each fix
  created the next defect. The author never stepped back to ask whether the restriction
  itself was wanted. 대표님 answered that in one sentence ("인사이트도 mcp 사용가능하도록
  해") and three defects evaporated. **Repeated implementation difficulty is evidence
  about the requirement, not just the implementation.**
- **Surface the policy question early, with its cost.** The author treated "are Insight
  AIs allowed MCP servers?" as settled by an inferred analogy to `AllowedToolNames`,
  and spent five rounds on mechanics. Asking "do we actually want this restriction?"
  before designing the gate would have saved all of it.
- **A gate is only as reachable as its enclosing condition.** v12 verified that both
  new gate points *hold the AI object*, and concluded the gate was sound. It never
  asked which requests *enter* the block the gate sits in — both call sites are inside
  `if req.McpServerIDs != nil`, so the one request that creates the violation (a type
  flip omitting the field) walks past it (D23). Verifying that a gate CAN run is not
  verifying that it WILL run for the case it targets.
- **Name the request that violates the new rule, then trace whether it reaches the
  gate.** Not "is the gate correct" but "what is the cheapest request that breaks this
  rule, and what happens to it line by line".
- **State the predicate, not the policy.** "Insight AIs are denied MCP servers" is a
  policy; "reject a NON-EMPTY whitelist when the resolved type is Insight" is a
  predicate. The policy left the empty list ambiguous, and the literal reading would
  have 400'd every Insight AI save (D23c). Every prescribed rejection needs its exact
  predicate and its exempt cases written down.
- **A frontend cleanup keyed on an event only covers rows that fire that event.**
  Clearing state in `onValueChange` does nothing for a row that loads already in the
  target state — and hiding the control removes the only affordance to fix it.
- **Verify a test citation is live before using it as evidence.** v11/v12 cited two
  api-validator tests as proof of real traffic; the whole module is skipped at `:4`.
  A skipped test proves nothing and cannot regress.
- Use `grep -i` for content/convention greps.
- For any producer function, grep its callers and check whether the return value is
  discarded (`_,`).
- For any struct unmarshalling an external wire format, verify the json tag against
  the actual specification, not local naming convention.
- When a value crosses a process/language boundary, read the receiving side's
  parsing code before asserting the contract holds.
- Before declaring scope, enumerate EVERY path that creates the same resource, and
  verify each one independently — never assume two paths are symmetric.
- Before citing a function, confirm which implementation the caller actually binds
  to (same name, different package, different signature).
- Before citing a code path as live, grep for its callers.
- **When prescribing a fix, verify the fix produces the observable you promise to
  test.** A `WHERE` predicate only rejects if the caller inspects `RowsAffected`; an
  enum in a spec only validates if a validator runs. Trace the fix forward the same
  way you trace a defect.
- **When editing a file for one defect, read it in full and fix every defect in it.**
  A second PR touching the same paragraph is wasted review.
- **When quoting a function to justify a gate, read the WHOLE function from its first
  line, not the neighborhood of the line you care about.** Early returns, guard
  clauses, and short-circuits above your quote can make the gate unreachable (D20).
- **For every gate, name the request that bypasses it.** If you cannot think of one,
  you have not looked hard enough at the branches.
- **When a backend gate starts rejecting data the frontend already holds and
  re-submits, the frontend is part of the change.** Trace every new rejection to the
  client that will hit it, across repository boundaries. "Not our repo" is how a fix
  becomes an outage (D21).
- **A judgement about existing behavior ("this is only cosmetic") expires the moment
  you prescribe a change that touches it.** Re-evaluate every such judgement against
  the post-fix code, not the current code.
- **Never assert that code does NOT do something without grepping for it.** v10 claimed
  the PUT path performs no pre-write read; it performs one, with an explanatory
  comment. A negative claim about code needs the same evidence as a positive one, and
  is easier to get wrong because nothing contradicts it on the screen you are reading.
- **A new policy needs an enforcement point, and the write path is rarely it.**
  Before prescribing a validation gate, ask which layer actually decides the behavior
  at runtime. If the consume path does not check the predicate, a write-time gate is
  documentation, not enforcement — it only constrains rows created after the deploy
  (D22).
- **Every new rejection predicate needs its own frontend trace; an exemption written
  for one predicate does not cover another.** Round 10 proved this lesson was written
  and then violated in the same document: §5 item 11's exemption was implemented as a filter on
  the id LIST, which silently exempted existence and ownership along with `tm_delete`
  (D25, §3.5a). **Write exemptions as predicate-scoped conditions inside the validator,
  never as a narrowed input set** — a narrowed input set cannot express "skip check A but
  still run checks B and C".
- **"No signature change needed" is a red flag when the caller holds state the callee
  needs.** v12 retracted a correct earlier draft that asked for a parameter, on the
  grounds that the diff could be computed at the call site for free. It could — and doing
  so cost the ownership check. Cheapness at the call site is not evidence of correctness
  inside the callee.
- **A test file cited as a live constraint must be opened.** Round 9 caught a skipped
  module quoted as proof; round 10 caught a docstring quoted as fixture code, whose own
  text said the opposite (`cleanup_report.py:86` tolerates 404). Round 11 caught a third:
  "no test in the suite issues a repeat DELETE", asserted without the grep, when 13 do.
  Three times is not a pattern, it is a habit: **quote the assertion or the default
  argument, not the file name**, and run the grep before writing any "no X exists".
- **Locating harm correctly does not relocate the fix.** D25 (v16) moved the diagnosis to
  the resolution path and was verified sound. The gate-placement line it should have
  updated still said dispatch, because that line had been written eleven versions earlier
  for a different defect and nobody re-read it. **When a diagnosis moves, grep for every
  prescription that depended on the old location** — D26 existed for exactly one round
  because that grep was never run.
- **A gate on a path the document itself calls dead is not a gate.** §3.1 established
  dispatch is unreachable in production, and §5 put the only runtime ownership assertion
  there anyway. Two sections of the same document, one contradicting the other's premise.
  Cross-check every prescribed enforcement point against this document's own reachability
  findings before calling a defect closed.

  (superseded, kept for the record) §5 item 11's skip exempts already-stored ids
  from the deleted-server check, and that exemption gave false comfort about the type
  check, which is a different predicate hitting the same unconditional PUT body.
