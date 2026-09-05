# InsightAI Realtime Listen (Proactive Notification) — Design

Status: **Approved — 2 consecutive review approvals (rounds 19 and 20) reached on rev 22; this design-stage review sub-loop (rounds 13-20, covering rev 15's trigger-API/event-ordering rewrite and its follow-on per-AIcall lock) is CLOSED. rev 23 fixes two non-blocking prose-attribution nitpicks round 20 itself flagged as not warranting further review (rev 9 approved via review round 7/8; rev 10 was a `main`-rebase reconciliation; rev 11-14 added and then stabilized the confbridge participant-count guard through review rounds 9-12, closing with round 12's second consecutive approval on rev 14 — see §0 rows 11-14, §10.7-§10.10 for that sub-loop. Rev 15 reopened the design with a substantive, CEO/CTO-directed change — an explicit trigger API replacing the implicit `Start`-hook, plus a related event-ordering fix — resetting the consecutive-approval count per policy. Review round 13 returned REQUEST_CHANGES on rev 15 (1 BLOCKING + 3 HIGH + 4 MEDIUM + 3 LOW, §10.11), fixed in rev 16. Review round 14 found rev 16's ordering fix, once collapsed into a single write-before-create sequence, collided with two pre-existing §5.2.4 rules once concurrent `runListenStart` goroutines for the same AIcall are considered (2 HIGH + 3 MEDIUM + 5 LOW, §10.12), fixed in rev 17 with a per-AIcall create-or-reuse lock and a scoped `owns`-merge. Review round 15 confirmed rev 17's diagnosis and rule were correct, but found the lock itself under-specified — an undersized TTL relative to a hardcoded 30s dependency, and an ownerless release that could delete a different goroutine's still-valid lock, reopening the same race the lock exists to close — plus a description in §5.2.4 that stated the resulting stop-path consequence backwards, and citations for the error-discrimination pattern that didn't show what they claimed (1 HIGH + 3 MEDIUM + 7 LOW, §10.13), fixed in rev 18 with a TTL sized against the goroutine's own outer timeout (not the RPC clients' own internal ones), a per-goroutine ownership token released via compare-and-delete, and the described inaccuracies corrected. Review round 16 independently re-verified rev 18's fixes against actual source (confirming the TTL derivation, the `TranscribeV1TranscribeList` 30000ms citation, and both corrected citations/consequence-descriptions were all genuinely right) but found rev 18's own TTL fix was only half-applied — §5.12's config table and §11 item 13 still carried the old `15`s default and its withdrawn "sum the RPC timeouts" rationale, directly contradicting §5.2.2's new rule — plus three further defects the TTL-vs-timeout change itself exposed: the lock's release ran under the same `ctx` the goroutine's own outer timeout cancels, stranding the lock in exactly the scenario the new TTL sizing was meant to keep working; raising the TTL above the goroutine timeout silently reversed §7 item 2's own "a later goroutine can proceed normally" test claim without updating it; and the lock's new compare-and-delete semantics had no regression test of their own (1 HIGH + **4** MEDIUM + 11 LOW, §10.14 — **§0 row 19 and this line itself originally undercounted this as "3 MEDIUM," a discrepancy review round 17 caught; row 19's own text is left as the historical record per this document's convention, corrected here since this line, unlike §0's rows, is continuously-updated current-state summary, not an append-only log**), fixed in rev 19 by actually updating §5.12/§11 item 13's TTL default and rationale, detaching the lock's release from the acquiring goroutine's own context (`context.WithoutCancel`, precedented elsewhere in this monorepo), correcting §7 item 2's test claim to the TTL-elapsed condition it now actually requires, adding release-layer regression coverage, and sweeping the LOW findings. **Review round 17 independently re-verified all six of round 16's findings against current text and actual source (including the `context.WithoutCancel` precedent citation and this monorepo's Go version) and confirmed every one genuinely fixed, but found rev 19's own sweep left several sections it touched internally inconsistent with each other: the miscount above; a still-uncompiling illustrative code snippet that LOW-7's fix claimed had gained an error check but had not; a brand-new brittle doc-internal line citation introduced by the very fix (LOW-10) meant to remove that pattern, already off by one line; §5.8's file-list bullet now claiming `rollbackListenState` lives in `pkg/cachehandler` alongside the lock/removal primitives, contradicting §9's placement of it in `listen_trigger.go`; §9's own flag count left at "twelve" after this same revision added a thirteenth; §6/§7's lock-error-handling text left describing an acquire-only failure mode ("no transcribe is started") as if it also covered the deferred release, which by construction runs after `TranscribeV1TranscribeStart`; and an acquire-succeeds-but-client-errors edge case that could strand the lock for the new, longer TTL without the crash the design's own residual-risk paragraph names as the only cause (7 MEDIUM + 3 LOW, plus 5 non-blocking nitpicks, §10.15) — fixed in rev 20, see §0 row 20. **Review round 18 independently re-verified all ten of round 17's findings (including re-checking the `context.WithoutCancel`/Go-version citations and confirming the B-7 best-effort-release code is valid, non-leaking Go) and confirmed nine of the ten genuinely fixed with no reservations, but found two new defects inside rev 20's own edit radius: a code comment citing a review finding ("B-9") that does not exist in this document's own numbering (the actual finding was LOW-1); and, more substantively, the `dupFilters` illustrative block B-2's fix rewrote still would not compile — its map was keyed by bare strings (`map[string]any`) rather than `TranscribeV1TranscribeList`'s actual `map[tmtranscribe.Field]any` parameter type, the same "real Go would not compile this" class as two of this loop's own recent findings — plus 6 LOW-severity attribution/wording residue items from B-6/B-7/LOW-1's own fixes (a mislabeled compound finding-id, an "accepted residual" paragraph and a §5.12 row that named crash-only stranding without also naming B-7's new ambiguous-acquire-then-release-also-fails residual, two lingering `tr` identifier references from LOW-1's own rename left in §7 item 2 and §5.2.4, an unflagged intentional-constant 12h TTL, and one self-undermining rationale clause in §10.15's own B-1 row) (0 BLOCKING + 2 MEDIUM + 6 LOW, §10.16)** — fixed in rev 21, see §0 row 21. **Review round 19 independently re-derived all eight of round 18's findings from current text and re-verified every load-bearing citation a second time (`TranscribeV1TranscribeList`'s real signature and every `tmtranscribe.Field` constant used in the corrected map, `TranscribeV1TranscribeStart`'s full 11-argument call, `context.WithoutCancel`'s precedent, this monorepo's Go version) — full compile-read of §5.2.2's entire snippet, not just the two lines MEDIUM-2 named — and returned this design-stage sub-loop's first APPROVE since rev 15 reopened it: 0 BLOCKING, 0 HIGH, 0 MEDIUM. It recorded 5 LOW/nitpick items (a stray "(above)" that should read "(below)," a third leftover `tr.ID` reference in §5.2.4 beyond the two LOW-4 fixed, a mis-citation inside §10.16's own LOW-6 row pointing at the wrong section for a quoted phrase, two now-stale doc-internal line citations in §10.15's nitpick paragraph, and two dangling "the List bullet above" references left behind by B-2's own fix removing that bullet's List call) — none blocking, all fixed here in rev 22 (§10.17) purely to maximize round 20's odds of delivering the second consecutive APPROVE this sub-loop needs to close, given round 20 is this policy's last round before its 20-round cap.** **Review round 20 independently re-derived all five of round 19's findings from current text, performed a second full compile-read of §5.2.2's entire lock snippet against real `bin-common-handler`/`bin-transcribe-manager` source (variable liveness, every RPC call's argument count/types, every cited line range), and confirmed all clean — no BLOCKING/HIGH/MEDIUM findings. It recorded 2 LOW attribution nitpicks inside rev 22's own fixes (one "since rev 21" that should say "since rev 20"; one "trailing clause" descriptor that no longer matched its re-pointed referent) and 3 non-blocking style/philosophical observations, explicitly assessing none of them as warranting a further review round — APPROVE, the second consecutive approval.** Fixed in rev 23 (§0 row 23) purely as a doc touch-up, per round 20's own recommendation, not as a new review-loop iteration — see §0)
Branch: `NOJIRA-Insight-AI-realtime-listen`
Owner: CPO-directed backend feature

## 0. Revision history

| Rev | Date | Change |
|---|---|---|
| 1 | 2026-09-03 | Initial draft. Fed each transcript segment through `aicallHandler.Send` as a `role=user` message; proactive origin derived from tool-call history; `TranscribeV1TranscribeStop(transcribeID)`; hangup cleanup via the existing `EventCMCallHangup` lookup. |
| 2 | 2026-09-03 | **Rewritten after architect review round 1 (REQUEST_CHANGES, 7 BLOCKING + 4 HIGH).** Transcript segments no longer become `Message` rows and no longer go through `Send`. New two-layer architecture: a Redis-backed transcript buffer plus a debounced *listen evaluation turn* that runs on its own pipecatcall. `notify_agent` gets `RunLLM:false`. Proactive origin becomes a first-class `Message.Origin` field, stored as `role=assistant`. Hangup cleanup uses a new indexed `listen_call_id` column. Transcribe session runs under `IDAIManager`. STT stop on hangup delegated entirely to transcribe-manager's own handler. §10 maps every review item to its resolution. |
| 3 | 2026-09-03 | **Revised after independent review round 2 (REQUEST_CHANGES, 3 BLOCKING + 4 HIGH + 4 MEDIUM + 2 LOW).** Fixed a real correctness bug: the 1:1 Redis resolver key breaks the moment a second Case listens to the same call (§5.2.4/§5.3.3, now a set). Re-diagnosed §5.6.3 — the "tool_calls ordering" fix in rev 2 was a no-op; the actual mechanism (`ToolHandle` writes the tool-call row with empty `content`, which pipecat-manager's own context filter drops) surfaces a **pre-existing production defect predating this design**, now called out explicitly and routed to its own ticket rather than papered over here. Replaced the sole `RunLLM:false` defense for `notify_agent` with an explicit reject-if-called-from-a-real-Q&A-turn guard, closing the "the agent's question silently gets no answer" hole. Added the missing `InsightSystemPrompt` guardrails to the listen-turn context. Stated plainly (§4, §5.6.4) that a proactive notification surfaces multiple rows today and added a frontend render filter as the mitigation. Decided the listen-vs-AI-summary transcribe collision (§5.2.2). Corrected the hangup-cleanup justification (§5.7.1) and the STT-stream-count claim (§5.11) to what the cited code actually establishes, rather than overstating it. Fixed metric-name namespacing (§5.13). §10 gains a round-2 matrix. |
| 4 | 2026-09-03 | **Revised after independent review round 3 (REQUEST_CHANGES, 4 BLOCKING + 3 HIGH + 5 MEDIUM).** Two of rev 3's own new mechanisms turned out broken: §5.4.4(c)'s reject-guard assumed a pipecatcall id ai-manager doesn't actually receive (fixed by a real, scoped cross-service change, §5.4.3a) and §5.2.2a's summary-transcribe reuse-tolerance broke `summaryhandler`'s read/lifecycle assumptions (replaced by giving listen its own system customer id, §5.2.1, so it never shares a transcribe with `ai_summary` at all). Separately, rev 1's original defect — Q&A context eviction via `getPipecatcallMessages`'s 100-row window — resurfaced through listen-turn tool-call rows and is closed at the source (§5.4.5: a new `Origin=listen_internal` tag excluded from replay at the query level), which also narrows the orphaned-tool-message finding back to a non-blocking follow-up. Narrowed the pipecatcall-identity guard to the two handlers that can actually fire from a listen turn and scoped its cache-bypass re-read to the one handler that persists (§5.4.4(b)). Fixed a self-contradictory cleanup step order, a stale citation, and frontend field-name mismatches (camelCase vs. the actual snake_case wire fields) that would have made the render filter never fire. §10.2 gains a round-3 matrix. |
| 5 | 2026-09-03 | **Revised after independent review round 4 (REQUEST_CHANGES, 4 BLOCKING + 5 HIGH + 9 MEDIUM + 3 LOW).** Reviewer's own assessment: no remaining structural blockers, only implementation-level bugs in rev 4's three new mechanisms. Fixed §5.4.4(c)'s inverted branch logic (the cache-bypass re-read was in the wrong condition, both failing to catch the case it was meant for and creating a new false-allow) with a single always-fresh-read design. Added explicit `pipecatcallID == uuid.Nil` handling (safe default: treat as a real Q&A turn, never as listen-internal) to protect against permanent message-tagging corruption during a rolling deploy, and confirmed the wire field can be optional so no forced deployment order is needed. Corrected `ApplyFields`'s actual location (`bin-common-handler`, not `bin-ai-manager`) and replaced the unspecified `FieldOriginNot` with a concrete, generic `databasehandler.NotEq` wrapper type. Closed the remaining system-prompt-eviction gap (proactive/real-Q&A-tool rows still competing for the 100-row window) by fetching the leading system row(s) unconditionally, separate from the capped window. Scoped `Origin` tagging to `contact_case` only, so ordinary conversation-AIcall pipecatcall rotation is never mistagged. Fixed the cache-bypass re-read's actual code path (an RPC client argument, not a direct `dbhandler` call), a stop-time `Send`-cooldown collision, a missing OpenAPI tool-name enum update, and several citation errors. §10.3 gains a round-4 matrix. |
| 6 | 2026-09-03 | **Revised after independent review round 5 (REQUEST_CHANGES, 2 BLOCKING + 2 HIGH + 3 MEDIUM — reviewer's own assessment: architecture stabilized, remaining findings all localized to specific sections).** Unified the "is this a listen turn?" decision to one fresh, cache-bypassing read in `ToolHandle`, consumed by both §5.4.5's `Origin` tagging and §5.4.4(c)'s reject-guard — rev 5 had fixed the guard's own cache-bypass logic but left the tagging step trusting a stale cache, which is the more dangerous of the two (permanent data mistagging vs. one rejected tool call). Added a positive "is this AIcall actually in a listening session" check (`ListenCallID != uuid.Nil`) to the listen-turn predicate, closing a false-positive window from best-effort pipecatcall interruption timing. Corrected `getPipecatcallMessages`'s two-fetch restructure to match `MessageList`'s actual newest-first ordering (both fetches reversed the same way, not one assumed oldest-first) and fixed a false claim about `InsightSystemPrompt` not applying to this path. Replaced §8's "zero behaviour change with the flag off" claim with an explicit table of what is and isn't gated — several fixes (context-assembly, stale-reply guard) are general and code-deploy-scoped, not listen-specific. Corrected the service boundary and scope claims for the cache-bypass RPC change and the `NotEq` wrapper's safe value-type range. §10.4 gains a round-5 matrix. |
| 7 | 2026-09-03 | **Revised after independent review round 6 (a scoped, targeted review per round 5's own recommendation — REQUEST_CHANGES, 2 BLOCKING + 3 MEDIUM, reviewer's own assessment: "light," architecture confirmed stable).** Replaced the `ListenCallID != uuid.Nil` term (which contributed no discrimination once a listen session was already active) with a genuinely positive signal: §5.4.3 now registers each listen turn's throwaway pipecatcall id in a Redis set the moment it's minted, and `ToolHandle`'s `listenTurn` resolution becomes a direct membership check — closing a race where a real Q&A tool call, delayed behind a best-effort-interrupted turn, could still be mistagged. This also removes the `AIcallGet` fresh read §5.4.5 previously needed (Redis membership is authoritative on its own), and gates the read that remains (`ReferenceType == ContactCase`) on an immutable, cache-safe field first, so the cost is confined to `contact_case` AIcalls again. Added the flag check `RunListenTurn` was missing, making §8's "rollback stops in-flight sessions" claim literally true rather than aspirational. Corrected several places across §6, §7, §9 that still described the mechanism's now-superseded rev-5/rev-6 shape. §10.5 gains a round-6 matrix. |
| 8 | 2026-09-03 | **Review round 7 APPROVEd rev 7** (0 BLOCKING; 7 localized findings, N-1 through N-7, offered as recommendations for before implementation starts rather than approval blockers). This revision closes all 7 rather than deferring any, since the review loop's own history is that a deferred finding tends to resurface: (N-1) a Redis `SISMEMBER` failure in §5.4.5 step 2 now degrades to `listenTurn=false` instead of failing the tool call closed — provably correct during a Redis outage, since no genuine listen turn can exist then either, and it restores §6's "Q&A unaffected by Redis outage" claim rev 7 had quietly broken; (N-2) a failed turn-id registration (§5.4.3) now aborts that turn instead of proceeding unregistered, which would have reproduced the exact permanent-mistagging failure mode §5.4.5's B1 fix exists to prevent; (N-3) the flag check moved from a standalone step 0 (which ran before the AIcall was fetched) into step 1's require-list, alongside the `c` it and `clearListenState` both need; (N-4) that same path now performs a full stop (releasing an owned transcribe) rather than a bare state clear, so a flag-off rollback doesn't strand a still-running, now-unreachable STT session; (N-5/N-6/N-7) new metric labels, TTL/cleanup documentation, and an explicit statement of where in `ToolHandle` the listen-turn check runs. §10.6 gains a round-7 matrix. |
| 9 | 2026-09-03 | **Review round 8 APPROVEd rev 8 — the second consecutive approval; the review loop concludes.** Round 8 flagged five non-blocking polish items, closed here: (M-1) §8's "next scheduled turn" language corrected — `RunListenTurn` is segment-triggered, not timer-scheduled, so a flag-off rollback's effect lands on the next transcript segment (or, in a quiet call, at hangup), not on a fixed clock; (M-2) named and scoped `stopListening(ctx, c)` as a single two-step helper (§5.7.2's stop snippet, then `clearListenState`) that never calls `ProcessTerminate`, closing the ambiguity that could have wired a listening rollback into ending the agent's whole Q&A session; (L-2/L-4) added test coverage for §5.4.3's registration-failure abort path and the new `skipped_disabled`/`skipped_invalid` outcomes. (L-3, the one remaining citation in §10.5's historical round-6 matrix pointing at the since-renamed step, is left as an accurate record of what rev 7 did at the time, per this document's own convention of not rewriting past rounds' matrices.) |
| 10 | 2026-09-04 | **`main`-rebase reconciliation, not a new review round** — the branch was rebased onto two commits merged to `main` after rev 9: `NOJIRA-Extract-call-transcript-pagination-helper` (pure refactor of `toolHandleGetCallTranscript`, no behaviour change — confirmed via its own PR's test-suite-unchanged claim) and `NOJIRA-Allow-caller-specified-transcribe-id` (adds an optional `id` parameter to `TranscribeV1TranscribeStart`, the RPC this design's §5.2.2 already calls). Reconciled directly against the current source rather than re-reviewed, since both changes are additive/mechanical, not design-affecting: added the new `id` parameter (`uuid.Nil`, preserving server-generated behaviour) to §5.2.2's snippet; updated two `tool_insight.go` line citations shifted by the pagination-helper refactor (§5.1 step 6, §5.2.1's second consequence) — confirmed the underlying logic they describe is unchanged, and that the refactored code's own comments already generically anticipate a second system-initiated transcriber (i.e. `IDAIManagerListen`), which needed no accommodation as a result; documented why the new caller-specified-transcribe-id capability (built for a different use case — pre-binding a per-transcribe RabbitMQ subscription) doesn't change §3's earlier decision against dynamic per-transcribe bindings. No other cited line ranges in `bin-ai-manager` shifted (spot-checked: `AllInsightToolNames`, `mapFunctions`, `ToolHandle`, `toolHandleGetContactInteractions`'s tenant checks, `pkg/toolhandler/definitions.go`'s `RunLLM` lines all unchanged). |
| 11 | 2026-09-04 | **Answers the CEO/CTO's own architectural question — "a conversation is two bridged calls (A-leg, B-leg); which leg do we base listening on?" — and closes part of §11's remaining blocking item. Revised in place after review round 9 (REQUEST_CHANGES, 1 BLOCKING + 3 HIGH + 3 MEDIUM + 3 LOW) — see below for what changed and §10.7 for the full matrix.** First pass confirmed, by reading the actual routing and confbridge code rather than assuming, that (a) `Case.ReferenceID` names the **A-leg (customer) call** because system-generated B-leg flows (`generateFlowForAgentCall` and the `connect` action's B-leg flow) can never carry `case_create`; (b) the two legs are separate `Call` rows joined through per-leg auxiliary join channels into a shared `Confbridge`, without either leg's original channel moving; and (c) the Snoop/ExternalMedia tap attaches to the listened call's own primary channel regardless of that topology, so the join-channel mechanism does not change §5.9's channel-relative `in`/`out` reading. §5.1.1 gained a new participant-count guard (originally numbered step 6a) requiring the listened call's confbridge to have exactly 2 parties before proceeding. **Round 9 found four defects in that first pass, all fixed in this revision:** (1) BLOCKING — the guard's one-shot form fails closed on a perfectly normal call, since `Call.ConfbridgeID`/`ChannelCallIDs` only reach "2 parties" after the agent *answers*, not when `ensureListen` may first run (e.g. at screen-pop/ring time); fixed by turning it into a bounded, metered retry (renumbered as step 7, §5.1.1). (2) HIGH — the "A-leg is structural" claim was over-stated (citing only `case_create`/`ai_talk` broadly); narrowed to the specific system-generated flows that can't carry it, with `actionHandleCall`/the agent-dial RPC named as the (out-of-scope) escape hatches, and a stronger, independent invariant added (`in == Case.Peer`, guaranteed CRM-eligible by `case_create`'s own `isCRMEligiblePeer` check). (3) HIGH — the declared "click-to-call inversion" residual risk is very likely impossible given that same `isCRMEligiblePeer` check (agent/extension/SIP peers can't produce a Case at all); reframed in §5.9/§11 as the narrower "staff calling in as a CRM-eligible peer" vector. (4) HIGH — the guard was undocumented as start-time-only; §5.9/§6/§11 now say plainly it does not catch a 3rd party joining mid-session. Mechanical fixes: corrected stale `§5.2.3`/`step 6` cross-references (the guard is step 7, not 6a), added `§6` error-table rows, `§5.12`/`§5.13` config and metric entries, `§7` test coverage, and `§2`/`§3`/`§8` scope/rollout notes for the new retry and the narrowed listen-eligibility scope — none of which existed after the first pass. |
| 12 | 2026-09-04 | **Fixes review round 10's findings against rev 11's round-9 fix itself — see §10.8 for the full matrix.** Round 10 confirmed BLOCKING-1/HIGH-1/HIGH-2/HIGH-3/MEDIUM-1/MEDIUM-2/MEDIUM-3/LOW-2/LOW-3 from round 9 were genuinely and honestly fixed, but found one new HIGH defect introduced by the BLOCKING-1 fix itself, plus 3 MEDIUM/4 LOW accuracy problems. **HIGH-A**: step 7's "give up on `len >= 3` once the call is `progressing`" fast-fail used `call.Status` (the *listened leg's own* answer state) as a proxy for "has the extra party lingered," but the listened leg is `progressing` for this design's entire target window (queue-wait + agent-ring), so the fast-fail was in practice unconditional the instant any 3rd party appeared — including during a documented, legitimate transient state (`connect` with `early_media: true` and multiple destinations bridges several ringing legs before the losing ones hang up, `bin-call-manager/pkg/confbridgehandler/joined.go:87-97` explicitly anticipates this). Fixed by dropping the fast-fail entirely: the guard now keeps polling on any non-2 count for the full wait budget, with no attempt to distinguish "still converging" from "stably wrong" (removing the now-meaningless `skipped_confbridge_invalid_topology` label). **MEDIUM-A**: the diagnosis of BLOCKING-1 itself misattributed the mechanism — `Call.ConfbridgeID` is set at *each leg's own* join, not only the B-leg's; what stays at 1 through the wait is the party *count*, not `ConfbridgeID`'s nil-ness — corrected. **MEDIUM-B**: the retry's wait budget was claimed to be "well inside the goroutine's own outer timeout," a value the document never actually set (and the two cited precedents are unbounded or unrelated); added an explicit `aicall_listen_ensure_goroutine_timeout_seconds` config (§5.12, §11 item 13). **MEDIUM-C**: §0/status undercounted round 9 as "2 HIGH" against its own 3-item enumeration; corrected. **LOW-A/B/C/D**: an unsound "len==0 impossible" aside (resolved as a byproduct of the HIGH-A rewrite), a "§7 gains item 1a" claim in §10.7 that itself used the broken list-marker pattern LOW-3 flagged (corrected to describe the actual sub-paragraph), an inverted-reading §8 sentence about the flag gate, and an over-attribution in §11 item 1's pointer to items 11/12 — all corrected in place. |
| 13 | 2026-09-04 | **Fixes review round 11's findings — APPROVE, 0 BLOCKING/HIGH, 1 MEDIUM + 6 LOW — see §10.9 for the full matrix.** Round 11 confirmed round 10's HIGH-A/MEDIUM-A/MEDIUM-B/MEDIUM-C/LOW-A/B/C/D were genuinely and completely fixed, and found no new structural defect — only accuracy items left over from round 10's own fix landing in one place without a full sweep. **MEDIUM-1**: §2 Goal 1 still said a 3+-party confbridge is "out of scope entirely," which was true before round 10's HIGH-A fix but not after — a *transiently* 3+ confbridge (the early-media scenario) now resolves via the same retry and has its own §7 regression test; only a *stably* non-2-party call is out of scope. Corrected. **LOW-1**: the status line's own round number was stale relative to the fix it was describing; corrected, and restated as "N of 2 consecutive approvals" to make the loop's remaining condition explicit. **LOW-2**: a stray blank line between rev 10's and rev 11's rows made this table split into two tables under GFM, silently breaking rendering of the two most load-bearing rows in this section; removed. **LOW-3**: §5.1.1's intro still cited `tool.go:191-199` as a "`context.Background()` + timeout" precedent after step 7/§5.12 had already disclaimed it as unbounded; corrected at the source. **LOW-4**: §5.2.4 named the `tm_update`-bypass as covering "this one write path" while concluding "start or stop," and separately still carried a "negligible" timing rationale for the start-write path that the bounded retry (rev 11/12) makes false — a start write can now land 30s+ after panel open, squarely in a window where `Send`'s 3s cooldown could reject a real agent question if an implementer trusted the stale rationale instead of the bypass. Restated plainly: both writes use the bypass, the timing rationale is dropped. **LOW-5**: credited `AddChannelCallID` alone with setting `Call.ConfbridgeID`; named the actual sibling RPC (`CallV1CallUpdateConfbridgeID`) that performs that specific write. **LOW-6**: added an explicit note that repeated panel re-opens during a long ring now spawn multiple concurrent bounded retry loops (bounded, detached, and covered by §5.2.2's existing transcribe-reuse race handling — not a new failure mode, but previously undocumented and worth stating so `skipped_confbridge_not_ready`'s rate isn't misread without it). |
| 14 | 2026-09-04 | **Fixes review round 12's findings — APPROVE, the second consecutive approval; the review loop concludes — see §10.10 for the full matrix.** All four items are non-blocking accuracy polish, none changing a design decision. **MEDIUM-1**: rev 10's rebase sweep (row 10 above) was explicitly scoped to `bin-ai-manager` and never re-checked `bin-transcribe-manager/pkg/transcribehandler/start.go`, which the same merge (`NOJIRA-Allow-caller-specified-transcribe-id`) shifted by ~46-52 lines; 7 citations across §5.1.1, §5.2.1, §5.2.2, §5.2.3, §5.11, and §6 were stale pointers to code that still exists and still does what was claimed — corrected to current line numbers. **MEDIUM-2**: §5.1.1 step 7's own bounded retry means the same AIcall can have two concurrent `ensureListen` goroutines (from repeated panel re-opens during one long ring, §11 item 13's LOW-6 note) resolve to the same transcribe id and race to write `listen_owns_transcribe` on the same row — a naive last-write-wins could leave the actual owner recorded as `owns=false`, which would make §5.7.2's stop path skip a session it should stop (bounded by the call-hangup backstop, but avoidable). §5.2.4 now specifies `UpdateListenState` must OR a `true` into `owns`, never overwrite one with `false`. **LOW-1**: §2 Goal 1's own attribution said its wording was "corrected in rev 12," when the wording itself landed in rev 13 (the *mechanism* it describes was rev 12's); corrected to name rev 13. **LOW-2**: §5.12's `aicall_listen_ensure_goroutine_timeout_seconds` row said "new in rev 11," contradicting its own next clause ("added after review round 10 finding MEDIUM-B") since round 10 postdates rev 11 — the config was actually introduced in rev 12; corrected. |
| 15 | 2026-09-04 | **Substantive rev, opens a new review sub-loop (round 13+) — CEO/CTO-directed architectural change, reached through a `superpowers:brainstorming`-skill dialogue, not an independent review round.** Two changes: (1) **the trigger becomes an explicit public API, `POST /v1/aicalls/{id}/listen`, replacing rev 1-14's implicit `Start`-hook design** (§5.1, rewritten). Rationale (CEO/CTO's own, confirmed in dialogue): bundling "start listening" as a side effect of "create/reuse the Q&A AIcall" conflates two independent concerns that should be independently callable and observable at the API layer — not a correctness fix, a separation-of-concerns one. `ensureListen` is split into `EnsureListenPrecheck` (steps 1-6, synchronous, runs inline in the new HTTP handler) and `ensureListenAsync` (steps 7-8, still a detached goroutine, unchanged in shape). The endpoint returns immediately after the synchronous prechecks — it does not block for step 7's up-to-45s confbridge wait — matching the API's stated purpose (no status-visibility requirement was raised in the dialogue; see §11 item 14 if that changes). Follows the existing `POST /v1/aicalls/{id}/terminate` action-endpoint idiom exactly (verified against source, not assumed — `main.go:348-351`, `ai_aicalls.go:113-131`, `servicehandler/aicall.go:250-270`), so this is a new instance of an established pattern, not a new one. `Start`'s three-success-return problem (§5.1's original subject, rev 1-14) needs no fix under rev 15, since `Start` no longer triggers listening at all. (2) **A real event-ordering gap the same dialogue surfaced is closed**: §5.2.2's transcribe-creation path used to register this AIcall's Redis resolver-set membership *after* `TranscribeV1TranscribeStart` succeeded, leaving a short window in which the freshly-created transcribe's own earliest events would be silently dropped as `dropped_unknown` — closed by pre-generating the transcribe id, registering it in Redis first, then passing that id to `TranscribeV1TranscribeStart` via the caller-specified-id capability (adopted for this ordering purpose only — the dynamic-RabbitMQ-binding purpose that capability was built for, and that §3 already rejected, is explicitly unaffected). §5.2.4, §5.8, §5.10.1a (new), §6, §7, §9, and §11 (items 14-15, new) updated accordingly. Neither change touches §5.2-§5.9's Layer 1/2 architecture, §5.4-§5.7's turn/tool/lifecycle mechanisms, or §5.9's speaker-mapping conclusions — those are unaffected and not re-reviewed by this revision's own review round. |
| 16 | 2026-09-04 | **Fixes review round 13's findings against rev 15 — REQUEST_CHANGES, 1 BLOCKING + 3 HIGH + 4 MEDIUM + 3 LOW — see §10.11 for the full matrix.** Round 13 confirmed rev 15's core motivation was sound (the event-ordering gap is real, the caller-specified-id mechanism works exactly as documented, adopting it doesn't quietly adopt dynamic RabbitMQ binding too) but found rev 15's own two changes each introduced a new defect, plus citation drift. **BLOCKING-1**: the endpoint was designed at the top-level `POST /v1/aicalls/{id}/listen`, mirroring `terminate` — but `terminate` is gated on Admin-console permissions (`PermissionCustomerAdmin`/`PermissionCustomerManager`), while the panel's own existing `Start` call is on the Agent-facing `/service_agents/*` surface (`PermissionAll`) per `bin-api-manager/docs/auth.md`'s explicit rule that Agent frontends must never call the top-level path. An ordinary agent in square-talk would have been denied. Moved to `POST /service_agents/aicalls/{id}/listen`, mirroring the existing `POST /service_agents/contact_addresses/:id/claim` precedent instead. **HIGH-1**: the `EnsureListenPrecheck`/`ensureListenAsync` split lost `kase`/`callID`/`call` across the function boundary (only a bare `bool` crossed it), forcing the async stage to silently re-fetch them — a duplicate RPC pair and a re-derived tenant boundary. **HIGH-2**: `ensureListenAsync` was specified lowercase/unexported, unreachable from `pkg/listenhandler` and unmockable on the `AIcallHandler` interface, making §7's own test items for it unimplementable. Both fixed together by collapsing into one exported `ProcessListen`, closing over `checkListenEligible`'s already-resolved values directly for `runListenStart` — also resolving **MEDIUM-4** (this now matches `ProcessTerminate`'s one-call shape exactly, where rev 15 had put orchestration logic in `listenhandler` instead). **HIGH-3**: rev 15's ordering fix pre-registered only the Redis `SADD`, leaving §5.2.4's DB write exactly where it was (after `TranscribeV1TranscribeStart`) — reopening a *different*, worse race: an early `transcript_created` event could resolve through the now-registered set before the DB write lands, fail `RunListenTurn`'s precondition, and trigger `stopListening`/`clearListenState`, deleting the very state rev 15's own fix had just created and killing the session for the whole call. Fixed by moving the DB write earlier too — both writes now happen together, speculatively, before `TranscribeV1TranscribeStart`, with an explicit `rollbackListenState` undo path on failure. **MEDIUM-3**: that undo path, as first drafted for HIGH-3's fix, treated every `TranscribeV1TranscribeStart` error identically, silently dropping §6's already-documented `TRANSCRIBE_ALREADY_PROGRESSING` reuse-on-conflict behaviour — restored as an explicit discriminated branch. **MEDIUM-2**: rev 15's public, arbitrarily-callable endpoint removed the implicit "just created/reused, therefore live" guarantee `Start`'s old hook relied on; a terminated/deleted AIcall could still reach step 7 and spawn a billed STT session. Step 2 (renamed "AIcall gate") now also requires `c.Status == progressing && c.TMDelete == nil`. **MEDIUM-1**: the precheck's own worst-case sequential RPC latency (~9s across three non-cache-first calls) exceeded `AIV1AIcallListen`'s inherited 3s default RPC timeout; given an explicit 10s timeout instead, following the same per-call-override pattern `TranscribeV1TranscribeStart` already uses. **LOW-1/LOW-2/LOW-3**: a sweep of now-stale bare `ensureListen` references in current-state prose (not the historical §0/§10.x rows, left as an accurate record per this document's own convention); three drifted citations (`ai_aicalls.go`, `servicehandler/aicall.go`, `transcribehandler/start.go`) corrected to their actual current lines; a new open item (§11 item 16) recording that rev 15/16's new failure branches still need explicit metric labels rather than folding silently into existing buckets. |
| 17 | 2026-09-04 | **Fixes review round 14's findings against rev 16 — REQUEST_CHANGES, 0 BLOCKING + 2 HIGH + 3 MEDIUM + 5 LOW — see §10.12 for the full matrix.** Round 14 independently re-verified every one of round 13's fixes against current source (not §10.11's own description) and confirmed all genuinely correct — BLOCKING-1's permission-surface fix in particular checked out line-for-line, and the reviewer traced the full round-trip proving the `TRANSCRIBE_ALREADY_PROGRESSING` discrimination is achievable (though rev 16's own helper for it didn't exist, MEDIUM-1 below). What round 14 found instead was two new defects created by rev 16's own HIGH-3 fix, both arising from a premise rev 16 carried forward without re-checking: that concurrent `runListenStart` goroutines for the *same* AIcall (already documented in §5.1.1's own LOW-6 note as an expected consequence of the bounded retry) would still resolve to the *same* transcribe id, as they did before rev 16 moved the write earlier. They no longer do — each goroutine now mints its own speculative id before either writes anything. **HIGH-1**: `UpdateListenState`'s "OR a `true` in, never overwrite with `false`" `owns`-merge rule (rev 14) was unconditional on transcribe id — so writing `owns=false` against a *different* id (the create-then-fall-back-to-reuse branch) could incorrectly carry forward a stale `owns=true` from an abandoned speculative write, making this AIcall believe it owns a session it does not, which §5.7.2's stop path would then wrongly stop. Fixed by scoping the OR-merge to same-transcribe-id writes only; a differing-id write sets `owns` directly, no carry-over. **HIGH-2**: two concurrent goroutines for the same AIcall, each pre-writing against its own freshly-generated id, can have the second `SREM` the first's already-live session out of the resolver set (rev-4's stale-id-replacement logic, applied incorrectly to a session that was never actually stale), or have a later rollback from the second goroutine delete DB/Redis state belonging to the first's live, billed session. Fixed by wrapping the whole reuse-check-through-write sequence in a new per-AIcall `SET NX EX` lock (`ai:listen:startlock:<aicall_id>`, new config `aicall_listen_start_lock_ttl_seconds`), reversing an earlier "considered and rejected" decision whose reasoning (§5.2.2's cross-AIcall dedup guard already prevents the worse outcome) covered only cross-AIcall races, not this same-AIcall one rev 16's write-ordering change newly exposed. **MEDIUM-1**: rev 16's `cerrors.IsReason` helper does not exist anywhere in this codebase; replaced with the actual established `errors.As(err, &ve) && ve.Reason == "..."` pattern, verified end-to-end against transcribe-manager's actual error construction and ai-manager's actual error-recovery path. **MEDIUM-2**: BLOCKING-1's endpoint-path sweep missed four current-state mentions of the old top-level path (§7 item 20, §8 step 4, §5.10.2 ×2) — corrected. **MEDIUM-3**: the 10s RPC timeout's stated justification ("none of the three RPCs are cache-first") was itself wrong for `CallV1CallGet`, which is cache-first — restated on the timeout-per-hop reasoning that actually holds regardless of caching. **LOW-1 through LOW-5**: a `checkListenEligible` signature mismatch between §5.1's snippet and §5.1.1's prose; two citation slips introduced by the BLOCKING-1 fix itself (`serviceagent_transcribe.go` line range, and using `AIV1AIcallGet` directly instead of the mandated two-level `aicallGet` helper); a handful of `ensureListen` references the rev-16 sweep missed in truly current-state (non-historical) prose; a redundant-but-harmless SREM now explained rather than left implicit; and two stale internal cross-references (§11 item numbering, §5.8-vs-§9 file placement) — all corrected. |
| 18 | 2026-09-04 | **Fixes review round 15's findings against rev 17 — REQUEST_CHANGES, 0 BLOCKING + 1 HIGH + 3 MEDIUM + 7 LOW — see §10.13 for the full matrix.** Round 15 confirmed rev 17's core diagnosis and its `owns`-merge rule were both correct, and independently re-verified round 14's other four findings as genuinely fixed (re-tracing the full `TRANSCRIBE_ALREADY_PROGRESSING` error round-trip end to end). What it found was that rev 17's own new lock — the mechanism meant to *close* round 14's HIGH-2 — was itself under-specified in two ways that could reopen the exact race it exists to prevent. **HIGH-1**: the lock's TTL (`15`s) was sized against only `TranscribeV1TranscribeStart`'s 5000ms timeout, but the lock also wraps up to two `TranscribeV1TranscribeList` calls, whose RPC client hardcodes a **30000ms** timeout not exposed to the caller — nominally summing to 65s, comfortably exceeding 15s. Separately, the lock's release (`h.cache.Del`, unconditional) and acquisition (a constant value) meant one goroutine's release could delete a *different* goroutine's still-valid lock if the first's TTL had already lapsed — reopening round 14's HIGH-2 outright. Fixed by re-deriving the TTL from the goroutine's own outer timeout instead of the RPC clients' internal ones (no call inside the lock can outlive the `ctx` it runs under, regardless of what its own timeout constant claims, so a TTL that exceeds the outer `AIcallListenEnsureGoroutineTimeoutSeconds` — default raised to `60`, strictly above the `45` default — can never expire under genuinely ongoing work), and by giving the lock a per-goroutine ownership token released only via an atomic Redis `EVAL` compare-and-delete (`ListenStartLockRelease`, new cache primitive), not a bare `Del`. **MEDIUM-1**: §5.2.4's own description of the bug HIGH-1 (round 14) fixed stated the consequence backwards — a stale carried-forward `owns=true` makes §5.7.2's stop path **incorrectly stop** a session this AIcall doesn't own (`!owns` evaluates false, so the "never touch it" branch is skipped), not "correctly skip stopping it" as an earlier draft said; corrected, and §10.11's own matrix (which had it right) is now the two sections agree with. **MEDIUM-2**: the two source citations offered as evidence for round 14 MEDIUM-1's fix (`disabled.go:24-28`, `bin-direct-manager/.../main.go:104-112`) don't actually contain the `errors.As`+`Reason` pattern they were cited for — neither has an `errors.As` call performing a `Reason` comparison at all; repointed to the two citations that do (`transcribehandler/stop.go:196-205`, and `bin-storage-manager/pkg/filehandler/signing.go:79` for the exact one-line-wrapper shape used here). **MEDIUM-3**: §5.2.4 overclaimed the lock's scope ("within one AIcall only one write sequence is ever in flight") — true only for the create-or-reuse sequence the lock actually wraps, not for teardown paths (`clearListenState`, `stopListenByCallID`), which do not take this lock and can still interleave with it; restated narrower, with the existing bounded-harm reasoning (§6, `isValidReference`) named as why this is an accepted precision gap rather than a new correctness issue. **LOW-1 through LOW-7**: the lock's own two outcomes (a losing goroutine, a `SetNX`/release Redis error) had no §6 row or §5.13 label — added, and folded into §11 item 16's existing open-item tracking; §5.1.1's own LOW-6 note still described the same-AIcall race as "already covered" by mechanisms round 14 found insufficient — repointed at the lock; §9 still said "seven flags" against §5.12's actual twelve, and omitted the lock's own two cache primitives from its `cachehandler` bullet and Redis-primitive enumeration — both corrected; a genuine Go-level bug in rev 17's own snippet (the reuse-path `List` call's error return was silently dropped, so an RPC failure would have read as "no existing session" and started a duplicate) — fixed, verified against the two other `List` call sites in the same snippet which already handled it correctly; §5.1's claim that the two-level `aicallGet` helper performs the ownership compare itself was imprecise (the helper only fetches; the compare is the caller's own, matching the sibling `ServiceAgentAIcallGet`'s actual division of labour) — restated; the "row's current value" the `owns`-merge rule and the rev-4 SREM rule both depend on was never stated as a fresh DB read (as opposed to the calling goroutine's own, potentially stale, in-hand `c`) — clarified, and confirmed the ambiguity was harmless for `owns` specifically (only the SREM half was ever exposed to it) given the lock already serializes this AIcall's own writers; and four spots this document's own MEDIUM-2 (round 14) sweep touched were attributed to "rev 16" when the text at each spot actually changed in rev 17 — corrected to name the revision that actually touched each one, the same attribution-precision class this document has now caught and fixed three separate times (rounds 12, 13, 15). |
| 19 | 2026-09-04 | **Fixes review round 16's findings against rev 18 — REQUEST_CHANGES, 0 BLOCKING + 1 HIGH + 3 MEDIUM + 11 LOW — see §10.14 for the full matrix.** Round 16 independently re-verified round 15's TTL-derivation reasoning and both corrected citations/prose against actual source and confirmed them genuinely right, but found rev 18's own TTL fix had only been applied to §5.2.2's prose, not to §5.12's config-table row or §11 item 13, which still carried the withdrawn `15`s default and its "sum the RPC timeouts" rationale — the document simultaneously mandated two contradictory TTL constraints. Rev 18's ownership-token/compare-and-delete release fix, once actually applied consistently, also exposed a defect of its own: the release ran under the acquiring goroutine's own `ctx`, so the one case the new TTL-vs-timeout margin exists for (a goroutine reaching its own outer timeout while still legitimately working) was exactly the case where the release would fail, stranding the lock anyway. **HIGH-1**: §5.12's `aicall_listen_start_lock_ttl_seconds` row and §11 item 13 still said `15` with the old rationale, contradicting §5.2.2's `60`/exceed-the-outer-timeout rule. Fixed by actually updating both to `60` with §5.2.2's derivation. **MEDIUM-1**: §5.2.2's own prose named `newTranscribeID` as the lock's ownership token, contradicting its own code block four lines later, which correctly mints a separate `lockToken`; corrected to match the code, and the prose now states plainly why the two identifiers must stay independent (`newTranscribeID` isn't minted until after lock acquisition, and isn't minted at all on the reuse path). **MEDIUM-2**: the deferred release ran under `ctx`, so a goroutine reaching its own outer timeout — cancelling `ctx` — would have its release call fail immediately, stranding the lock for the full TTL in exactly the scenario the TTL-vs-timeout margin was supposed to keep working; fixed by detaching the release's context from `ctx`'s own cancellation (`context.WithoutCancel`, precedented in this monorepo at `bin-schedule-manager/pkg/dispatchhandler/manual.go:102`) with its own short timeout (`aicall_listen_start_lock_release_timeout_seconds`, new config, proposed default `3`), and withdrawing the earlier, false "margin for the deferred release itself to run" justification. **MEDIUM-3**: raising the TTL strictly above the goroutine timeout means a genuinely crashed goroutine (pod loss — the release `defer` never runs at all, unaffected by MEDIUM-2's fix) now strands the lock for the full TTL rather than for less than the outer timeout as rev 17's original `15`s default implied; §7 item 2's own "a later goroutine can acquire it and proceed normally" test claim was stale against this and is corrected to the TTL-elapsed condition it now actually requires, with the trade-off (slower crash recovery, in exchange for the TTL never lapsing under legitimate work) stated explicitly in §5.12/§5.2.2 rather than left implicit. **LOW-1 through LOW-11**: §10.13's own LOW-6 "Where" column incorrectly named §5.10.1a (never touched by that sweep; correctly still "rev 16") and omitted §9 (the actual fifth site) — corrected; a broken doc-internal line citation (`` `:1013-1014` above ``) in §5.2.4's LOW-7 fix replaced with a named-subsection reference; §6's `skipped_start_locked` mention read as though §5.13 already carried that label, when §11 item 16 says the label is still an open decision — reworded to match; §0's own rev-18 row's "default raised to `60`" aside read as attached to the wrong config (the goroutine timeout, not the lock TTL) — left as the historical record per this document's convention, not rewritten, since the row is not factually wrong, only ambiguously worded; §5.8's file-list bullet still omitted the lock's two cache primitives that §9's parallel enumeration already carried — added; the lock's acquire/release API was asymmetric (a raw inline `SetNX` call paired with a named `Release` function, duplicating the key format in two places) — given a matched `ListenStartLockAcquire`/`ListenStartLockRelease` pair, key format built in exactly one place; and §9's "same primitive as the existing debounce lock" phrasing, read literally, implied §5.3.4's lock predates this whole feature rather than an earlier revision of this same still-unshipped design — reworded. Also fixed, closer to housekeeping: the illustrative `dupFilters` pseudo-block (§5.2.2) now actually binds the name the lock sequence references and includes its own error check, rather than showing the exact dropped-error shape round 15's LOW-4 had just fixed a few paragraphs later; the reuse-path `switch` statement's conflict-recovery `List` call renamed (`existingRetry`/`errListRetry`) so it no longer shadows the create path's own `existing`/`errList` names in a section whose most recent bug was about that exact pair; and a pre-existing (not rev-18) conflation of the confbridge-wait budget (`aicall_listen_confbridge_ready_max_wait_seconds`, `30`s) with the goroutine's own outer timeout (`45`s) in §5.1's response-timing prose, corrected to name the right config. §7 item 2 also gains direct regression coverage for the release's compare-and-delete no-op semantics and for the detached-context release itself — neither had a test of its own before this revision. |
| 20 | 2026-09-04 | **Fixes review round 17's findings against rev 19 — REQUEST_CHANGES, 0 BLOCKING + 7 MEDIUM + 3 LOW + 5 non-blocking nitpicks — see §10.15 for the full matrix.** Round 17 independently re-verified all six of round 16's own findings (the TTL derivation, the `context.WithoutCancel` precedent and this monorepo's Go version, the token-naming fix, and the corrected §7 test claim) and confirmed every one genuinely fixed. What it found instead was that rev 19's own sweep, while fixing round 16's specific findings, left several sections it touched internally inconsistent with each other or with sections just outside its own edit radius — the same "fixed the cited defect, broke something adjacent" pattern this loop has now shown five rounds running. **B-1 (MEDIUM)**: the Status line and §0 row 19 itself undercounted round 16 as "3 MEDIUM" against §10.14's own 4-row enumeration. Fixed on the Status line (continuously-updated current-state summary, not a frozen historical row); row 19's own text is left alone, per this document's own distinction between factually-wrong text (corrected) and merely historical text (not rewritten) — the miscount is noted on the Status line instead. **B-2 (MEDIUM)**: §10.14's LOW-7 claimed the illustrative `dupFilters` pseudo-block had been rewritten "with its own error check," but it still declared `err`/`existing` from a `List` call with no `if err != nil` — the same dropped-error shape it claimed to remove, just wearing an `existing, err :=` mask instead of `existing :=`. Fixed by dropping the call from the illustrative block entirely — it now only binds the filter map, with both real call sites (each with their own, already-correct error handling) referenced by name. **B-3 (MEDIUM)**: rev 19's own LOW-10 fix introduced a new brittle doc-internal line citation ("line-990") in the same breath as replacing a different one (LOW-2) — and it was already off by one. Replaced with a description naming the create-path call, not a line number. **B-4 (MEDIUM)**: §5.8's file-list bullet, rewritten in rev 19 to add the lock's two primitives, ended up claiming all four names (including `rollbackListenState`, an `aicallHandler`-level helper that issues an `AIcallUpdate`, not a cache primitive) lived in `pkg/cachehandler` — contradicting §9's placement of `rollbackListenState` in `listen_trigger.go`, and §9 itself still omitted `ListenTranscribeAIcallRemove` from its own `cachehandler` enumeration. Both sections corrected to agree: `rollbackListenState` stays with the trigger logic in `listen_trigger.go`; `ListenTranscribeAIcallRemove`/`ListenStartLockAcquire`/`ListenStartLockRelease` are the three cache primitives, listed in both §5.8 and §9. **B-5 (MEDIUM)**: §9's flag count was left at "twelve" the moment rev 19 added a thirteenth flag (`aicall_listen_start_lock_release_timeout_seconds`) without updating this count — corrected to "thirteen." **B-6 (MEDIUM)**: §6's lock-error row and §7 item 2's matching test both described `ListenStartLockAcquire`/`ListenStartLockRelease` errors identically ("no transcribe is started," "metered `failed`," attributed to `checkListenEligible`) — which cannot hold for the deferred `Release` call, since it runs *after* `TranscribeV1TranscribeStart` may already have succeeded, its own error is discarded (`_ =`) by design, and the lock lives in `runListenStart` (§5.2.2, step 8), never in `checkListenEligible` (steps 1-6). Split into two rows/two test cases: an acquire-error path (still fail-closed, still metered `failed`, zero `TranscribeV1TranscribeStart` calls) and a release-error path (best-effort, never metered, the lock falls back to its own TTL if the release genuinely didn't happen). **B-7 (MEDIUM)**: an acquire call that succeeds server-side (the `SET NX` lands) but errors client-side (timeout, connection reset mid-response) registers no `defer`, so nothing would ever release that lock — a stranding path the design's own "reserved for an actual crash" claim didn't cover. Fixed by attempting a best-effort release with the same token on the acquire-error path itself, before returning the error, collapsing the residual (a second failure on that same attempt) into the already-accepted crash case rather than leaving it as a distinct, undocumented one. **LOW-1/LOW-2/LOW-3**: `tr` was declared from `TranscribeV1TranscribeStart`'s return and never used (real Go would not compile this) — replaced with `_`; `skipped_start_locked` was asserted as settled in two places in §7 item 2's own text after §6 had already been reworded (rev 19, round 16 LOW-3) to call it "proposed, not yet added to §5.13" — both §7 mentions reworded to match; and the "already uses"/"already established" wording split rev 19 introduced between §5.2.2 and §9 for the debounce-lock cross-reference is now consistent in both places. Five non-blocking nitpicks (a scoped, unambiguous-in-context "already uses" phrasing since generalized anyway; a `45s`-attributed-to-step-7 imprecision distinct from LOW-11's already-fixed conflation; a defensible internal-route-path mention; and confirmation that the §0-row-freezing convention itself was applied correctly and consistently) recorded in §10.15, not separately fixed. |
| 21 | 2026-09-04 | **Fixes review round 18's findings against rev 20 — REQUEST_CHANGES, 0 BLOCKING + 2 MEDIUM + 6 LOW — see §10.16 for the full matrix.** Round 18 independently re-derived all ten of round 17's findings from current text (not §10.15's description) and re-verified the load-bearing citations a second time — `context.WithoutCancel` at `bin-schedule-manager/pkg/dispatchhandler/manual.go:102`, this monorepo's `go 1.27.1`, `TranscribeV1TranscribeList`'s hardcoded `30000`ms client timeout, and `TranscribeV1TranscribeStart`'s 10-parameter signature — confirming all still accurate, and further confirmed B-7's new best-effort-release code is valid Go with no leaked `cancel()`. Nine of round 17's ten findings were confirmed genuinely and completely fixed with no reservations. **MEDIUM-1**: §5.2.2's own code comment, explaining why `TranscribeV1TranscribeStart`'s return is discarded, cited "round 17 finding B-9" — a finding id that does not exist anywhere in this document's numbering (B-1 through B-7 is round 17's actual range); the finding that actually motivated this specific edit is LOW-1. Corrected. **MEDIUM-2**: more substantively, the `dupFilters` illustrative block B-2's fix (rev 20) rewrote to remove its dropped-error bug still would not compile — it declared `map[string]any` with bare string keys, but `TranscribeV1TranscribeList`'s actual signature (`bin-common-handler/pkg/requesthandler/transcribe_transcribes.go:40`) takes `filters map[tmtranscribe.Field]any`, a distinct named type Go does not implicitly convert between; the block also mixed an untyped string-keyed map with an already-typed value (`tmtranscribe.StatusProgressing`). Fixed by keying the map with the actual `tmtranscribe.Field` constants (`FieldCustomerID`, `FieldReferenceID`, `FieldStatus`, `FieldDeleted`, all verified present at `bin-transcribe-manager/models/transcribe/field.go`). **LOW-1 through LOW-6**: §7 item 2's new deferred-release test case attributed itself to "round 17 finding B-6/MEDIUM-4," reading as if MEDIUM-4 were also round 17's when it is round 16's — reworded to "B-6, extending round 16's MEDIUM-4 coverage"; both the §5.2.2 "accepted residual" paragraph and §5.12's lock-TTL row named crash-only ("pod loss") stranding as the sole cause of a full-TTL strand, without also naming B-7's own new residual (an ambiguous acquire error whose best-effort release also fails) — both now name it explicitly; §7 item 2's happy-path assertion and §5.2.4's `UpdateListenState` description both still referenced `tr.ID`, a variable that either no longer exists (§7, after LOW-1's rev-20 rename) or was never in scope there in the first place (§5.2.4, whose own parameter is `transcribeID`) — both corrected; the Redis resolver set's `12h` membership TTL was the only listen timing constant in this design with no §5.12 config row and no stated reason why — a sentence added stating this is deliberate (a safety-margin bound, not a tuning knob), not an oversight, so it is not promoted to a fourteenth flag; and §10.15's own B-1 row argued from a slightly different, self-undermining framing ("factually-wrong text (corrected) … merely historical text (not rewritten)" — row 19's miscount *is* factually wrong, which argues for correcting it, not leaving it) than the actually-operative rule its own trailing clause stated correctly (a still-current summary line is corrected; a frozen historical row is not) — **left as the historical record of round 17's own review, per this document's now well-established convention, rather than rewritten; noted here instead**, the same treatment §0 row 18's LOW-4 (round 16) and row 19's B-1 (round 17) both received before it. |
| 22 | 2026-09-04 | **Fixes review round 19's findings against rev 21 — APPROVE (first of 2 consecutive needed), 0 BLOCKING/HIGH/MEDIUM + 5 LOW — see §10.17 for the full matrix.** Round 19 independently re-derived all eight of round 18's findings from current text, re-verified every load-bearing citation a second time, and additionally performed a full line-by-line compile-read of §5.2.2's entire lock snippet (not just the two lines MEDIUM-2's fix touched) — every variable declaration, every RPC call's argument count and types, against the actual signatures in `bin-common-handler`/`bin-transcribe-manager` — and found the snippet clean: no unused variables, no dropped errors, no type mismatches. This is the sub-loop's first APPROVE since rev 15 reopened it. Round 19 also independently confirmed §5.8/§9's file placement and §9's "thirteen" flag count (a fresh count of §5.12) were undisturbed by rev 21, and that §10.15's B-1 row was correctly left unrewritten. Five LOW/nitpick items, none blocking, fixed here anyway to maximize round 20's odds of the second consecutive APPROVE this sub-loop needs before its 20-round cap: **LOW-1**: §5.2.2's own "accepted residual" paragraph, extended in rev 21 to name a second stranding cause, pointed readers "(above)" to text that is actually ~40 lines below — corrected to "(below)." **LOW-2**: a third `tr.ID` reference survived in §5.2.4's historical description of rev 2's original (since-replaced) key format, beyond the two rev-21 LOW-4 already fixed — corrected to `transcribeID`, matching the other two. **LOW-3**: §10.16's own LOW-6 row quoted a framing phrase and attributed it to "§10.15's own B-1 row," but that exact phrase is not there — it is verbatim in §0's row 20; corrected the attribution (row 20 itself, being a frozen historical row, is left unrewritten; only the matrix's mis-citation of *which* row carries the phrase is corrected). **LOW-4**: §10.15's own nitpick paragraph carried two doc-internal line citations that had already drifted stale by the time round 19 checked them — replaced with descriptive references, the same fix this exact anti-pattern has already received twice elsewhere in this document (round 16's LOW-2, round 17's B-3). **LOW-5**: two mentions of "the `List` bullet/call ... above" in §5.2.2 went stale the moment rev 20's B-2 fix removed the illustrative block's own `List` call, leaving nothing above either mention that still shows one — reworded to describe the reuse-check bullet and the `dupFilters` binding directly, without implying a `List` call sits where it no longer does. |
| 23 | 2026-09-04 | **Fixes review round 20's two LOW nitpicks against rev 22 — APPROVE (second consecutive), 0 BLOCKING/HIGH/MEDIUM + 2 LOW + 3 non-blocking observations — see §10.17 (this row extends its scope; round 20 did not open a new §10.18, since it found nothing warranting a further review round).** Round 20 independently re-derived all five of round 19's findings from current text and performed a second full compile-read of §5.2.2's entire snippet against real `bin-common-handler`/`bin-transcribe-manager` source, confirming the sub-loop's second consecutive APPROVE — **closing the review sub-loop opened by rev 15** (rounds 13-20). It found rev 22's own LOW-5 fix (§5.2.2) had itself introduced an attribution slip: "not, since rev 21, shown as its own illustrative `List` call" — the `List` call was actually removed in **rev 20** (round 17 finding B-2), not rev 21 (rev 21 only re-keyed the map's type, round 18 finding MEDIUM-2); corrected, and the finding's origin (round 17 B-2) named explicitly. It also found rev 22's own LOW-3 fix (§10.16) had left a stale positional descriptor: "its own trailing clause states correctly" was accurate for §10.15's B-1 row (whose operative-rule statement is trailing) but not for §0's row 20 (the corrected referent), whose statement appears earlier in the same sentence, in a parenthetical, not a trailing clause; reworded to name the parenthetical precisely. Three further observations — a tension between §0 row 9's "matrices are never rewritten" framing and this loop's actual practice since rev 19 of correcting factually-wrong matrix rows in place, a self-referential "(§5.2.2 above)" citation, and a locally-inconsistent (but not convention-violating) `err`-vs-`errUpdate` naming choice inside one `if`-init — were recorded by round 20 as pre-existing, cosmetic, and explicitly not worth their own fix or a further review round; left as-is. No further review round was dispatched for this row: round 20's own verdict already closed the sub-loop, and both fixes here are the kind of trivial, non-substantive prose correction this document's own policy treats as ordinary editing rather than a new design change. |
| 24 | 2026-09-06 | **CEO/CTO decision: both listen switches removed; listening is always on.** `aicall_listen_enabled` / `AICALL_LISTEN_ENABLED` no longer exists, so §5.4.1's flag precondition, §5.12's `aicall_listen_enabled` row and §8's flag-based rollout and rollback no longer apply; the §5.13 `skipped_disabled` result is retired. The companion conversation design (`docs/plans/2026-09-05-insight-ai-conversation-listen-design.md`, rev 8) records the same decision for `aicall_listen_conversation_enabled`. Two documents that still describe the switches are deliberately left untouched as frozen implementation records: `docs/plans/2026-09-04-insightai-realtime-listen-plan.md` and `docs/superpowers/plans/2026-09-05-insight-ai-conversation-listen.md`. |

Every code reference below was re-verified against the worktree at rev 2 authoring time; file:line citations are load-bearing and were read, not assumed.

---

## 1. Problem statement

Today's Case Insight Assistant (`AI.Type == insight`) is purely reactive: the
agent asks a question in the Case Insight Assistant panel
(`square-admin/src/views/contacts/CaseInsightAssistantPanel.js`,
`square-talk/src/features/cases/CaseInsightAssistantPanel.jsx`), the LLM
calls read-only tools (`get_contact_interactions`, `get_call_transcript`, …
— `bin-ai-manager/pkg/aicallhandler/tool_insight.go`) to answer. The AI
never initiates. It has no visibility into a call that is happening right
now.

대표님's request: while an agent is on a call tied to an open Case, InsightAI
should follow the live conversation as it happens and, if it judges the
situation warrants it, speak up first — surfacing a warning or suggestion to
the agent before being asked.

## 2. Goals

1. When an agent opens/resumes a Case whose linked call is still in
   progress **and bridged 1:1 to exactly one other party** (the normal
   agent-answers-the-call shape), InsightAI automatically starts listening
   to that call's live transcript. **Scope narrowed in rev 11 (the retry
   mechanism), the wording here corrected in rev 13, review round 11
   finding MEDIUM-1** (§5.1.1 step 7): a Case's call that is still queued, still
   ringing, or *transiently* in a 3+-party confbridge (e.g. an
   early-media multi-destination `connect` before the losing legs hang
   up) all resolve themselves via the same bounded retry once the call
   settles to a normal 2-party bridge — round 10's own HIGH-A finding is
   precisely that a fast-fail on 3+ was unsound and had to be removed. Only
   a call that is *stably* not 2-party for the entire retry budget is out
   of scope (§3); rev 11's first draft of this goal understated that
   distinction before that fix existed.
2. The **same** Insight AIcall the agent already chats with is the one that
   watches the call and speaks up — no second AI config, no second AIcall,
   no separate "watcher" session record.
3. Proactive messages land in the existing panel thread over the existing
   delivery path (message row → webhook event → WebSocket / poll), and are
   structurally distinguishable in the UI from an answer to a question.
4. Listening stops automatically when the underlying call ends. No new
   billed STT session is left running.
5. What triggers a proactive message is entirely customer-configurable —
   ships as an extension of the AI's existing `init_prompt`, not a
   hardcoded rule set.
6. **(new in rev 2)** Watching a call must not: pollute the agent-visible
   message thread, emit a customer webhook per spoken sentence, evict the
   system prompt or the agent's Q&A history from the LLM context, kill an
   in-flight answer to the agent, or scale LLM cost with speech volume.

## 3. Non-goals (explicit scope cuts)

| Item | Why cut | Re-engagement signal |
|---|---|---|
| Detecting a call that starts *after* the Case is already open, when that call is not the Case's own `ReferenceID` call | `Case.ReferenceID` (VOIP-1253) is fixed at Case-creation time to the call that produced the Case. A later, different call touching the same contact is a distinct scenario (peer/contact matching) with real ambiguity (which of possibly several concurrent calls?) that the CPO has not asked for | A concrete request for "listen to whatever call this contact is on right now, even if it's not the one that opened the Case" |
| Rule-based / keyword-only condition detection | CPO explicitly directed LLM judgment over customer-defined prompt instead (2026-09-03 design discussion) | N/A — deliberately rejected for MVP |
| Separate watcher AI / second AIcall session | CPO explicitly directed single-AI consolidation (2026-09-03 design discussion) | A demonstrated need to decouple watch-cadence from the agent-facing chat session (e.g. cost isolation) |
| square-talk WebSocket push (replacing its 2s poll) | Existing poll cadence is adequate to surface a proactive message with acceptable latency; SQUARE-52 already tracks square-talk WebSocket parity as a separate concern | SQUARE-52 lands, or proactive-message latency is reported as a problem |
| Multi-party (3+) speaker attribution | `transcript.Direction` is binary (`in`/`out`); conferences already only distinguish two "sides" (`call_media.rst`: "direction indicates speaker relative to conference"). Out of reach without a data-model change upstream in `bin-transcribe-manager`. **Enforced at listen-start in rev 11** by §5.1.1 step 7's confbridge participant-count guard — a call that is not (or no longer, at start-check time) a stable 2-party bridge simply does not begin listening. That guard is start-time only, not a running invariant (§11 item 12) | A concrete multi-party Insight request |
| Transcript content stored as `Message` rows (any role) | **Changed in rev 2.** Rev 1 proposed `role=user` with a speaker tag. That makes every spoken line a customer webhook delivery (`messagehandler/db.go:81` → `notifyhandler/publish.go:24-26`), a panel-visible bubble, and a consumer of the 100-row LLM replay window (`aicallhandler/start.go:620-661`) that would evict the system prompt itself. Transcript is now ephemeral Redis state (§5.3), never a row | A concrete need to persist/audit what the AI heard — which should then be a purpose-built table, not the Q&A message thread |
| Per-turn LLM tool restriction (listen turns limited to `notify_agent` only) | `PipecatV1PipecatcallStart` takes no tool list — pipecat-manager resolves tools from the AI record via `toolhandler.GetByNames` (`bin-pipecat-manager/pkg/toolhandler/main.go:91-108`). Restricting per session means a new field on the pipecatcall start contract, in both Go and the Python runner | Telemetry shows listen turns burning cost on unnecessary read-only tool calls |
| Billing Insight listening to the customer | The listen STT session runs under `cmcustomer.IDAIManagerListen` (§5.2.1), so it is not attributed to the customer's transcription usage. Whether Insight listening becomes a billed line item is a pricing decision, not an architecture one | A pricing decision to monetise Insight listening |
| Dynamic per-transcribe RabbitMQ bindings | Available (`transcript.EventSubscriptionID()` returns `TranscribeID`, `models/transcript/transcript.go:33-35`) and would eliminate wasted deliveries, but introduces a bind/unbind lifecycle whose failure mode is a permanently leaked binding, and breaks the "all bindings are static wildcards" convention the golden test pins. The static-wildcard cost is bounded to one Redis `GET` per platform speech turn (§5.3) | Measured broker or ai-manager CPU cost from `transcript_created` fan-in; the escape hatch and its leak-sweeper requirement are pre-documented in §5.3 |

---

## 4. Architecture overview

The central rev-2 change: **listening and answering are two different
workloads on one AIcall, and they must not share a code path.**

```
                     transcribe-manager
                     transcript_created (per final STT result, platform-wide)
                              │
                              ▼
      ┌───────────────────────────────────────────────────────┐
      │ LAYER 1 — intake (no LLM, no DB write, no webhook)     │
      │  subscribehandler.processEventTMTranscriptCreated      │
      │   • drop if TMDelete != nil  (H3)                      │
      │   • Redis SMEMBERS ai:listen:transcribe:<transcribe_id>│
      │     → empty = not ours, drop; else fan out per AIcall  │
      │   • RPUSH pending + window lists, LTRIM, EXPIRE        │
      └───────────────────────────┬───────────────────────────┘
                                  │ try Redis SET NX EX (debounce lock)
                                  │ not acquired → return, stays buffered
                                  ▼
      ┌───────────────────────────────────────────────────────┐
      │ LAYER 2 — evaluation turn (≤ 1 per interval per AIcall)│
      │  aicallHandler.RunListenTurn                           │
      │   • LPOP-all pending; if empty → skip                  │
      │   • assemble a BOUNDED context explicitly:             │
      │       system prompt snapshot (from AIcall.Metadata)    │
      │     + listen-turn system prompt                        │
      │     + last 10 user/assistant Q&A rows                  │
      │     + rolling transcript window (last 40 lines)        │
      │   • fresh, throwaway pipecatcall id (NOT c.Pipecatcall)│
      │   • PipecatcallStart + TerminateWithDelay              │
      └───────────────────────────┬───────────────────────────┘
                                  │
              ┌───────────────────┴────────────────────┐
              │                                        │
   LLM calls notify_agent                   LLM emits text / nothing
   (RunLLM:false → no follow-up text)                  │
              │                                        ▼
              ▼                          every pipecat message event whose
   ToolHandle → tool rows                PipecatcallID != AIcall.PipecatcallID
   → then one assistant row              is DROPPED (§5.4.4). Nothing is
     with Origin=proactive               persisted, no webhook, no panel row.
   → webhook + panel (intended)
```

Four properties fall out of this shape, and they are the answers to the
review's blocking items:

- **The agent's Q&A path is untouched.** `Send` /
  `SendReferenceTypeOthers` (`pkg/aicallhandler/send.go:16-149`) is never
  invoked by this feature, so its 3-second cooldown
  (`send.go:27-32`, `internal/config/main.go:37`) never rejects a
  transcript, and its `interruptPreviousPipecatcall` +
  `UpdatePipecatcallID` sequence (`send.go:116-122`) never rotates the
  AIcall's pipecatcall out from under an in-flight answer.
- **LLM invocations are decoupled from speech volume.** One turn per
  interval per AIcall, regardless of how many sentences were spoken.
- **Context size is a constant**, assembled from known-bounded inputs,
  not from `getPipecatcallMessages`' newest-100 replay
  (`start.go:620-661`) — which, because the system prompt is itself
  message row #1 (`start.go:812-819`), would have evicted the AI's own
  instructions after ~100 spoken lines.
- **Nothing a listen turn produces reaches the customer's (tenant's) own
  webhook endpoint or the agent's panel** unless the LLM deliberately
  calls `notify_agent` — and **corrected in rev 3**: when it does, or when
  it uses any other Insight tool during a listen turn (§6, still allowed),
  more rows go out than the one intended notification, because
  `ToolHandle` writes a tool-call row and a tool-result row for every
  tool invocation, and both are webhook-published and panel-rendered
  today. So the guarantee is "silence unless the LLM acts," not "exactly
  one row when it does" — see §5.6.4 for what the tenant's webhook
  consumer and the agent's panel actually receive, and the mitigation.

---

## 5. Design

### 5.1 Trigger: an explicit `POST /service_agents/aicalls/{id}/listen` API (rev 15 — replaces the `Start`-hook design; endpoint surface corrected in rev 16, review round 13 finding BLOCKING-1)

**Rev 1-14 hooked listening inside `Start` as an implicit side effect of
creating or reusing the Q&A AIcall.** That design itself required solving
a real problem: `startReferenceTypeContactCase` has **three** success
returns and only two of them transition status, so a hook keyed on the
`initiating → progressing` transition (rev 1's original placement) would
never fire on the *common* path (every panel re-open reuses an
already-active AIcall with no status write at all). Verified in
`pkg/aicallhandler/start.go`:

| Return | Line | Goes through `startContactCaseTurn` (which owns the `UpdateStatus(...Progressing)` at `start.go:542`)? |
|---|---|---|
| Fresh create | `start.go:439` | yes |
| Existing row stuck at `Initiating` | `start.go:509` | yes |
| **Reuse an already-active AIcall** | `start.go:512-513` | **no — returns `existing` with no status write at all** |

Rev 2-14's fix was to hook `Start` itself (the sole caller of
`startReferenceTypeContactCase`, `start.go:168-199,190-191`) rather than
the status transition, covering all three returns with one call.

**Rev 15 removes this hook entirely — the problem above no longer needs
solving, because `Start` no longer triggers listening at all.** The
CEO/CTO's own architectural review of this design (2026-09-04 design
discussion) concluded that bundling "start listening" as a side effect of
"create/reuse the Q&A AIcall" conflates two independent concerns: the
caller has no way to observe whether listening was attempted or
succeeded, no way to trigger one without the other, and — as this
document's own revision history shows (§0, rev 11's confbridge guard,
rev 12/13's fixes to it) — listening has accreted enough of its own
failure modes and observability needs that treating it as a bare side
effect of AIcall creation understates how independent a capability it
actually is. The two capabilities are now independently callable and
independently observable at the API layer, even though today only one
caller (the panel-open flow, §5.10.1) ever exercises the second one right
after the first.

**The fix: a new, dedicated endpoint on the Agent-facing surface, not the
top-level Admin one — corrected in rev 16 after review round 13's
BLOCKING-1.** Rev 15's first draft put this at the top-level
`POST /v1/aicalls/{id}/listen`, mirroring `terminate`
(`pkg/listenhandler/main.go:348-351`). That was wrong: `AIcallTerminate`
(`servicehandler/aicall.go:249-277` — line range corrected in rev 16,
review round 13 finding LOW-2) gates on
`amagent.PermissionCustomerAdmin|PermissionCustomerManager`
(`aicall.go:258`), an **Admin-console** permission tier. The panel's own
existing "Start" call — `ServiceAgentAIcallCreate`
(`pkg/servicehandler/serviceagent_aicall.go`) — is on the **Agent**
surface instead, gated only on `amagent.PermissionAll` via
`h.hasPermission(ctx, a, a.CustomerID, amagent.PermissionAll)`, the same
shape `ServiceAgentTranscribeList` uses (`serviceagent_transcribe.go:27-29`
for the `IsAgent()` check, `:45-48` for the permission check — corrected
in rev 17, review round 14 finding LOW-2; an earlier draft cited `:41-44`,
which is the unrelated `if token == "" { … }` block in between).
`bin-api-manager/docs/auth.md:119`
states this in the imperative, as a rule this design must not violate:
"`square-talk` (and any other Agent-facing frontend) MUST call ONLY
`/service_agents/*` paths — never the top-level `/<resource>` path
directly, even if the top-level path's permission bitmask happens to
allow Agent-level access," and `:124`: "Do NOT 'fix' a missing
Agent-facing capability by relaxing the top-level endpoint's permission
bitmask." Putting `listen` at the top-level path would have meant an
ordinary agent in square-talk (holding neither admin nor manager
permission) gets `ErrPermissionDenied` calling it — **listening would
never start in the feature's actual primary use case** — and relaxing
`terminate`'s own bitmask to fix that is explicitly the wrong move per
`auth.md:124`.

- **Route**: `POST /service_agents/aicalls/{id}/listen`
  (`bin-ai-manager/pkg/listenhandler`), following the existing id-scoped
  action-verb idiom already used on this exact surface —
  `POST /service_agents/contact_addresses/:id/claim`
  (`bin-openapi-manager/openapi/paths/service_agents/contact_addresses/id_claim.yaml`,
  generated handler `PostServiceAgentsContactAddressesIdClaim`,
  `gen.go:23354`) is the closest precedent: verb as the trailing path
  segment, `POST`, no request body, same as every other action route
  cited in this design. New `regV1AIcallsIDListen =
  regexp.MustCompile("/v1/aicalls/" + regUUID + "/listen$")` and a
  `processV1AIcallsIDListenPost` switch case in
  `bin-ai-manager/pkg/listenhandler/main.go` — **the internal ai-manager
  RPC surface itself stays at the plain `/v1/aicalls/{id}/listen` path**;
  it is the *public, api-manager-facing* path that moves to
  `/service_agents/aicalls/{id}/listen`. These are two different
  services' routes with the same trailing segment, not one route
  reachable two ways.
- **RPC client**: `AIV1AIcallListen(ctx, aicallID uuid.UUID) (*amaicall.AIcall, error)`
  in `bin-common-handler/pkg/requesthandler/ai_aicalls.go`, mirroring
  `AIV1AIcallTerminate`'s shape (`ai_aicalls.go:110-127` — corrected in
  rev 16, review round 13 finding LOW-2) — `POST` to
  `/v1/aicalls/<id>/listen` with `ContentTypeNone` (no request body,
  same as `terminate`). **Given a longer explicit timeout than
  `requestTimeoutDefault` (3000ms,
  `bin-common-handler/pkg/requesthandler/main.go:150`) — new in rev 16,
  review round 13 finding MEDIUM-1, corrected in rev 17 (review round 14
  finding MEDIUM-3)**: `ProcessListen` (below) runs up to three
  **sequential**, cross-service RPCs (`TranscribeV1TranscribeGet`,
  `ContactV1CaseGet`, `CallV1CallGet`), each already subject to its own
  `requestTimeoutDefault`. (An earlier draft of this paragraph justified
  the 10s figure by claiming none of the three is cache-first —
  incorrect for `CallV1CallGet`, which *is* cache-first
  `bin-call-manager/pkg/dbhandler/call.go:115-130`; the actual
  `bin-call-manager/pkg/callhandler/db.go:171-185` this design cited only
  shows the handler delegating to `dbhandler`, not bypassing its cache.
  Withdrawn as the stated reason.) The real justification is simpler and
  holds regardless of caching: **each hop can independently take up to
  its own default timeout**, so three sequential RPC hops can add up to
  roughly 3× a single hop's timeout worst-case — comfortably exceeding
  `AIV1AIcallListen`'s own inherited 3s default if it used one, and
  failing the *client's* request even if ai-manager's own precheck later
  succeeds. `AIV1AIcallListen` passes an explicit `10000` (10s) timeout,
  the same pattern
  `TranscribeV1TranscribeStart` already uses for its own `5000`
  (§5.2.2), rather than inheriting the default.
- **Public exposure**: new
  `bin-openapi-manager/openapi/paths/service_agents/aicalls/id_listen.yaml`
  (mirroring `contact_addresses/id_claim.yaml`'s shape: no request body,
  200 returns the existing `AIManagerAIcall`), regenerated into
  `bin-api-manager/gens/openapi_server/gen.go`
  (`router.POST(options.BaseURL+"/service_agents/aicalls/:id/listen",
  wrapper.PostServiceAgentsAicallsIdListen)`, alongside the existing
  `PostServiceAgentsAicalls` registration at `gen.go:23344`), wired
  through a new handler in `bin-api-manager/server/service_agents_aicalls.go`
  → `pkg/servicehandler/serviceagent_aicall.go`'s new
  `ServiceAgentAIcallListen(ctx, a *auth.AuthIdentity, id uuid.UUID) (*amaicall.WebhookMessage, error)`,
  which: checks `a.IsAgent()` (else `ErrAuthenticationRequired`, matching
  `ServiceAgentTranscribeList`'s and `ServiceAgentAIcallList`'s own first
  line); checks `h.hasPermission(ctx, a, a.CustomerID,
  amagent.PermissionAll)` (else `ErrPermissionDenied`); fetches the
  AIcall via the private two-level helper `h.aicallGet(ctx, id)`
  (**corrected in rev 17, review round 14 finding LOW-2**, which found
  the first draft calling `AIV1AIcallGet` directly — `serviceagent_aicall.go:112`
  already wraps this exact fetch, and `bin-api-manager/CLAUDE.md`'s
  two-level handler pattern expects it reused rather than re-inlined),
  **then performs the ownership compare itself** (`tmp.CustomerID !=
  a.CustomerID → ErrPermissionDenied`) — **restated precisely in rev 18,
  review round 15 finding LOW-5**: `aicallGet` only fetches; the sibling
  `ServiceAgentAIcallGet` (`serviceagent_aicall.go:117-120`) does the
  compare itself, in the public method, not inside the helper, and
  `ServiceAgentAIcallListen` follows that same division of labour, not
  (as an earlier draft implied) a compare bundled into `aicallGet` —
  then calls `h.reqHandler.AIV1AIcallListen(ctx, id)` and
  `.ConvertWebhookMessage()`.
  **A top-level Admin-console
  `/v1/aicalls/{id}/listen` public route is deliberately not added** —
  nothing in this design's scope needs an admin-console caller for this
  action, and adding an unused surface is scope creep (YAGNI); add one on
  its own ticket if a genuine admin-console use case appears.
- **Caller**: square-admin and square-talk call this once, automatically,
  when the Case Insight Assistant panel opens — right after the existing
  `Start` call that creates/reuses the Q&A AIcall (§5.10.1), which is
  itself already on this same `/service_agents/*` surface
  (`ServiceAgentAIcallCreate`). This is the **same trigger timing** as
  rev 1-14's hook: nothing about §5.1.1 step 7's confbridge-readiness
  guard (which exists precisely because this can fire as early as ring
  time) changes. Only the entry point moves from an implicit `Start` side
  effect to an explicit second call.

**Response shape and timing.** The endpoint returns the current
`*amaicall.AIcall` immediately, having run only `ProcessListen`'s fast
synchronous prechecks (§5.1.1 steps 1-6) — it does **not** block for step
7's up-to-30s confbridge-readiness wait (`aicall_listen_confbridge_ready_max_wait_seconds`,
§5.12 — distinct from the goroutine's own 45s outer timeout budget,
**conflated here before review round 16 finding LOW-11**). Blocking an
HTTP request even for that shorter window would be a bad pattern on its
own merits, and the API's stated purpose
here (separation of concerns, §5.1's rewrite above) does not require
synchronous status visibility — that remains a deliberate scope cut
(§3), not an oversight; see §11 item 14 if that changes. If the fast
prechecks pass, the confbridge-wait-and-start stage continues in the same
detached, best-effort goroutine as before; if they fail (flag off,
non-Insight AI, no live Case-linked call, etc.), nothing further runs and
the response simply reflects the AIcall's unchanged state. The caller has
no way to observe from this response alone whether listening ultimately
started.

**Interface shape — a single exported method, not a two-function split —
corrected in rev 16 after review round 13's HIGH-1/HIGH-2/MEDIUM-4.**
Rev 15's first draft split `ensureListen(ctx, a, c)` into
`EnsureListenPrecheck(ctx, c) (proceed bool, err error)` (steps 1-6) and
a separately-named `ensureListenAsync(c)` (steps 7-8), on the theory that
the new HTTP handler only starts with an AIcall id, not an
already-resolved `*ai.AI`. Two defects followed: (a) steps 4-6 resolve
`kase`, `callID`, and `call` — locals steps 7-8 also need (step 7 polls
`call.ConfbridgeID`; §5.2.2 passes `call.ActiveflowID` and `callID` to
`TranscribeV1TranscribeStart`) — and a two-function split with only a
bare `bool` crossing the boundary discards all of them, forcing
`ensureListenAsync` to silently re-fetch the Case and the call before it
could even start step 7 (a duplicate RPC pair and a re-derived tenant
boundary, contradicting §5.1.1 step 4's own description of that check as
"the tenant boundary for the whole feature"); (b) `ensureListenAsync`
was specified lowercase — unexported, unreachable from `pkg/listenhandler`
and unmockable on the `AIcallHandler` interface
(`pkg/aicallhandler/main.go:31-71`, where every method is exported, as
Go requires for cross-package interface satisfaction) — making §7's own
test items for it unimplementable as written. The fix collapses both
into **one exported method**, matching `terminate`'s own one-call shape
exactly (`ProcessTerminate`, called once by
`processV1AIcallsIDTerminatePost` — `v1_aicalls.go:191-227`):

```go
// ProcessListen is the sole entry point for listen's trigger, called
// once by processV1AIcallsIDListenPost — the same one-call shape as
// ProcessTerminate. It resolves the AIcall and AI record, runs §5.1.1
// steps 1-6 inline (synchronously — never anything slower than the
// three RPCs the caller's own longer timeout above already budgets
// for), and — only if every step passes — spawns a detached goroutine
// for steps 7-8, closing over the already-resolved a/c/kase/callID/call
// values directly. No value crosses a function boundary by itself, so
// there is nothing to re-fetch and nothing to silently lose.
func (h *aicallHandler) ProcessListen(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error) {
    c, err := h.Get(ctx, id) // cache-first, same as every other single-AIcall route
    if err != nil {
        return nil, err
    }

    a, kase, callID, call, proceed, err := h.checkListenEligible(ctx, c) // §5.1.1 steps 1-6, inline
    if err != nil {
        return nil, err
    }
    if proceed {
        go func() {
            ctx, cancel := context.WithTimeout(context.Background(), listenEnsureGoroutineTimeout)
            defer cancel()
            h.runListenStart(ctx, a, c, kase, callID, call) // §5.1.1 steps 7-8
        }()
    }
    return c, nil // unchanged by steps 1-6 themselves; steps 7-8 write asynchronously
}
```

`processV1AIcallsIDListenPost` itself is thin: parse `id`, call
`h.aicallHandler.ProcessListen(ctx, id)`, marshal the result, return 200
— matching `processV1AIcallsIDTerminatePost`'s shape exactly (one
business-handler call, no orchestration logic in `listenhandler`,
resolving review round 13's MEDIUM-4).

`Start`'s own three success-return paths (`start.go:439/509/512-513` —
the historical problem this section used to be about) need no fix at
all under rev 15/16: `Start` calls neither `ProcessListen` nor any of its
internal helpers any more. The new listenhandler route is the only
caller.

#### 5.1.1 `ProcessListen` — `checkListenEligible` (steps 1-6) and `runListenStart` (steps 7-8)

**One exported entry point, `ProcessListen`, since rev 16 (§5.1, review
round 13 findings HIGH-1/HIGH-2/MEDIUM-4) — steps 1-6 run synchronously
before the HTTP response, steps 7-8 run in a detached goroutine closing
directly over steps 1-6's own already-resolved values, same as every
prior revision's underlying two-stage shape.** The step numbering and
the logic within each step are otherwise unchanged from rev 1-14; only
which function owns which steps, and who calls them, changed. (Rev 15's
first draft split this into two separately-callable functions,
`EnsureListenPrecheck`/`ensureListenAsync`, connected only by a bare
`bool` — losing `kase`/`callID`/`call` across that boundary and forcing
steps 7-8 to silently re-fetch them, plus specifying `ensureListenAsync`
unexported and therefore unreachable from `pkg/listenhandler`. Rev 16
collapses both into `ProcessListen`, per §5.1's own snippet.)

`ProcessListen(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)`
fetches `c` (cache-first), then calls `checkListenEligible(ctx, c)`
(§5.1's own snippet — **signature corrected in rev 17, review round 14
finding LOW-1**, which found this prose and §5.1's code disagreeing on
whether `a` is resolved by `ProcessListen` first or by
`checkListenEligible` itself; `checkListenEligible` owns it, matching
step 2's own resolution of `a`, and returns it alongside `kase`/`callID`/
`call`) for steps 1-6 below — the caller only has `c`'s id from the URL
path, not an already-resolved `*ai.AI` the way `Start`'s old hook had.
Every step is a cache-first read or a single RPC — never the
confbridge-readiness wait — which is what makes running all six inline
before the HTTP response correct, not just convenient (§5.1's explicit
longer RPC timeout is the budget for this).

If `checkListenEligible` returns `proceed=true`, `ProcessListen` spawns a
detached goroutine — with its own `context.Background()`, following the
same detached-goroutine shape already used at `tool.go:191-199` (which
itself runs unbounded, with no timeout) and `start.go:97-100` (a 5s
timeout, but for an unrelated best-effort member-AI fetch — not a
precedent this design's own timeout value is drawn from; **corrected in
rev 13, review round 11 finding LOW-3**: this goroutine's own timeout is
step 7's purpose-built `AIcallListenEnsureGoroutineTimeoutSeconds`,
§5.12) — running `runListenStart(ctx, a, c, kase, callID, call)` for
steps 7-8, with every value it needs passed directly from
`checkListenEligible`'s own return values, not re-derived. It is
fire-and-forget by design, as before; §6 covers the failure modes.

**Steps 1-6 (`checkListenEligible`):**

1. **Feature gate.** `config.Get().AIcallListenEnabled` false → return.
2. **AIcall gate.** `a.Type != ai.TypeInsight` → return. (`contact_case`
   AIcalls are Insight in practice, but this is deny-by-default.) **Also
   requires `c.Status == aicall.StatusProgressing && c.TMDelete == nil`
   — new in rev 16, review round 13 finding MEDIUM-2.** Rev 1-14 never
   needed this: `Start`'s own hook only ever ran against an AIcall it had
   just created or reused as active. Rev 15's public, arbitrarily-callable
   `POST /service_agents/aicalls/{id}/listen` removed that guarantee —
   any caller can `POST` against any AIcall id it owns, including one
   already `terminated`/deleted — so the type gate is extended into a
   combined AIcall-liveness gate rather than left to rely on the implicit
   "this was just created" assumption that no longer holds. Without this,
   a terminated AIcall could still pass steps 3-6, spawn step 7's 45s
   goroutine, and start a billed STT session that only §5.4.1's own
   `c.Status == progressing` require-list check (unrelated to this one)
   would eventually reap on the first transcript segment — later and
   less directly than catching it here.
3. **Idempotency.** If `c.Metadata[listen_transcribe_id]` is set and
   `TranscribeV1TranscribeGet(thatID)` reports
   `Status == progressing && TMDelete == nil && ReferenceID == <the call
   we are about to resolve>` → already listening, return. This is what
   makes repeated panel opens free.
4. **Case lookup.** `ContactV1CaseGet(c.CustomerID, c.ReferenceID)` — the
   same customer-scoped RPC `toolHandleGetContactInteractions` already
   uses (`tool_insight.go:125`), followed by the same
   `kase.CustomerID != c.CustomerID || kase.CustomerID == uuid.Nil`
   recheck (`tool_insight.go:135-138`). **This is the tenant boundary for
   the whole feature** — see §5.3 for why it cannot be re-checked at
   event time.
5. **Reference typing.** `Case.ReferenceType` and `Case.ReferenceID` are
   plain `string` (`bin-contact-manager/models/kase/kase.go:65-66`), and
   there is no `ReferenceTypeCall` constant anywhere in the repo — call
   sites use the bare literal `"call"`. This design adds one:

   ```go
   // bin-contact-manager/models/kase/kase.go
   // ReferenceTypeCall is the stored ReferenceType value for a Case
   // created from a call. The field is a plain string (it mirrors
   // contact_interactions.reference_type's existing vocabulary), so this
   // is an untyped string constant rather than a typed enum member.
   const ReferenceTypeCall = "call"
   ```

   The checks then typecheck:
   ```go
   if kase.ReferenceType != kmkase.ReferenceTypeCall { return }
   callID, errParse := uuid.FromString(kase.ReferenceID)
   if errParse != nil || callID == uuid.Nil { return }
   ```
   (Rev 1's `kase.ReferenceID == uuid.Nil` did not compile against a
   `string` field. The design doc it cited also lives at
   `bin-contact-manager/docs/plans/2026-07-24-case-reference-id-design.md`,
   not under `monorepo/docs/plans/`.)
6. **Call liveness + ownership.** `CallV1CallGet(callID)`; require
   `call.CustomerID == c.CustomerID` (defence in depth, same shape as
   `tool_insight.go:854` — line renumbered by the `paginateUntilExact`
   refactor (`NOJIRA-Extract-call-transcript-pagination-helper`, merged to
   `main` 2026-09-04); logic unchanged), `call.TMDelete == nil`, and
   `call.Status ∈
   {dialing, ringing, progressing}` — the exact set
   `transcribehandler.isValidReference` treats as transcribable
   (`bin-transcribe-manager/pkg/transcribehandler/start.go:160-163`, drift
   from `107-115` fixed in rev 14/§10.10, off-by-two corrected in rev
   16/§10.11 review round 13 finding LOW-2 — `NOJIRA-Allow-caller-specified-transcribe-id`
   shifted this file by ~46-50 lines after rev 10's sweep, scoped to
   `bin-ai-manager`, missed it).
   Anything else → the call is over; return (the agent can still use
   `get_call_transcript` on the finished call, unchanged).

**Steps 7-8 (`runListenStart`, detached goroutine, called with `a`, `c`,
`kase`, `callID`, `call` already resolved by `checkListenEligible` —
rev 16):**

7. **Confbridge participant-count guard, with bounded retry (rev 11;
   revised after review round 9 found the first version fails closed on
   the normal path — see below).**

   **Why the A-leg is the right call.** `Case.ReferenceID` (from step 4)
   resolves to the **A-leg (customer) call**, not the agent's B-leg. The
   correct claim is narrower than "structural for all of `case_create`" —
   it is: **no *system-generated* leg flow can carry `case_create` or
   `ai_talk`.** Two such generators were checked, not one:
   `bin-queue-manager/pkg/queuecallhandler/execute.go:generateFlowForAgentCall`
   (`74-97`) builds the agent's B-leg flow from a single hardcoded
   `confbridge_join` action, and `actionHandleConnect`'s own B-leg flow
   (`bin-flow-manager/pkg/activeflowhandler/actionhandle.go:495-514`) is
   likewise hardcoded to `confbridge_join` + `hangup`. Neither can ever
   contain `case_create`. This is **not** a categorical guarantee across
   the whole platform: `actionHandleCall` can chain a **customer-authored**
   flow onto a new leg (`actionhandle.go:899-937`, `masterCallID =
   af.ReferenceID` at `~932-934` when `opt.Chained`), and
   `bin-agent-manager`'s agent-dial RPC similarly accepts a caller-supplied
   `FlowID` on a leg carrying `MasterCallID`
   (`bin-agent-manager/pkg/listenhandler/models/request/agents.go:87-91`).
   Either of those *could* in principle place `case_create` on a B-leg.
   This design's guarantee is therefore scoped to the standard
   queue-routing flow (`generateFlowForAgentCall`) and the `connect`
   action, not to every conceivable flow shape — `ai_talk` is dropped from
   this claim entirely, since it only ever targets `ReferenceTypeCall`/
   `ReferenceTypeConversation` AIcalls (`actionhandle.go:1083-1086`), never
   `contact_case`, and was never actually load-bearing here.

   A stronger and simpler guarantee covers the gap: `actionHandleCaseCreate`
   itself only creates a Case when the call's peer is CRM-eligible —
   `crmIneligiblePeerTypes` excludes `agent`/`extension`/`sip`/`conference`/
   `ai`/`ai_team`/`none` (`actionhandle.go:1259-1266`), checked via
   `isCRMEligiblePeer` (`:1272-1275`) at `:1354`, against the peer resolved
   by `deriveEndpointsForCase` (`:1287-1300`: incoming → peer=source,
   outgoing → peer=dest — i.e. always the far-end party on *that call's own
   channel*). So the real invariant this design relies on is **`in` = the
   listened channel's own remote party = `Case.Peer`, which `case_create`
   already guarantees is an external contact**, not merely "A-leg." See
   §5.9 for how this reframes the click-to-call residual risk.

   **What bridging does and does not change.** Bridging the two legs does
   not move either leg's original channel: each keeps its own local
   Asterisk bridge and joins the shared `Confbridge` through an auxiliary
   join channel (`bin-call-manager/pkg/confbridgehandler/join.go:Join`,
   `~21-90`; bridge creation `~106-137`; `StartChannelWithBaseChannel`
   dialing `PJSIP/conf-join/...`, `~85-88`/`~150-163`). §5.9's
   Snoop/ExternalMedia tap attaches to the call's own primary channel
   regardless (`bin-call-manager/pkg/externalmediahandler/start.go:startReferenceTypeCall`,
   `~60-90`; `channelhandler/start.go:StartSnoop`, `~14-42`), so this
   2-stage join-channel mechanism does not change §5.9's channel-relative
   `in`/`out` reading — only who ends up on the other end of the channel.
   That "who's on the other end" question is exactly what this guard
   checks: the clean `in=customer`/`out=agent` reading assumes precisely
   one other party.

   **The check, and the bounded retry review round 9 found missing —
   revised again after review round 10 found the first retry design's
   give-up rule unsound (see below).**
   `Joined()` (`bin-call-manager/pkg/confbridgehandler/joined.go:33-44`)
   makes two separate writes for *whichever* call joins: `AddChannelCallID`
   (`:33`) extends `len(ChannelCallIDs)` on the confbridge side, and the
   sibling `CallV1CallUpdateConfbridgeID` RPC (`:40`) sets that call's own
   `Call.ConfbridgeID` (**named explicitly here per review round 11
   finding LOW-5**, which is the write that actually sets the field, not
   `AddChannelCallID`) — so the A-leg's own `ConfbridgeID` is already
   non-nil from its own join, at `execute.go:66`'s forward, well before
   the agent answers. What actually stays at 1 through the whole
   agent-ring window is `len(ChannelCallIDs)`, not `ConfbridgeID`'s
   nil-ness — round 9's original diagnosis conflated the two, which round
   10 caught as a self-contradiction (the very next sentence already
   describes the A-leg "sitting alone" post-forward, which requires a
   non-nil `ConfbridgeID`). In the standard queue-routing flow, `len`
   reaches 2 only once the B-leg's own join channel completes
   `ChannelEnteredBridge`
   (`bin-queue-manager/pkg/queuecallhandler/service.go:124-151` — the
   A-leg's queue-wait actions are `fetch_flow`/`empty` only;
   `execute.go:48,66` create the B-leg and forward the A-leg's activeflow
   essentially at the same time, before the agent has answered). A
   one-shot check at `ProcessListen`/`runListenStart` time (rev 16
   naming) — which can run as early as panel-open, and a screen-pop UI
   opening the Case panel at ring time is entirely plausible — would
   therefore see `len(ChannelCallIDs) == 1` on a perfectly normal call
   and silently never listen, with no retry and nothing recorded to
   explain why. That is a reliability regression this guard must not
   introduce.

   So the check retries, bounded, inside `runListenStart`'s own
   goroutine — which this feature gives an **explicit**
   `AIcallListenEnsureGoroutineTimeoutSeconds` (new config, default `45`;
   review round 10 found the intro's cited precedents, `tool.go:191-199`'s
   unbounded `context.Background()` and `start.go:97-100`'s unrelated 5s
   fetch timeout, do not actually bound anything for this path, so this
   feature does not inherit either — it sets its own) rather than firing
   once: poll every `AIcallListenConfbridgeReadyPollIntervalSeconds` (new
   config, default `2`) for up to
   `AIcallListenConfbridgeReadyMaxWaitSeconds` (new config, default `30` —
   strictly less than the goroutine timeout above, leaving margin for the
   RPC calls themselves), re-running step 6's call-liveness check and this
   confbridge check together each poll — a call that goes from
   `dialing`/`ringing` to terminated during the wait exits the loop via
   step 6, not this one. On each poll:
   - `call.ConfbridgeID == uuid.Nil`, or `CallV1ConfbridgeGet` returns a
     confbridge with `TMDelete != nil` or `Status != progressing`
     (`bin-call-manager/models/confbridge/main.go:~35-37,~64-68` —
     checked in addition to the participant count, since
     `confbridgehandler/leaved.go:~43-48` resets `Call.ConfbridgeID` to
     `uuid.Nil` from a goroutine that can outlive its own request context,
     making a stale non-nil id reachable in principle even though today's
     code paths happen to avoid it) → **not ready yet**, keep polling.
   - `len(confbridge.ChannelCallIDs) == 2` and the confbridge is live →
     proceed to §5.2.
   - `len(confbridge.ChannelCallIDs) != 2` — **keep polling regardless of
     the count or the call's own status, full stop (revised after review
     round 10 finding HIGH-A).** The first version of this retry tried to
     fast-fail on `len >= 3` once `call.Status == progressing`, reasoning
     that a `progressing` call was "past" the pre-answer window where an
     extra party could still be transient noise. That reasoning does not
     hold: `call.Status` reflects only the *listened* leg's own answer
     state, and in this design's own target flow the A-leg is already
     `progressing` from before `case_create` even runs, for the entire
     subsequent queue-wait and agent-ring window — so the fast-fail
     condition was, in practice, always true the instant a 3rd party
     appeared, not just once one had lingered. A concrete, documented
     platform pattern hits exactly this: a `connect` action with
     `early_media: true` and multiple destinations
     (`bin-api-manager/docsdev/source/flow_advanced_patterns.rst:1317-1338`)
     runs each ringing destination's `confbridge_join` **before answer**
     (`actionhandle.go:opt.EarlyMedia`; `bin-call-manager/pkg/callhandler/status.go:35-38`
     fires `ActionNext` on `ringing` for an early-execution call), so the
     confbridge can transiently hold the A-leg plus several ringing B-legs
     — `confbridgehandler/joined.go:87-97` explicitly iterates members
     looking for a `dialing`/`ringing` outgoing call, confirming this
     state is expected — before settling to 2 within seconds as the
     losing legs hang up. The original fast-fail would have given up on
     that call permanently, on the very first (and, per BLOCKING-1's
     screen-pop scenario, possibly only) `ProcessListen` invocation — the
     same failure class round 9 blocked on, reintroduced by its own fix.
     There is no cheap, sound way to distinguish "stably wrong" from
     "transiently 3+ while settling" mid-wait, and the loop is already
     bounded, so it does not try: any non-2 count just keeps polling until
     the budget runs out.
   - The wait budget elapses (whether the count was stuck at 1, stuck at
     3+, or oscillating — this design does not distinguish those cases),
     `CallV1ConfbridgeGet` errors, or step 6's own liveness check fails
     during a poll → stop, do not listen. This is still a silent no-op
     from the *AIcall*'s perspective (§6), but it is **not silent
     operationally**: each distinct outcome gets its own
     `aicall_listen_start_total` label (§5.13) —
     `skipped_confbridge_not_ready` for the max-wait timeout with the last
     observed count logged (not label-cardinality-bearing, but present in
     the log line for diagnosis), `skipped_confbridge_error` for the RPC
     failure — so the false-negative rate this retry exists to bound stays
     visible in production rather than merely bounded on paper. §7 gains
     explicit coverage for each new branch, including a 3→2 settle within
     the wait budget.

   **What this guard does not do, stated plainly rather than assumed.**
   It is enforced only inside `checkListenEligible`/`runListenStart`,
   before listening starts —
   there is no ongoing re-check once a listen session is live. A 3rd party
   joining *after* listening has started (barge-in, an attended-transfer
   leg — `bin-transfer-manager/pkg/transferhandler/attended.go:139` — a
   `conference`-type join) is not caught by this guard and is documented
   as open residual risk in §5.9/§11, not mitigated here. Note also that a
   3rd party degrades only the `out` side of the mapping — `in` (the
   listened channel's own party) stays correct regardless of who else is
   in the bridge, since it never depended on the other parties' identities
   in the first place (see the `in == Case.Peer` framing above).

   **One more consequence of the retry, noted explicitly per review round
   11 finding LOW-6, corrected in rev 18 (review round 15 finding
   LOW-2):** the idempotency check (step 3) only short-circuits
   once `listen_transcribe_id` is set, which never happens while step 7 is
   still polling — so if the agent re-opens the Case panel several times
   during a long ring, each open spawns its *own* concurrent, independently
   bounded retry loop. This is not a new failure mode: it is bounded (each
   loop still respects the same wait budget and goroutine timeout),
   detached (§5.1.1 intro), and the resulting race to actually start the
   transcribe session once the confbridge is ready is now serialized by
   §5.2.2's per-AIcall lock (an earlier draft of this paragraph said the
   race was "already covered by §5.2.2's reuse rule and §6's
   `TRANSCRIBE_ALREADY_PROGRESSING` row" alone — the exact unexamined
   premise review round 14's HIGH-2 found insufficient once rev 16 moved
   the write before creation) — but it does mean
   `skipped_confbridge_not_ready`'s raw rate can be inflated by repeated
   re-opens of the same still-ringing call, not just by distinct calls,
   which matters when interpreting that metric per §5.13's guidance.
8. Proceed to §5.2.

No new event subscription is needed for the trigger — it is a one-shot
check triggered by the `POST /v1/aicalls/{id}/listen` call (§5.1, rev 15
— previously tied to AIcall-start time under rev 1-14's `Start`-hook
design), not a standing watch for "some future call on this contact."

### 5.2 The transcribe session: owner, start, and reuse

#### 5.2.1 Owner — a listen-only system customer id, distinct from `IDAIManager` (resolves review-round-1 B6, revised in rev 4 to also resolve review-round-3 B3)

Rev 1 left this unresolved. Rev 2/3 decided `IDAIManager` (the same
system id `summaryhandler.startReferenceTypeCall` uses,
`bin-ai-manager/pkg/summaryhandler/start.go:84-99`) and then, when review
round 2 found that this makes listen collide with a concurrent AI
summary's transcribe session, tried to fix the collision by making the
two features share and hand off ownership of one session (§5.2.2a in rev
3). **Review round 3 (finding B3) showed that hand-off design is itself
broken two ways**: `summaryhandler.contentGetTranscripts`
(`pkg/summaryhandler/content.go:141-172`) reads with `size=1` and no
transcribe-id pin, so it can silently pick up listen's session instead of
its own; and §5.7.2's `owns=true` stop path can tear down a transcribe
that a *later-arriving* summary is now also depending on, cutting a paid
summary's STT off mid-call while the call is still live.

**Decision, revised in rev 4: give listen its own system customer id,
separate from `IDAIManager`.** `cmcustomer.IDAIManagerListen` (exact
value TBD at implementation time — a new constant alongside `IDAIManager`
in `bin-customer-manager/models/customer/customer.go`). This makes
listen's and summary's transcribe sessions **provably independent** at
the `startLive` dedup-guard layer (`transcribehandler/start.go:248-266`,
drift fixed in rev 14/§10.10 — see §5.1.1 step 6's citation note,
scoped by `customer_id`), because they are never the same owner — no
hand-off logic, no shared lifecycle, no read-path ambiguity, and §5.2.2a
is **deleted, not fixed**: `summaryhandler.startReferenceTypeCall` needs
no change at all.

| | `IDAIManagerListen` (chosen, rev 4) | shared `IDAIManager` (rev 2/3, reverted) | tenant `customer_id` (rejected) |
|---|---|---|---|
| Customer's transcribe list | clean | clean | shows a session they never started |
| Billing | platform-borne | platform-borne | silent surprise transcription charge |
| Collision with the customer's own live transcribe | none (dedup guard scoped by `customer_id`) | none | a same-language customer session 409s our `Start` |
| Collision with `ai_summary`'s transcribe | **none — different owner, guard never fires** | real, and its fix (§5.2.2a) introduced B3 | n/a |
| Cost when listen and a summary are both live on the same call | up to 2 separate transcribe sessions (up to 4 STT streams, §5.11) instead of 1 shared session — accepted, bounded, platform-borne | 1 shared session, but with B3's correctness risk | n/a |
| Precedent in-repo | new sentinel id, same *pattern* as `IDAIManager`'s existing precedent | yes (summary) | none |

**Implementation-time confirmation needed:** whether a new system
`customer_id` sentinel requires an actual row in `bin-customer-manager`'s
customer table (FK-backed) or is usable as a bare UUID constant the way
`IDAIManager` apparently already is (per round 1/2's verified read-only
usage) — flagged in §11 rather than assumed here.

**Deliberately not added, stated rather than silently decided (review
round 4, finding M1):** `bin-customer-manager`'s hardcoded
"known-system-id" whitelist (`models/customer/ids.go` and its inline
duplicate in `bin-call-manager/pkg/callhandler/validate.go`) is used to
gate certain call-origination paths — not anything on listen's transcribe
start/list/stop path, so `IDAIManagerListen` not being in that whitelist
does **not** block this feature (verified: listening never calls into
whatever checks that list). It is left out deliberately, not by
oversight: adding a new sentinel to a validation whitelist it was never
designed to need would be an unrelated, unrequested change to
`bin-customer-manager`'s and `bin-call-manager`'s call-path validation —
exactly the kind of scope creep root `CLAUDE.md`'s "smallest change that
works" principle argues against here. Revisit only if a future feature
needs `IDAIManagerListen` to pass that specific gate.

**Consequence, stated explicitly:** the event-time tenant check rev 1
proposed (`AIcall.CustomerID == transcript's CustomerID`) is impossible —
it would *always* fail, because the transcript's `CustomerID` is the
system id (`IDAIManagerListen`), never a tenant id. The tenant boundary is
therefore enforced **once, at listen-start time** (§5.1 steps 4 and 6:
customer-scoped `CaseGet` + `CustomerID` recheck on both the Case and the
call), and the event path instead verifies *provenance*: "is this
`transcribe_id` one we ourselves started and recorded?" (§5.3). That is a
stronger property than a field comparison — the id is one ai-manager
generated and persisted, not attacker-influenceable.

**Second consequence:** `get_call_transcript`'s own listing is filtered by
`tmtranscribe.FieldCustomerID: c.CustomerID` (`tool_insight.go:869` — line
renumbered by the `paginateUntilExact` refactor, see above; logic
unchanged), so it will not see the listen session. That is correct and
intended: the agent reads *finished* transcripts of *the customer's own*
sessions through that tool; the live listen session is an internal,
platform-owned stream. Nothing regresses. **Confirmed directly against
the current code (2026-09-04)**: the exclusion is stated generically in
the surrounding comment as "exclude any row whose `CustomerID !=
c.CustomerID`," not hardcoded to `IDAIManager` specifically — the
comment names exactly this scenario, a future system-initiated
transcriber, as already anticipated. `IDAIManagerListen` needs no change
to `get_call_transcript` to stay correctly excluded.

#### 5.2.2 Reuse rule — listen-to-listen only, language-tolerant

`startLive`'s duplicate guard is scoped `(customer_id, reference_id,
language, status=progressing, deleted=false)`
(`transcribehandler/start.go:248-266`). Under §5.2.1's rev-4 decision
(listen's own `IDAIManagerListen`, never shared with `ai_summary`'s
`IDAIManager`), the **only** sessions listening can ever collide with are
**other listen sessions on the same call** — the two-Cases-one-call case
(§5.11) — never a concurrent `ai_summary`.

```go
// dupFilters — bound once here, referenced by name from both
// TranscribeV1TranscribeList calls in the lock sequence below. Shown as
// a standalone map, not a full call, deliberately: both actual call
// sites (below) own their own error handling, and round 17 (finding
// B-2) found an earlier draft of this illustrative block that combined
// the two — a `List` call with a declared, never-checked `err` — read as
// exactly the dropped-error bug round 15's LOW-4 had just fixed a few
// paragraphs later.
dupFilters := map[tmtranscribe.Field]any{ // keyed by the typed Field, not
    tmtranscribe.FieldCustomerID:  cmcustomer.IDAIManagerListen, //   a bare
    tmtranscribe.FieldReferenceID: callID,                       //   string —
    tmtranscribe.FieldStatus:      tmtranscribe.StatusProgressing, // matching
    tmtranscribe.FieldDeleted:     false,                        //   TranscribeV1TranscribeList's
                                                                  //   actual `filters map[tmtranscribe.Field]any`
                                                                  //   parameter (review round 18 finding
                                                                  //   MEDIUM-2 — an earlier draft used
                                                                  //   map[string]any, which would not
                                                                  //   compile against that signature)
}
```

- **Any** progressing `IDAIManagerListen` session on this call is reused,
  *regardless of its language*, and `Metadata[listen_owns_transcribe] =
  false`.
  Rationale: starting a second session only because the language string
  differs would double the STT cost on one call to gain nothing — the LLM
  reads whatever language comes out. Maximising reuse is the cheaper and
  simpler rule. This is the explicit answer to review round 2's "the
  reuse rule must account for language/owner."
- Otherwise start one, with `Metadata[listen_owns_transcribe] = true`.
  **Revised in rev 15, again in rev 16 (review round 13 finding HIGH-3),
  and once more in rev 17 (review round 14 findings HIGH-1/HIGH-2)** —
  each round found the previous fix closed one race while opening
  another, which is why this whole bullet is now guarded by an explicit
  per-AIcall lock rather than relying on write ordering alone.

  **The three problems, in the order they were found, and what each fix
  actually closes:**
  1. *(rev 15's problem, still the root motivation.)* Registering this
     AIcall in the Redis resolver set (§5.2.4) used to happen *after*
     `TranscribeV1TranscribeStart` succeeded, leaving a short window in
     which the freshly-created transcribe could already be emitting
     `transcript_created` events for lines nobody had registered to
     receive yet — silently dropped as `dropped_unknown` (§6).
  2. *(rev 16's problem, per review round 13 HIGH-3.)* Pre-registering
     only the Redis `SADD` (rev 15's fix) left §5.2.4's DB write
     (`listen_call_id`, `metadata[listen_transcribe_id]`) *after*
     creation — so an early event could resolve through the
     now-registered set, fail `RunListenTurn`'s precondition (§5.4.1 step
     1, which requires the metadata **set**, not just the Redis
     membership present), and trigger `stopListening`/`clearListenState`,
     deleting the state the fix had just created. Rev 16 moved the DB
     write earlier too, so both land together, before creation.
  3. *(rev 17's problem, per review round 14 HIGH-1/HIGH-2.)* Rev 16's
     fix pre-writes speculatively **per goroutine**, using a freshly
     generated id each time. §5.1.1 step 7's own retry (and its LOW-6
     note, §11 item 13) already establishes that the *same* AIcall can
     have multiple concurrent `runListenStart` goroutines in flight from
     repeated panel re-opens — and two of them can now both pass the
     `TranscribeV1TranscribeList` check (below) before either finishes
     writing, each minting its **own** `newTranscribeID` and pre-writing
     against it. Depending on interleaving this either (a) has the
     second goroutine's write `SREM` the first goroutine's *already-live*
     session out of the resolver set (§5.2.4's rev-4 stale-id logic,
     applied to a session that was never actually stale), silently
     dropping events for a live session, or (b) has a later rollback
     (e.g. the second goroutine's own `TranscribeV1TranscribeStart`
     failing) delete DB/Redis state that in fact belongs to the first
     goroutine's live, billed session.

  Problem 3 is not fixable by write-ordering alone — it needs mutual
  exclusion. So the whole reuse-check-through-write sequence for a given
  AIcall (both the reuse-check bullet above — **not**, since rev 20
  (review round 17 finding B-2), shown as its own illustrative `List`
  call; this mention corrected to match in rev 23, review round 20
  finding LOW-1 — and everything below) is now
  wrapped in a Redis lock. **Revised again in rev 18 (review round 15
  finding HIGH-1): rev 17's first version of this lock had two defects
  of its own — an undersized TTL and an unconditional, ownerless
  release — each of which could reopen problem 3 rather than close it.**

  **TTL sizing (rev 18).** Rev 17 sized the TTL (`15`s) against only
  `TranscribeV1TranscribeStart`'s explicit 5000ms timeout, but the lock
  also wraps up to two `TranscribeV1TranscribeList` calls (the reuse
  check below, and the conflict-recovery re-run further down) — and that
  RPC client hardcodes its own **30000ms** timeout, not caller-adjustable
  (`bin-common-handler/pkg/requesthandler/transcribe_transcribes.go:48`;
  changing it would touch every existing caller across the monorepo,
  out of scope here). Summed naively that is up to 65s, comfortably
  exceeding a 15s TTL. But every one of these calls runs under the same
  `ctx` this goroutine derived from
  `AIcallListenEnsureGoroutineTimeoutSeconds` (§5.1.1) — so no call can
  actually run longer than whatever budget remains on that outer
  context, regardless of what its own internal timeout constant claims.
  A goroutine can therefore never *legitimately* hold this lock longer
  than its own outer timeout allows. The correct TTL is not "sum the
  RPC timeouts" but "exceed the outer goroutine timeout, so the lock can
  never expire while a goroutine that is still within its own legitimate
  budget holds it": `aicall_listen_start_lock_ttl_seconds` default
  changes from `15` to `60` (§5.12) — strictly greater than
  `aicall_listen_ensure_goroutine_timeout_seconds`'s own default `45`.

  **This margin is deliberately about legitimate work, not about the
  release itself — corrected in rev 19, review round 16 finding
  MEDIUM-2/MEDIUM-3.** An earlier draft of this paragraph justified the
  `60`-vs-`45` gap partly as "margin for the deferred release itself to
  run," which does not hold: the release below is a `defer`, and if the
  goroutine reaches its own outer timeout, `ctx` is already cancelled at
  that point, so a release call still keyed off `ctx` would fail
  immediately rather than get any extra time from the gap. The gap exists
  for one reason only — so the TTL cannot lapse under a goroutine that is
  still legitimately working — and the release is instead made
  independent of `ctx`'s own cancellation (below), which is the actual
  fix for the case this paragraph used to (incorrectly) credit to the
  TTL margin. **The accepted residual**, stated plainly rather than
  papered over: a goroutine that genuinely crashes (pod loss, process
  kill — anywhere the `defer` itself never runs, as opposed to `ctx`
  merely expiring) still strands the lock for the full TTL, since nothing
  is left to release it early — **and, more narrowly (review round 18
  finding LOW-2), so does an ambiguous `ListenStartLockAcquire` error
  (below) whose own best-effort release also fails**: that second
  failure collapses into this same residual rather than getting its own
  handling, since by that point there is nothing left this goroutine can
  do differently from a genuine crash. §7 item 2 covers both as the
  expected, accepted behaviour rather than a defect: a shorter TTL would
  only reopen rev 17's original race (HIGH-2, round 14) in exchange for
  faster crash recovery, which is the wrong trade for a lock whose entire
  purpose is preventing two writers from clobbering a live, billed STT
  session.

  **Ownership token + compare-and-delete release, run on a context
  detached from the goroutine's own cancellation (rev 18, release
  context fixed in rev 19).** Rev 17's release (`h.cache.Del(ctx,
  lockKey)`, unconditional) and acquisition (a constant value, `"1"`)
  meant any holder's release could delete a *different* holder's lock if
  the first's TTL had already expired and a second goroutine had since
  acquired it — silently reopening problem 3 rather than closing it,
  since the debounce lock's own "anyone may release it" shape (§5.3.4) is
  safe only because stealing that lock merely delays a turn, not because
  it is a generally-safe pattern for every lock in this design. Rev 18
  fixed the ownership half of this (a per-goroutine token, checked before
  delete) but, as first drafted, still ran the release under this
  goroutine's own `ctx` — so the one case the TTL-vs-timeout margin above
  exists for (a goroutine reaching its own outer timeout while still
  correctly finishing its work) was exactly the case where the release
  would fail, since a cancelled `ctx` fails any further Redis call
  immediately (**review round 16 finding MEDIUM-2**). Fixed by detaching
  the release's context from `ctx`'s own deadline/cancellation —
  `context.WithoutCancel`, already used for this same
  detach-from-the-triggering-context purpose elsewhere in this monorepo
  (`bin-schedule-manager/pkg/dispatchhandler/manual.go:102`) — with its
  own short bound so a truly stuck Redis call still returns:

  Acquisition and release are now a matched, named pair — **not** a raw
  `SetNX` call plus a named release, which round 16 (LOW-6) noted let the
  key format drift between the two call sites:

  ```go
  lockToken := h.utilHandler.UUIDCreate() // this goroutine's own identity for
                                          //   the lock — independent of
                                          //   newTranscribeID, minted below
  acquired, err := h.cache.ListenStartLockAcquire(ctx, c.ID, lockToken.String(), listenStartLockTTL) // new config default 60s, §5.12
  if err != nil {
      // Ambiguous outcome — new in rev 20, review round 17 finding B-7:
      // the SET NX may have landed server-side even though the client
      // saw an error (timeout, connection reset mid-response — a Redis
      // client cannot always tell "definitely not set" from "set, but
      // the response was lost"). Attempt a best-effort release with our
      // own token so an ambiguous acquire error doesn't strand the lock
      // for the full TTL the same way a genuine crash does. If this
      // second call also fails, the outcome collapses into that same
      // crash case below — accepted, not specially handled further.
      releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), listenStartLockReleaseTimeout)
      _ = h.cache.ListenStartLockRelease(releaseCtx, c.ID, lockToken.String())
      cancel()
      return err // fail closed, same as every other §5.2 RPC failure — no
                 // transcribe list/start call has been made yet
  }
  if !acquired {
      // Another goroutine for this exact AIcall is already inside this
      // sequence (§5.1.1 step 7's own retry, or a second panel-open
      // during the same ring). Let it finish — this goroutine's job is
      // now redundant, and racing it is exactly problem 3 above.
      return nil
  }
  defer func() {
      // Detached from ctx's own cancellation/deadline (review round 16
      // finding MEDIUM-2) so a goroutine that reaches its own outer
      // timeout still releases promptly instead of stranding the lock
      // for the full TTL — combined with the best-effort release on the
      // acquire-error path above (round 17 finding B-7), stranding for
      // the full TTL is now reserved for an actual crash (pod loss,
      // process kill — anywhere this `defer` itself never runs), not
      // merely an error return from either call.
      releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), listenStartLockReleaseTimeout) // new, small — e.g. 3s
      defer cancel()
      _ = h.cache.ListenStartLockRelease(releaseCtx, c.ID, lockToken.String()) // compare-and-delete, best-effort — see below
  }()

  existing, errList := h.reqHandler.TranscribeV1TranscribeList(ctx, "", 10, dupFilters) // implements the reuse-check bullet above, using dupFilters (§5.2.2 above), now inside the lock
  if errList != nil {
      return errList // fail closed — an unhandled error here previously
                      // read as "no existing session found" (rev 17's
                      // bug, review round 15 finding LOW-4) and could
                      // have started a duplicate session
  }
  if len(existing) > 0 {
      _, errUpdate := h.UpdateListenState(ctx, c.ID, callID, existing[0].ID, false) // reuse bullet, unchanged logic, now serialized
      return errUpdate
  }

  newTranscribeID := h.utilHandler.UUIDCreate()
  if _, err := h.UpdateListenState(ctx, c.ID, callID, newTranscribeID, true); err != nil {
      return err // fail closed: no transcribe created yet, nothing to roll back
  }
  _, err = h.reqHandler.TranscribeV1TranscribeStart(
      ctx,
      newTranscribeID,              // id — caller-specified, not uuid.Nil;
                                    //   this ordering fix is the one and only
                                    //   reason this design uses that capability
      cmcustomer.IDAIManagerListen, // customerID  (§5.2.1)
      call.ActiveflowID,           // activeflowID — the call's, not the AIcall's:
                                   //   a panel-started contact_case AIcall has
                                   //   ActiveflowID == uuid.Nil. Mirrors
                                   //   summaryhandler/start.go:79-82.
      uuid.Nil,                    // onEndFlowID — no on-end flow for listening
      tmtranscribe.ReferenceTypeCall,
      callID,
      language,                    // §5.2.3
      tmtranscribe.DirectionBoth,  // both legs; §5.9
      tmtranscribe.ProviderEmpty,  // provider: default order gcp → aws
      5000,                        // timeout ms, same as summaryhandler
  )
  switch {
  case err == nil:
      // The created transcribe's id equals newTranscribeID (caller-
      // specified, above) — not captured into its own variable (round
      // 17 finding LOW-1: an earlier draft did, as `tr`, and never used
      // it) since the DB/Redis state written above already matches;
      // nothing further to write for this path.
  case isAlreadyProgressing(err): // helper below; the read-then-create
      // race §6 already documents — this AIcall's own List() above ran
      // just before another writer (a different AIcall on the same
      // call, §5.11's two-Cases-one-call case; the lock above only
      // serializes writers sharing this same AIcall) won the create.
      // Re-run the list once (§6, unchanged behaviour) and, if a winner
      // is found, rewrite our state to point at it instead of giving up
      // — the discrimination review round 13's MEDIUM-3 required, since
      // a blanket rollback-and-fail here would silently drop the
      // reuse-on-conflict behaviour §6 already promises.
      existingRetry, errListRetry := h.reqHandler.TranscribeV1TranscribeList(ctx, "", 10, dupFilters) // deliberately
                                    //   re-declared, not reused — this is a
                                    //   second, later List() call, and round
                                    //   16 (LOW-10) flagged shadowing the
                                    //   create-path List() call's own
                                    //   `existing`/`errList` names, above,
                                    //   as confusable with that call's
                                    //   already-fixed dropped-error bug
                                    //   (round 15 LOW-4; round 17 finding
                                    //   B-3: an earlier draft of this
                                    //   comment cited a doc line number
                                    //   here instead, already wrong by the
                                    //   time it was written)
      if errListRetry != nil || len(existingRetry) == 0 {
          _ = h.rollbackListenState(ctx, c.ID, newTranscribeID) // no winner found either; give up
          return err
      }
      if _, errUpdate := h.UpdateListenState(ctx, c.ID, callID, existingRetry[0].ID, false); errUpdate != nil {
          _ = h.rollbackListenState(ctx, c.ID, newTranscribeID)
          return errUpdate
      }
      // our own speculative id never got created — remove only that
      // membership, never touch the winner's (UpdateListenState above
      // already registered us against the winner correctly)
      _ = h.cache.ListenTranscribeAIcallRemove(ctx, newTranscribeID, c.ID)
  default:
      // Any other TranscribeV1TranscribeStart failure: give up, undo the
      // speculative write.
      _ = h.rollbackListenState(ctx, c.ID, newTranscribeID)
      return err
  }
  ```

  `isAlreadyProgressing(err)` — **named explicitly in rev 17, review
  round 14 finding MEDIUM-1: rev 16's snippet invented a
  `cerrors.IsReason` helper that does not exist anywhere in this
  codebase.** The actual, established pattern is
  `errors.As(err, &ve) && ve.Reason == "TRANSCRIBE_ALREADY_PROGRESSING"`
  against `*cerrors.VoipbinError` (`bin-common-handler/models/errors/voipbin_error.go:26-30`),
  used at `bin-transcribe-manager/pkg/transcribehandler/stop.go:196-205`
  and, in exactly the one-line wrapper shape used here, at
  `bin-storage-manager/pkg/filehandler/signing.go:79` (**citations
  corrected in rev 18, review round 15 finding MEDIUM-2** — rev 17's
  citations, `streaminghandler/disabled.go:24-28` and
  `bin-direct-manager/pkg/listenhandler/main.go:104-112`, turned out not
  to contain this pattern at all: the former is a doc comment describing
  it happening elsewhere, the latter is an unrelated `errors.Is`/generic
  error-mapper pair). `isAlreadyProgressing` is a one-line local wrapper
  around that pattern, named for readability in the `switch` above; no
  new helper is added to `bin-common-handler` itself (§9).

  `ListenStartLockAcquire(ctx, aicallID, token, ttl)` / `ListenStartLockRelease(ctx, aicallID, token)`
  — **new in rev 18, given a matched, symmetric pair of names and moved
  off a raw `SetNX` call in rev 19 (review round 16 finding LOW-6)** — are
  this lock's only two entry points; the `ai:listen:startlock:<aicallID>`
  key format is built in exactly one place (inside these two functions),
  not once inline at the call site and once inside the release helper as
  rev 18 first had it, so the two can no longer drift apart. `Acquire` is
  a thin `SetNX` wrapper (same underlying Redis command already
  established for §5.3.4's debounce lock — no new primitive there).
  `Release` performs the
  lock's compare-and-delete: `GET`s the current value at the key, and
  `DEL`s it only if that value still equals `token`, via a single Redis
  `EVAL` (a short Lua script — `if redis.call("GET",KEYS[1])==ARGV[1]
  then return redis.call("DEL",KEYS[1]) else return 0 end` — so the
  compare-and-delete is atomic, not a separate `GET` then `DEL` that
  could itself race). If the value no longer matches (this goroutine's
  TTL already expired and someone else acquired it), the call is a
  deliberate no-op — releasing would delete a lock this goroutine no
  longer legitimately holds, which is precisely the defect rev 18 fixes
  and §7 item 2 now tests directly (round 16 finding MEDIUM-4). `Release`
  is always called on a context detached from the acquiring goroutine's
  own `ctx` (above) — the two functions do not otherwise share a context
  source, and neither should: `Acquire` must respect the caller's
  deadline like any other step-7 RPC, `Release` deliberately must not.

  `rollbackListenState(ctx, aicallID, transcribeID)` is a small,
  dedicated helper — not a reuse of `clearListenState` (§5.7.3), whose
  own contract reads `listen_transcribe_id` from an AIcall struct
  "already in hand," an assumption that does not cleanly hold here since
  `UpdateListenState` writes through the DB rather than mutating the
  caller's in-memory `c`. `rollbackListenState` instead takes the known
  `transcribeID` directly: `SREM`s the Redis membership for exactly that
  id, and clears `listen_call_id`/the two metadata keys via a targeted
  `AIcallUpdate` — the same shape `clearListenState` uses, just addressed
  by an explicit id instead of a re-read one.

  Signature verified against
  `bin-common-handler/pkg/requesthandler/transcribe_transcribes.go:64-76`
  as of `NOJIRA-Allow-caller-specified-transcribe-id` (merged to `main`
  2026-09-04, after rev 9). Rev 1 omitted `provider` and `onEndFlowID`
  entirely; this `id` parameter did not exist before rev 10.

  **On the new caller-specified-id capability: adopted in rev 15 for
  ordering only, still not for its originally-intended purpose.** Its
  purpose (per its own design doc,
  `bin-transcribe-manager/docs/plans/2026-09-03-caller-specified-transcribe-id-design.md`)
  is letting a caller pre-declare a transcribe's id so it can bind a
  *dynamic, per-transcribe* RabbitMQ subscription
  (`transcribe-manager.transcript.<id>.#`) before the session starts
  producing events. §3's non-goals table already considered and rejected
  exactly that binding pattern for this design — not because of the
  ordering race rev 15 now fixes, but because of the bind/unbind
  lifecycle it would add on top of the wildcard subscription this design
  already uses (§5.3.1) with a Redis-based filter (§5.3.2) that needs no
  per-transcribe binding at all. **That rejection is unchanged.** Rev 15
  uses only the *id-predeclaration* half of the capability (so the
  Redis resolver set can be populated before creation) without the
  *binding* half (a new per-id RabbitMQ subscription) — the pre-declared
  id only ever feeds the existing Redis resolver set, never a queue
  binding.

  **On the `SET NX` lock (rev 17): this reverses an earlier decision, and
  says so.** §11's original item 9-adjacent reasoning (and this
  document's own §5.1.1 LOW-6 note, §11 item 13) had argued deduping
  concurrent `ProcessListen`/`runListenStart` calls outright was
  "unnecessary complexity: §5.2.2's guard already prevents the far worse
  outcome — two live transcribe sessions." That reasoning covered
  cross-AIcall duplication (transcribe-manager's own `startLive` dedup
  guard, §5.2.2's `List` check) but not this same-AIcall, same-goroutine-
  class race, which only became reachable once rev 16 moved the write
  before creation. The lock is scoped narrowly — one short-TTL key per
  AIcall, held only for the duration of this one sequence — not a general
  concurrency-control layer.

  **Scope of the wider fix: the create-or-reuse sequence for one AIcall's
  own attempts, not the reuse-of-another-AIcall's-session case above.**
  When *this* AIcall's own `List` call (inside the lock) finds an
  existing session another AIcall already created, that session was
  already running and producing events before this AIcall ever looked —
  there is no "before creation" moment for *this* AIcall to register
  ahead of, since it did not create that session. That remains a
  pre-existing, narrower, and effectively unclosable race (shared across
  every revision of this design), distinct from the same-AIcall race the
  lock above closes.

- A session ai-manager does **not** own — i.e. one started by the customer
  under their own `customer_id`, or one started by `ai_summary` under
  `IDAIManager` — is never reused and never touched. We cannot see it
  (different owner in our filter) and must not affect its lifecycle. This
  is now structurally guaranteed for `ai_summary` specifically, not just
  a filtering convention — see §5.2.1's rev-4 revision. (§5.2.2a, which
  in rev 3 tried to fix the `ai_summary` collision by making
  `summaryhandler` reuse-tolerant, is **deleted in rev 4**: review round
  3 (finding B3) showed that fix broke `summaryhandler`'s own read path
  and lifecycle assumptions. §5.2.1's separate-owner decision removes the
  collision at its source instead, so `summaryhandler` needs no change at
  all.)

#### 5.2.3 Language selection

`language` for a session we start: `c.STTLanguage` if non-empty, else
`config.Get().AIcallListenDefaultLanguage` (default `"en-US"`).
`transcribe-manager` normalises to BCP47 itself
(`transcribehandler/start.go:73-75`), so no client-side validation.

#### 5.2.4 Persisting listen state

```go
h.UpdateListenState(ctx, c.ID, callID, transcribeID, owns)
```

**Two calling conventions as of rev 16 (review round 13 finding
HIGH-3), not one.** On the **reuse** path (§5.2.2), this is still called
*after* the `List` call finds an existing session, exactly as rev 1-14
always did — there is nothing to pre-write ahead of when reusing someone
else's already-running session. On the **create** path (§5.2.2), this is
now called *before* `TranscribeV1TranscribeStart`, speculatively, against
the id that call generates for itself and passes in — closing the
teardown race rev 15's narrower fix (pre-registering only the Redis
`SADD`) reopened; see §5.2.2 for the full before/after account, the
per-AIcall lock added in rev 17, and the rollback path when
`TranscribeV1TranscribeStart` then fails.
**`UpdateListenState`'s own `owns`-merge rule *did* need a change to
support the create path safely — corrected in rev 17, review round 14
finding HIGH-1; see below.**

**New in rev 4 (review round 3, finding M2):** if `c.Metadata` already
carries a *different* `listen_transcribe_id` — the §5.1.1 step-3
idempotency check found the old session no longer valid and started a
fresh one — `UpdateListenState` first `SREM`s this AIcall's own id from
the **old** transcribe's resolver set before `SADD`-ing it to the new
one. Without this, the stale membership survives until its 12h TTL,
which does no *functional* harm (the old transcribe's own events have
stopped, so nothing is buffered against it) but leaves an unnecessary,
undocumented dangling set entry. **Correction, rev 17, review round 14
finding LOW-5**: an earlier draft of this paragraph additionally claimed
"§5.4.1's precondition would also refuse to act on a mismatched
`listen_transcribe_id`" — false as stated; §5.4.1 step 1 only requires
the field to be **set**, it never compares it against anything, so this
parenthetical is removed rather than relied upon.

`UpdateListenState` performs **one** `AIcallUpdate` writing:
- column `listen_call_id = callID` (§5.8),
- `metadata[listen_transcribe_id] = transcribeID` (**`tr.ID` in an earlier
  draft — corrected in rev 21, review round 18 finding LOW-4: `tr` is
  §5.2.2's local variable, not in scope here; this section's own
  parameter is `transcribeID`**),
- `metadata[listen_owns_transcribe] = owns`,

and then adds this AIcall to the Redis resolver **set**:
`SADD ai:listen:transcribe:<transcribeID> <c.ID>`, `EXPIRE ai:listen:transcribe:<transcribeID> 12h`.
This `12h` is a hardcoded constant, not one of §5.12's flags — deliberately,
not by omission (**noted explicitly in rev 21, review round 18 finding
LOW-5**): unlike §5.12's timing flags, it bounds a worst-case safety
margin (how long a genuinely-orphaned resolver-set entry can outlive its
transcribe, §5.11) rather than a value expected to need real-world
tuning, so it is not added as a fourteenth flag.

**`owns` must never be allowed to downgrade for the same transcribe id —
new in rev 14 (review round 12 finding MEDIUM-2), the OR-merge itself
scoped to same-id writes only in rev 17 (review round 14 finding
HIGH-1).** §5.1.1 step 7's bounded retry (§11 item 13's own LOW-6 note)
means the *same* AIcall can have two concurrent `runListenStart`
goroutines racing to reach this call (one per panel re-open during a long
ring). Historically (rev 14's own framing) both were assumed to resolve
to the **same** transcribe id — `startLive`'s dedup guard (§5.2.2)
guarantees that *across* AIcalls, but rev 16 broke the assumption
*within* one AIcall's own two racing goroutines, each of which could
mint a different speculative id before either finished writing. Rev 17/18's
per-AIcall lock (§5.2.2) now serializes those two goroutines *for the
create-or-reuse sequence specifically* — teardown paths
(`clearListenState`/§5.7.3, `stopListenByCallID`/§5.7.1) do not take this
lock and can still interleave with it (§5.2.2's own scope note); the
claim here is narrower than "only one write sequence is ever in flight
for this AIcall, full stop" (**precision corrected in rev 18, review
round 15 finding MEDIUM-3**). Within the create-or-reuse sequence itself,
though, the same row can still legitimately be written against **two
different transcribe ids in sequence** (the create-then-fall-back-to-
reuse branch, §5.2.2's `switch`), and an unconditional OR-merge is wrong
across that boundary: it would carry a stale `owns=true` from the
abandoned speculative id forward onto the row now describing a *different*
transcribe id — one this AIcall does not actually own. **Corrected in
rev 18, review round 15 finding MEDIUM-1**: this makes §5.7.2's stop
path (`if !owns { … never touch it }`) **incorrectly stop** that
session — since `!owns` evaluates to `false`, the "never touch it"
branch is skipped and the session is torn down — not, as an earlier
draft of this paragraph said, correctly skip stopping it. In §5.11's
two-Cases-one-call scenario this AIcall would stop the *other* Case's
still-live, still-listening session out from under it.

The rule is therefore: `UpdateListenState` writes
`listen_owns_transcribe` as `owns || <the row's current value>` **only
when `transcribeID` equals the row's current `listen_transcribe_id`** —
i.e. this write is *re-affirming* the same session another concurrent
write already touched, which is exactly rev 14's original same-AIcall
race. When `transcribeID` **differs** from the row's current value —
whether via §5.1.1 step 3's idempotency check finding a stale session and
starting fresh (rev 4's original SREM-from-old-id case, just above), or
via §5.2.2's create-then-reuse fallback — `owns` is set directly to the
caller's given value, with no carry-over: the row is now describing a
different transcribe relationship entirely, and the previous `owns` value
said nothing meaningful about it.

**Which read "the row's current value" means — clarified in rev 18,
review round 15 finding LOW-7.** `UpdateListenState` takes `c.ID`, not
the in-hand `*aicall.AIcall`, and §5.2.2's `rollbackListenState`
discussion already notes the caller's own `c` is never mutated by this
write (**a bare doc-internal line citation here was fixed in rev 19,
review round 16 finding LOW-2** — every revision shifts line numbers, so
this points at the named subsection instead) — so "the row's current
value" means a fresh `AIcallGet` inside `UpdateListenState`
itself, immediately before the merge decision, not whatever the calling
goroutine's own stale `c.Metadata` happens to hold. This matters only
for the `SREM`-from-old-id half of the rule (rev 4, just above) — a
stale in-hand `c` could name the wrong old id to `SREM` — not for the
`owns` value itself, which is identical either way inside §5.2.2's
create-then-fall-back-to-reuse branch (both reads agree once the lock,
§5.2.2, has serialized this AIcall's own writers).

(Deduping the two `runListenStart` goroutines entirely, via the same
`SET NX` primitive §5.3.4 already uses, was considered and rejected in
rev 1-16 as unnecessary complexity — "§5.2.2's guard already prevents the
far worse outcome, two live transcribe sessions." Review round 14 found
that reasoning covered only *cross*-AIcall duplication, not the
*same*-AIcall race rev 16's write-ordering change newly exposed; rev 17
adds exactly that lock, scoped narrowly, in §5.2.2. This paragraph's own
`owns`-merge fix is a second, independent layer on top of that lock —
the lock prevents the race from happening at all in the common case; the
scoped merge rule keeps the single remaining legitimate same-AIcall,
different-id sequence — the create-then-fall-back-to-reuse branch — from
writing stale ownership.)

**Set, not a single value — fixed in rev 3.** Rev 2 wrote a single key
(`ai:listen:transcribe:<transcribeID> = <c.ID>`) (**`tr.ID` in an earlier
draft — corrected in rev 22, review round 19 finding LOW-2, the third
survivor of the same out-of-scope `tr` reference §5.2.4's other two
mentions were already fixed for in rev 21**), which directly contradicts
§5.2.2's own reuse rule and §5.11's own edge case: **N AIcalls can share
one listen transcribe** (two Cases on one call, §5.11). With a
single-valued key, the second AIcall's `UpdateListenState` would silently
overwrite the first's mapping — the first AIcall stops receiving segments
for the rest of the call, with no error and no metric — and either
AIcall's `clearListenState` would delete the shared key out from under the
other. A set fixes both: every listening AIcall adds itself
(`SADD`), every listening AIcall removes only itself on cleanup (`SREM`,
§5.7.3), and Redis deletes the key automatically once the set is empty.
§5.3.2's intake step becomes `SMEMBERS`, fanning the same segment out to
every AIcall in the set (still one Redis round trip per platform speech
turn — `SMEMBERS` costs the same order as `GET` for the tiny (≤2-3 member,
in every observed case) sets this key ever holds).

**This is the only `ai_aicalls` write the feature makes during a listening
session** (one at start, one at stop). It is *not* per turn. That bounds
the known `tm_update` ↔ `Send`-cooldown coupling
(`dbhandler/aicall.go:240` bumps `tm_update`; `send.go:27-32` reads it) to
two ~3s windows per listening session, not an unbounded number — but both
are genuine risk, not just the stop one. **Rev 5 (review round 4 finding
H1) flagged the *stop*-time write as a real cost**: listening stops on
call hangup, which is exactly when an agent is likely to ask the Insight
AI a follow-up ("what was that about?"), and a `Send()` landing inside
that ~3s window would be rejected by a cooldown it did nothing to
deserve. **The *start*-time write is no longer safe to dismiss as
negligible either, since rev 11/12's bounded confbridge-readiness retry
(§5.1.1 step 7) means the start write can now land up to
`AIcallListenConfbridgeReadyMaxWaitSeconds` (default 30s) after panel
open, not immediately — comfortably inside a window where the agent may
already have typed a real question** (corrected in rev 13, review round
11 finding LOW-4; earlier revisions' "negligible" framing predates the
retry and no longer holds). Rather than accept either window silently,
**both** `UpdateListenState` (the start write, §5.2.4) and
`clearListenState` (the stop write, §5.7.3) skip their `AIcallUpdate`'s
`tm_update` bump: the write uses
`dbhandler.AIcallUpdateWithoutTouchingTMUpdate` (or an equivalent
targeted-column update that bypasses the standard `tm_update`-on-any-write
convention) for both write paths, so listen's own bookkeeping never
contributes to the cooldown at all — start or stop. This is narrower and
safer than the `tm_last_send` decoupling recorded as a follow-up in §11:
it fixes listen's own two writes specifically, without touching `Send`'s
cooldown semantics for every other AIcall write path.

### 5.3 Layer 1 — event intake

#### 5.3.1 Binding

`bin-ai-manager/pkg/subscribehandler/main.go` gains one static pattern
appended to `topicPatterns` (`main.go:52-64`):

```
transcribe-manager.transcript.*.created
```

Adding it requires editing three coupled places, all confirmed:
1. `topicPatterns` — **append at the end**; the golden test is
   position-sensitive.
2. `processEvent`'s switch (`main.go:179-220`) — a new case using the
   already-declared-but-currently-unused `publisherTranscribeManager`
   constant (`main.go:34`).
3. `binding_golden_test.go` — the `expected` slice (append
   `"transcribe-manager.transcript.*.created"`), the hardcoded
   `len(topicPatterns) != 11` → `12` (two occurrences: the check and the
   message), and the doc comment above `topicPatterns`.

#### 5.3.2 Volume, and why the wildcard is affordable (resolves review H4)

`transcript_created` fires per final STT result for **every** transcription
session platform-wide — flow-driven, summary-driven, customer-started —
not just calls we listen to. `processEventRun` spawns an unbounded
goroutine per event (`main.go:161-165`, prefetch 10). Rev 1 did not
account for this.

The per-event work is therefore made unconditionally cheap:

```go
func (h *subscribeHandler) processEventTMTranscriptCreated(ctx context.Context, m *sock.Event) error {
    var evt tmtranscript.Transcript
    if err := json.Unmarshal([]byte(m.Data), &evt); err != nil { ... }

    // H3: transcripthandler.dbDelete publishes EventTypeTranscriptCreated on
    // DELETE too (bin-transcribe-manager/pkg/transcripthandler/db.go:33 — a
    // known bug, documented in models/transcribe/routingkey_golden_test.go:
    // 182-184). Without this guard a deleted line replays into the LLM as
    // freshly-spoken content.
    if evt.TMDelete != nil || strings.TrimSpace(evt.Message) == "" {
        return nil
    }

    return h.aicallHandler.EventTMTranscriptCreated(ctx, &evt)
}
```

and `EventTMTranscriptCreated` opens with a single Redis `SMEMBERS`
(§5.2.4's fix — a set, not a single value):

```
aicallIDs, ok := cache.ListenAIcallIDsGet(ctx, evt.TranscribeID)  // SMEMBERS ai:listen:transcribe:<id>
if !ok || len(aicallIDs) == 0 { return nil }   // not a session we started — 99.9% of platform events end here
for _, aicallID := range aicallIDs {
    // buffer + debounce (§5.3.3/§5.3.4) independently per listening AIcall
}
```

**Sized cost of keeping the wildcard:** per final STT result anywhere on
the platform — one AMQP delivery, one goroutine, one JSON unmarshal, one
Redis `SMEMBERS`. No DB query, no RPC. At VoIPBin's current single-node
scale that is a rounding error; the escape hatch (dynamic per-transcribe
binding) and its leak-sweeper requirement are pre-documented in §3 so the
switch is a decision, not a redesign.

**On the Redis resolver being the sole filter:** the key is written
explicitly at listen start and deleted (well, `SREM`'d) explicitly at
listen stop (§5.7). It is *not* part of `cachehandler.AIcallSet`'s
snapshot-index scheme (`pkg/cachehandler/handler.go:79-97`), which writes
secondary keys (`ai:aicall:reference_id:<id>`, `ai:aicall:pipecatcall_id:<id>`)
and never invalidates the old key when the indexed field changes. Reusing
that scheme for listen state would leave stale keys pointing at stale
snapshots and would collide every non-listening AIcall on a shared
nil-UUID key (review round-1 M1). This key is a purpose-built,
explicitly-managed pointer, not a snapshot index — that distinction is
the fix.

**Cache-loss behaviour (stated, not hidden):** a Redis flush drops the
resolver keys, so in-flight calls stop being listened to until the panel
is reopened (which re-runs §5.1 and repopulates). There is deliberately no
DB fallback on miss, because a DB fallback would put a query on the
platform-wide hot path — exactly the cost this design removes. Losing
best-effort proactive notifications for the remainder of a call is an
acceptable, self-healing degradation; the DB column remains the source of
truth for cleanup.

#### 5.3.3 Buffering

Two Redis lists per AIcall, both `EXPIRE`d to
`AIcallListenBufferTTLHours` (default 6h) on every push:

| Key | Op | Purpose |
|---|---|---|
| `ai:listen:pending:<aicall_id>` | `RPUSH` | lines not yet evaluated; drained atomically by the turn |
| `ai:listen:window:<aicall_id>` | `RPUSH` + `LTRIM -W -1` | rolling last `W` lines (default 40) for continuity across turns |

The line format is the structural speaker tag (see §5.9):
`"[CUSTOMER] …"` / `"[AGENT] …"`.

Two lists rather than one list plus a counter: both operations are single
atomic Redis commands, so no cross-command consistency reasoning is
needed. A line briefly present in `window` but not yet popped from
`pending` is harmless — it is context either way.

#### 5.3.4 Debounce

After buffering:

```
if !cache.ListenTurnTryLock(ctx, aicallID, interval) {  // SET ai:listen:lock:<id> NX EX <interval>
    return nil   // a turn ran recently; this line waits in the buffer
}
go h.aicallHandler.RunListenTurn(detachedCtx, aicallID)
```

`interval = config.Get().AIcallListenEvaluateIntervalSeconds` (default
20). This is a leaky-bucket debounce that:
- works across replicas (both ai-manager pods share Redis),
- needs no timers, no goroutine-per-AIcall, no in-process state,
- self-heals on pod loss (the lock TTL expires).

**Known behaviour, stated:** the last few lines before a silence are not
evaluated until the *next* line arrives. In practice a call ends shortly
after and §5.7's hangup path performs one final flush turn, so the tail is
not lost. A wall-clock flush timer is deliberately not introduced.

### 5.4 Layer 2 — the listen evaluation turn

`aicallHandler.RunListenTurn(ctx, aicallID)`:

#### 5.4.1 Preconditions

1. `c := h.Get(aicallID)`. **Reordered in rev 8 (review round 7, finding
   N-3), placement clarified in rev 9 (review round 8, finding L-3): `c`
   is not needed by the flag comparison itself, only by what runs when a
   condition fails** — `stopListening(ctx, c)` below (and, indirectly,
   `clearListenState`, per §5.7.3's "callers already hold `c`"). Rev 7's
   standalone step 0 ran before `c` existed and could not have called
   either. Require `config.Get().AIcallListenEnabled`,
   `c.Status == progressing`, `c.ReferenceType == contact_case`,
   `c.Metadata[listen_transcribe_id]` set — any failing → `stopListening`
   and return (`skipped_disabled` if the flag was the failing condition,
   `skipped_invalid` otherwise).

   **`stopListening(ctx, c)` — named and scoped precisely in rev 9
   (review round 8, finding M-2), replacing the informal "stop listening
   entirely" phrase rev 8 used without a single definition.** It is
   exactly two calls, in order, and nothing else:
   1. If `c.Metadata[listen_owns_transcribe]`, run §5.7.2's stop
      snippet (fresh `TranscribeV1TranscribeGet` + conditional
      `TranscribeV1TranscribeStop`, non-fatal on failure).
   2. `clearListenState(ctx, c)` (§5.7.3).

   **It never calls `ProcessTerminate`** (`process.go:38-68`) — that
   function ends the *AIcall itself* (`UpdateStatus` to terminated +
   `FlowV1ActiveflowServiceStop`), which would kill the agent's entire
   Insight Q&A session, directly contradicting §6's "the Q&A panel keeps
   working normally" row. §5.7.2's own heading ("Hooked into
   `ProcessTerminate`... for `contact_case` AIcalls") describes a
   *different* call site — the AIcall-is-ending path — reusing the same
   stop snippet; `stopListening` is the shared helper both that path and
   `RunListenTurn`'s preconditions call into, not a call to
   `ProcessTerminate` itself. §7's cleanup tests (item 7) cover both
   call sites.
2. `lines := cache.ListenPendingPopAll(ctx, aicallID)` — a single `LPOP
   key count` (Redis ≥6.2), atomic, so no concurrent appender can lose a
   line between a read and a trim. Empty → return (`skipped_empty`).
3. Cost cap: a per-AIcall turn counter (`INCR ai:listen:turns:<id>`, same
   TTL as the buffer) bounded by
   `AIcallListenMaxTurnsPerAIcall` (default 60 ≈ 20 minutes of continuous
   speech at a 20s interval). Exceeded → `stopListening(ctx, c)` (step 1's
   helper) and return (`skipped_cap`). This is the hard backstop against a
   pathological long call.

**On intake (§5.3) needing no flag check of its own, reconfirmed in rev
8**: a flag-off rollback still lets §5.3 buffer incoming segments for up
to one more `AIcallListenEvaluateIntervalSeconds`, but step 1 above stops
evaluating them (zero further LLM turns) and clears the resolver-set
membership that intake depends on, so the very next segment after that
stops matching anything at §5.3.2's `SMEMBERS` check. Bounded, self-limiting
staleness — not a gap.

**Trigger cadence, precision added in rev 9 (review round 8, finding
M-1): `RunListenTurn` is segment-triggered (§5.3.4's debounce lock), not
timer-scheduled.** A flag-off rollback's `stopListening` call therefore
fires on the *next transcript segment that arrives* for an in-flight
session, not on a fixed clock — if the call has gone quiet, that is
whenever the next word is spoken, or, in the limit, never before the call
itself ends (at which point §5.7.1's hangup path is the actual backstop,
independent of the flag). §8's "within one `AIcallListenEvaluateIntervalSeconds`"
framing describes the common case (an active conversation), not a
guarantee; corrected there too.

#### 5.4.2 Context assembly (resolves review B2)

Built explicitly — **`getPipecatcallMessages` is not called**:

| # | Role | Content | Bound |
|---|---|---|---|
| 1 | `system` | `InsightSystemPrompt` (`pkg/aicallhandler/main.go:264-282`) — **added in rev 3**. `startInitMessages` normally puts this first for every `ai.TypeInsight` AIcall (`start.go:790-797`), ahead of the customer's own `init_prompt`. Rev 2's context assembly read only `Metadata[prompt_snapshots]`, which holds **just** the substituted `init_prompt` (`buildPromptSnapshots`, `start.go:128-166`) — it never captured `InsightSystemPrompt`. Without it, a listen turn ran with none of the platform's own Insight guardrails ("base every answer strictly on retrieved data", "never expose raw JSON or tool responses", "never mention tool names/JSON/backend logic") — exactly the rules that keep *unsolicited* output sane. `InsightSystemPrompt` is a fixed platform constant (not per-customer), so it needs no DB read either | 1 message |
| 2 | `system` | The frozen prompt snapshot from `c.Metadata[prompt_snapshots]` (`models/aicall/main.go:12-22`) — for `AssistanceTypeAI` there is exactly one; for `AssistanceTypeTeam`, the one whose `MemberID == c.CurrentMemberID`, else the first. Already substituted at AIcall start (`start.go:128-166`), so **no DB read and no re-substitution** | 1 message |
| 3 | `system` | `ListenTurnSystemPrompt` — a new constant beside `InsightSystemPrompt`, describing the watch task and the `notify_agent` contract (§5.5.3) | 1 message |
| 4 | `user`/`assistant` | The last `AIcallListenQAContextSize` (default 10) rows of this AIcall with `Role ∈ {user, assistant}`, oldest-first. Fetched as `messageHandler.List(ctx, 30, "", {FieldAIcallID: c.ID, FieldDeleted: false})` then filtered in-process (`ApplyFields` has no `IN` support) and truncated. Gives the AI continuity with what the agent asked and with its own earlier notifications | ≤10 messages |
| 5 | `user` | The transcript block: `cache.ListenWindowGet` (≤40 lines) rendered with a marker separating already-seen lines from the newly popped ones | 1 message, ≤40 lines |

Total: a constant-shaped, small prompt, independent of call length. Both
system prompts can never be evicted, because they are not competing with
transcript rows for a 100-row window — transcript lines are not rows at
all. (The flow-parameter JSON block `startInitMessages` also appends for
some AIcalls is deliberately *not* replayed here — listen turns are never
flow-parameterized, since a panel-started `contact_case` AIcall has
`ActiveflowID == uuid.Nil`, so there is nothing for it to carry.)

#### 5.4.3 Session start

```go
turnPipecatcallID := h.utilHandler.UUIDCreate()   // NOT written to c.PipecatcallID
// New in rev 7 (review round 6, finding F1): register this turn id as a
// genuine, positively-known listen turn — see §5.4.5 step 2 for why a
// negative check ("not the currently-bound id") isn't enough on its own.
// TTL comfortably exceeds the terminate-with-delay below so the entry
// outlives the turn; self-expiring, no explicit cleanup needed.
//
// Error handling tightened in rev 8 (review round 7, finding N-2): a
// silently-ignored SADD failure here would let the turn proceed
// unregistered, so any tool call it makes would resolve listenTurn=false
// at §5.4.5 step 2 — the tool-call/tool-result rows get permanently
// tagged Origin=OriginNone (never excluded from future Q&A replay) and
// notify_agent gets rejected, i.e. exactly the "worse failure mode"
// §5.4.5 step 2's B1 fix exists to avoid, reintroduced through the one
// write this section owns. Abort the turn instead of proceeding
// unregistered.
if errAdd := cache.ListenTurnPipecatcallIDAdd(ctx, c.ID, turnPipecatcallID, listenTurnPipecatcallIDTTL); errAdd != nil { // SADD ai:listen:turnpcid:<aicall_id> <turnPipecatcallID>, EXPIRE
    log.Warnf("could not register listen turn id, skipping this turn: %v", errAdd)
    promListenTurnTotal.WithLabelValues("skipped_register_failed").Inc()
    return
}
pc, err := h.startListenPipecatcall(ctx, c, turnPipecatcallID, llmMessages)
// → PipecatV1PipecatcallStart(ctx, turnPipecatcallID, c.CustomerID, c.ActiveflowID,
//      pmpipecatcall.ReferenceTypeAICall, c.ID, llmType, llmMessages,
//      STTTypeNone, "", TTSTypeNone, "", "")
_ = h.reqHandler.PipecatV1PipecatcallTerminateWithDelay(ctx, pc.HostID, pc.ID, defaultListenTurnTimeout) // 60s
```

`listenTurnPipecatcallIDTTL` (default 180s — 3× `defaultListenTurnTimeout`)
is a new config-derived constant alongside the others in §5.12; generous
headroom is cheap (a handful of UUIDs per AIcall, self-expiring) and the
cost of it being too short is exactly the bug this section fixes.

`startListenPipecatcall` is a sibling of `startPipecatcall`
(`start.go:697-744`) that takes the pipecatcall id and the message list as
parameters instead of reading `c.PipecatcallID` and calling
`getPipecatcallMessages`.

**Not writing `turnPipecatcallID` to the AIcall row is the load-bearing
decision.** It means:
- no `AIcallUpdate` per turn → no `tm_update` bump → no `Send` cooldown
  interference (review L3),
- `interruptPreviousPipecatcall` is never called → an in-flight answer to
  the agent is never killed (review B1),
- and the mismatch itself becomes the drop signal (§5.4.4).

Tool calls still route correctly: pipecat POSTs to
`…/<pipecatcall_id>/tools` (`scripts/pipecat/tools.py:107`), pipecat-manager
resolves the aicall from the *pipecatcall's* `ReferenceID` (= `c.ID`), not
from `AIcall.PipecatcallID`. `ToolHandle` therefore operates on the right
AIcall.

#### 5.4.3a Threading pipecatcall identity into `ToolHandle` (new in rev 4 — infrastructure prerequisite for §5.4.4(c), §5.6.5, and B1's fix below)

**Review round 3 (finding B4) showed §5.4.4(c) as written in rev 3 cannot
be implemented**: it assumed `ToolHandle` already knows which pipecatcall
a tool call arrived on, but it does not. Traced end to end:
`bin-pipecat-manager/pkg/pipecatcallhandler/runner.go:457` calls
`AIV1AIcallToolExecute(ctx, pc.ReferenceID, request.ID, ...)` — `pc.ID`
(the pipecatcall id) is in hand but never passed. The RPC signature
(`bin-common-handler/pkg/requesthandler`), the wire DTO
(`bin-ai-manager/pkg/listenhandler/models/request/aicalls.go`), and
`ToolHandle(ctx, id, toolID, toolType, function)`
(`aicallhandler/tool.go:24`) all lack the field.

**Decision: thread it through, as a small, explicit cross-service
addition — not a workaround.** This single addition is what makes
§5.4.4(b)'s drop-if-foreign guard, §5.4.4(c)'s reject-guard, and B1's fix
(below) all implementable from one shared signal, rather than three
separate mechanisms:

1. `runner.go:457` passes `pc.ID` as an added `pipecatcallID uuid.UUID`
   argument to `AIV1AIcallToolExecute`.
2. `bin-common-handler/pkg/requesthandler`'s `AIV1AIcallToolExecute`
   signature and its wire marshalling gain the field; regenerate its
   mock.
3. `V1DataAIcallsIDToolExecutePost` (`listenhandler/models/request/aicalls.go`)
   gains `pipecatcall_id`, **`json:"pipecatcall_id,omitempty"` — omittable,
   not required** (see the rollout note below); the `listenhandler`
   handler for `POST /v1/aicalls/<uuid>/tool_execute`
   (`pkg/listenhandler/v1_aicalls.go`) passes it through as an unwrapped
   argument (root `CLAUDE.md`'s transport-DTO-ownership rule — this stays
   a domain argument from here on, never a `request.*` value past
   `listenhandler`).
4. `ToolHandle(ctx, id, toolID, toolType, function, pipecatcallID
   uuid.UUID)` — one new parameter on the interface method itself
   (`pkg/aicallhandler/main.go`'s `AIcallHandler` interface, not just its
   implementation). **Scope corrected in rev 5, review round 4 finding
   M2**: this is not "one new parameter" in isolation — `mapFunctions`'
   value type (`tool.go:54-76`) is
   `func(ctx, *aicall.AIcall, *message.ToolCall) *messageContent`, shared
   by all 21 `toolHandleXxx` functions. `pipecatcallID` is consumed by
   `ToolHandle` itself — used to resolve `listenTurn bool` (§5.4.5 step 2;
   as of rev 7, via a Redis membership check, not an `AIcallGet`) — and
   only the resulting **`listenTurn bool`**, not `pipecatcallID` itself,
   is passed explicitly to `toolHandleNotifyAgent` (§5.4.4(c)) — the
   other 20 handlers' signatures are **unchanged**, avoiding a
   21-function signature churn for a value only one handler needs.
   Regenerate the `AIcallHandler` mock (any other file that implements or
   mocks this interface — checked at implementation time — picks up the
   new method signature too).
5. Inside `ToolHandle`, the same comparison used for §5.4.5's `Origin`
   tagging and §5.4.4(c)'s reject-guard — see both for the exact,
   corrected logic (rev 5 fixes both; rev 4's version of each had a
   distinct bug).

This is a real, scoped cross-service change (pipecat-manager +
common-handler + ai-manager's listen surface + `ToolHandle`'s callers),
correctly reflected in §8's rollout and §9's impacted-files list — rev 3
under-scoped this as a single `mapFunctions` line; that was wrong.

**Rollout ordering, new in rev 5 (review round 4, finding B2).** Making
the wire field optional (step 3) means a rolling deploy where
`bin-pipecat-manager` still runs the old binary sends no
`pipecatcall_id` at all — unmarshalled as `uuid.Nil`. §5.4.4(c) and
§5.4.5 both already treat `uuid.Nil` as "assume this is the agent's real
turn" (fail toward doing nothing new, never toward mistagging real
content — see §5.4.4(c)'s worked rationale), so an old `pipecat-manager`
talking to a new `ai-manager` **degrades safely**: `notify_agent` calls
simply get rejected (harmless — nothing calls it before listening ships
anyway) and no row is ever mistagged. The unsafe direction — new
`pipecat-manager` behavior reaching an old `ai-manager` that doesn't
expect the field — cannot happen (an old `ai-manager` ignores an unknown
JSON field). Net: **no forced deployment order is required between the
three touched services**, but `bin-pipecat-manager` should still land
first as ordinary good practice (its change is additive and inert until
`ai-manager`'s side consumes it).

#### 5.4.4 Suppressing all output except `notify_agent` (resolves review-round-1 B1)

Three independent mechanisms, not two — rev 2's "belt and braces" was
missing a buckle. Review round 2 (finding F3) showed `RunLLM: false` is
not the reliable primitive rev 2 treated it as, so it is now the weakest
of three layers rather than the primary one.

**(a) `notify_agent` is defined with `RunLLM: false` — a best-effort
hint, not a guarantee.** Verified end to end: `tool.Tool.RunLLM`
(`models/tool/main.go:74-83`) is serialised to pipecat,
`_build_run_llm_defaults` reads it (`scripts/pipecat/tools.py:58-85`), and
the happy path passes `FunctionCallResultProperties(run_llm=should_run_llm)`
(`tools.py:105,142-152`), suppressing the follow-up `bot_llm` text frame.
**Three caveats review round 2 found, all confirmed by re-reading
`tools.py`:**
1. **Every error path drops `properties` entirely** (`tools.py:135-138`
   HTTP≥400, `:156-159` timeout, `:163-166` `ClientError`, `:170-173`
   generic) — a failed or slow `notify_agent` call re-runs the LLM and can
   still emit follow-up text.
2. **The model can override the default.** `args.pop("run_llm", …)`
   (`tools.py:105` — **line corrected in rev 5, review round 4 finding
   M5**: rev 4 cited `tools.py:~60`) takes the LLM's own `run_llm`
   argument first if it supplies one — `models/tool/main.go:75-77`'s own
   comment says as much: *"The LLM can still override this per-call via a
   `run_llm` argument."*
3. **Correcting a false claim from rev 2**: it is not true that "every
   other tool in `definitions.go` uses `RunLLM: true`" — 9 of the 21 tools
   do not (`connect_call`, `send_email`, `send_message`, `stop_media`,
   `stop_service`, `stop_flow`, `set_variables`, `get_variables`,
   `get_aicall_messages`; `definitions.go` lines 10, 83, 163, 240, 276,
   306, 335, 376, 409). The accurate, and stronger, statement is that
   **all six existing Insight tools** use `RunLLM: true` — their tool
   *definitions* (as opposed to their handler implementations, which do
   live in `tool_insight.go`) are registered in
   `pkg/toolhandler/definitions.go` at **lines 754-755, 785-786, 824-825,
   849-850, 879-880, 906-907** (**line numbers corrected in rev 5, review
   round 4 finding M4**: rev 4's fix for review round 3's M3 corrected the
   *file* but reused the *wrong* line numbers — the ones for the
   `RunLLM: false` tools listed just above, not the six Insight tools'
   own lines) — which is exactly why `notify_agent` being the one Insight
   tool with `RunLLM: false` is a deliberate outlier, not a "usual
   pattern," and needs (b)/(c) below to actually hold.

**(b) A pipecatcall-identity guard on every inbound pipecat message event
that can actually fire from a listen turn.** `messagehandler` gains one
shared helper:

```go
// isForeignPipecatcall reports whether evt.PipecatcallID differs from the
// AIcall's currently-bound PipecatcallID. True means the event came from a
// session the AIcall no longer (or never did) consider its conversational
// turn — a listen evaluation turn, or a genuinely stale reply — and MUST
// NOT be persisted or delivered.
func (h *messageHandler) isForeignPipecatcall(ac *aicall.AIcall, evtPipecatcallID uuid.UUID) bool
```

applied for `ac.ReferenceType == aicall.ReferenceTypeContactCase` in the
**two** handlers that can actually fire from a listen turn — **narrowed
from four in rev 3, review round 3 finding H2**: `EventPMMessageUserLLM`
(`event.go:293-307`) and `EventPMMessageUserTranscription`
(`event.go:115-133`) are both driven by an STT leg, and a listen turn is
started with `STTTypeNone` (§5.4.3's `startListenPipecatcall` call), so
neither event can ever originate from a listen turn's pipecatcall. **Cost
argument corrected in rev 5, review round 4 finding M6**: both handlers
already call `resolveActiveAIID` → `AIV1AIcallGet` per event today
(`event.go:73-83`, called from `:125` and `:299`), so adding the guard
would not be a *new* AIcall lookup — the real reason to leave them
unguarded is purely structural (the condition the guard checks for cannot
occur on these two paths, per `STTTypeNone` above), not a cost argument
this design invented. Left unchanged, exactly as rev 1 had them.

| Handler | File:line | Today | Rev 4 |
|---|---|---|---|
| `EventPMMessageBotLLM` | `messagehandler/event.go:167-180` | persists **any** non-empty text unconditionally on the non-conversation branch | drop if foreign; also pass `WithPipecatcallID(evt.PipecatcallID)` on the row it does persist |
| `EventPMMessageBotLLMIntermediate` | `event.go:260-291` | publishes an `EventTypeMessageIntermediate` **webhook per token chunk**, no aicall check | drop if foreign |

This is a strict improvement beyond this feature: it extends to
`contact_case` the same stale-response guard the `conversation` branch
already has (`event.go:182-189`), so today's silently-persisted stale
contact_case replies stop appearing too. Metric:
`ai_manager_aicall_foreign_pipecatcall_dropped_total{handler}`.

**Correctness caveat found in review round 2 (F4): a stale cache read can
turn (b) into a false positive against a genuine answer.**
`AIcallGet` is cache-first (`pkg/dbhandler/aicall.go:112-115`), and
`AIcallUpdate`'s cache refresh discards its own error
(`_ = h.aicallUpdateToCache(ctx, id)`). If the Redis write right after a
real `Send()`'s `UpdatePipecatcallID` (`aicallhandler/db.go:244-248`)
transiently fails, the cached AIcall keeps the *old* `PipecatcallID` for
up to its TTL — and (b) would then drop the agent's genuine answer as
"foreign."

**Fix, path corrected in rev 5 (review round 4, finding H4).**
`EventPMMessageBotLLM` gets the AIcall via
`h.reqHandler.AIV1AIcallGet(ctx, ...)` (`messagehandler/event.go:160`) —
a RabbitMQ RPC to ai-manager's own `listenhandler`, not a direct
`dbhandler` call, and rev 4's `dbhandler.AIcallGet(skipCache: true)`
citation was never on this code path. `AIV1AIcallGet`'s underlying
`listenhandler` route already resolves through the same cache-first
`dbhandler.AIcallGet` rev 4 assumed `messagehandler` called directly, so
the fix is: `AIV1AIcallGet` gains an optional cache-bypass argument (or a
sibling method), threaded down to the `dbhandler.AIcallGet(skipCache:
true)` call rev 4 correctly identified as the eventual target — it is
just one RPC hop further away than rev 4's snippet showed. On a
mismatch, `messagehandler` issues this cache-bypassing variant of the
same RPC it already makes, following the same shape as the `conversation`
branch's own stale-reply guard at `event.go:209-219`, which this design's
guard is explicitly modelled on. Only drop if the DB-authoritative read
still disagrees. **Corrected in rev 6, review round 5 finding M2**: this
is *not* confined to `bin-ai-manager` — `AIV1AIcallGet`'s client is
`bin-common-handler/pkg/requesthandler/ai_aicalls.go`, so its signature
change (or a sibling method) lands in `bin-common-handler` like §5.4.3a's
and §5.4.5's changes do, just a smaller one (one client method, not a new
cross-service field). Concretely: the RPC is a bodyless GET
(`ContentTypeNone, nil`), so "cache-bypass" is expressed as a query
parameter on the existing route or a sibling
`GET /v1/aicalls/<uuid>?skip_cache=true`-style variant — `listenhandler`'s
route regex and its `dbhandler.AIcallGet` call both need the plumbing,
not just the RPC signature.

**Scoped to the one handler that actually persists — review round 3
finding H1.** `EventPMMessageBotLLMIntermediate` fires once per streamed
token chunk, not once per message; re-reading the AIcall bypassing cache
on every mismatched intermediate chunk would put an uncached DB read on
that hot path for the entire duration of every listen-turn (or genuinely
stale) reply. It only ever *publishes a webhook*, never persists a row —
so a false-positive drop there costs nothing more than one skipped
intermediate-token webhook, which is not user-visible (only the final
`EventPMMessageBotLLM` message matters to the agent). The cache-bypass
re-read therefore applies **only** to `EventPMMessageBotLLM`;
`EventPMMessageBotLLMIntermediate` drops on a plain in-memory mismatch,
no re-read.

Two existing handlers were checked and need **no** change:
- `EventPMPipecatcallTerminated` returns early unless
  `ac.ReferenceType == ReferenceTypeConversation` (`event.go:405-408`) —
  **noted as a real gap, not dismissed**: this means `contact_case` has no
  termination-triggered "Sorry, I'm having trouble responding right now"
  backstop the way `conversation` does, so (b)'s cache-bypass re-read in
  the paragraph above is the only safety net for a stale-cache false
  drop on this reference type. Acceptable because it is a re-read, not a
  guess.
- `EventPMPipecatcallInitialized` returns early unless
  `cc.ReferenceType == ReferenceTypeCall` (`aicallhandler/event.go:110-112`).

**(c) `toolHandleNotifyAgent` itself rejects the call outright when it did
not arrive on a listen turn — closing the review-round-2 F3 hole, now
actually implementable via §5.4.3a.** (a) alone has a failure mode rev 2
did not analyze: if `notify_agent` is invoked during the agent's *own*
Q&A turn (on `c.PipecatcallID` itself, not a throwaway listen-turn id) and
`run_llm=False` actually takes effect, the agent's real question gets
**no answer at all** — just an unrelated notification. §6 ("LLM calls
`notify_agent` during a normal Q&A turn → Allowed... Harmless") was wrong
about this in rev 2; it is not harmless. **Rev 3's claim that pipecat
"already sends the pipecatcall id on every `tool_execute` POST" to
`ToolHandle` was wrong** (review round 3, finding B4) — `tools.py:107`
puts it in the URL path from pipecat to pipecat-manager only;
pipecat-manager's own `runner.go:457` never forwards it to ai-manager. §5.4.3a
fixes exactly this, and `toolHandleNotifyAgent` now receives the real
`pipecatcallID` as a parameter (threaded from `ToolHandle`, itself
threaded per §5.4.3a).

**Rewritten three times now.** Rev 5 (review round 4, finding B1) fixed
the inverted two-branch logic. Rev 6 (review round 5, finding B1) moved
resolution into `ToolHandle` so both this guard and §5.4.5's tagging
shared one fresh `AIcallGet` — but the *comparison* that read still fed
(`pipecatcallID != c.PipecatcallID`) remained a negative test, which rev
6's own `ListenCallID` addition didn't actually fix (review round 6,
finding F1). Rev 7 replaces the comparison itself: `ToolHandle` (§5.4.5
step 2) now resolves `listenTurn` via a direct Redis membership check
against the set of ids §5.4.3 registers when it mints a listen turn — a
positive, race-free signal that makes the `AIcallGet` read this section
used to describe **unnecessary for this decision** (it is still used
elsewhere, by §5.4.4(b)'s stale-reply guard, for an unrelated reason):

```go
// toolHandleNotifyAgent(ctx, c, tool, listenTurn) — listenTurn is
// resolved once by ToolHandle (§5.4.5 step 2) via Redis set membership,
// not re-derived here. One source of truth, not two that can disagree.
if !listenTurn {
    // This tool fired on the agent's own conversational turn (or the
    // membership check couldn't run at all — §5.4.5 step 2 degrades a
    // Redis failure there to listenTurn=false rather than failing the
    // tool call closed, since that is the provably correct value under
    // a Redis outage, review round 7 finding N-1) — reject rather than
    // let RunLLM's best-effort suppression silently eat the agent's
    // real question.
    fillFailed(res, fmt.Errorf("notify_agent is only usable while proactively monitoring a call; you were asked a question — answer it directly instead"))
    return res
}
```

This makes `notify_agent` fail closed exactly when (a)'s guarantee is
weakest, independent of whether (a) actually suppressed the follow-up,
and fails closed (reject, not silently corrupt) on the unknown-id case
too.

Net effect: **a listen turn produces exactly zero persisted rows and zero
webhooks unless the LLM calls `notify_agent` from that turn — and a call
to `notify_agent` from any other context is rejected rather than silently
eating a real answer.**

#### 5.4.5 Keeping listen turns from evicting the agent's own Q&A context (new in rev 4, resolves review-round-3 B1)

**Review round 3 (finding B1) showed rev 1's original defect —
`getPipecatcallMessages`'s newest-100-row replay evicting the AIcall's
own system prompt — returns through a different door.** §5.4.2 stops the
*listen turn's own* context from using that replay path, but says nothing
about the rows a listen turn *writes*. Per §6, an Insight tool (including
`notify_agent`) can fire during a listen turn, and every tool call writes
two `Message` rows via `ToolHandle` (tool-call, tool-result —
`tool.go:47,88`). At the `AIcallListenMaxTurnsPerAIcall=60` cap, even one
tool call per turn is 120 rows — enough on its own to push the real
Q&A history (and the leading system-prompt rows) out of `getPipecatcallMessages`'s
top-100 window (`start.go:620-661`) the **next** time the agent asks the
Insight AI a real question. This is exactly rev 1's B2 defect, recurring
through a mechanism rev 2 did not create and rev 3 did not check for.

**Fix: tag every row a listen turn writes, exclude tagged rows from the
agent's own Q&A replay at the query level, and — new in rev 5 — guarantee
the leading system-prompt row(s) survive independent of that window
entirely.**

1. `message.Origin` (§5.6.2) gains a second value:
   ```go
   OriginListenInternal Origin = "listen_internal"  // tool-call/tool-result
   // rows written during a listen evaluation turn — never replayed into
   // any future context, listen or Q&A. The listen turn's own context
   // (§5.4.2) is assembled explicitly and never reads message rows for
   // the transcript/tool-call portion either, so this exclusion only
   // matters for getPipecatcallMessages (a real Q&A turn's context).
   ```
2. `ToolHandle` (extended per §5.4.3a to know `pipecatcallID`) tags the
   tool-call and tool-result rows it writes. **Rewritten across rev 6
   (review round 5, findings B1/B2) and rev 7 (review round 6, findings
   F1/F2) — the predicate went through three shapes before landing on one
   that is both correct and cheap:**

   - **B1 (rev 6): the tagging decision used the cached `c.PipecatcallID`
     while §5.4.4(c)'s reject-guard used a fresh, cache-bypassing read —
     the tagging step, which has the *worse* failure mode (permanent
     exclusion vs. one rejected tool call), was trusting the weaker
     source.** Fixed by resolving `listenTurn` once and sharing it with
     both consumers (kept in rev 7, see the snippet below).
   - **B2 (rev 6, incompletely) → F1 (rev 7): `pipecatcallID !=
     c.PipecatcallID` proves only "not the currently-bound turn," not "a
     listen turn," and rev 6's fix (`ListenCallID != uuid.Nil`) doesn't
     close the gap — inside an active listening session that term is a
     constant `true` and adds no discrimination.** Concretely: agent asks
     Q1 (pipecatcall A minted); before `ToolHandle` processes Q1's tool
     call, agent asks Q2, `Send` best-effort-interrupts A and rotates to
     B (`send.go:113-122`); Q1's tool call now arrives with
     `pipecatcallID=A`, and `A != fresh.PipecatcallID(B)` is true — indistinguishable
     from a genuine listen turn under rev 6's predicate, even though
     `ListenCallID` being set is unrelated to which turn this call came
     from. **The fix needs a signal that is positively "this specific id
     was minted as a listen turn," not "this id currently isn't the bound
     one."** §5.4.3 now registers exactly that, in Redis, at the moment a
     listen turn's throwaway id is minted (`SADD
     ai:listen:turnpcid:<aicall_id> <turnPipecatcallID>`, self-expiring
     TTL) — so the predicate becomes a direct membership check instead of
     an inference from what it *isn't*:
     ```go
     // New in rev 7: cheap, cache-safe pre-filter before paying for a
     // fresh DB read at all — c.ReferenceType is immutable (never among
     // AIcallUpdate's written fields, pkg/dbhandler/aicall.go), so the
     // cached copy is trustworthy for this one check regardless of
     // staleness elsewhere. Also closes review round 6's F2: without
     // this gate, every tool call on every AIcall type (all 21
     // mapFunctions handlers, not just contact_case) would pay an
     // uncached read that only ever matters for contact_case.
     //
     // Placement (review round 7, finding N-7): this runs between
     // ToolHandle's existing h.Get (tool.go:33) and its first
     // messageHandler.Create (tool.go:47) — i.e. before any row for this
     // tool call is written, so there is no already-persisted row left
     // orphaned if this returns an error below.
     listenTurn := false
     if c.ReferenceType == aicall.ReferenceTypeContactCase {
         isMember, errMember := cache.ListenTurnPipecatcallIDIsMember(ctx, c.ID, pipecatcallID) // SISMEMBER ai:listen:turnpcid:<aicall_id> <pipecatcallID>
         if errMember != nil {
             // Degrade, don't fail closed — corrected in rev 8, review
             // round 7 finding N-1. A Redis outage means §5.4.3's SADD and
             // §5.3.4's debounce lock are ALSO failing, so no genuine
             // listen turn can be running platform-wide at that moment —
             // every tool call arriving during a Redis outage is
             // structurally a real Q&A call. Failing the tool call closed
             // here would make an Insight Q&A session's ordinary tool use
             // (get_call_transcript, etc.) go down with Redis, contradicting
             // §6's "Redis unavailable → Q&A is completely unaffected"
             // row. listenTurn=false is therefore not a guess; it is the
             // provably correct value under this specific failure.
             log.Warnf("listen-turn membership check failed, assuming a real Q&A turn: %v", errMember)
             promListenMembershipCheckFailedTotal.Inc()
         } else {
             listenTurn = isMember
         }
     }
     ```
     This is a direct positive test — pipecatcall A from the race above
     was never `SADD`'d as a listen-turn id, so `isMember` is `false`
     regardless of what `c.PipecatcallID` happens to be at read time. The
     fresh, cache-bypassing `AIcallGet` rev 6 introduced for this
     decision is **no longer needed at all**: Redis set membership is
     already the authoritative, race-free source `AIcallGet` was being
     used as a proxy for. `toolHandleNotifyAgent` (§5.4.4(c)) takes the
     resulting `listenTurn bool` as a parameter, unchanged from rev 6.

   `Origin = OriginListenInternal` when `listenTurn`, `Origin = OriginNone`
   otherwise — unchanged from today for every AIcall this feature doesn't
   touch, and now also unchanged for a `contact_case` AIcall that simply
   isn't currently listening. The proactive `notify_agent` output row
   itself keeps `Origin = OriginProactive` (§5.6.2) — it is real
   conversational content the agent should see and the AI should
   remember, unlike the mechanical tool-call/result exchange that
   produced it.
3. **Mechanism and location corrected in rev 5, review round 4 finding
   B3.** `ApplyFields` does **not** live in `bin-ai-manager` — it is
   `bin-common-handler/pkg/databasehandler/main.go:61-110`, shared by
   every service in the monorepo (`bin-ai-manager`, `bin-call-manager`,
   `bin-message-manager`, …), and today it only ever builds
   `squirrel.Eq{...}` per field, with one hardcoded special case for the
   `"deleted"` key (`main.go:76-85`). Passing a `FieldOriginNot` key
   straight through would build `squirrel.Eq{"origin_not": ...}` — a SQL
   error on a nonexistent column, on every single `getPipecatcallMessages`
   call.

   **Decision: add a typed exclusion wrapper to `databasehandler`, not
   another hardcoded field-name special case** (the `"deleted"` special
   case is not a pattern to extend — see `bin-common-handler/CLAUDE.md`'s
   3-service admission rule, which already governs what this shared
   package may grow):
   ```go
   // bin-common-handler/pkg/databasehandler/main.go
   // NotEq wraps a filter value to signal "!=" instead of ApplyFields'
   // default "=". Generic in principle (any field, any service, not a
   // string-keyed special case like "deleted") — but see the scope note
   // below before treating it as safe for every value type.
   type NotEq struct{ Value any }

   // inside ApplyFields' per-field switch:
   if ne, ok := value.(NotEq); ok {
       sb = sb.Where(squirrel.NotEq{string(field): ne.Value})
       continue
   }
   ```
   **Scope note, review round 5 finding M3: this snippet is correct for
   `Origin` (a string-kind value), and this design uses it for nothing
   else — it is not a claim that `NotEq` handles every value type
   `ApplyFields` handles today.** `ApplyFields`'s existing per-field
   switch already does type-specific normalization before building `Eq`
   — a `uuid.UUID` value is converted via `.Bytes()` before comparison
   (`main.go:67-68`), and `bool` gets the `"deleted"` special case
   (`main.go:76-85`) — and the bare `squirrel.NotEq{string(field):
   ne.Value}` above bypasses both. Implementation should apply this
   design's `NotEq` only to `Origin`-shaped (string) fields until/unless
   it is extended to route through the same per-type normalization `Eq`
   already gets, since `bin-common-handler` is consumed monorepo-wide and
   a `NotEq{Value: someUUID}` from an unrelated service would silently
   compare against the wrong byte representation.
   `getPipecatcallMessages` then passes
   `{FieldAIcallID: c.ID, FieldOrigin: databasehandler.NotEq{Value:
   message.OriginListenInternal}}` — no new `Field` constant needed
   (`FieldOrigin` — added by §5.8 alongside the `Origin` field itself, and
reused as-is here rather than needing a second constant, since `Field`
values are just string keys and the wrapper type is what changes the
comparison, not the field name),
   and no `FieldStruct`/`ConvertFilters` allowlist entry either (review
   round 4 finding M3: `FieldStruct` only gates what an *external RPC
   caller* may filter by, `pkg/listenhandler/v1_messages.go:45` —
   `getPipecatcallMessages` builds its filter map directly in Go code, so
   exposing an `origin_not` RPC filter would be pure unused surface area,
   not a requirement). Being a `bin-common-handler` change, it needs the
   full monorepo-wide verification workflow for every consumer, not just
   the services this design otherwise touches (§8).
4. **New in rev 5 (review round 4 finding B4), corrected in rev 6 (review
   round 5 finding H2): excluding listen-internal rows narrows the
   eviction risk but does not eliminate it** — rev 4's own worked example
   (60 turns × 2 tagged rows = 120 rows evicting the system prompt)
   applies just as well to 60 *proactive* rows (never tagged, since they
   are real content) plus the agent's own Q&A tool rows (also never
   tagged). The fix has to guarantee the system prompt specifically, not
   merely reduce the row count competing for the window.
   `getPipecatcallMessages` (`start.go:620-661`) is restructured into two
   fetches instead of one:
   ```go
   // Both fetches use dbhandler.MessageList's actual, unchanged behaviour
   // (message.go:205-216): WHERE tm_create < token ORDER BY tm_create
   // DESC LIMIT size — i.e. "newest N", newest-first, same as today.
   // Rev 5's description of fetch (1) as "oldest first" was wrong; fixed
   // here, and the reasoning about InsightSystemPrompt corrected too —
   // startInitMessages DOES write it as the first row for exactly this
   // (Insight, contact_case) path (start.go:792-793), not "not applicable
   // here" as rev 5 claimed.

   // (1) The system row(s) for this AIcall. In production there are never
   //     more than 3 (InsightSystemPrompt + substituted init_prompt +
   //     optional parameter-JSON, all written once at AIcall creation and
   //     never again — startInitMessages, start.go:790-819), so "newest 5"
   //     and "all of them" are the same fetch; size=5 is headroom, not a
   //     truncation risk. Independent of the capped window in (2), so it
   //     can never be evicted no matter how much conversation follows.
   systemRowsDesc, _ := h.messageHandler.List(ctx, 5, "", map[message.Field]any{
       message.FieldAIcallID: c.ID,
       message.FieldRole:     message.RoleSystem,
   })

   // (2) The newest 100 non-system, non-listen-internal rows — Q&A
   //     history and proactive notifications, exactly as before, minus
   //     §5.4.5's listen-internal exclusion.
   restDesc, _ := h.messageHandler.List(ctx, 100, "", map[message.Field]any{
       message.FieldAIcallID: c.ID,
       message.FieldRole:     databasehandler.NotEq{Value: message.RoleSystem},
       message.FieldOrigin:   databasehandler.NotEq{Value: message.OriginListenInternal},
   })

   // Both results are newest-first; reverse BOTH the same way start.go's
   // existing single-fetch path already does (start.go:633-635) before
   // use, then concatenate: reverse(systemRowsDesc) ++ reverse(restDesc).
   // This is the only structural change from today's single-fetch,
   // single-reversal shape — two fetches, two reversals, one concat.
   ```
   This guarantees the AI's own instructions are never lost regardless of
   conversation or notification volume — the thing rev 1's B2, and its
   rev-4 recurrence, were both actually about. The 100-row cap on *the
   rest* still means very old Q&A history can eventually be pushed out by
   a long call full of proactive notifications, same as any capped window
   already accepts for an unusually long conversation; that is a bounded,
   pre-existing trade-off, not the defect being fixed here.

**This also resolves review round 3's B2** (the "orphaned `tool`-role
message" finding, §5.6.3, §11 item 3), for the instances this feature
itself creates: a listen turn's tool-call/tool-result rows are now
permanently excluded from ever being replayed into any future context —
`getPipecatcallMessages` (step 4 above) and the listen turn's own
explicit assembly (§5.4.2, which never reads message rows for this
portion in the first place). They cannot become "orphaned" in a context
they are never placed into. **§11 item 3 is narrowed accordingly**: the
general defect (an agent-initiated tool call's own tool-call row already
gets filtered by `run.py:450`'s empty-content check today, independent of
listening) is still real and still worth confirming, but it is no longer
gated on or made worse by this feature shipping — restored to a
follow-up-ticket item rather than a rollout blocker.

**Known remaining gap, stated rather than silently left (review round 4
finding M7): `get_aicall_messages`.** This existing tool
(`toolHandleGetAIcallMessages`, `tool.go:761-763`) reads up to 1000
messages by `FieldAIcallID` alone and hands them to the LLM verbatim — it
does not go through `getPipecatcallMessages` and is therefore unaffected
by this section's `NotEq` filtering. A `contact_case` AIcall's `Origin =
OriginListenInternal` rows can reach the LLM through this tool. This is
lower severity than the context-eviction defect (it is a content-leak of
mechanical tool-call JSON into a Q&A answer, not a lost system prompt or
lost history), and is left as a follow-up rather than fixed in this
design — recorded in §11.

### 5.5 The `notify_agent` tool

#### 5.5.1 Definition

```go
{
    Name:   tool.ToolNameNotifyAgent,
    RunLLM: false,   // deliberate: the notification IS the output; no follow-up text
    Description: `Pushes a short, actionable note to the human agent's Insight
Assistant panel, without the agent having asked anything.

WHEN TO USE:
- You are watching a live call transcript and something just happened that the
  agent needs to know right now, per your configured instructions.

WHEN NOT TO USE:
- The agent asked you a question — answer normally instead; do not call this.
- You have nothing new or actionable to say. Saying nothing is the correct and
  expected outcome for most checks.
- You want to repeat something you already notified about on this call.

ARGUMENTS:
- message (required): one or two sentences, written for a busy human mid-call.

This is the only way to reach the agent proactively. It writes into the same
panel thread the agent is already reading; it cannot place calls, send email or
SMS, change CRM records, or spend money.`,
    Parameters: map[string]any{
        "type": "object",
        "properties": map[string]any{
            "message": map[string]any{
                "type":        "string",
                "description": "The note to show the agent. One or two sentences.",
            },
        },
        "required": []string{"message"},
    },
}
```

Registration: `tool.ToolNameNotifyAgent` is added to
`tool.AllInsightToolNames` (`models/tool/main.go:65-72`) and **not** to
`tool.AllToolNames`. Gating then works through the existing, verified
mechanism: `amai.AllowedToolNames(TypeInsight)`
(`models/ai/tool_validation.go:36-47`) re-applied at expansion time in
`bin-pipecat-manager/pkg/toolhandler/GetByNames`
(`toolhandler/main.go:91-108`). Also needs
`message.FunctionCallNameNotifyAgent` (`models/message/tool.go`) and an
entry in `ToolHandle`'s `mapFunctions` (`aicallhandler/tool.go:54-76`).

**OpenAPI surface — missing from rev 4, added in rev 5 (review round 4
finding H3).** The tool-name vocabulary is also public API surface, not
just an internal Go constant: `bin-openapi-manager/openapi/openapi.yaml`
carries a tool-name enum, and `paths/ais/main.yaml` /
`paths/ais/id.yaml` document the Insight-allowed tool list in prose —
both were updated for every prior Insight tool (`get_call_transcript`,
etc.) and must be updated here too, followed by the standard
`go generate ./...` regen in both `bin-openapi-manager` and
`bin-api-manager`. Listed explicitly in §9, not left implicit under
"`Origin` in the spec."

#### 5.5.2 Relaxing the Insight read-only invariant (resolves review H1)

`models/tool/main.go:62-64` currently says *"Every entry MUST be read-only
(no side effects)"*, and
`models/ai/allowed_tools_test.go:72-88`
(`TestAllInsightToolNamesAreReadOnly`) hard-fails on any name missing from
its hardcoded `knownReadOnly` map. `notify_agent` persists and delivers a
message. **This design deliberately relaxes that invariant, narrowly:**

- The comment on `AllInsightToolNames` is rewritten to: *"Every entry must
  be read-only with respect to customer data and external systems. The
  single sanctioned exception is `notify_agent`, whose only effect is to
  write a message into the AIcall's own conversation thread — the same
  thread the agent is already reading. It cannot place calls, send
  email/SMS, mutate CRM records, or spend money. See
  docs/plans/2026-09-03-insight-ai-realtime-listen-design.md §5.5.2."*
- The test keeps its name and its fail-loudly property but gains a second,
  separate map so that any *other* write tool still fails:

  ```go
  knownReadOnly := map[tool.ToolName]bool{ /* the existing six */ }
  // Sanctioned write exception -- see the design doc cited above. Adding a
  // name here requires the same explicit design-level justification.
  knownSanctionedWrite := map[tool.ToolName]bool{
      tool.ToolNameNotifyAgent: true,
  }
  for _, n := range tool.AllInsightToolNames {
      if !knownReadOnly[n] && !knownSanctionedWrite[n] { t.Errorf(...) }
  }
  ```
- `docs/plans/2026-07-30-case-insight-assistant-tool-expansion-design.md`
  §2.6 gains a one-line pointer noting the exception, so the two documents
  do not silently disagree.
- `TestValidateToolNames_WriteToolNeverAllowedForInsight`
  (`allowed_tools_test.go:94-108`) needs **no** change: it iterates
  `tool.AllToolNames`, and `notify_agent` is not a member.

**Auto-grant blast radius, acknowledged:** Insight AIs typically store
`tool_names=["all"]`, which `AllowedToolNames` expands at runtime, so every
existing Insight AI gains `notify_agent` on deploy with no re-consent
step. **Corrected in rev 3**: the worst case if a model calls it during an
ordinary Q&A turn is *not* "one extra harmless message" — §5.4.4(c) now
rejects that call outright, precisely because rev 2's "harmless" framing
was wrong (§5.4.4(c)'s own rationale). So the actual worst case is a
failed tool call the agent never sees (it just gets its real answer,
unaffected) — no external action, no spend, no data change, and no
silently-eaten answer either. The tool description still explicitly
tells the model not to do this; (c) is the backstop for when the
description is not followed.

#### 5.5.3 Customer-configurable triggering

The Insight AI's `init_prompt` (existing field, existing editing UI) is
where the customer defines *when* to notify, e.g. *"if the customer
mentions cancellation, a compliance keyword, or requests something
requiring approval, call notify_agent with a short actionable note;
otherwise say nothing."* The frozen snapshot of that prompt is message #1
of every listen turn (§5.4.2), so no schema change and no new field is
needed to make triggering customer-defined — directly per 대표님's
direction.

`ListenTurnSystemPrompt` supplies only the mechanics (you are watching a
live call; most checks should produce no output; use `notify_agent` and
nothing else to reach the agent; do not repeat a notification you already
sent), never the business conditions.

### 5.6 How a proactive message is stored (resolves review H2)

#### 5.6.1 Rejecting `role=notification`

`message.RoleNotification` exists (`models/message/main.go:68`), is
produced today only by `EventPMTeamMemberSwitched` (`event.go:348`), and
**is skipped when assembling LLM context**: `getPipecatcallMessages` does
`if m.Role == message.RoleNotification { continue }` with the comment
*"skip non-LLM roles (e.g. notification) that would cause API errors"*
(`start.go:637-641`).

That skip is exactly why it is the **wrong** home for a proactive message.
A proactive notification is a genuine assistant utterance in the
conversation. If it were stored as `notification`, then when the agent
replies *"what did you mean by that?"*, the Q&A turn's context would not
contain what the AI had just told them — the AI would have no memory of
its own notification. That is a functional defect, not a desirable
property. `RoleNotification`'s existing use (a machine-readable
member-switch JSON blob that is genuinely not conversational) does not
generalise here.

Checked on the UI side too: neither panel special-cases roles —
square-admin styles `msg.role === 'user'` one way and everything else the
other (`CaseInsightAssistantPanel.js:44`); square-talk does the same
(`CaseInsightAssistantPanel.jsx:62`). So `role=notification` would give no
UI distinction for free anyway.

#### 5.6.2 Decision: `role=assistant` + a new `Message.Origin` field

```go
// models/message/main.go
Origin Origin `json:"origin,omitempty" db:"origin"`

type Origin string
const (
    OriginNone      Origin = ""          // default; every message that answers or asks
    OriginProactive Origin = "proactive"  // AI-initiated, not a reply to anything
)
```

- `role=assistant` → the AI remembers what it told the agent; ordinary LLM
  context replay is correct with no special-casing.
- `Origin` is an explicit, first-class marker written by ai-manager itself,
  which is possible here precisely because the row is created by
  `ToolHandle` — not, as rev 1 assumed, reconstructed from tool-call
  archaeology. Rev 1's premise was wrong: the eventual assistant *text*
  reply is created independently in `EventPMMessageBotLLM` with
  `toolCalls=nil` and no correlation column back to the tool call, so
  "did this message's generation include a `notify_agent` call" is not
  answerable from the schema.
- A string enum rather than a bool, matching `Role` / `Direction` /
  `DeliveryStatus`, so a future third origin does not need another column.
- Exposed in `message.WebhookMessage` + `ConvertWebhookMessage` — the
  frontends key their badge off it (§5.10).

#### 5.6.3 Message ordering — re-diagnosed in rev 3; surfaces a pre-existing production defect

Rev 2 claimed creating the proactive row *inside* `ToolHandle`'s step 2
would produce `assistant(tool_calls) → assistant(text) → tool(result)`,
which OpenAI rejects unless the `tool_calls` message is immediately
followed by its results — and "fixed" this by moving the proactive row to
*after* the tool-result row. **Review round 2 (finding F2) showed this
diagnosis was wrong, and the fix was consequently a no-op.** Re-verified
directly against the code for rev 3:

- `ToolHandle` creates the tool-call row with **empty content**:
  `h.messageHandler.Create(ctx, uuid.Nil, c.CustomerID, c.ID, c.ActiveflowID,
  message.DirectionIncoming, message.RoleAssistant, "", []message.ToolCall{*tool}, ...)`
  (`tool.go:47` — the `""` is the `content` argument).
- `bin-pipecat-manager/scripts/pipecat/run.py:450` builds the replayed
  context as `valid_messages = [m for m in messages if m.get("role") and
  m.get("content")]` — an empty-string `content` is falsy in Python, so
  **the `assistant(tool_calls)` row is filtered out of the context before
  it ever reaches `LLMContext`**, regardless of which row was created
  first or second on the DB side. The same filter runs on the
  team-conversation path (`run.py:637`). Rev 2's reordering therefore
  changes nothing about what pipecat-manager actually sends to the LLM —
  **both orderings produce byte-identical payloads.**
- What *does* reach the LLM, because it has real content, is the
  `role=tool` result row (`toolCreateResultMessage`, `tool.go:88`) — with
  **no preceding `tool_calls` entry in the same context**, because that
  entry was just filtered out. This is the actual OpenAI-API-incompatible
  shape (a `tool`-role message must reference a `tool_calls` entry present
  in the same request), just the reverse of what rev 2 diagnosed.

**This is not a defect this feature introduces.** Every existing Insight
tool (`get_call_transcript`, `get_contact_profile`, …) follows the exact
same `ToolHandle` path today, so **any multi-turn `contact_case` Q&A
session where the agent's first question triggers a tool call, and the
agent then asks a follow-up question, already replays this orphaned
`tool`-role row into the follow-up's context** — independent of listen,
independent of `notify_agent`. This design does not change that shape for
`notify_agent`'s own tool-call/result pair (they follow the identical
`ToolHandle` sequence as every other tool), and does not attempt to fix
it here: fixing `run.py:450`'s content-truthiness filter (or giving the
tool-call row non-empty placeholder content) is a change to shared
`ToolHandle`/pipecat-manager behaviour affecting every AI tool call
platform-wide, not something scoped to Insight listening.

**Action item, separate from this design (§11 item 3, escalated in rev
3 from "follow-up" to "recommend filing and investigating immediately"):**
confirm empirically whether this is actually causing production failures
today (it may be masked if OpenAI's Chat Completions API is in practice
lenient about an orphaned `tool` message, or if few real sessions exercise
a tool call followed by a genuine follow-up question on the same AIcall —
that needs checking, not assuming). If confirmed, it is a platform-wide
correctness bug predating and independent of this feature and should be
triaged on its own ticket, at whatever urgency the confirmation warrants.

The proactive `Origin=proactive` row itself needs no special ordering
relative to the tool-call/result pair — it is a wholly separate `Message`
row, written by `toolHandleNotifyAgent` (§5.5) once it validates the
argument, with no OpenAI-payload constraint linking it to the tool-call
sequence above.

#### 5.6.4 What actually surfaces per `notify_agent` call (new in rev 3, resolves review-round-2 F7)

Rev 2 asserted a single new row per notification. **That is wrong.**
`ToolHandle` (`aicallhandler/tool.go:24-100`) writes, for **every** tool
invocation including `notify_agent`:

1. `role=assistant`, `content=""`, carrying `ToolCalls` (`tool.go:47`),
2. `role=tool`, the raw JSON result (`tool.go:88` →
   `toolCreateResultMessage`),
3. — and only for `notify_agent` — the new `role=assistant`,
   `Origin=proactive` row (§5.6.2).

`messageHandler.Create` publishes `aimessage_created` to the customer's
(tenant's) configured webhook **unconditionally** for every row
(`pkg/messagehandler/db.go:81` → `notifyhandler/publish.go:24-26`), and
neither panel special-cases roles when rendering — square-admin styles on
`msg.role === 'user'` and treats everything else uniformly
(`CaseInsightAssistantPanel.js:44`); square-talk does the same
(`.jsx:62`). So **one proactive notification is, today, three
webhook-published, panel-rendered rows**: an empty bubble, a raw-JSON
blob, and the intended note.

This is **not new to this feature** — every existing Insight tool call
already produces rows 1 and 2 and already surfaces them the same way; it
predates this design (same root cause as §5.6.3's finding: `ToolHandle`'s
two-row-per-tool-call shape). This design makes it materially more
visible because listening is the first thing that can trigger a tool call
*without the agent having asked for it*, so the noise now appears
unprompted mid-call, not as a byproduct of a question the agent typed.

**Mitigation shipped with this design (frontend, §5.10.1): filter, don't
suppress.** Both `CaseInsightAssistantPanel` components stop rendering a
message if `msg.role === 'tool'` or (`msg.role === 'assistant' &&
!msg.content && msg.tool_calls?.length`) — **field names corrected in
rev 4, review round 3 finding H3**: `WebhookMessage`'s wire field is
`tool_calls` (`models/message/webhook.go:23`,
`json:"tool_calls,omitempty"`), snake_case like every other field the
panels already read (`msg.role`, `msg.content`, `msg.tm_create` —
`CaseInsightAssistantPanel.js:44,46,47`); rev 3's `toolCalls` (camelCase)
would never have matched and the filter would silently never have fired.
This is a generically useful filter, not scoped to listening, since it
also cleans up every existing Insight Q&A tool call's panel noise today
(scope note: it does **not** hide the `role='system'` rows
`startInitMessages` writes at AIcall creation, `start.go:812-819` — those
predate this feature and are a separate, out-of-scope cleanup). It is a
client-side render filter only; it does **not** touch the webhook
delivery, `Create`, or `PublishWebhookEvent` — those keep firing for rows
1 and 2 exactly as before, so a tenant's own webhook-consuming automation
still sees every tool-call row it does today. Suppressing *those*
webhooks is a separate, larger decision (whether tenants rely on
tool-call webhooks for their own automation is unknown) and is recorded
as a follow-up (§11 item 6), not attempted here.

### 5.7 Lifecycle and cleanup

#### 5.7.1 Call hangup (resolves review B4 and simplifies B5)

**Lookup.** `EventCMCallHangup` currently resolves via
`GetByReferenceID(evt.ID)` (`aicallhandler/event.go:53-58`). For an
Insight AIcall `ReferenceID` is the **Case** id, so that lookup can never
find the listening AIcall from a call id. Rev 1's "add one more check
here" does not work.

A genuinely new lookup is required:

```go
func (h *aicallHandler) EventCMCallHangup(ctx context.Context, evt *cmcall.Call) {
    // existing path, unchanged: the AIcall whose reference IS this call
    if cc, err := h.GetByReferenceID(ctx, evt.ID); err == nil {
        _, _ = h.ProcessTerminate(ctx, cc.ID)
    }

    // new path: every contact_case AIcall listening to this call
    h.stopListenByCallID(ctx, evt.ID)
}
```

`stopListenByCallID` runs `AIcallList` with
`{FieldReferenceType: contact_case, FieldListenCallID: evt.ID,
FieldDeleted: false}` — hence the indexed **column** in §5.8 — and clears
each match. Plural on purpose: two Cases on one call each get their own
AIcall (see §5.11), and both must be cleared.

**STT stop: not ai-manager's job on this path — reasoning corrected in
rev 3.** `bin-transcribe-manager/pkg/transcribehandler/event.go:51-81`
(`EventCMCallHangup`) does list every non-deleted transcribe with
`reference_id == call.ID`, owner-agnostic, and call `Stop` on each — but
**review round 2 (finding F8) showed the DB-status-flip half of that path
is pod-local, not the platform-wide guarantee rev 2 stated.** `Stop` →
`stopLive` → `streamingHandler.Stop` reads the **per-pod in-memory**
`mapStreaming`; on whichever pod happens to consume the shared hangup
subscribe queue (`cmd/transcribe-manager/main.go:201`) that is *not* the
session's owning pod, the in-memory lookup misses,
`isSafeToConsiderStopped` treats that as "already gone," the physical
streaming stop is skipped, and `UpdateStatus(StatusDone)` is written
anyway — regardless of whether the physical STT stream actually stopped.
The safety comment at `stop.go:155-166` that justifies this branch
explicitly assumes routed-to-owning-pod delivery (*"the RPC can only ever
reach the pod identified by the transcribe's `HostID`"*), which is true of
the direct `TranscribeV1TranscribeStop`/health-check RPCs but **not** of
this in-process hangup-event path.

The *actual* backstop this design relies on is simpler and does not
depend on that DB write: hanging up the call closes Asterisk's external-
media WebSocket connection that was feeding the streaming session's audio
(`bin-transcribe-manager/CLAUDE.md`'s own description of the transport —
"Go dials out to Asterisk's `chan_websocket` endpoint... raw 8kHz slin
binary frames"). Once that socket closes, the STT read loop ends and
billing for that stream stops, independent of whether `ai_transcribes`'s
row shows `progressing` or `done` on a non-owning pod. **This is a
pre-existing property of every transcribe session on the platform today,
not something this design introduces or changes** — listening rides the
same guarantee every flow-driven and summary-driven transcribe already
relies on. ai-manager's hangup path therefore still issues no stop RPC (a
persisted `HostID` could address a queue that no longer exists after a
transcribe-manager restart, per `bin-transcribe-manager/CLAUDE.md`), but
the justification is "the audio transport itself terminates," not "the DB
row is guaranteed accurate."

Before returning, `stopListenByCallID` runs one final flush turn
(`RunListenTurn`, bypassing the debounce lock) if `pending` is non-empty,
so the last words of the call are still evaluated. Then it clears state.

#### 5.7.2 AIcall terminates while the call is still live

Agent closes the panel, session idles out, or the turn cap (§5.4.1) trips.
Hooked into `ProcessTerminate` (`pkg/aicallhandler/process.go:38`) for
`contact_case` AIcalls. Here ai-manager *does* own the stop:

```go
if !owns { /* someone else's session (e.g. a concurrent ai_summary) — never touch it */ }
else {
    tr, err := h.reqHandler.TranscribeV1TranscribeGet(ctx, listenTranscribeID)  // shared queue, always reachable
    if err == nil && tr.Status == tmtranscribe.StatusProgressing {
        // signature verified: (ctx, hostID, transcribeID)
        //   bin-common-handler/pkg/requesthandler/transcribe_transcribes.go:113-117
        if _, errStop := h.reqHandler.TranscribeV1TranscribeStop(ctx, tr.HostID, tr.ID); errStop != nil {
            log.Warnf(...)   // NOT fatal — see below
            promListenStopFailed.Inc()
        }
    }
}
```

`HostID` is fetched **fresh** via `TranscribeV1TranscribeGet` rather than
read from a persisted column, precisely because it is regenerated on every
transcribe-manager restart.

**Stated fallback, rather than asserting cleanup always succeeds:** if the
owning pod restarted, the per-pod queue
`bin-manager.transcribe-manager-<host_id>.request` no longer exists and
the stop RPC times out. That is logged, metered, and tolerated — the
session's audio transport is guaranteed to end when the call itself ends
(§5.7.1's corrected reasoning), which is at most one call-duration away.
The failure mode is a slightly-longer-than-necessary STT session, never a
permanently orphaned one.

**Second-order consequence of §5.7.1's correction, noted here rather than
hidden:** the `tr.Status == tmtranscribe.StatusProgressing` gate above can
itself already read `done` on a call that hung up moments earlier but
whose STT-stop RPC never reached the owning pod (§5.7.1) — in which case
this branch is simply skipped, which is the correct outcome (nothing left
to stop).

#### 5.7.3 Clearing state (all paths)

`clearListenState(ctx, aicallID)` — **step order corrected in rev 4,
review round 3 finding M1**: rev 3's numbered steps cleared the AIcall's
`listen_transcribe_id` metadata *before* using that same value to `SREM`
the resolver set, which cannot work (the value needed for step 2 no
longer exists once step 1 runs). Actual order:
1. Read `transcribeID := c.Metadata[listen_transcribe_id]` from the
   AIcall already in hand (no extra fetch — `clearListenState`'s callers
   already hold `c`).
2. Redis: `SREM ai:listen:transcribe:<transcribeID> <aicallID>` (§5.2.4's
   set fix — removes only this AIcall's membership; Redis deletes the key
   itself once the set empties, so a shared transcribe stays resolvable
   for whichever AIcall(s) are still listening to it), plus
   `DEL ai:listen:pending:<aicallID>`, `ai:listen:window:<aicallID>`,
   `ai:listen:lock:<aicallID>`, `ai:listen:turns:<aicallID>` (these four
   are per-AIcall, never shared, so a plain `DEL` is correct for them).
   **Not included, deliberately (review round 7, finding N-6): any
   `ai:listen:turnpcid:<aicallID>` entries §5.4.3 registered.** They are
   short-TTL (§5.12) and self-expiring by design, and leaving a stale one
   past this stop causes no incorrect behaviour — a tool call arriving
   late for an already-stopped listen turn still correctly resolves
   `listenTurn=true` and gets `Origin=OriginListenInternal`, exactly as
   it should for a row that genuinely came from that turn.
3. `AIcallUpdate` → `listen_call_id = uuid.Nil`, remove both metadata keys
   (one write) — last, since nothing downstream needs the old value once
   step 2 has consumed it. **Uses the `tm_update`-bypassing write variant
   (§5.2.4), same as the start-time `UpdateListenState` write** — named
   explicitly here per review round 11 finding LOW-4, since this section
   previously said only "`AIcallUpdate`" and left the bypass implicit.

Removing this AIcall's set membership before clearing the DB metadata is
what guarantees a stale `(transcribe_id, aicall_id)` pairing can never be
matched again by §5.3.

### 5.8 Data model and plumbing scope (resolves review M2)

The rev-1 proposal of three columns is reduced to **one column + two
metadata keys**, on the principle that only a field we must *query by*
needs to be a column.

| Field | Where | Why |
|---|---|---|
| `listen_call_id` | **new column** `ai_aicalls.listen_call_id binary(16)`, **indexed** | `EventCMCallHangup` must run `WHERE listen_call_id = ?` (§5.7.1). JSON metadata is not usefully indexable |
| `listen_transcribe_id` | `AIcall.Metadata["listen_transcribe_id"]` | only ever read with the row already in hand (idempotency check, stop path). Resolution in the hot path goes through Redis, not a DB query, so no index is needed |
| `listen_owns_transcribe` | `AIcall.Metadata["listen_owns_transcribe"]` | same |

`Metadata` is the established home for per-AIcall flags — see
`MetaKeyPromptSnapshots` / `MetaKeyAutoAuditEnabled`
(`models/aicall/main.go:21-27`). Two new constants
`MetaKeyListenTranscribeID`, `MetaKeyListenOwnsTranscribe` follow that
pattern.

**On `ai_aicalls.transcribe_id` having existed before:** migration
`bad27b40fe8e` deliberately dropped a `transcribe_id` column from
`ai_aicalls` (per `docs/plans/2026-02-24-aicall-schema-cleanup-design.md`
§3). This design does **not** re-add it. The old column was the
chatbot-era per-AIcall transcribe binding for a feature that no longer
exists; the transcribe id now lives in `Metadata`. What *is* added is a
different thing: a **call** reference, needed for an event lookup that has
no other key.

Full plumbing checklist for the one column:

- `models/aicall/main.go` — field, `json:"listen_call_id,omitempty" db:"listen_call_id,uuid"`
- `models/aicall/field.go` — `FieldListenCallID Field = "listen_call_id"`
- `models/aicall/field_test.go` — golden constant list
- `models/aicall/filters.go` — `FieldStruct` entry (`filter:"listen_call_id"`),
  required for `AIcallList` filtering via `ApplyFields`
  (`dbhandler/aicall.go:206`)
- `models/aicall/filters_test.go`
- **not** added to `models/aicall/webhook.go` / `ConvertWebhookMessage` —
  internal plumbing, same treatment as `Message.PipecatcallID`
  (`models/message/main.go:26`, `json:"-"`). No OpenAPI change, no
  `aicall_struct_aicall.rst` change follows from it.
- `bin-dbscheme-manager` migration, **generated** with
  `alembic -c alembic.ini revision -m "..."` (never a hand-picked revision
  id), adding the column and `create index idx_ai_aicalls_listen_call_id
  on ai_aicalls(listen_call_id)`. AI drafts the file; a human applies it.
- `bin-ai-manager/docs/domain.md` — AIcall entity + Metadata keys.
- `bin-ai-manager/docs/architecture.md` — the new subscription
  (`subscribeTargets` / `topicPatterns` change triggers the
  `scripts/check-service-docs.sh` warning).
- `bin-ai-manager/docs/operations.md` — new config flags and metrics.

For `Message.Origin` (§5.6.2), which **is** user-visible:

- `models/message/main.go` (field + `Origin` type), `field.go`,
  `field_test.go`, `filters.go`
- `models/message/webhook.go` + `ConvertWebhookMessage`
- `bin-dbscheme-manager` migration:
  `ai_messages.origin varchar(16) not null default ''`
- `bin-openapi-manager/openapi/openapi.yaml` → `go generate ./...`, then
  `bin-api-manager` → `go generate ./...`
- RST (§5.10.2)

For the new `POST /service_agents/aicalls/{id}/listen` endpoint (§5.1,
**new in rev 15, endpoint surface corrected in rev 16**), user-visible (a
new public API surface):

- `bin-ai-manager/pkg/listenhandler/main.go` — `regV1AIcallsIDListen`,
  route table entry, `processV1AIcallsIDListenPost` (the internal
  ai-manager route stays at the plain `/v1/aicalls/{id}/listen` path —
  §5.1 — it is only the public api-manager path that moved)
- `bin-ai-manager/pkg/aicallhandler/{main,start}.go` — `ProcessListen`
  (single exported method, rev 16), the `Start` hook removed (§5.1.1)
- `bin-common-handler/pkg/requesthandler/ai_aicalls.go` + mock — `AIV1AIcallListen`
  (with its own longer RPC timeout, §5.1)
- `bin-openapi-manager/openapi/paths/service_agents/aicalls/id_listen.yaml`
  (new file, mirroring `service_agents/contact_addresses/id_claim.yaml`)
  → `go generate ./...` in both `bin-openapi-manager` and `bin-api-manager`
- `bin-api-manager/server/service_agents_aicalls.go` — `PostServiceAgentsAicallsIdListen`
- `bin-api-manager/pkg/servicehandler/serviceagent_aicall.go` + interface
  in `main.go` — `ServiceAgentAIcallListen`
- RST: `ai_overview.rst`'s new "Insight Assistant: live call listening"
  subsection (§5.10.2) gains the endpoint's method/path/response shape;
  no new `*_struct_*.rst` file, since the response is the existing
  `AIManagerAIcall`/`WebhookMessage` shape, unchanged by this endpoint
- `bin-ai-manager/pkg/aicallhandler/listen_trigger.go` (§9's own file
  list — **corrected in rev 17, review round 14 finding LOW-5**, which
  found this bullet and §9 disagreeing on which file holds
  `rollbackListenState`) — `rollbackListenState` (§5.2.2's rev-16/17
  rewrite of the ordering fix): an `aicallHandler`-level helper, not a
  cache primitive — it reads `c.ID`/`transcribeID` and issues a targeted
  `AIcallUpdate` (§5.2.2) — so it stays in this file with `ProcessListen`
  and the rest of the trigger's own logic, distinct from the three cache
  primitives below (**this split stated explicitly in rev 20, review
  round 17 finding B-4 — an earlier draft folded all four names into one
  "all in `pkg/cachehandler`" sentence, which was wrong for this one**)
- `pkg/cachehandler/{main,handler}.go` **or the new package (§9's scope
  note)** — `ListenTranscribeAIcallRemove` (§5.2.2's conflict-recovery
  branch — **added to this bullet in rev 20, review round 17 finding
  B-4, to match §9's own placement of it as a cache primitive; an
  earlier draft of this bullet had grouped it with `rollbackListenState`
  above instead**) and `ListenStartLockAcquire`/`ListenStartLockRelease`
  (§5.2.2's per-AIcall lock, added in rev 19, review round 16 finding
  LOW-5) — the `Add` primitive rev 15 introduced is **removed in rev
  16**: its job is now folded into `UpdateListenState`'s own existing
  `SADD`, called earlier (§5.2.4)

`bin-contact-manager` is touched only by the one-line
`kmkase.ReferenceTypeCall` constant (§5.1.1 step 5) and must be run
through the full verification workflow as well.

### 5.9 Speaker mapping — general semantics confirmed empirically; the A-leg/B-leg question closed structurally in rev 11; one empirical gap remains open

**What is now confirmed.** Two independent things back the `in=customer`/
`out=agent` reading, from two different kinds of evidence, plus a third,
stronger invariant found while answering the CEO/CTO's own question this
session (review round 9 pushed this from "assumed" to "reasoned from
code"):

1. **General channel-relative semantics, confirmed against real production
   data, not just documentation.** Traced through
   `bin-transcribe-manager/pkg/transcribehandler/start.go` and
   `docsdev/source/transcribe_tutorial.rst`, then checked against a real
   transcribe sample (`transcribe_id=BEED29D2D0C64848B9899B357E974BB1`)
   pulled directly from the production `bin_manager` database. Stated
   precisely, to avoid the ambiguity round 9 flagged in an earlier
   phrasing: `direction` is relative to the transcribed *channel*'s own
   read/write direction — `in` is the audio Asterisk *reads from* that
   channel (i.e. what the channel's own owner said), `out` is the audio
   Asterisk *writes to* that channel (i.e. what was played to them, which
   in a bridge is whoever else is on the call). This is not merely a
   documentation reading; the production `transcribe_transcripts` rows for
   that session were consistent with it, and it also matches
   `streaminghandler/start.go:67-68`'s snoop *spy* direction and
   `transcribehandler/start.go:268-271`'s split of a `both`-direction
   session into two independent per-direction snoops.
2. **Which leg is transcribed, confirmed in rev 11 (§5.1.1 step 7) —
   narrower than "structural for all of `case_create`."** The precise,
   code-checked claim is: no *system-generated* B-leg flow
   (`generateFlowForAgentCall` or the `connect` action's B-leg flow, both
   cited in §5.1.1 step 7) can ever contain `case_create`, so for the
   flows this design targets, `Case.ReferenceID` cannot name a B-leg
   produced by them.
3. **The stronger invariant, which is what actually closes most of the
   speaker-identity question (§5.1.1 step 7):** `actionHandleCaseCreate`
   only creates a Case when the call's peer is CRM-eligible
   (`isCRMEligiblePeer`/`crmIneligiblePeerTypes`,
   `bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1259-1275,1354`),
   which excludes `agent`/`extension`/`sip`/`conference`/`ai`/`ai_team`/
   `none` peer types outright — and the peer is always resolved as the far
   end of *that call's own channel*
   (`deriveEndpointsForCase`, `:1287-1300`). So **`in` = the listened
   channel's own remote party = `Case.Peer`**, and `case_create` itself
   already guarantees that party is not an internal agent/extension/SIP
   endpoint. This is a stronger, non-circular guarantee than "it happens
   to be the A-leg," and it is what actually bounds point (2) below.

Once the listened leg is bridged to an agent, audio "sent out" through it
(point 1's `out`) is the bridged agent's voice, so `in=customer`/
`out=agent` follows directly.

**What is still open — do not overstate the above as full closure.**

- **The one item rev 1-10 already flagged is still open:** none of (1),
  (2), or (3) is a substitute for capturing one real (or staged)
  **agent-bridged** call's transcript segments and confirming `in`/`out`
  against known speaker identity end-to-end. The production sample in (1)
  confirms the channel-relative *mechanism*; it does not by itself confirm
  that a live agent-bridged session labels the *specific* speaker pair
  correctly. Still blocks §11's sign-off, not the rest of this design.
- **New from rev 11, then narrowed by review round 9's own verification —
  the "click-to-call inverts the mapping" risk as first stated in rev 11
  is very likely impossible, not merely guarded.** Rev 11's first draft of
  this section named an agent-outbound call (e.g. click-to-call dialing
  the agent's own SIP/extension leg) as an unguarded inversion risk. But
  per invariant (3), that exact shape cannot produce a Case at all — the
  agent/extension/SIP peer types are CRM-ineligible, so
  `actionHandleCaseCreate` returns without creating one. The genuine,
  narrower residual vector is different: **an *inbound* call whose peer
  address happens to be CRM-eligible (`tel`/`email`) but is actually staff
  in disguise** — a supervisor or remote agent calling in via a plain DID
  rather than through the agent-dial path, for instance — which
  `case_create` cannot distinguish from a real customer, and which
  inversion would then affect. §5.1.1 step 7's participant-count guard
  does not catch this either (a 2-party bridge still reads as 2 parties).
  This is documented residual risk, not mitigated in this revision — a
  caller-role signal beyond address type would be needed to close it, and
  is out of scope here.
- **New from rev 11, not yet investigated:** call-transfer scenarios via
  `bin-transfer-manager` were not traced this session — whether a
  transfer changes which leg is transcribed, or produces a transient
  3-party bridge, is unconfirmed. §5.1.1 step 7's guard is enforced only
  at listen-start and is **not** re-checked once a listen session is
  live, so a transfer occurring mid-session is unguarded, not merely
  "refused" — see §5.1.1 step 7's closing paragraph and §11 item 12.
- **Unchanged from rev 1-10, precision added in rev 11:** a 3rd party in
  the bridge degrades only the `out` side of the mapping (`out` was never
  more specific than "whoever else is on the call," and a 3rd party makes
  that ambiguous); `in` stays correct regardless, since invariant (3)
  never depended on how many other parties there are. §5.1.1 step 7's
  guard refuses to *start* listening on a stably-non-2-party bridge; it
  has no mechanism to stop an already-listening session if a 3rd party
  joins mid-call (see the point above).

Tags are structural (`[CUSTOMER]`/`[AGENT]`), not localized, so prompt
behaviour does not fork by call language. `transcript.Direction` also
carries a `both` value (`models/transcript/transcript.go:41-45`); a
segment with `direction=both` is tagged `[SPEAKER]` rather than guessed.

### 5.10 Frontend (`monorepo-javascript`)

Correction to the rev-1 review's M6: `square-admin` and `square-talk` are
**not** separate repositories. Both live in the single
`monorepo-javascript` repo
(`/home/pchero/gitvoipbin/monorepo-javascript/square-admin`,
`…/square-talk`). So this feature is **two PRs total**: one in `monorepo`
(backend, lands first), one in `monorepo-javascript` (both apps).

#### 5.10.1 Both panels

- **square-admin** (`src/views/contacts/CaseInsightAssistantPanel.js`):
  already subscribes to `customer_id:{id}:aicall:{aicallId}` and receives
  every new message over the existing WebSocket path — no transport
  change. `MessageThread` currently styles on `msg.role === 'user'`
  (line 44); it gains:
  - a render filter (resolves review-round-2 F7, field names corrected in
    rev 4 per review-round-3 H3, detailed in §5.6.4): skip rendering any
    message with `msg.role === 'tool'`, or `msg.role === 'assistant' &&
    !msg.content && msg.tool_calls?.length`. This is a client-side filter
    only — it hides the two noise rows every tool call (not just
    `notify_agent`) already produces, without touching what the tenant's
    own webhook consumer receives (§5.6.4, §11 item 6);
  - a third branch for `msg.origin === 'proactive'` (distinct surface + a
    `Sparkles`/bell affordance + an accessible label such as "Proactive
    insight"), so a notification is never mistaken for an answer.
- **square-talk** (`src/features/cases/CaseInsightAssistantPanel.jsx`):
  unchanged transport (2s poll); identical render-filter and
  `origin`-driven treatment at line 62.
- No backend read-surface work is needed:
  `ServiceAgentAImessageList` returns `ConvertWebhookMessage()` output
  (`bin-api-manager/pkg/servicehandler/serviceagent_aimessage.go:68-72`),
  so adding `Origin` to `WebhookMessage` is sufficient. Rev 1's open item
  #2 ("does the read surface expose what the badge needs?") is hereby
  **closed**: it does, once the field exists.

#### 5.10.1a Triggering listen explicitly (new in rev 15, replaces the implicit trigger; endpoint path corrected in rev 16)

Both panels now make **two** sequential calls when opening (previously
one): the existing call that creates/reuses the Q&A AIcall (`Start`,
already `POST /service_agents/aicalls` per `ServiceAgentAIcallCreate`),
then a new call to `POST /service_agents/aicalls/{id}/listen` (§5.1 —
**not** the top-level `/v1/aicalls/{id}/listen`; that path is
Admin-console-only and would reject an ordinary agent, §5.1's BLOCKING-1
fix) using the AIcall id `Start`'s response returned. The second call is
fire-and-forget from the frontend's own perspective too — its response
(the current AIcall, §5.1) carries no listening-status field to act on
(§5.1's own scope cut), so the panel does not need to branch on it; a
failed or slow response does not block rendering the panel, and a
repeated call (e.g. a fast double-open) is free (§5.1.1 step 3's
idempotency check, now reached via `checkListenEligible`).

- **square-admin** (`src/views/contacts/CaseInsightAssistantPanel.js`):
  fire the `listen` call immediately after `Start` resolves, in the same
  effect that currently calls `Start`.
- **square-talk** (`src/features/cases/CaseInsightAssistantPanel.jsx`):
  same pattern, same call site as its own `Start` invocation.
- No new WebSocket/poll subscription is needed for this call itself — it
  is a one-shot trigger, not a data source. Any resulting proactive
  message still arrives over each panel's existing message transport
  (WebSocket for square-admin, 2s poll for square-talk), unchanged.

#### 5.10.2 RST docs (resolves review M7)

Mandatory per root `CLAUDE.md` for user-visible changes:

- `bin-api-manager/docsdev/source/ai_struct_message.rst` — new `origin`
  field (compared against `WebhookMessage`, not the internal struct).
- `bin-api-manager/docsdev/source/ai_struct_tool.rst` and/or
  `ai_overview.rst` — the `notify_agent` tool and the Insight-only tool
  set.
- `ai_overview.rst` — a short "Insight Assistant: live call listening"
  subsection: that it is triggered by
  `POST /service_agents/aicalls/{id}/listen` (**updated in rev 15, the
  route itself moved to this path in rev 16 (§5.1's BLOCKING-1 fix), this
  specific mention corrected to match in rev 17, review round 14 finding
  MEDIUM-2** — no longer described as automatic on AIcall creation), that
  it is `init_prompt`-driven, and that proactive messages are marked
  `origin=proactive`.
- New endpoint doc for `POST /service_agents/aicalls/{id}/listen` itself
  (**new in rev 15; the route moved here in rev 16 per review round 13
  finding BLOCKING-1; this specific mention corrected to match in rev
  17, review round 14 finding MEDIUM-2**) — method, path, empty request body, 200 response shape
  (the existing `AIManagerAIcall` struct, already documented), following
  whichever existing pattern documents
  `POST /service_agents/contact_addresses/:id/claim` today (same
  directory, same doc generation path) — **not**
  `POST /v1/aicalls/{id}/terminate`, which is on the Admin-console
  surface this endpoint deliberately is not (§5.1).
- Build procedure: `cd bin-api-manager/docsdev && rm -rf build &&
  python3 -m sphinx -M html source build`, then
  `git add -f bin-api-manager/docsdev/build/`, RST + HTML in the same
  commit.

### 5.11 Cost and concurrency bounds

**Concurrency.** Correcting rev 1: the bound is *not* "one Insight AI per
customer." The DB constraint is one active AIcall per
`(customer_id, reference_type, reference_id)` — i.e. per **Case** — via
the `active_reference_key` unique index (`start.go:359-368`). A customer
with N open Cases on N live calls therefore gets N concurrent listen
sessions. Bounds that actually apply:

| Bound | Value |
|---|---|
| **Transcribe sessions** per call — for listen alone | 1 — any progressing `IDAIManagerListen` session on that call is reused (§5.2.2) |
| **Transcribe sessions** per call — listen + a concurrent `ai_summary` (rare) | up to 2, since rev 4 (§5.2.1) deliberately no longer shares ownership with `ai_summary`'s `IDAIManager` session |
| **STT streams** per call (corrected in rev 3, review-round-2 F11) | 2, not 1 — `DirectionBoth` expands to two independent streamings, one per direction (`transcribehandler/start.go:268-271`, drift from `~216-219` fixed in rev 14/§10.10: `directions := []transcript.Direction{DirectionIn, DirectionOut}`), each its own external-media leg and provider stream. One shared *transcribe session* still means one shared *billing/lifecycle record* (§5.2.2's reuse rule dedupes at that level), but the underlying STT cost is two streams, not one |
| LLM turns per AIcall per minute | `60 / AIcallListenEvaluateIntervalSeconds` = 3 at the default |
| LLM turns per AIcall, total | `AIcallListenMaxTurnsPerAIcall` = 60 (hard stop, then listening ends) |
| Tokens per turn | constant-shaped: 3 system messages (`InsightSystemPrompt` + prompt snapshot + `ListenTurnSystemPrompt`, §5.4.2) + ≤10 Q&A messages + ≤40 transcript lines |
| Concurrent listen sessions | number of open Case panels whose Case call is live |

**Worst case per listened call at defaults:** 3 small LLM turns/min,
capped at 60 turns (~20 min of continuous speech), one shared transcribe
session (two STT streams). Contrast with rev 1, which was one
*unbounded-context* LLM call per spoken sentence.

**Superseded by rev 24 (2026-09-06): the switch was removed; listening is always on.** ~~**Kill switch.** `AIcallListenEnabled` defaults to **false**. The feature
ships dark and is enabled deliberately.~~

### 5.12 New configuration

| Flag / env | Default | Purpose |
|---|---|---|
| `aicall_listen_enabled` / `AICALL_LISTEN_ENABLED` | `false` | master kill switch |
| `aicall_listen_evaluate_interval_seconds` | `20` | debounce window (§5.3.4) |
| `aicall_listen_window_size` | `40` | rolling transcript lines in context |
| `aicall_listen_qa_context_size` | `10` | Q&A rows in context |
| `aicall_listen_max_turns_per_aicall` | `60` | hard per-AIcall turn cap |
| `aicall_listen_buffer_ttl_hours` | `6` | Redis TTL on the buffer/lock/turn-count keys (§5.3.3, §5.3.4, §5.4.1) — **not** `ai:listen:turnpcid:*`, which uses its own, much shorter `aicall_listen_turn_pipecatcall_id_ttl_seconds` (rev 7, review round 6 F1) since a turn-id entry only ever needs to outlive one listen turn |
| `aicall_listen_default_language` | `en-US` | fallback when `STTLanguage` is empty |
| `aicall_listen_turn_pipecatcall_id_ttl_seconds` | `180` | **new in rev 7**: TTL on the `ai:listen:turnpcid:<aicall_id>` set entries (§5.4.3, §5.4.5 step 2) |
| `aicall_listen_confbridge_ready_poll_interval_seconds` | `2` (proposed — §11 item 13) | **new in rev 11**: poll interval for §5.1.1 step 7's bounded confbridge-readiness retry |
| `aicall_listen_confbridge_ready_max_wait_seconds` | `30` (proposed — §11 item 13) | **new in rev 11**: total wait budget for the same retry, before giving up with `skipped_confbridge_not_ready` |
| `aicall_listen_ensure_goroutine_timeout_seconds` | `45` (proposed — §11 item 13) | **new in rev 12, added after review round 10 finding MEDIUM-B** (corrected from a mistaken "rev 11" attribution in review round 12 finding LOW-2): the steps-7-8 goroutine's (`runListenStart`, renamed rev 16) own `context.WithTimeout`, explicit and specific to this feature (§5.1.1 step 7) — must stay strictly greater than `aicall_listen_confbridge_ready_max_wait_seconds` to leave margin for the RPC calls the retry loop itself makes each poll; the two pre-existing detached-goroutine patterns cited in §5.1.1's intro (`tool.go:191-199`, unbounded; `start.go:97-100`, an unrelated 5s fetch) do not bound this path and are not reused here |
| `aicall_listen_start_lock_ttl_seconds` | `60` (proposed — §11 item 13; **raised from `15` in rev 18, kept at `60` — this row's own derivation text corrected in rev 19, review round 16 finding HIGH-1, which found the `15` default had never actually been updated here despite §5.2.2 saying it was**) | **new in rev 17, review round 14 finding HIGH-2**: TTL on the `ai:listen:startlock:<aicall_id>` key (§5.2.2), the per-AIcall `SET NX EX` lock serializing concurrent `runListenStart` goroutines' create-or-reuse sequence. Must strictly **exceed** `aicall_listen_ensure_goroutine_timeout_seconds`'s own default (`45`) — not merely one RPC call's own timeout — so the lock cannot expire out from under a goroutine that is still legitimately working within its own outer budget (§5.2.2's derivation; an earlier "sum the RPC timeouts inside the lock" derivation was wrong, review round 15 finding HIGH-1). A lock-holder that genuinely crashes (pod loss — the release `defer` never runs at all, §5.2.2), or whose acquire call errors ambiguously and whose own best-effort release attempt also fails (review round 18 finding LOW-2, §5.2.2), still strands the lock for the full TTL; accepted, since a shorter TTL only reopens review round 14's HIGH-2 in exchange for faster crash recovery |
| `aicall_listen_start_lock_release_timeout_seconds` | `3` (proposed — §11 item 13) | **new in rev 19, review round 16 finding MEDIUM-2**: bound on the detached `context.WithTimeout(context.WithoutCancel(ctx), …)` the lock's `Release` call runs under (§5.2.2), so a genuinely stuck Redis call during cleanup cannot hang the releasing goroutine indefinitely — independent of, and much shorter than, the lock's own TTL above |

All in `internal/config/main.go` with `SetXxxForTest` helpers, following
the existing pattern (`config/main.go:159-177`), and documented in
`bin-ai-manager/docs/operations.md`.

### 5.13 New metrics

**Naming corrected in rev 3 (review-round-2 F12).** Existing ai-manager
metrics are declared with `Namespace: metricsNamespace` (`"ai_manager"`)
plus a bare `Name:` (e.g. `aicall_create_total`,
`aicall_tool_execute_total`, `message_create_total`) — the namespace is
prepended by the Prometheus client library, not typed into the name
string. Rev 2's names already included `ai_manager_` as a literal prefix,
which would render as `ai_manager_ai_manager_aicall_listen_start_total`.
The table below gives the `Name:` value only; the namespace is implicit,
exactly like every existing ai-manager metric.

| Metric (full name = `ai_manager_` + this) | Labels | Meaning |
|---|---|---|
| `aicall_listen_start_total` | `result` = started / reused / skipped_not_listenable / skipped_confbridge_not_ready / skipped_confbridge_error / failed | §5.1–5.2 outcomes. **New in rev 11** (review round 9 BLOCKING-1/MEDIUM-2, revised after review round 10 HIGH-A removed the unsound `skipped_confbridge_invalid_topology` fast-fail label — see §5.1.1 step 7): the two `skipped_confbridge_*` labels give step 7's bounded retry the observability it needs — a sustained non-zero `skipped_confbridge_not_ready` rate is the direct signal that the retry budget (§11 item 13) is too short for real ring times (or genuinely non-2-party topology, which this design does not attempt to distinguish from a slow ring — see step 7), which a bare `skipped_not_listenable` bucket would have hidden |
| `aicall_listen_segment_total` | `result` = buffered / dropped_deleted / dropped_unknown | §5.3 intake |
| `aicall_listen_turn_total` | `result` = ran / skipped_locked / skipped_empty / skipped_cap / skipped_disabled / skipped_invalid / skipped_register_failed / failed | §5.4 turns. **New in rev 8**: `skipped_disabled` (§5.4.1 step 1, flag off — review round 6 F3), `skipped_invalid` (same step, any of the other require-list conditions), `skipped_register_failed` (§5.4.3, the turn-id `SADD` failed — review round 7 N-2) |
| `aicall_listen_notify_total` | — | proactive messages actually delivered |
| `aicall_foreign_pipecatcall_dropped_total` | `handler` | §5.4.4(b) guard firings (also covers pre-existing stale contact_case replies) |
| `aicall_listen_stop_failed_total` | — | §5.7.2 stop RPC failures falling back to the call-hangup-ends-the-transport backstop (§5.7.1) |
| `aicall_listen_membership_check_failed_total` | — | **new in rev 8, review round 7 N-1**: §5.4.5 step 2's `SISMEMBER` errored and degraded to `listenTurn=false`. Near-zero expected; a sustained non-zero rate means Redis is unhealthy, not that anything listen-specific is wrong |

`aicall_listen_turn_total{result="skipped_locked"}` is the
direct measure of how much LLM spend the debounce is saving; if it is near
zero, the interval is too short for the traffic.

### 5.14 Code hygiene: the commented-out ghost (resolves review M3)

`bin-ai-manager/pkg/subscribehandler/transcribemanager.go` currently
contains nothing but a fully commented-out `processEventTMTranscriptCreated`
that does `aicallHandler.GetByTranscribeID(evt.TranscribeID)` →
`aicallHandler.ChatMessage(...)` — i.e. precisely the naive
one-LLM-call-per-segment design this revision rejects. It is **not
revived**. The file is rewritten to hold the real §5.3.1 implementation,
and the commented block is deleted in the same change (it would otherwise
read as an endorsed alternative).

---

## 6. Error handling and edge cases

| Case | Behaviour |
|---|---|
| Case lookup, call lookup, transcribe list/start fails | Logged, metered `skipped_*`/`failed`, listening simply does not start. Never fails the triggering call — steps 1-6 (`checkListenEligible`) fail fast and cheap inside the `POST /service_agents/aicalls/{id}/listen` request itself, and steps 7-8 run detached (§5.1.1, rev 16 naming) |
| Terminated/deleted AIcall passed to `ProcessListen` (**new in rev 16, review round 13 finding MEDIUM-2**) | Step 2's now-combined AIcall gate rejects `c.Status != progressing \|\| c.TMDelete != nil` before any of steps 3-6 run — no transcribe list/start RPCs are made at all, since rev 15's public endpoint removed the implicit "just created/reused" guarantee `Start`'s old hook relied on |
| `UpdateListenState`'s speculative pre-write (DB + Redis) fails, before `TranscribeV1TranscribeStart` is even called (**rev 16, review round 13 finding HIGH-3 — supersedes rev 15's Redis-only pre-registration**) | Fail closed: return without calling `TranscribeV1TranscribeStart` at all — no transcribe is created, nothing to roll back. Logged, metered as a `skipped_*`/`failed` outcome same as any other step-7 RPC failure |
| `TranscribeV1TranscribeStart` fails *after* the speculative pre-write above succeeded, for any reason other than `TRANSCRIBE_ALREADY_PROGRESSING` (**rev 16**) | `rollbackListenState` (§5.2.2) explicitly undoes both the DB write and the Redis `SADD` against the pre-generated id, rather than leaving either for TTL/next-idempotency-check cleanup — no functional harm either way, but explicit rollback is preferred, matching this design's existing convention (§5.2.4's stale-membership case) |
| Call ends between the §5.1.1 step-6 liveness check and `TranscribeV1TranscribeStart` | `isValidReference` rejects a non-active call (`transcribehandler/start.go:160-163`, line drift fixed in rev 14/§10.10, corrected again in rev 16/§10.11 LOW-2); treated as a no-op, logged |
| `TranscribeV1TranscribeStart` returns `TRANSCRIBE_ALREADY_PROGRESSING` despite the list showing none (read-then-create race, acknowledged at `transcribehandler/start.go:242-247`) — **rev 16, review round 13 finding MEDIUM-3: rev 15's rollback-on-any-error snippet had silently dropped this discrimination; restored explicitly** | Re-run the list once (§5.2.2's `switch` case) and rewrite this AIcall's state to reuse the winner, undoing the speculative write against our own pre-generated id instead of keeping it; if still nothing, roll back and give up, same as any other failure |
| Two concurrent `runListenStart` goroutines for the *same* AIcall reach §5.2.2's create-or-reuse lock at once (**new in rev 18, review round 15 finding LOW-1**) | The losing goroutine's `ListenStartLockAcquire` returns `acquired=false`; it returns immediately without any RPC call, metered under `aicall_listen_start_total` as one of `skipped_*` — **`skipped_start_locked` is this design's proposed label for it, not yet added to §5.13's enumerated set; §11 item 16 tracks the decision (corrected in rev 19, review round 16 finding LOW-3, from wording that read as already settled)** — the winning goroutine's sequence is unaffected |
| §5.2.2's lock `ListenStartLockAcquire` call errors (Redis unavailable) (**new in rev 18, renamed in rev 19 with the acquire/release symmetry fix**) | Fail closed, same as every other Redis-dependent step in this design (§6's Redis-unavailable row below): no `TranscribeV1TranscribeList`/`TranscribeV1TranscribeStart` call has been made yet, so no transcribe is started; a best-effort release is attempted with the same token first (§5.2.2, review round 17 finding B-7 — the underlying `SET NX` may have landed despite the client-visible error), but the outcome is metered `failed` regardless of whether that release itself succeeds |
| §5.2.2's lock `ListenStartLockRelease` call errors, on the normal (deferred) release path after a successful acquire (**split out from the row above in rev 20, review round 17 finding B-6, which found the row above's "no transcribe is started" and "metered `failed`" claims did not hold for this path — by construction the deferred release runs *after* `TranscribeV1TranscribeStart` may already have run**) | Best-effort only: the return value is deliberately discarded (`_ =`, §5.2.2), so there is nothing to meter here specifically. If the underlying `DEL` genuinely did not happen, the lock simply falls back to expiring on its own TTL (§5.2.2's accepted residual) — no different in outcome from the crash case the TTL is already sized to tolerate |
| Two Cases on one live call | Two AIcalls (the unique key is per-Case), one shared STT session. The first to arrive owns it (`owns=true`); the second reuses (`owns=false`) and never stops it. Both are cleared on hangup by `stopListenByCallID`'s plural lookup |
| `transcript_created` arrives for a deleted transcript | Dropped on `TMDelete != nil` (§5.3.2, review H3) |
| `transcript_created` arrives after listening stopped | Redis resolver key already deleted → dropped |
| Redis unavailable | Buffering and the debounce lock fail → no listen turns run (and, since §5.4.3's `SADD` also fails, none *can* run — see below). **Q&A is unaffected — restated precisely in rev 8, review round 7 finding N-1**: as of rev 7, `ToolHandle` *does* touch `ai:listen:turnpcid:*` for every `contact_case` tool call (§5.4.5 step 2), so "never touches these keys" is no longer literally true — but a `SISMEMBER` error there degrades to `listenTurn=false` rather than failing the tool call, and that is provably the correct value during a Redis outage (no genuine listen turn can exist at that moment either, for the same reason). Net effect on Q&A is unchanged: it still runs normally, just via a degrade path instead of never touching the keys at all. Listening itself degrades to today's reactive-only behaviour |
| Redis flushed mid-call | Listening silently stops for in-flight calls until the panel is reopened (§5.3.2). Stated, accepted, self-healing |
| `RunListenTurn` fails after popping `pending` (pipecatcall start error, pod loss) (**new in rev 3**) | The popped lines are gone — `LPOP` already removed them, and only the ≤40-line `window` retains a copy for the *next* turn's context. Accepted, bounded data loss: at most one debounce interval's worth of transcript is skipped from evaluation, never from the call itself (nothing about the actual call or its recording is affected) |
| LLM emits text instead of calling `notify_agent` | Dropped by the pipecatcall-identity guard on the two pipecat message handlers it applies to (§5.4.4(b), narrowed from four in rev 3); metered. Nothing persisted, no webhook |
| LLM calls `notify_agent` with empty/whitespace/oversized `message` | Rejected in `parseNotifyAgentMessage`; `fillFailed` (same style as the other tools in `tool_insight.go`); tool-result row records the failure; **no** proactive message row |
| LLM calls `notify_agent` during a normal Q&A turn (**corrected in rev 3, was "harmless" in rev 2**) | **Rejected outright** by the `listenTurn` check `ToolHandle` resolves and passes to `toolHandleNotifyAgent` (§5.4.5 step 2, §5.4.4(c)) — the call fails, the agent's real answer proceeds unaffected. Rev 2 called this harmless; it was not (§5.4.4(c)) |
| LLM calls other Insight tools during a listen turn | Allowed (no per-turn tool restriction — §3). Adds tool-call/result rows, webhook-published as always; hidden from the panel by the render filter (§5.6.4, §5.10.1) but still delivered to the tenant's webhook consumer. Discouraged by `ListenTurnSystemPrompt` |
| Turn cap reached on a very long call | Listening stops cleanly with `skipped_cap`; the Q&A panel keeps working normally |
| Agent asks a question while a listen turn is mid-flight | Both proceed independently on separate pipecatcalls. `Send` rotates `c.PipecatcallID`, which makes the still-running listen turn's id "foreign" — so if that turn later emits text it is dropped, and if it calls `notify_agent` the notification still lands correctly (tool routing goes through the pipecatcall's `ReferenceID`, not `AIcall.PipecatcallID`) |
| transcribe-manager pod restarted; stop RPC unreachable | Logged + metered; transcribe-manager's own hangup handler is the guaranteed backstop (§5.7.2) |
| `ProcessListen` runs while the call is still queued/ringing, or the confbridge is transiently deleted/non-`progressing`/non-2-party (agent hasn't joined yet, or an early-media multi-destination `connect` has extra ringing legs still in the bridge) (**new in rev 11, review round 9 BLOCKING-1, retry logic revised after review round 10 HIGH-A**) | Not treated as failure: §5.1.1 step 7 polls (bounded) until the confbridge is live with exactly 2 parties, the call ends, or the wait budget elapses. Never fails the triggering `POST` (still detached, §5.1.1). The loop does not attempt to distinguish "still converging" from "will never converge" — see step 7 for why that distinction was tried and dropped |
| `CallV1ConfbridgeGet` errors while polling (**new in rev 11**) | Logged, metered `skipped_confbridge_error`; listening does not start; retried on the *next* `POST /service_agents/aicalls/{id}/listen` call (next panel open), not within the same poll loop |
| The wait budget (`AIcallListenConfbridgeReadyMaxWaitSeconds`) elapses without ever reaching a live 2-party confbridge (**new in rev 11**) | Logged, metered `skipped_confbridge_not_ready` (the last observed party count is included in the log line, not as a metric label); listening does not start for this `ProcessListen` invocation; retried on the next panel open |
| A 3rd party joins the confbridge *after* listening has already started (**new in rev 11, review round 9 HIGH-3 — documented, not mitigated**) | §5.1.1 step 7's guard is start-time only; there is no re-check once listening is live. `out` becomes ambiguous (no longer reliably "the agent"); `in` is unaffected. Open residual risk, §11 item 12 |

---

## 7. Testing strategy

**`bin-ai-manager` unit (gomock, table-driven, following
`pkg/aicallhandler/start_test.go`):**

1. `checkListenEligible` — every branch of §5.1.1 steps 1-6: disabled
   flag; non-Insight AI; **terminated/deleted AIcall (new in rev 16,
   review round 13 finding MEDIUM-2 — must make zero transcribe list/start
   calls, unlike every other branch below it need not even reach the
   idempotency check)**; already-listening idempotency (must make **zero**
   transcribe-start calls); `ReferenceType != "call"`; unparseable
   `ReferenceID`; cross-customer Case; cross-customer call; each non-live
   call status; each returns `proceed=false` synchronously with **zero**
   goroutines spawned. Separately, happy-path `proceed=true` asserts
   `runListenStart` is invoked exactly once by `ProcessListen`, receiving
   the already-resolved `a`/`c`/`kase`/`callID`/`call` values directly
   (asserting **zero** additional `ContactV1CaseGet`/`CallV1CallGet` calls
   inside the goroutine — the direct regression test for review round
   13's HIGH-1) rather than re-fetching them (**rewritten in rev 16**;
   rev 15's first draft split this into two separately-callable
   functions connected only by a bare `bool`, which is what created the
   re-fetch risk this test now pins against).

   *Step 7's confbridge participant-count guard, added in rev 11 (review
   round 9 required this after BLOCKING-1; the retry logic itself was
   revised after review round 10 finding HIGH-A), gets its own dedicated
   coverage within this same test suite:* `ConfbridgeID == uuid.Nil`
   while the call is `ringing`, polled until it resolves and the party
   count reaches 2 (asserts the retry actually re-polls, not just that it
   eventually succeeds — the false-negative regression test for
   BLOCKING-1); wait budget exhausted while still `len == 1` →
   `skipped_confbridge_not_ready`, no transcribe-start call; **`len == 3`
   while the call is `progressing`, then settling back to `len == 2`
   within the wait budget → proceeds to §5.2 normally, does NOT give up**
   (the direct regression test for review round 10's HIGH-A — an
   early-media multi-destination `connect` transiently over-populating the
   confbridge before settling); a *stable* `len == 3` for the entire wait
   budget → `skipped_confbridge_not_ready` (same label as any other
   timeout — this design does not distinguish the two causes, §5.1.1 step
   7); `CallV1ConfbridgeGet` error → `skipped_confbridge_error`; a
   confbridge with `TMDelete != nil` or non-`progressing` `Status` treated
   the same as "not ready," not as "2 parties confirmed"; call transitions
   to a non-live status *during* the poll loop → step 6's own liveness
   check ends the loop, not the confbridge check; happy path (`len == 2`,
   confbridge `progressing`) → proceeds to §5.2 with **zero** extra polls
   once satisfied.
2. **`POST /service_agents/aicalls/{id}/listen` handler coverage
   (rewritten in rev 15, endpoint surface and handler shape corrected in
   rev 16 — `Start` no longer has a hook to test here at all).** At the
   `serviceHandler` layer (`ServiceAgentAIcallListen`): non-agent identity
   → `ErrAuthenticationRequired`, no RPC calls; agent without
   `PermissionAll` → `ErrPermissionDenied`, no `AIV1AIcallListen` call
   (the direct regression test for review round 13's BLOCKING-1);
   cross-customer AIcall (fetched but `CustomerID != a.CustomerID`) →
   rejected before `AIV1AIcallListen` is called. At the `aicallHandler`
   layer (`ProcessListen`, gomock, single exported method — collapsed in
   rev 16 per review round 13's HIGH-2/MEDIUM-4, so this is the only
   handler-layer test surface now, not two): unknown `id` → 404, no
   `checkListenEligible` call. `checkListenEligible` returning
   `proceed=false` → 200 with the unchanged AIcall, **zero**
   `runListenStart` invocations (asserts the method actually branches on
   the returned bool, not just always firing the goroutine). Happy path →
   returns within test-reasonable latency (asserts `ProcessListen` does
   **not** block on step 7's confbridge wait — the single most
   safety-critical property of the sync/async split, §5.1). Repeated
   calls on an already-listening AIcall are free (rev 1's original
   reuse-path regression, now exercised via two consecutive HTTP calls
   instead of two `Start` returns, since `Start` itself no longer
   triggers this at all).
   **Event-ordering fix (§5.2.2, new in rev 15, restructured in rev 16
   per review round 13's HIGH-3/MEDIUM-3):** `UpdateListenState`'s
   speculative pre-write (DB + Redis, against a pre-generated id) happens
   before `TranscribeV1TranscribeStart` is called (assert call order via
   gomock `gomock.InOrder`, not just that both eventually happen);
   pre-write failure → `TranscribeV1TranscribeStart` never called;
   `TranscribeV1TranscribeStart` failure with a **non**-`TRANSCRIBE_ALREADY_PROGRESSING`
   error → `rollbackListenState` called with the pre-generated id;
   `TranscribeV1TranscribeStart` failure **with**
   `TRANSCRIBE_ALREADY_PROGRESSING` and the re-run list finds a winner →
   `UpdateListenState` called again with the winner's id and `owns=false`,
   `ListenTranscribeAIcallRemove` called for the AIcall's own
   never-created pre-generated id, **not** `rollbackListenState` (the
   direct regression test for review round 13's MEDIUM-3 — rev 15's
   snippet would have rolled back and given up here instead of reusing
   the winner, silently dropping §6's documented reuse-on-conflict
   behaviour); same case but the re-run list also comes up empty →
   `rollbackListenState`, give up; **an early `transcript_created` event
   arriving between the pre-write and `TranscribeV1TranscribeStart`
   returning must not trigger `stopListening`/`clearListenState`** (the
   direct regression test for review round 13's HIGH-3 — assert
   `RunListenTurn`'s precondition check reads the pre-written
   `listen_transcribe_id` correctly and processes the segment normally,
   not as a `skipped_invalid` teardown); happy path → the transcribe id
   the mocked `TranscribeV1TranscribeStart` reports back equals the
   pre-generated id already written.
   **Per-AIcall create-or-reuse lock, and the scoped `owns`-merge (rev
   17, review round 14 findings HIGH-1/HIGH-2; lock coverage extended in
   rev 19, review round 16 finding MEDIUM-4):** two concurrent
   `runListenStart` invocations for the *same* AIcall — the second
   `ListenStartLockAcquire` call must fail (`acquired=false`) while the
   first still holds `ai:listen:startlock:<aicall_id>`, and the second
   goroutine returns immediately with **zero**
   `TranscribeV1TranscribeStart`/`UpdateListenState` calls of its own (the
   direct regression test for HIGH-2's clobbering scenario — assert the
   first goroutine's session survives untouched), metered under this
   design's proposed (not yet added to §5.13, §11 item 16) label
   `skipped_start_locked`. A goroutine that completes normally —
   including one whose own `ctx` has already been cancelled by
   `aicall_listen_ensure_goroutine_timeout_seconds` — always calls
   `ListenStartLockRelease` and the key is gone immediately after,
   asserted by mocking `ListenStartLockRelease` to capture the context it
   receives and confirming that context is **not** `ctx` and is not
   already `Done()` (the direct regression test for round 16's MEDIUM-2 —
   a release still keyed off the cancelled `ctx` would silently no-op).
   **`ListenStartLockRelease`'s compare-and-delete semantics** (round 15
   HIGH-1(b), re-verified in round 16): a release call whose token no
   longer matches the key's current value — this goroutine's own TTL
   already lapsed and a different goroutine has since acquired the same
   key — is a no-op; the second goroutine's still-live lock is
   unaffected (the direct regression test for the exact clobbering this
   lock exists to prevent, now exercised at the release layer directly
   rather than only inferred from HIGH-2's create-path test above).
   **Simulated crash (no `defer` ever runs, e.g. the goroutine is killed
   mid-sequence rather than merely timing out):** the lock is held for
   the full `aicall_listen_start_lock_ttl_seconds`; a goroutine that
   attempts `ListenStartLockAcquire` before that window elapses observes
   `acquired=false` (proposed label `skipped_start_locked` again, not an
   error); one that attempts after the window elapses acquires normally
   and proceeds (**corrected in rev 19, review round 16 finding
   MEDIUM-3**, from an earlier claim that any "later goroutine" could
   acquire and proceed — true only once the TTL has actually elapsed,
   since the TTL now exceeds a single goroutine's own outer timeout
   budget). **Acquire-error path, including the best-effort release
   attempt (rev 20, review round 17 finding B-7):** `ListenStartLockAcquire`
   returning a Redis error → `runListenStart` (not `checkListenEligible`,
   which never reaches this lock — **mislabeled before rev 20, review
   round 17 finding B-6**) fails closed, metered `failed`, **zero**
   `TranscribeV1TranscribeStart` calls, regardless of whether the
   best-effort `ListenStartLockRelease` attempt on this path itself
   succeeds (asserted by making that release call also fail and
   confirming `runListenStart` still returns the original acquire
   error). **Deferred-release-error path (new in rev 20, review round 17
   finding B-6, extending round 16's MEDIUM-4 coverage):** a
   `ListenStartLockRelease` error on the
   normal, successful-acquire path is swallowed (`_ =`) by design — assert
   this does **not** propagate as a `runListenStart` failure and does
   **not** get separately metered, distinguishing it from the
   acquire-error path above.
   `UpdateListenState`'s `owns`-merge: writing the **same**
   `transcribeID` as the row's current one with `owns=false` after a
   prior `owns=true` write → merged result is `true` (rev 14's original
   case, unchanged); writing a **different** `transcribeID` (the
   create-then-fall-back-to-reuse branch above) with `owns=false` after
   a prior `owns=true` write **against the old id** → merged result is
   `false`, not carried forward (the direct regression test for HIGH-1 —
   asserts this AIcall does not end up believing it owns a session it
   fell back away from).
3. `EventTMTranscriptCreated` — `TMDelete != nil` drop; empty-message
   drop; empty-set drop (asserting **no** DB call); buffered-but-locked
   (no turn); buffered-and-unlocked (turn runs); **new in rev 3, pins F1's
   fix**: two AIcalls in the same resolver set both get the segment
   buffered independently, and clearing one (`SREM`) leaves the other's
   membership and buffering intact.
4. `RunListenTurn` — empty pending → skip; turn cap → `stopListening`
   called, `skipped_cap`; flag off → `stopListening` called,
   `skipped_disabled` (**new in rev 9, review round 8 finding L-4**);
   each other failing require-list condition → `stopListening` called,
   `skipped_invalid`; context assembly golden test asserting exact
   message count and order — `InsightSystemPrompt` first, the prompt
   snapshot second, then `ListenTurnSystemPrompt` (this is the direct
   regression test for both the rev-1 context-eviction defect and
   rev-2's missing-guardrails defect, F6); asserts `getPipecatcallMessages`
   is **not** called and `c.PipecatcallID` is **not** written.
5. `messagehandler.isForeignPipecatcall` — for each of the **two**
   handlers it applies to (§5.4.4(b), narrowed from four in rev 3):
   matching id persists/publishes, mismatched id drops, and (for
   `EventPMMessageBotLLMIntermediate`) **no webhook is published** on
   mismatch. Also: a mismatch that resolves to a match on the
   cache-bypassing re-read still persists (pins review round 4's F4 fix,
   scoped in rev 5 to `EventPMMessageBotLLM` only).
6. `toolHandleNotifyAgent` — success writes exactly one proactive row with
   `Role=assistant` and `Origin=proactive`; called with `listenTurn=false`
   (§5.4.4(c)) is rejected with no row written; empty/whitespace/
   oversized argument writes no proactive row.
7. Cleanup — `stopListenByCallID` clears **all** matching AIcalls (two-row
   case); `ProcessTerminate` stops only when `owns=true`; stop-RPC failure
   is non-fatal and metered; `clearListenState` `SREM`s only its own
   membership, not the whole resolver set. **New in rev 9, review round 8
   finding M-2:** `stopListening` itself — asserts it calls the §5.7.2
   stop snippet then `clearListenState`, in that order, and that it
   **never** calls `ProcessTerminate` (a regression test for the
   AIcall-termination mixup the naming ambiguity risked); both its call
   sites (§5.4.1's require-list failure, §5.4.1 step 3's turn cap) are
   covered by item 4 above.
8. **New in rev 9, review round 8 finding N-2/L-2:** §5.4.3's registration
   failure — `ListenTurnPipecatcallIDAdd` erroring aborts before
   `startListenPipecatcall`/`PipecatV1PipecatcallStart` is ever called
   (no pipecat session started, nothing to clean up), metered
   `skipped_register_failed`; the turn's `pending` lines were already
   popped by §5.4.1 step 2 before this point and are not requeued —
   consistent with the existing "lost lines beyond the 40-line window"
   trade-off (§6), not a new one.
9. **New in rev 3, pins F4:** the pipecatcall-identity guard's
   cache-bypass re-read — a mismatch against the cached `PipecatcallID`
   that resolves to a match on a DB-authoritative re-read persists the
   message; a mismatch that still disagrees on re-read drops it.
10. **New in rev 4, pins B3:** listen and a concurrent `ai_summary` on the
   same call each get their own transcribe session (`IDAIManagerListen`
   vs. `IDAIManager`); `TranscribeV1TranscribeList` scoped to
   `IDAIManagerListen` never returns a summary's session and vice versa;
   `summaryhandler.startReferenceTypeCall`'s own tests are unaffected by
   listen having run on the same call (a direct regression test for the
   rev-3 defect this replaces).
11. **`ToolHandle`'s `listenTurn` resolution (§5.4.5 step 2) — rewritten
    across rev 4 through rev 7, current shape tested here:**
    - `c.ReferenceType != ReferenceTypeContactCase` → `listenTurn=false`
      unconditionally, no Redis call made (pins review round 6 F2: cost
      confined to the one reference type that needs it).
    - `pipecatcallID` registered via §5.4.3's `SADD` for this AIcall →
      `listenTurn=true` (a genuine listen turn).
    - `pipecatcallID` **not** registered (a real Q&A turn's id, including
      one from a just-interrupted, since-rotated-away turn — the specific
      race review round 6's F1 named) → `listenTurn=false` (pins F1: no
      longer inferred from "not currently bound," directly tested by
      registering turn A, rotating `c.PipecatcallID` to B, and confirming
      a tool call arriving with id A still resolves `listenTurn=false`).
    - `pipecatcallID == uuid.Nil` → `listenTurn=false` (pins review round
      4's B2: unknown id treated as a real turn, never as listen).
    - Redis `SISMEMBER` erroring → **degrades to `listenTurn=false`**
      (review round 7 finding N-1, not a hard tool-call failure) — the
      tool call proceeds normally as a real Q&A turn; metered via
      `promListenMembershipCheckFailedTotal`, and `toolHandleNotifyAgent`
      still reached (with `listenTurn=false`, so it still correctly
      rejects if this happened to be a `notify_agent` call).
12. `getPipecatcallMessages`'s two-fetch context assembly (§5.4.5,
    revised across rev 5-7): a golden test seeding 150+ listen-internal
    rows interleaved with 10 real Q&A rows and the leading system-prompt
    rows asserts (a) the system-prompt row(s) are always present
    regardless of how many listen-internal or proactive rows follow, in
    their original creation order (pins review round 6's H2 — both
    fetches are newest-first and must be reversed before use, not just
    the first), and (b) the "rest" fetch excludes every listen-internal
    row via the `NotEq` filter.
13. `toolHandleNotifyAgent`'s reject logic (§5.4.4(c)) — takes the
    pre-resolved `listenTurn bool` directly (no `AIcallGet` call of its
    own to test/mock here, since rev 7 removed it): `listenTurn=true` →
    allowed; `listenTurn=false` → rejected with no proactive row written.

**`bin-ai-manager` model/golden:**

14. `models/ai/allowed_tools_test.go` — `notify_agent` passes via
    `knownSanctionedWrite`; a hypothetical unlisted write tool still fails;
    `TestValidateToolNames_WriteToolNeverAllowedForInsight` still passes
    unchanged.
15. `pkg/subscribehandler/binding_golden_test.go` — updated to 12 patterns
    with the new one appended last.
16. `models/aicall/field_test.go`, `filters_test.go`,
    `models/message/field_test.go`, `webhook_test.go` — new fields,
    including `Origin`'s two values.

**Boundary:**

17. `requesthandler` mock expectations pinning the exact
    `TranscribeV1TranscribeStart` argument list including `provider` and
    `onEndFlowID` (§5.2.2), and `TranscribeV1TranscribeStop(ctx, hostID,
    transcribeID)` argument order (§5.7.2) — both were wrong in rev 1, so
    both get an argument-shape test. `AIV1AIcallToolExecute`'s new
    `pipecatcallID` argument (§5.4.3a), both at the `bin-common-handler`
    client and the `bin-pipecat-manager` call-site.

**Deferred until §5.9's empirical check lands:**

18. A pinned golden-transcript test for the `in`/`out` → `[CUSTOMER]`/
    `[AGENT]` mapping. This is exactly the silent-wrong-attribution class
    that deserves a pinned test rather than a happy-path assertion.

**Frontend (`monorepo-javascript`):**

19. Both `CaseInsightAssistantPanel` suites: renders an
    `origin: 'proactive'` message with its distinct treatment and
    accessible label; renders a normal assistant message unchanged;
    renders a message with no `origin` field (backward compatibility with
    every existing row) unchanged; the tool-call/tool-result render
    filter (§5.6.4) hides `role='tool'` and empty-content
    `role='assistant'` rows (field names matching the real `tool_calls`/
    `content` wire fields, not camelCase).
20. **New in rev 15, path corrected in rev 17 (§5.10.1a, review round 14 finding MEDIUM-2):** both panels
    call `POST /service_agents/aicalls/{id}/listen` exactly once per
    panel-open, using the AIcall id from the `Start` response, and do
    **not** block rendering
    on that call's response; a rejected/slow `listen` response does not
    prevent the panel from rendering or from receiving/displaying Q&A
    messages. A second rapid open/close/reopen does not spawn duplicate
    in-flight `listen` calls beyond what the existing effect-dependency
    array already guards against for `Start` itself (no new
    frontend-side dedup logic is required — §5.1.1 step 3's server-side
    idempotency check is what actually makes repeats free).

---

## 8. Rollout

1. Backend PR in `monorepo` — code + migrations (drafted, **not**
   applied), service docs, RST + rebuilt HTML. **Corrected in rev 4**:
   `bin-common-handler` is touched unconditionally now (§5.4.3a's
   `pipecatcallID` parameter, and §5.4.5's `databasehandler.NotEq`), not
   "if touched." Full verification workflow in `bin-ai-manager`,
   `bin-common-handler` (monorepo-wide, since `databasehandler` is shared
   by every service — §5.4.5 step 3), `bin-pipecat-manager` (§5.4.3a),
   `bin-contact-manager`, `bin-customer-manager` (§5.2.1, pending §11 item
   5), `bin-api-manager`, `bin-openapi-manager`.
2. **Migration-before-deploy ordering — new in rev 5, review round 4
   finding H5.** `messageHandler.List`/`MessageList` builds its `SELECT`
   column list by reflecting the `Message` struct
   (`commondatabasehandler.GetDBFields`, `dbhandler/message.go:210`);
   `AIcallList` does the same for `AIcall`. The instant the `Origin` field
   (or `ListenCallID`) exists in the Go struct, every message/AIcall
   query — including ones this feature never touches — selects that
   column. **A code deploy landing before its migration is a hard outage
   for every AIcall/message read in `bin-ai-manager`** (`Unknown column`),
   not a soft degradation. Human applies the Alembic migrations *before*
   the code deploy that references the new columns reaches any pod — not
   merely "before implementation sign-off" as §9 lists them, but as an
   explicit, ordered deploy-gate step here.
3. **Deploy with `aicall_listen_enabled=false`. "Zero behaviour change"
   corrected in rev 6, review round 5 finding H1 — it is not literally
   true; here is what actually does and doesn't change with the flag
   off:**

   | Change | Behaviour with flag **off** | Why |
   |---|---|---|
   | §5.1's `POST /service_agents/aicalls/{id}/listen` endpoint and `ProcessListen`/`checkListenEligible`/`runListenStart`, §5.2–§5.7's transcribe/Redis/turn machinery | Endpoint reachable, but fully inert downstream (**endpoint existence itself is rev 15/16's own change, not flag-gated — see below**) | `checkListenEligible` step 1 returns `proceed=false` immediately on the flag (§5.1.1) — `runListenStart` is never even called, so nothing downstream ever runs. The endpoint always exists once this PR deploys; calling it with the flag off is a harmless no-op, same as every other §5.1/§5.2 step failing its precheck |
   | §5.4.5's `getPipecatcallMessages` two-fetch restructure | **Always active**, for every `call`/`conversation`/`task`/`contact_case` AIcall, regardless of the flag | It is a general context-assembly fix (guarantees the system prompt, §5.4.5 step 4), not listen-specific machinery — there is no listen state to gate it on for AIcalls that were never going to listen anyway |
   | §5.4.4(b)'s `isForeignPipecatcall` guard on `EventPMMessageBotLLM`/`…Intermediate` | **Always active** for `contact_case` AIcalls | Same reasoning: it is a general stale-reply guard (the design's own §5.4.4(b) text calls it "a strict improvement beyond this feature"), not conditioned on a listen session existing |
   | §5.4.5's `Origin` tagging in `ToolHandle` | Inert in practice — `listenTurn` (§5.4.5 step 2, rewritten in rev 7) checks Redis membership in `ai:listen:turnpcid:<aicall_id>`, which is never populated while the flag keeps `checkListenEligible`/§5.4.3 from ever running | Gated *indirectly*, through the positive listen-turn-id registration (review round 6's F1 fix), not through an explicit flag read in `ToolHandle` |
   | `ToolHandle`'s per-tool-call `ReferenceType` check (§5.4.5 step 2, review round 6 F2) | **Always active**, for every AIcall type | A single cached-field comparison (`c.ReferenceType == ReferenceTypeContactCase`), not a DB read — cheap enough to run unconditionally rather than threading the flag through `ToolHandle` for one branch |
   | Frontend render filter (§5.10.1) | **Always active** | Client-side; ships with its own PR regardless of the backend flag (step 4 below) |
   | §5.1.1 step 7's confbridge participant-count guard and its bounded retry (**new in rev 11**) | Fully inert | Never reached — step 1's flag check returns first, same as every other §5.1/§5.2 step; the new `skipped_confbridge_*` metrics (§5.13) stay at true zero with the flag off |

   **Confirm accordingly**: `ai_manager_aicall_listen_*` metrics all flat
   (true zero-activity signal), but do **not** expect
   `foreign_pipecatcall_dropped_total` or existing Insight Q&A response
   *shape* to be bit-identical to pre-deploy — both are expected to change
   slightly (dropping genuinely-stale replies that used to be silently
   persisted; system-prompt rows now always present in context) the
   moment the code deploys, independent of the flag. Watch these two
   specifically during the code-only deploy window, not just the
   `listen_*` metrics.
4. Frontend PR in `monorepo-javascript` (both apps). Safe to deploy while
   the backend flag is off — no message will ever carry
   `origin=proactive`, and the render filter only ever hides rows that
   were noise before this feature too. **New in rev 15, path corrected in
   rev 17 (review round 14 finding MEDIUM-2)**: both panels' new
   `POST /service_agents/aicalls/{id}/listen` call (§5.10.1a) is equally
   safe to deploy first — it always exists
   once step 1's backend PR lands (the table above), and is a harmless
   no-op with the flag off.
5. Enable the flag for one pilot customer. Watch
   `listen_turn_total{result}` (especially `skipped_locked` vs `ran`),
   `listen_notify_total`, `foreign_pipecatcall_dropped_total`, and LLM
   spend.
6. Tune `aicall_listen_evaluate_interval_seconds` from observed data
   before wider enablement.

**Rollback:** set `aicall_listen_enabled=false`. **Made literally true in
rev 7 (review round 6, finding F3), tightened in rev 8 (review round 7,
findings N-3/N-4), timing corrected in rev 9 (review round 8, finding
M-1)**: §5.4.1 step 1 checks the flag directly inside `RunListenTurn`
(merged with the existing require-list, not a separate step 0 that ran
before the AIcall was even fetched — N-3), so an in-flight session's
*next evaluated turn* sees the flag off and calls `stopListening`
(§5.4.1's helper — M-2), including releasing any transcribe session this
AIcall itself owns (§5.7.2) — not just clearing Redis/DB bookkeeping and
leaving an owned session to run until hangup, which is what a bare state
clear would have left standing (N-4). **`RunListenTurn` is
segment-triggered, not on a fixed timer (§5.3.4)**, so "next evaluated
turn" means the next time the call actually produces a transcript
segment — typically within one `AIcallListenEvaluateIntervalSeconds`
(default 20s) for an active conversation, but not a guarantee for a
quiet stretch; §5.7.1's hangup-triggered cleanup is the actual backstop
if the call goes silent before another segment arrives. Before rev 7's
fix, nothing on the intake or evaluation path read the flag at all, so a
session that started while the flag was on would have run to call-end or
the turn cap regardless of a rollback. A session this AIcall doesn't own
is never touched by any of this and is reaped by transcribe-manager on
call hangup either way, same as always; the `listen_call_id` column and
`origin` field are inert **for the listening path specifically**. The §8
table above still applies — rollback does not roll back the always-active
context-assembly fix or the stale-reply guard, which are code-deploy
changes, not flag-gated ones. No migration rollback is required.

---

## 9. Impacted files (indicative)

`bin-ai-manager`
- `models/tool/main.go` — `ToolNameNotifyAgent`, `AllInsightToolNames`, invariant comment
- `models/ai/allowed_tools_test.go` — sanctioned-write map
- `models/message/{main,field,filters,webhook}.go` + tests — `Origin`
  (both values: `proactive` and `listen_internal`, §5.4.5)
- `models/aicall/{main,field,filters}.go` + tests — `ListenCallID`, metadata keys
- `pkg/aicallhandler/main.go` — **the `AIcallHandler` interface itself**
  (`ToolHandle`'s new `pipecatcallID` parameter, §5.4.3a — corrected in
  rev 5: this is an interface-level change, not just the implementation
  file below), plus `ListenTurnSystemPrompt`, config-derived constants
- `pkg/dbhandler/{main,aicall}.go` + mocks — **new in rev 6, review round
  5 finding M1, narrowed in rev 7**: `AIcallGet` gains an optional
  cache-bypass argument, consumed via `AIV1AIcallGet`'s new argument by
  `messagehandler`'s stale-reply guard (§5.4.4(b)) — **no longer** also
  needed by §5.4.5's `listenTurn` resolution, which moved to a Redis
  membership check in rev 7 (§5.4.3, §5.4.5 step 2) instead of an
  `AIcallGet` call. Also: a targeted `AIcallUpdate` variant that writes
  `listen_call_id`/metadata without bumping `tm_update` (§5.2.4/§5.7.3,
  resolving review round 4's H1). Both are `DBHandler` interface changes,
  so the interface, its implementation, and its mock all move together.
- `pkg/aicallhandler/start.go` — **the `Start` hook is removed in rev
  15** (§5.1), not added — `getPipecatcallMessages`'s two-fetch rewrite
  (leading system rows + `databasehandler.NotEq`-filtered rest, §5.4.5)
  stays here since it is unrelated to the trigger
- `pkg/aicallhandler/listen_trigger.go` *(new, rev 15, single-method
  shape corrected in rev 16)* — `ProcessListen` (the sole exported entry
  point), `checkListenEligible` (§5.1.1 steps 1-6), `runListenStart`
  (§5.1.1 steps 7-8), `rollbackListenState` (§5.2.2); `startListenPipecatcall`
  (**new in rev 7**: registers `turnPipecatcallID` in Redis, §5.4.3)
  stays with whichever of this file or `listen.go` below owns turn
  execution — implementation-time call, not a design decision
- `pkg/aicallhandler/listen.go` *(new)* — `EventTMTranscriptCreated`,
  `RunListenTurn` (`AIcallListenEnabled` precondition, merged into
  §5.4.1 step 1's require-list as of rev 8), context assembly,
  `stopListenByCallID`, `clearListenState`
- `pkg/aicallhandler/tool.go` — `ToolHandle`'s implementation of the new
  `pipecatcallID` parameter, the `listenTurn` resolution (§5.4.5 step 2 —
  `ReferenceType` pre-gate + Redis `SISMEMBER`, not an `AIcallGet`),
  `mapFunctions` entry for `notify_agent` (unchanged signature, §5.4.3a
  step 4), `Origin` tagging on tool-call/tool-result rows (§5.4.5)
- `pkg/aicallhandler/tool_insight.go` — `toolHandleNotifyAgent` (takes
  `listenTurn bool`, no `AIcallGet` of its own — §5.4.4(c)),
  `parseNotifyAgentMessage`
- `pkg/aicallhandler/event.go` — `EventCMCallHangup` second lookup
- `pkg/aicallhandler/process.go` — terminate-path stop
- `pkg/listenhandler/v1_aicalls.go`, `pkg/listenhandler/models/request/aicalls.go`
  — **new in rev 5**: `pipecatcall_id` on `V1DataAIcallsIDToolExecutePost`
  and its handler (§5.4.3a step 3; missing from rev 3/4's file list).
  **New in rev 15**: `regV1AIcallsIDListen`, route table entry,
  `processV1AIcallsIDListenPost` (§5.1)
- `pkg/messagehandler/main.go`, `event.go` — `isForeignPipecatcall`,
  applied to the two handlers that can actually fire from a listen turn
  (§5.4.4(b)); the cache-bypass re-read now goes through `AIV1AIcallGet`'s
  RPC client with a new optional argument, not a direct `dbhandler` call
  (§5.4.4(b), path corrected in rev 5)
- `pkg/toolhandler/definitions.go` — `notify_agent` (`RunLLM:false`)
- `pkg/cachehandler/{main,handler}.go` **or a new `pkg/listencachehandler`
  package** — see the scope note below. **New in rev 7**: also
  `ListenTurnPipecatcallIDAdd`/`IsMember` (§5.4.3, §5.4.5 step 2).
  `ListenTranscribeAIcallRemove` (§5.2.2's conflict-recovery branch —
  **added to this bullet in rev 20, review round 17 finding B-4**; it
  existed in the design since rev 4 but this enumeration had never
  listed it). **New in rev 18** (corrected in the same revision, review
  round 15 finding LOW-3, from an earlier draft that listed the lock TTL
  config but not the primitives it backs): `ListenStartLockAcquire` (the
  per-AIcall start lock's `SetNX`, §5.2.2 — same underlying Redis
  command as the debounce lock already established in §5.3.4, no new
  Redis command; **named as a symmetric pair with `Release` in rev 19,
  review round 16 finding LOW-6**, replacing a raw inline `SetNX` call)
  and `ListenStartLockRelease` (the lock's compare-and-delete release, a
  Lua `EVAL`, genuinely new — §5.2.2).
- `pkg/subscribehandler/{main,transcribemanager,binding_golden_test}.go`
- `internal/config/main.go` — thirteen flags (**corrected in rev 20,
  review round 17 finding B-5**, from "twelve," which rev 19 left stale
  the moment it added `aicall_listen_start_lock_release_timeout_seconds`
  — see §5.12 for the current enumerated list)
- `docs/{domain,architecture,operations}.md`

`bin-contact-manager` — `models/kase/kase.go` (`ReferenceTypeCall`)

`bin-customer-manager` — **new in rev 4**: `models/customer/customer.go`,
new `IDAIManagerListen` system customer constant (§5.2.1), pending §11
item 5's confirmation of whether it needs a backing row

`bin-pipecat-manager` — **new in rev 4 (§5.4.3a)**:
`pkg/pipecatcallhandler/runner.go` forwards `pc.ID` on `tool_execute`

`bin-common-handler` — `pkg/requesthandler`'s `AIV1AIcallToolExecute`
signature gains a `pipecatcallID` parameter (§5.4.3a) and its
`AIV1AIcallGet` client gains an optional cache-bypass argument (§5.4.4(b),
corrected in rev 5); mock regen for both. **New in rev 5 (§5.4.5, review
round 4 finding B3)**: `pkg/databasehandler/main.go` gains the `NotEq`
wrapper type and its handling in `ApplyFields` — this is the
monorepo-wide-consumed change, requiring the full verification workflow
across every service that calls `ApplyFields`, not just the ones this
design otherwise touches (§8). **New in rev 15**: `pkg/requesthandler/ai_aicalls.go`
gains `AIV1AIcallListen` (§5.1); mock regen.

`bin-dbscheme-manager` — three generated migrations (two from rev 1–3,
plus `bin-customer-manager`'s system-customer seed if §11 item 5
concludes one is needed)

`bin-openapi-manager`, `bin-api-manager` — `Origin` in the spec, both
values (**corrected in rev 5, review round 4 finding M8**: rev 4 said to
document only `proactive` and leave `listen_internal` undocumented, but
`listen_internal` rows are created through the same `ToolHandle` →
`messageHandler.Create` → `ConvertWebhookMessage` path as any other
message, so the value genuinely reaches a tenant's webhook payload —
leaving it undocumented while it's on the wire is worse than documenting
it plainly as "internal bookkeeping; do not depend on this value's
presence or meaning" in the RST `origin` field description, §5.10.2),
regen, RST + build. `ToolHandle`'s new parameter and
`AIV1AIcallToolExecute`'s new argument are internal RPC surface, not
public API — no OpenAPI change from those. **New in rev 15, and this one
*is* public API surface — on the Agent-facing `/service_agents/*` tree,
not the top-level Admin one (corrected in rev 16, review round 13
finding BLOCKING-1)**:
`openapi/paths/service_agents/aicalls/id_listen.yaml` (new file), regen;
`bin-api-manager/server/service_agents_aicalls.go`'s
`PostServiceAgentsAicallsIdListen`,
`pkg/servicehandler/serviceagent_aicall.go`'s
`ServiceAgentAIcallListen` + its `ServiceHandler` interface entry in
`main.go` (§5.1).

`monorepo-javascript` — both `CaseInsightAssistantPanel` files + tests,
**plus (new in rev 15) the `POST /service_agents/aicalls/{id}/listen`
call site in each (§5.10.1a, path corrected in rev 17)**

**Scope note on `cachehandler` (review-round-2 F10), acknowledged rather
than resolved here:** today the package is a pure JSON entity-snapshot
cache — two primitives, `getSerialize`/`setSerialize`
(`handler.go:17-41`), fixed 24h TTL, no raw Redis data-structure or lock
primitive of any kind. §5.3.3/§5.3.4/§5.4.1 add `SADD`/`SREM`/`SMEMBERS`,
`RPUSH`/`LTRIM`/`EXPIRE`, `LPOP count`, `SET NX EX`, and `INCR`; §5.2.2
(rev 18) adds a compare-and-delete `EVAL` for the start lock's release
(**added to this enumeration in rev 18, review round 15 finding LOW-3**)
— a second, structurally different responsibility (ephemeral buffers +
distributed rate limiting + a mutual-exclusion lock) that does not fit
the existing package's shape. This design does not resolve which is right (extend
`cachehandler` with a new file, or give listen state its own small
package sharing the same Redis client) — it belongs with whoever
implements this, informed by how the team wants shared-infra Redis usage
organised. Flagged, not decided, in §11 item 9 (renumbered as this
document grew; corrected in rev 17, review round 14 finding LOW-5, from
a stale "item 7" reference).

---

## 10. Review-response matrix (round 1 → rev 2)

| # | Review finding | Resolution | Where |
|---|---|---|---|
| B1 | `messagehandler` is not the dispatch path; `SendReferenceTypeOthers` rotates the pipecatcall and kills in-flight answers; 3s cooldown drops most segments; assistant text always persisted | `Send` is not used at all. Listen turns run on their own throwaway pipecatcall id, never written to the AIcall row. Output suppressed by `RunLLM:false` **and** a pipecatcall-identity guard on all four pipecat message handlers | §4, §5.4.3, §5.4.4 |
| B2 | Per-segment 100-message replay blows up cost and evicts the agent's Q&A history (and, in fact, the system prompt) | Transcript is never a message row; context is assembled explicitly from the frozen prompt snapshot + ≤10 Q&A rows + ≤40 transcript lines. Batching/debounce is core, not an optimisation | §5.3.3, §5.3.4, §5.4.2 |
| B3 | Every segment becomes a customer webhook **and** a panel-visible `role=user` bubble | Segments live only in Redis. The sole new row per notification is the intended proactive message | §5.3.3, §3 (non-goal row) |
| B4 | Hangup cleanup cannot work — `GetByReferenceID(evt.ID)` never matches an Insight AIcall (its `ReferenceID` is the Case id) | New indexed `listen_call_id` column + a second, plural lookup in `EventCMCallHangup` | §5.7.1, §5.8 |
| B5 | `TranscribeV1TranscribeStop` signature wrong; per-pod `HostID` regenerated on restart | Hangup path needs **no** stop RPC (transcribe-manager's own `EventCMCallHangup` already stops every transcribe on the call, owner-agnostic). Terminate path uses the correct `(ctx, hostID, transcribeID)` with `HostID` fetched fresh, and a stated fallback on failure | §5.7.1, §5.7.2 |
| B6 | Which `customer_id` the listen transcribe runs under is unresolved and load-bearing | Decided: `IDAIManager`, with the tenant check relocated to listen-start time and provenance-checked at event time; reuse rule made owner-aware and language-tolerant | §5.2.1, §5.2.2 |
| B7 | Trigger never fires on the Case-resume path (`start.go:512-513` returns with no status transition) | Hook moved to `Start`'s dispatch branch, covering all three success returns; a named regression test per return | §5.1 |
| H1 | `notify_agent` breaks the documented read-only invariant and its guarding test; `tool_names=["all"]` auto-grants | Invariant explicitly relaxed with a named exception; comment, test (`knownSanctionedWrite`), and the 2026-07-30 design doc all updated; auto-grant blast radius stated and bounded | §5.5.2 |
| H2 | "Origin comes free from the tool-call record" is false; `RoleNotification` is the existing precedent | Rev-1 premise confirmed wrong and dropped. `role=notification` **evaluated and rejected** (it is skipped in LLM context, so the AI would forget its own notification). Decision: `role=assistant` + first-class `Message.Origin`. Ordering constraint discovered and handled | §5.6.1, §5.6.2, §5.6.3 |
| H3 | `transcript.*.created` also carries DELETE events | `TMDelete != nil` guard at intake, with the upstream bug cited and left for its own ticket | §5.3.2, §11 |
| H4 | Event-intake volume unbounded and unaccounted for | Per-event work reduced to one Redis `GET` (no DB, no RPC); volume explicitly sized; dynamic per-transcribe binding documented as a pre-analysed escape hatch with its leak caveat | §5.3.2, §3 |
| M1 | `AIcallSet` secondary keys are never invalidated; a listen cache index would go stale and collide on nil-UUID | No cache index. The resolver key is explicitly written/deleted by the listen lifecycle and is deliberately outside `AIcallSet`'s snapshot scheme; hangup lookup uses a filtered `AIcallList` on an indexed column | §5.3.2, §5.7.1 |
| M2 | Schema/plumbing scope understated; `transcribe_id` was deliberately dropped; `Metadata` already exists for flags | Reduced to one column + two `Metadata` keys, with a full plumbing checklist and an explicit note that `transcribe_id` is **not** being re-added | §5.8 |
| M3 | Commented-out `processEventTMTranscriptCreated` ghost | Deleted; the file holds the real implementation | §5.14 |
| M4 | `kase.ReferenceType`/`ReferenceID` are `string`; no `ReferenceTypeCall` constant; wrong doc path | Constant added to the owning model; checks rewritten to `!= ReferenceTypeCall` + `uuid.FromString`; doc path corrected | §5.1 step 5 |
| M5 | `TranscribeV1TranscribeStart` real signature | Full argument list written out with `provider` and `onEndFlowID` justified, plus an argument-shape test | §5.2.2, §7.11 |
| M6 | "square-admin/square-talk are separate repos" | **Corrected**: both live in the single `monorepo-javascript` repo. Two PRs total, backend first | §5.10 |
| M7 | RST sync missing from the plan | Added, with named files and the clean-rebuild procedure | §5.10.2 |
| L1 | `transcript.Transcript` embeds `CustomerID` | Noted, but under B6's resolution it holds `IDAIManager`, so it is not usable as a tenant check; used only as a provenance sanity assertion | §5.2.1 |
| L2 | Concurrency bound is per-Case, not per-customer | Corrected, with the N-Cases-N-sessions consequence and the shared-STT mitigation spelled out | §5.11 |
| L3 | `AIcallUpdate` bumps `tm_update`, feeding the `Send` cooldown | Reduced to exactly two writes per listening session (start, stop) — never per turn — so the window is bounded and pre-agent-input; decoupling the cooldown recorded as a follow-up | §5.2.4, §11 |
| — | §3 non-goals table, §4.4 self-flagged speaker mapping (confirmed correct; do not re-litigate) | Preserved verbatim / carried forward unchanged | §3, §5.9 |

### 10.1 Review-response matrix (round 2 → rev 3)

Round 2 (an independent review, deliberately skeptical of rev 2's own
claims, run against the actual code rather than the review-round-1
matrix) confirmed most of rev 2's fixes hold, and found 3 BLOCKING + 4
HIGH + 4 MEDIUM + 2 LOW new issues, all introduced by rev 2's own new
mechanisms.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| F1 | Single-valued Redis resolver key contradicts the N-AIcalls-share-one-transcribe design; second listener silently steals the first's mapping, either's cleanup deletes the shared key | Resolver key changed to a Redis set (`SADD`/`SREM`/`SMEMBERS`); every listening AIcall keeps its own membership | §5.2.4, §5.3.2, §5.7.3 |
| F2 | The "tool_calls ordering" defect and fix in rev 2's §5.6.3 were mis-diagnosed; the reordering is a no-op because the tool-call row is filtered out of context by empty `content`, not by ordering | Re-diagnosed against `run.py:450` directly. The real (and pre-existing, feature-independent) defect is an orphaned `tool`-role row with no preceding `tool_calls` entry. Documented plainly, not silently fixed here, escalated to an immediate-verification item rather than a deferred ticket | §5.6.3, §11 item 2 |
| F3 | `RunLLM:false` has undocumented error-path, override, and Q&A-turn-collision holes; the "every other tool uses RunLLM:true" supporting claim is false | Rewrote as a three-layer defense: (a) `RunLLM:false` as a best-effort hint with its real caveats stated, (b) the pipecatcall-identity guard, (c) a new explicit reject-if-invoked-from-the-agent's-own-turn check in `toolHandleNotifyAgent`. Corrected the false claim (all six *Insight* tools use `RunLLM:true`, not "every other tool") | §5.4.4(a)(c) |
| F4 | The pipecatcall-identity guard can false-positive against a genuine reply if a post-`Send` cache write transiently fails, and `contact_case` has no termination-triggered backstop the way `conversation` does | Guard now re-reads the AIcall bypassing cache before dropping on a mismatch; the missing backstop is stated explicitly rather than silently assumed covered | §5.4.4(b) |
| F5 | Listen-vs-`ai_summary` transcribe collision only analysed in one direction; the reverse (listen starts first) makes a later summary attempt fail with `TRANSCRIBE_ALREADY_PROGRESSING` | `summaryhandler.startReferenceTypeCall` made reuse-tolerant, symmetric with listen's own rule, via a shared `ensureIDAIManagerTranscribe` helper | §5.2.2a |
| F6 | Listen-turn context assembly omits `InsightSystemPrompt` (the platform's own hallucination/tool-leakage guardrails for Insight AIs) | Added as message #1, ahead of the customer's own prompt snapshot | §5.4.2 |
| F7 | A proactive notification is claimed to be "one new row"; it is actually three (tool-call row, tool-result row, proactive row), all webhook-published and panel-rendered, contradicting §4's "invisible unless notified" claim | Stated plainly as a pre-existing, feature-independent `ToolHandle` shape made more visible by listening; a frontend render filter (not a webhook suppression) is the shipped mitigation, with the larger webhook-level fix recorded as a separate follow-up | §5.6.4, §5.10.1, §11 item 6 |
| F8 | "transcribe-manager's `EventCMCallHangup` already stops every session, owner-agnostic" overstates a pod-local mechanism as a platform-wide guarantee | Reasoning corrected: the real backstop is that hanging up the call closes the Asterisk WebSocket feeding the STT stream, independent of whether the DB status write on a non-owning pod is accurate. Conclusion (no stop RPC needed here) unchanged; justification rewritten to match what the cited code actually establishes | §5.7.1, §5.7.2 |
| F9 | Redis `SET NX EX` debounce is a rate limiter, not a lock; a turn that fails after popping `pending` loses those lines beyond the 40-line window; `LPOP count` needs Redis ≥6.2, unverified against the deployed server | Both risks stated explicitly as accepted/bounded rather than left implicit; Redis-version confirmation added as an implementation-time open item | §6 (new row), §11 item 7 |
| F10 | The `cachehandler` change (six Redis primitives beyond its current two) is a second, structurally different responsibility, understated as "+ mock regen" | Acknowledged explicitly as an open scope question (extend `cachehandler` vs. a new package) rather than a decided detail | §9 scope note, §11 item 7 |
| F11 | "STT sessions per call: 1" understates cost — `DirectionBoth` is two independent STT streams, not one | Split into "transcribe sessions" (1, still deduped) vs. "STT streams" (2) in the bounds table | §5.11 |
| F12 | Metric names double the `ai_manager` namespace prefix | Corrected to bare `Name:` values, namespace implicit per existing convention | §5.13 |
| F13 | §5.3.1's "three coupled places" was read as undercounting the golden-test edit count | Table already enumerated all four edits within `binding_golden_test.go` (expected slice, length check, message string, doc comment) grouped under one of the three *places*; left as-is, no defect found on re-check |  — |

### 10.2 Review-response matrix (round 3 → rev 4)

Round 3 (another independent, skeptical-by-instruction review, run against
the code again rather than trusting rev 3's own §10/§10.1) confirmed most
of rev 3's fixes hold (F1, F6, F8, F11, F12, and the §5.6.3
re-diagnosis all checked out against the code directly), and found 4
BLOCKING + 3 HIGH + 5 MEDIUM new issues — every one of them introduced by
rev 3's own new mechanisms, not by anything earlier.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| B1 | Rev 1's context-eviction defect (system prompt/Q&A history pushed out of `getPipecatcallMessages`'s 100-row window) resurfaces through the tool-call/tool-result rows a listen turn's own tool use writes | New `Origin=listen_internal` tag on listen-turn tool rows, excluded from `getPipecatcallMessages`'s query via a new `FieldOriginNot` filter — excluded at the SQL layer, not by fetching more and filtering in Go | §5.4.5 |
| B2 | §11 item 2 (orphaned `tool`-role message) deferred to a follow-up ticket when this feature's own happy path (an unprompted `notify_agent` call) actively creates the condition | §5.4.5's fix also removes listen's own tool rows from ever being replayed anywhere, so this feature can no longer produce an instance of the defect; the general (agent-initiated) case is restored to a follow-up item since it is no longer made worse by this feature | §5.4.5, §11 item 3 |
| B3 | §5.2.2a's "make `summaryhandler` reuse-tolerant" fix for the listen/`ai_summary` transcribe collision breaks `summaryhandler`'s own read path (`contentGetTranscripts`'s unpinned `size=1` list) and lifecycle assumptions (listen's `owns=true` stop can cut off a summary that later attached to the same session) | §5.2.2a deleted. Listen gets its own system customer id (`IDAIManagerListen`, distinct from `ai_summary`'s `IDAIManager`), so the two features' transcribe sessions are provably independent — no reuse, no hand-off, no shared lifecycle, `summaryhandler` needs no change at all | §5.2.1, §5.2.2 |
| B4 | §5.4.4(c) (reject `notify_agent` when called from the agent's real Q&A turn) assumed `ToolHandle` already receives the invoking pipecatcall id; it does not — `runner.go:457` never forwards `pc.ID` to ai-manager | New §5.4.3a: pipecatcall id threaded through `runner.go` → `bin-common-handler`'s RPC → the wire DTO → `ToolHandle`'s signature, as a real, scoped cross-service change reflected in §8/§9. §5.4.4(c) and §5.4.5's `Origin` tagging both consume this one signal | §5.4.3a |
| H1 | The pipecatcall-identity guard's cache-bypass re-read, applied to all four original handlers, would put an uncached DB read on the highest-volume one (`EventPMMessageBotLLMIntermediate`, fired per token chunk) | Cache-bypass re-read scoped to `EventPMMessageBotLLM` only (the one that persists); `EventPMMessageBotLLMIntermediate` drops on a plain cached mismatch — its only cost on a false positive is one skipped, non-user-visible intermediate webhook | §5.4.4(b) |
| H2 | Two of the four guarded handlers (`EventPMMessageUserLLM`, `EventPMMessageUserTranscription`) can never fire from a listen turn (`STTTypeNone`), so guarding them only adds a per-utterance AIcall lookup to a platform-wide hot path for a condition that cannot occur | Guard narrowed to the two handlers that can actually fire from a listen turn; the other two left unchanged, exactly as before this feature | §5.4.4(b) |
| H3 | Frontend render-filter condition used `toolCalls` (camelCase); the real wire field is `tool_calls` (snake_case, matching every other field the panels already read) — the filter as written would never have fired | Corrected to `msg.tool_calls`, `msg.content`, `msg.role` throughout §5.6.4/§5.10.1 | §5.6.4, §5.10.1 |
| M1 | §5.7.3's clear-state steps used a value (the transcribe id) in step 2 that step 1 had already deleted in step 3's original ordering — self-contradictory as written | Reordered: read the transcribe id from the AIcall already in hand, `SREM` using it, *then* clear the DB metadata | §5.7.3 |
| M2 | `UpdateListenState` `SADD`s the new resolver-set membership but never `SREM`s a stale old one when the §5.1.1 idempotency check restarts listening with a fresh transcribe | Old membership is now explicitly `SREM`'d before the new `SADD`, when a prior `listen_transcribe_id` existed | §5.2.4 |
| M3 | The six Insight tools' `RunLLM: true` citation was attributed to `tool_insight.go` (which contains no `RunLLM` occurrences); the real location is `pkg/toolhandler/definitions.go` at the same line numbers | Citation corrected | §5.4.4(a) |
| M4 | §5.4.2 item 4's `Role ∈ {user, assistant}` filter still admits empty-content tool-call rows that `run.py:450` discards anyway, so the effective Q&A context is smaller than the configured 10-row budget | Noted as a known, minor inefficiency; not blocking (§5.4.5's `Origin` exclusion is the mechanism that matters for correctness, this is a small further headroom improvement left for implementation) |  — |
| M5 | §5.6.4's render-filter coverage claim ("every existing... panel noise") overstated: it does not hide `role=system` rows (`startInitMessages`, `start.go:812-819`), which both panels already render today | Scope note added: the filter covers tool-call/tool-result noise specifically; `role=system` rendering is a separate, pre-existing, out-of-scope concern | §5.6.4 |

### 10.3 Review-response matrix (round 4 → rev 5)

Round 4 (again an independent, skeptical-by-instruction review) found
that of rev 4's three new mechanisms, only §5.2.1 (the separate system
customer id) held up cleanly; §5.4.4(c) and §5.4.5 each had a real bug,
and the reviewer's own closing assessment was that the design's skeleton
is sound and consistent with the code — the remaining issues were all in
implementation-level details of the newest additions, not architecture.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| B1 | §5.4.4(c)'s cache-bypass re-read sat in the wrong branch: it neither caught the false-allow case it was written for, nor avoided creating a new one | Rewritten as a single always-fresh-read (not on a hot path, so no reason to trust the cache at all here), removing the two-branch logic entirely | §5.4.4(c) |
| B2 | A `pipecatcallID` of `uuid.Nil` (old `pipecat-manager` build, rolling-deploy window) was unhandled; depending on how §5.4.4(c)/§5.4.5 read it, could permanently mistag real Q&A content as `listen_internal` | `uuid.Nil` explicitly treated as "assume a real Q&A turn" in both places — the fail-safe direction, since guessing wrong that way costs one rejected tool call, never data corruption. Wire field made optional so no deploy ordering is forced | §5.4.3a, §5.4.4(c), §5.4.5 |
| B3 | `ApplyFields` was mis-located in `bin-ai-manager` (actually `bin-common-handler/pkg/databasehandler`, shared by every service); the proposed `FieldOriginNot` mechanism was unspecified and would have produced a SQL error on every Q&A turn | Correct location cited; mechanism decided as a generic `databasehandler.NotEq` wrapper type (not another hardcoded field-name special case), verification scope widened to the whole monorepo for this one change | §5.4.5 step 3, §8, §9 |
| B4 | Excluding only listen-internal rows narrows but does not close the context-eviction risk — proactive rows and the agent's own Q&A tool rows still compete for the 100-row window and can still evict the system prompt | `getPipecatcallMessages` restructured into two fetches: the leading system row(s), always included regardless of window pressure, plus the capped, `NotEq`-filtered rest | §5.4.5 step 4 |
| H1 | Guarding all four original pipecatcall-message handlers would put an uncached DB read on the highest-volume one; narrowed to two, but the *justification* given (the other two "can never fire from a listen turn") was correct while the guard itself was still overbroad on the two it did cover | (carried forward from round 3, no new action — round 4 confirmed round 3's H1/H2 fix as correct) |  — |
| H2 | `Origin` tagging in `ToolHandle` was not scoped by reference type, so an ordinary `conversation`/`task`/`none` AIcall's routine pipecatcall rotation (via `Send()`) would be mistagged `listen_internal` and silently lose history | Tagging rule now requires `ac.ReferenceType == ReferenceTypeContactCase` in addition to the pipecatcall-id mismatch | §5.4.5 step 2 |
| H3 | `notify_agent` was missing from the public OpenAPI tool-name enum and the Insight-tool-list prose docs, unlike every prior Insight tool | Added explicitly to §5.5.1 and §9 | §5.5.1 |
| H4 | The cache-bypass re-read's cited code path (`dbhandler.AIcallGet(skipCache)`) doesn't match how `EventPMMessageBotLLM` actually fetches the AIcall (an RPC, `AIV1AIcallGet`, not a direct `dbhandler` call) | Corrected: the RPC client gains the optional cache-bypass argument instead, one hop further down than rev 4's snippet showed | §5.4.4(b) |
| H5 | No explicit migration-before-deploy ordering; `Origin`/`ListenCallID` existing in the Go struct changes every message/AIcall query's `SELECT`, so a code deploy landing before its migration is a hard outage, not a soft degradation | Made an explicit, ordered rollout step rather than an implicit assumption | §8 |
| M1 | `IDAIManagerListen` not registered in `bin-customer-manager`'s known-system-id whitelist | Confirmed the listen path never traverses that gate (so this doesn't block), and stated the omission is deliberate scope discipline, not an oversight | §5.2.1 |
| M2 | §5.4.3a's "one new parameter" understated the change — `mapFunctions`' shared signature is used by 21 handlers | Clarified: only `ToolHandle` and `toolHandleNotifyAgent` need the new parameter; the other 20 handlers are unaffected | §5.4.3a step 4 |
| M3 | Adding `FieldOriginNot` to `FieldStruct` would expose an unused, unnecessary RPC filter surface (`FieldStruct` only gates external-RPC-visible filters; `getPipecatcallMessages` builds its filter map directly) | Not added — the existing `FieldOrigin` constant plus the new `NotEq` wrapper is sufficient, no `FieldStruct`/`ConvertFilters` change needed | §5.4.5 step 3 |
| M4 | The `RunLLM: true` line-number citation for the six Insight tools was still wrong after round 3's file-path fix — it reused the `RunLLM: false` tools' line numbers | Corrected to the six tools' own lines (754-755, 785-786, 824-825, 849-850, 879-880, 906-907) | §5.4.4(a) |
| M5 | `args.pop("run_llm", …)` cited at `tools.py:~60`; actual location `tools.py:105` | Corrected | §5.4.4(a) |
| M6 | The stated cost reason for not guarding two handlers ("adds a per-utterance AIcall lookup") was factually wrong — both already do that lookup today; the real reason is structural (the condition cannot occur) | Cost claim removed, structural reasoning kept (conclusion was already correct) | §5.4.4(b) |
| M7 | `get_aicall_messages` (an existing tool) bypasses `getPipecatcallMessages` entirely and can leak `listen_internal` rows' raw JSON into an LLM answer | Acknowledged as a real, lower-severity gap and left as a follow-up rather than fixed here | §5.4.5, §11 |
| M8 | `listen_internal` was to be left undocumented in the public RST while still appearing on actual webhook payloads | Documented plainly instead (as an internal-bookkeeping value, not relied upon), since hiding a value that's genuinely on the wire is worse than documenting it | §9 |
| M9 | Two items in §11 (Redis version, speaker mapping) are still open and self-marked blocking | Correct as stated — both remain open pending empirical/operational confirmation, not resolved by this revision |  — |
| L1–L3 | Minor citation/line-number errors (`PromptSnapshot` location, missing files in §9, raw-vs-`.String()` UUID convention) | Corrected where cited above; `.String()` convention left to implementation (both forms work through `ConvertFilters`) | various |

### 10.4 Review-response matrix (round 5 → rev 6)

Round 5's closing assessment: the design's skeleton reached a stable
point (no mechanism needed re-architecting), but two of rev 5's own new
mechanisms each had one remaining bug, both traced to the same root
cause — resolving "is this a listen turn?" in two different places with
two different levels of trust in the cache.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| B1 | §5.4.4(c)'s reject-guard used a fresh, cache-bypassing read (correct), but §5.4.5's `Origin` tagging — the more dangerous of the two consumers, since a wrong tag is permanent — still used the cached `c.PipecatcallID` | Unified: one fresh read in `ToolHandle`, shared by both the tagging decision and the guard (passed to `toolHandleNotifyAgent` as an already-resolved `listenTurn bool`, removing its separate read entirely — also cuts one RPC per tool call) | §5.4.5 step 2, §5.4.4(c) |
| B2 | `pipecatcallID != c.PipecatcallID` proves only "not the currently-bound turn," not "a listen turn" — a real Q&A tool call arriving just after `interruptPreviousPipecatcall`'s best-effort rotation could match by coincidence | Added a positive signal: `ListenCallID != uuid.Nil` (only set while a real listen session is active, per §5.8/§5.7.3), so the predicate can't fire outside an actual listening window regardless of pipecatcall-id timing | §5.4.5 step 2 |
| H1 | §8 claimed "zero behaviour change" with the flag off; at least 4 of rev 4/5's changes (context-assembly restructure, stale-reply guard, `Origin` tagging's code path, frontend filter) are not actually flag-gated | Replaced with an explicit table: which changes are flag-gated (none directly — `Origin` tagging is gated *indirectly* through the listen-state check) vs. always-active as general fixes, with guidance on what to actually watch during the code-only deploy window | §8 |
| H2 | The two-fetch context assembly assumed fetch (1) returns oldest-first and asserted `InsightSystemPrompt` "is not applicable" to this path; `MessageList` actually returns newest-first for both fetches, and `InsightSystemPrompt` is exactly what this path serves | Both fetches documented as newest-first, both reversed the same way before merging (matching the existing single-fetch reversal this replaces); the `InsightSystemPrompt` claim removed and corrected | §5.4.5 step 4 |
| M1 | §9 omitted the `dbhandler`/`DBHandler` interface changes rev 5 actually requires (cache-bypass `AIcallGet`, a `tm_update`-preserving `AIcallUpdate` variant) | Added explicitly | §9 |
| M2 | §5.4.4(b)'s claim that the cache-bypass RPC change is "confined to `bin-ai-manager`" is wrong — `AIV1AIcallGet`'s client lives in `bin-common-handler` | Corrected, with the concrete shape (query parameter or sibling route) specified | §5.4.4(b) |
| M3 | The `NotEq` wrapper's "any field, any service, generic" framing overstated its safety — `ApplyFields`'s existing per-type normalization (UUID `.Bytes()`, the `deleted` bool special case) isn't applied to `NotEq`'s bare pass-through | Scoped explicitly to `Origin` (a string field) for this design; broader safety left as an explicit non-claim rather than an implied one | §5.4.5 step 3 |

### 10.5 Review-response matrix (round 6 → rev 7)

Round 6 was explicitly scoped by the reviewer to rev 6's own changes
only, per round 5's recommendation — settled architecture and long-closed
findings were not re-litigated. It found rev 6's B1 fix (unifying the
fresh read) was real, but the comparison that read fed was still not a
genuine positive listen-turn signal, plus three document-consistency
gaps left over from the mechanism's rapid iteration across rev 4-6.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| F1 | `ListenCallID != uuid.Nil` is a constant `true` for the entire duration of an active listening session, so it adds no discrimination exactly when the predicate is ever consulted — a real Q&A tool call delayed behind a best-effort pipecatcall rotation could still be mistagged `listen_internal` | Replaced with a direct, positive signal: §5.4.3 registers each listen turn's throwaway pipecatcall id in a Redis set at mint time; `listenTurn` becomes a `SISMEMBER` check against that set, which a real Q&A turn's id was never added to, regardless of timing | §5.4.3, §5.4.5 step 2 |
| F2 | The unified fresh read (rev 6) ran unconditionally for every tool call on every AIcall type, with a debounce-based cost justification that only applies to listen turns | Fixed as a side effect of F1's rewrite: the (now Redis, not DB) check is pre-gated on `c.ReferenceType == ReferenceTypeContactCase` (an immutable, cache-safe field) before any Redis call is made, so non-`contact_case` AIcalls pay nothing | §5.4.5 step 2 |
| F3 | §8's "rollback stops in-flight sessions" claim was aspirational — nothing on the intake or evaluation path read `AIcallListenEnabled`, so a running session would ignore a rollback until call-end or the turn cap | Added as `RunListenTurn`'s first precondition; §8's rollback text and table updated to describe the real (bounded, ~one evaluate-interval) latency instead of "immediately" | §5.4.1 step 0, §8 |
| F4 | Several passages in §6, §7, §9 still described rev 5/rev 6's now-superseded mechanism (a fresh `AIcallGet` shared via `toolHandleNotifyAgent`'s own call, `pipecatcallID` passed as a raw parameter rather than the resolved `listenTurn bool`) | Swept and corrected in the same pass as F1's rewrite, since the mechanism change made them stale | §5.4.3a step 4, §6, §7 items 10-12, §9 |
| F5 | §6 and §7 still said "all four" pipecat message handlers, though §5.4.4(b) narrowed the guard to two back in rev 3 | Corrected to "two" in both places | §6, §7 item 5 |
| F6-F8 | Minor: the `NotEq` snippet's shape doesn't match `ApplyFields`' real `switch`/`case` structure; "same shape as the `conversation` branch's guard" overstates similarity (that guard is still cache-first); the two-AIcall-read-per-tool-call shape (`h.Get` plus the listen-turn check) wasn't stated as deliberate | Not addressed in rev 7 — reviewer's own assessment was these are cosmetic and can be fixed while touching the file during implementation, not blocking |  — |

### 10.6 Review-response matrix (round 7 → rev 8)

**Round 7 APPROVEd rev 7 outright — zero BLOCKING findings, architecture
confirmed stable, monotonically decreasing severity across rounds 5-7.**
It offered 7 localized findings (N-1 through N-7) explicitly as
pre-implementation recommendations, not approval conditions. rev 8
closes all 7, on the view that the loop's own five-round history is that
a "recommended, not required" finding left in the document tends to
resurface as a defect in the next round anyway (as F1 in round 6 did with
rev 5's own deferred residual, and B3 in round 4 did with rev 3's).

| # | Review finding | Resolution | Where |
|---|---|---|---|
| N-1 | §5.4.5 step 2's Redis `SISMEMBER` failure path (`return nil, err`) contradicts §6's "Redis unavailable → Q&A completely unaffected" row — a Redis outage would now fail every `contact_case` Insight tool call, not just listening | Degrades to `listenTurn=false` instead — provably correct during an outage, since §5.4.3's `SADD` and §5.3.4's lock are failing too, so no genuine listen turn can exist at that moment. §6's row restated to describe the degrade path precisely rather than claim the keys are never touched | §5.4.5 step 2, §6 |
| N-2 | §5.4.3's turn-id `SADD` error was silently discarded (`_ = ...`); a failed registration lets the turn proceed unregistered, reproducing the exact permanent-mistagging failure mode §5.4.5's B1 fix (round 5) exists to prevent | Turn aborts on a registration failure instead of proceeding, metered via a new `skipped_register_failed` result | §5.4.3 |
| N-3 | §5.4.1's flag check (step 0) ran before `c := h.Get(aicallID)` (step 1), but `clearListenState` (called on the flag-off path) is documented as requiring the caller to already hold `c` — the same class of self-contradictory step ordering review round 4's M1 caught in §5.7.3 | Flag check merged into step 1's require-list, after `c` is fetched | §5.4.1 |
| N-4 | The flag-off path said "clear listen state" while the turn-cap path said "stop listening entirely" for what should be the same kind of exit; a bare state clear doesn't stop a still-running owned transcribe, so a rollback could strand a billed STT session with its handle lost | Both paths now go through the same full stop (§5.7.2's owned-transcribe stop included) | §5.4.1, §8 |
| N-5 | New `skipped_disabled`/`skipped_register_failed` outcomes had no corresponding metric labels | Added to `aicall_listen_turn_total`'s label list, plus a new `aicall_listen_membership_check_failed_total` for N-1's degrade path | §5.13 |
| N-6 | `aicall_listen_buffer_ttl_hours`'s description ("TTL on all listen keys") no longer covered `ai:listen:turnpcid:*`'s separate, shorter TTL; §5.7.3's cleanup list didn't mention (or explain the deliberate omission of) that key | Both corrected/clarified | §5.12, §5.7.3 |
| N-7 | §5.4.5 step 2's listen-turn check didn't state where in `ToolHandle`'s existing flow it runs, relative to the tool-call row it's meant to gate | Stated explicitly: between the existing `h.Get` and the first `messageHandler.Create`, so a failure here never leaves an already-written row behind | §5.4.5 step 2 |

### 10.7 Review-response matrix (round 9 → rev 11, revised in place)

**Round 9 reviewed only rev 11's own new content** (§0 row 11, §5.1.1's
new guard, §5.9, §11 items 1/1a/1b), per this loop's established practice
of scoping a round to what actually changed rather than re-litigating
settled sections (rounds 6 and 8 did the same). Verdict: REQUEST_CHANGES —
every file:line citation checked out, but the *reasoning* built on top of
them did not: the structural claim was broader than the code supports,
the newly-declared residual risk was likely contradicted by code the
reviewer traced one function away from what rev 11 cited, and the new
guard's failure mode was both undiscussed and, on the normal path, wrong.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| BLOCKING-1 | The participant-count guard's one-shot form reads `len(ChannelCallIDs) != 2` for the entire queue-wait and agent-ring period (`Call.ConfbridgeID` is only set once the B-leg's join channel reaches `ChannelEnteredBridge`, which happens after answer) — so a screen-pop-timed or ring-timed `ensureListen` call would silently never listen on an entirely normal call, with no retry and nothing to explain why | Rebuilt as a bounded, metered retry inside the existing `ensureListenAsync` goroutine (renumbered step 7): polls until the confbridge reaches 2 live parties, the call itself becomes non-live (step 6 exits the loop), or a wait budget elapses; each distinct give-up reason gets its own metric label | §5.1.1 step 7, §6, §5.12, §5.13, §11 item 13 |
| HIGH-1 | "`case_create`/`ai_talk` only ever execute on the A-leg's own activeflow" is broader than the code supports — `actionHandleCall` can chain a customer-authored flow onto a new leg, and the agent-dial RPC accepts a caller-supplied `FlowID`; only the two *specific* system-generated B-leg flow builders actually checked (`generateFlowForAgentCall`, the `connect` action's B-leg flow) are guaranteed empty of `case_create`. The `ai_talk` half of the claim was also unverified and, on inspection, irrelevant (`ai_talk` never targets `contact_case`) | Restated as "no *system-generated* B-leg flow can carry `case_create`," citing both flow builders; named the two customer-authored-flow escape hatches explicitly as out of this guarantee's scope; dropped the `ai_talk` clause; added the independent, stronger `in == Case.Peer` invariant (backed by `case_create`'s own `isCRMEligiblePeer` check) as the thing that actually does most of the load-bearing work | §5.1.1 step 7, §5.9 |
| HIGH-2 | The declared "agent-outbound call inverts the mapping" residual risk is very likely impossible: `actionHandleCaseCreate`'s own `isCRMEligiblePeer` check excludes agent/extension/SIP/conference/AI peer types, so that exact scenario can't produce a Case at all — a real vulnerability was reframed as a phantom one, and a real, cheaper guarantee (`in == Case.Peer`) was left undiscovered | Reframed §5.9/§11 around the actual narrower vector (an inbound call whose CRM-eligible peer happens to be staff dialing in via a plain DID, not through the agent-dial path) and added the `in == Case.Peer` invariant as newly-stated, code-backed closure for the scenario rev 11 originally worried about | §5.9, §11 item 11 |
| HIGH-3 | §5.9 claimed 3+-party confbridges "break the assumption outright" and that the guard "is not trusted" mid-call, describing a re-check mechanism that doesn't exist — the guard is start-time only, and a 3rd party joining after listening starts is entirely unguarded | Stated plainly: start-time only, no ongoing re-check; precision added that only `out` degrades (`in` stays correct regardless of party count, since it never depended on other parties' identity) | §5.1.1 step 7 (closing paragraph), §5.9, §6, §11 item 12 |
| MEDIUM-1 | Stale cross-references: "§5.2.3's Snoop/ExternalMedia tap" (§5.2.3 is Language selection, unrelated) and three "§5.1.1 step 6" citations that should say "step 7" (the guard, not call-liveness) | Corrected throughout §5.1.1 and §5.9 | §5.1.1, §5.9, §11 item 1 |
| MEDIUM-2 | The new guard added an RPC and a gate with no corresponding §6 error row, no confbridge liveness check (`TMDelete`/`Status`, asymmetric with the neighbouring steps' checks), no §5.13 metric label, and no §7 test coverage for its new branches | All four added: §6 gains new rows, step 7 checks confbridge `TMDelete`/`Status` alongside party count, §5.13 gains new `skipped_confbridge_*` labels (reduced to 2 after round 10's own HIGH-A removed the third), and coverage for every new branch is folded into §7 item 1 as a labelled sub-paragraph — **not** a separate "item 1a," since ordered-list sub-numbering is exactly what LOW-3 in this same round flagged as broken | §6, §5.1.1 step 7, §5.13, §7 |
| MEDIUM-3 | The guard silently narrows Goal 1's scope (every non-confbridged live call becomes permanently non-listenable) without updating §2/§3/§8 | Goal 1 (§2) and the multi-party non-goal row (§3) now state the narrowed scope explicitly; §8's flag-gating table gains a row for the new mechanism | §2, §3, §8 |
| LOW-1 | (Verification only, no defect) Is the guard's silent-no-op failure mode consistent with the neighbouring gates' shape? | Confirmed consistent — no change needed | — |
| LOW-2 | §5.9's original "`in` is audio arriving into that channel from the far end" wording is ambiguous and, read literally, backwards from Asterisk's actual read/write convention, even though the conclusion (`in=customer`) was correct | Reworded to state the channel's own read/write direction explicitly (`in` = read from the channel's own owner, `out` = written to them) | §5.9 |
| LOW-3 | `6a.`/`1a.`/`1b.` markdown list markers don't continue an ordered list and would render as plain paragraphs, breaking §5.1.1's and §11's numbering | §5.1.1's guard renumbered to step 7 (old step 7 → 8); §11's new items appended as 11-13 instead of sub-numbered under item 1 | §5.1.1, §11 |

### 10.8 Review-response matrix (round 10 → rev 12)

**Round 10 confirmed round 9's own findings were genuinely and honestly
fixed** — it independently re-verified BLOCKING-1/HIGH-1/HIGH-2/HIGH-3/
MEDIUM-1/MEDIUM-2/MEDIUM-3/LOW-2/LOW-3 against the current source rather
than trusting §10.7's description, and confirmed each closure rather than
rubber-stamping it. What it did not approve was one genuinely new defect
the BLOCKING-1 fix itself introduced, plus accuracy problems in how that
fix and its neighbours were described.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| HIGH-A | Step 7's original retry design fast-failed on `len(ChannelCallIDs) >= 3` once `call.Status == progressing`, reasoning that a `progressing` call was past the window where an extra party could be transient pre-answer noise — but `call.Status` reflects only the *listened leg's own* answer state, which is already `progressing` for this design's entire target window (queue-wait + agent-ring), so the fast-fail condition was true the instant any 3rd party appeared, not just once one had lingered. A documented platform pattern (`connect` with `early_media: true` and multiple destinations) legitimately produces a transient 3+-party state that settles to 2 within seconds — `confbridgehandler/joined.go:87-97` explicitly anticipates ringing legs inside the bridge — and the original fast-fail would have given up on such a call permanently, on possibly the only `ensureListen` invocation it ever gets (the same screen-pop scenario BLOCKING-1 itself was about) | Removed the fast-fail: any non-2 count now just keeps polling for the full wait budget, with no attempt to distinguish "still converging" from "stably wrong." The now-meaningless `skipped_confbridge_invalid_topology` label is removed; §7 gains an explicit 3→2-settle-within-budget test as the direct regression case | §5.1.1 step 7, §6, §5.13, §7 |
| MEDIUM-A | The BLOCKING-1 diagnosis itself misattributed the mechanism: it said `Call.ConfbridgeID` "is only populated once the *B-leg's* join channel" completes, then two sentences later described the A-leg "sitting alone" post-forward — which requires the A-leg's own `ConfbridgeID` to already be non-nil. `AddChannelCallID` sets `ConfbridgeID` for whichever call joins, A-leg included; it is `len(ChannelCallIDs)`, not `ConfbridgeID`'s nil-ness, that stays at 1 through the wait | Corrected the diagnostic paragraph to attribute the stuck state to the party count, not to `ConfbridgeID` | §5.1.1 step 7 |
| MEDIUM-B | The 30s wait budget was described as "well inside the goroutine's own outer timeout," but the document never actually sets that outer timeout, and the two precedents §5.1.1's intro cites for the detached-goroutine pattern don't bound this path either (`tool.go:191-199` is an unbounded `context.Background()`; `start.go:97-100` is an unrelated 5s fetch, six times smaller than the proposed budget) — the claim could ship unchecked and silently truncate the retry | Added an explicit, feature-specific `aicall_listen_ensure_goroutine_timeout_seconds` config (default `45`, strictly greater than the 30s wait budget), and added it to §11 item 13's implementation sign-off list | §5.1.1 step 7, §5.12, §11 item 13 |
| MEDIUM-C | §0's status line and row 11 both recorded round 9 as "1 BLOCKING + 2 HIGH + 3 MEDIUM + 3 LOW," undercounting against §0 row 11's own enumeration and §10.7's table, both of which list 3 HIGH findings | Corrected to "3 HIGH" in both places | §0 (status line, row 11) |
| LOW-A | "`len == 0` impossible given `ConfbridgeID != uuid.Nil`" asserted impossibility the design's own reasoning ten lines earlier (a stale non-nil `ConfbridgeID` via `leaved.go`'s goroutine) contradicts, and was also the one count the retry's original branching left unhandled | Resolved as a byproduct of the HIGH-A rewrite — the retry no longer branches on the specific count at all, only on `== 2` vs. everything else | §5.1.1 step 7 |
| LOW-B | §10.7's own MEDIUM-2 resolution text claimed "§7 gains item 1a" — reusing the exact broken list-marker pattern that same round's LOW-3 finding required removing | Corrected to describe what actually shipped: a labelled sub-paragraph folded into §7 item 1, not a separate numbered item | §10.7 (MEDIUM-2 row) |
| LOW-C | §8's flag-gating table said the new guard is "reached only after step 1's flag check already returned" — worded as if the guard *is* reached, when the point is that it never is, with the flag off | Reworded to "never reached — step 1's flag check returns first" | §8 |
| LOW-D | §11 item 1 credited review round 9 with surfacing "the two new residual-risk vectors" in items 11-12, but item 12 (call-transfer) was rev 11's own first-draft addition — round 9 only corrected its "refused vs. unguarded" framing (§10.7's HIGH-3 row records this accurately) | Reworded item 1's pointer to attribute each item correctly | §11 item 1 |

### 10.9 Review-response matrix (round 11 → rev 13)

**Round 11 APPROVEd rev 12 — the first of the two consecutive approvals
this loop's policy requires to close.** It independently re-verified every
one of round 10's findings against current source (not against §10.8's
own description of them) and confirmed all were genuinely and completely
fixed, including the substantive one (HIGH-A) — tracing the early-media
multi-destination scenario through four separate files rather than taking
§10.8's citation at face value. What remained was one MEDIUM (a summary
section, §2, that fell out of sync with the HIGH-A fix landing two rounds
after it was written) and six LOW findings — wording, a stale citation,
and one real GFM table-rendering break introduced while fixing something
else.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| MEDIUM-1 | §2 Goal 1 still said a live call in a 3+-party confbridge is "out of scope entirely" — true of rev 11's first-pass fast-fail design, no longer true after round 10's HIGH-A removed it: a *transiently* 3+ confbridge (the early-media scenario) now resolves via the same retry, with its own §7 regression test proving it. Only a *stably* non-2-party call is genuinely out of scope | Reworded to state the distinction explicitly, crediting round 10's HIGH-A fix as the reason rev 11's original wording no longer held | §2 |
| LOW-1 | The status line's own "pending review round 10" had gone stale relative to the sentence describing round 10's fix having already happened, and to the "needs round 11" clause immediately after it | Corrected the round number and restated the line around the loop's actual remaining condition ("N of 2 consecutive approvals") so it can't drift the same way again | §0 (status line) |
| LOW-2 | A stray blank line between rev 10's and rev 11's rows split §0's revision-history table into two tables under GFM — silently breaking table rendering for the two most load-bearing rows in the section, the same rendering-defect class as round 9's own LOW-3 | Removed the blank line | §0 |
| LOW-3 | §5.1.1's intro still cited `tool.go:191-199` as a "`context.Background()` + timeout" precedent, 140 lines upstream of where step 7/§5.12 had already disclaimed it (rev 12, MEDIUM-B) as actually unbounded | Corrected at the source to state plainly that `tool.go:191-199` has no timeout and `start.go:97-100`'s 5s is for an unrelated fetch, and that this feature's own timeout is purpose-built, not inherited | §5.1.1 intro |
| LOW-4 | §5.2.4 said the `tm_update`-bypass covered "this one write path" while concluding "start or stop" (self-contradictory), and separately still called the start-time write's cooldown-collision risk "negligible" — a claim the bounded retry (rev 11/12) invalidated, since the start write can now land up to 30s after panel open, squarely in a window an agent could be typing a real question | Restated plainly: both `UpdateListenState` (start) and `clearListenState` (stop) use the bypass; the now-false "negligible" framing for the start write is dropped. §5.7.3 step 3 also now names the bypass variant explicitly, closing the implicit reference LOW-4 flagged there | §5.2.4, §5.7.3 step 3 |
| LOW-5 | Step 7's diagnostic paragraph credited `AddChannelCallID` alone with setting `Call.ConfbridgeID`, when that specific write is actually performed by the sibling `CallV1CallUpdateConfbridgeID` RPC `Joined()` also calls | Named both writes and which one does which, in the same paragraph MEDIUM-A (round 10) had already corrected once | §5.1.1 step 7 |
| LOW-6 | Repeated panel re-opens during a long ring now spawn multiple concurrent, independently bounded retry loops (step 3's idempotency check can't short-circuit until `listen_transcribe_id` is set, which never happens while step 7 is still polling) — bounded and already covered by §5.2.2's reuse rule and §6's race handling, but previously undocumented, and relevant to interpreting `skipped_confbridge_not_ready`'s rate | Added an explicit paragraph to step 7 stating this consequence and its interaction with the metric | §5.1.1 step 7 |

### 10.10 Review-response matrix (round 12 → rev 14) — second consecutive approval, loop closes

**Round 12 APPROVEd rev 13 — the second of the two consecutive approvals
this loop's policy requires.** It independently re-verified all seven of
round 11's findings against current document text and current source
(including a direct check of both cited goroutine-timeout precedents and
the `joined.go` write attribution), confirmed each was genuinely fixed,
and swept the whole document one more time as the round meant to close
the loop — table structure, list numbering, metric/config/error-row
consistency, and a spot-check of the freshest citations. It found zero
BLOCKING and zero HIGH findings. The two MEDIUM and two LOW items below
are non-blocking: one is citation drift from a merge two revisions
earlier that a scoped rebase sweep missed, one is a genuine but small and
already-bounded concurrency edge case worth closing at negligible cost,
and two are self-contradictory attribution sentences pointing at the
wrong revision number. None changes a design decision, a failure mode, or
a rollout step — which is why, per this loop's policy, they do not reset
the consecutive-approval count and this design is **Approved**.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| MEDIUM-1 | Rev 10's rebase-reconciliation sweep (row 10, §0) was explicitly scoped to `bin-ai-manager` and its own text says so — but the same merge (`NOJIRA-Allow-caller-specified-transcribe-id`) also shifted `bin-transcribe-manager/pkg/transcribehandler/start.go` by ~46-52 lines, and that file was never re-checked. 7 citations across §5.1.1 step 6, §5.2.1, §5.2.2, §5.2.3, §5.11, and §6 (two rows) pointed at stale line numbers for code that still exists and still does exactly what was claimed (confirmed independently: the transcribable-status set, the dedup guard and its filter scope, the BCP47 normalisation, the `both`-direction split, and the read-then-create race comment) | All 7 corrected to current line numbers, each flagged inline as a rev-14 fix so a future rebase sweep doesn't miss them again | §5.1.1 step 6, §5.2.1, §5.2.2, §5.2.3, §5.11, §6 (×2) |
| MEDIUM-2 | §5.1.1 step 7's bounded retry (LOW-6, round 11) means the *same* AIcall can have two concurrent `ensureListen` goroutines resolve to the *same* transcribe id (guaranteed by §5.2.2's dedup guard) and race to write `listen_owns_transcribe` on the same row via `UpdateListenState` — a naive last-write-wins could persist `owns=false` for the AIcall that actually started the session, making §5.7.2's stop path skip a session it should stop. Bounded (call-hangup transport closure is a hard backstop) but avoidable at negligible cost | `UpdateListenState` now specified to OR a `true` into `owns`, never overwrite one with `false`, rather than blindly writing the caller's value. Deduping the two `ensureListen` calls outright (a `SET NX` guard, mirroring §5.3.4) was considered and explicitly rejected as unnecessary — §5.2.2's guard already prevents the worse outcome (two live sessions), so this is a narrow single-column fix, not a new concurrency layer | §5.2.4 |
| LOW-1 | §2 Goal 1's own attribution said its wording was "corrected in rev 12," but the wording change itself landed in rev 13 (round 11's MEDIUM-1 fix) — rev 12 only fixed the underlying *mechanism* (step 7's fast-fail removal) the wording describes | Corrected to name rev 13 for the wording, rev 11 for the mechanism | §2 |
| LOW-2 | §5.12's `aicall_listen_ensure_goroutine_timeout_seconds` row said "new in rev 11, added after review round 10 finding MEDIUM-B" — self-contradictory, since round 10 (which found MEDIUM-B) reviewed rev 11 and could not have motivated a change already present in rev 11; the config was actually introduced in rev 12, in direct response to MEDIUM-B | Corrected to "new in rev 12" | §5.12 |

### 10.11 Review-response matrix (round 13 → rev 16)

**Round 13 was the first review of rev 15's own substantive change** —
not a continuation of the confbridge sub-loop (rounds 9-12), which
closed with 2 consecutive approvals on rev 14 and stays closed; rev 15
reopened the design with new content, resetting the consecutive-approval
count per this document's own policy. Round 13 confirmed rev 15's
underlying motivation independently against source — the event-ordering
gap is real (`bin-transcribe-manager/pkg/transcribehandler/start.go:273-279`
starts streaming before its own DB row is created), the caller-specified-id
mechanism works exactly as documented (`transcribehandler/start.go:80-91`),
and adopting id-predeclaration does not quietly also adopt the
dynamic-RabbitMQ-binding purpose §3 already rejected — but found that
rev 15's own two changes each introduced a new defect, on top of the
endpoint being wired onto the wrong permission surface for its actual
caller.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| BLOCKING-1 | The new endpoint was designed at the top-level `POST /v1/aicalls/{id}/listen`, mirroring `terminate` — but `terminate` (`servicehandler/aicall.go:258`) is gated on `PermissionCustomerAdmin`/`PermissionCustomerManager`, an Admin-console tier, while the panel's own existing `Start` call (`ServiceAgentAIcallCreate`) is on the Agent-facing `/service_agents/*` surface, gated only on `PermissionAll`. `bin-api-manager/docs/auth.md:119,124` states in the imperative that Agent frontends must never call the top-level path, and that relaxing the top-level bitmask is the wrong fix for a missing Agent-facing capability. As designed, an ordinary square-talk agent would have gotten `ErrPermissionDenied` on every call — listening would never start in the feature's actual primary use case | Moved to `POST /service_agents/aicalls/{id}/listen`, mirroring the existing `POST /service_agents/contact_addresses/:id/claim` precedent (`gen.go:23354`) instead of `terminate`. New `ServiceAgentAIcallListen` in `pkg/servicehandler/serviceagent_aicall.go`, following `ServiceAgentTranscribeList`'s auth shape (`IsAgent()` + `hasPermission(..., PermissionAll)`) plus an explicit ownership recheck. The internal ai-manager RPC route stays at the plain `/v1/aicalls/{id}/listen` path — only the public api-manager path moved | §5.1, §5.8, §5.10.1a, §7 item 2, §9 |
| HIGH-1 | `EnsureListenPrecheck(ctx, c) (proceed bool, err error)` and `ensureListenAsync(c)` were connected only by a bare `bool` — `kase`, `callID`, and `call`, all resolved by steps 4-6, were discarded at the function boundary, forcing `ensureListenAsync` to silently re-fetch the Case and the call before it could even start step 7 (a duplicate RPC pair and a re-derivation of what §5.1.1 step 4 calls "the tenant boundary for the whole feature"), contradicting the claim that `ensureListenAsync` was "unchanged in shape" from rev 1-14 | Collapsed into one exported `ProcessListen`, whose goroutine (`runListenStart`) closes directly over `checkListenEligible`'s already-resolved `a`/`c`/`kase`/`callID`/`call` — nothing crosses a function boundary by itself, so there is nothing to re-fetch and nothing to silently lose | §5.1, §5.1.1, §7 item 1 |
| HIGH-2 | `ensureListenAsync` was specified lowercase — unexported, and therefore unreachable from `pkg/listenhandler` and unmockable on the `AIcallHandler` interface (every method on which Go requires to be exported for cross-package interface satisfaction) — making §7's own test items asserting its invocation count unimplementable as written | Resolved by the same `ProcessListen` collapse as HIGH-1: one exported method, directly mockable | §5.1, §7 item 2 |
| HIGH-3 | Rev 15's ordering fix pre-registered only the Redis `SADD`, leaving §5.2.4's DB write (`listen_call_id`/`listen_transcribe_id` metadata) exactly where it always was — *after* `TranscribeV1TranscribeStart` succeeded. That reopened a different, worse race: in the interval between the pre-registered `SADD` and the DB write landing, a real `transcript_created` event could already resolve through the now-registered resolver set and reach `RunListenTurn`, whose precondition (§5.4.1 step 1) requires `listen_transcribe_id` to be **set** — which it is not yet — triggering `stopListening`/`clearListenState` and deleting the very membership and buffers rev 15's own fix had just created, killing the session for the rest of the call on what should have been a completely ordinary start | §5.2.4's DB write moved earlier too — both the DB write and the Redis `SADD` now happen together, speculatively, against a pre-generated id, *before* `TranscribeV1TranscribeStart` is called, with an explicit `rollbackListenState` undo (both the DB fields and the Redis membership) on any failure | §5.2.2, §5.2.4, §6, §7 item 2 |
| MEDIUM-1 | `checkListenEligible`'s own worst-case latency (~9s across three sequential, non-cache-first RPCs — `ContactV1CaseGet`, `CallV1CallGet`, `TranscribeV1TranscribeGet`) exceeds `requestTimeoutDefault` (3000ms), the timeout `AIV1AIcallListen` would otherwise inherit — under a degraded contact-manager/call-manager, the caller's own request could time out even though ai-manager's precheck later succeeds | `AIV1AIcallListen` given an explicit 10s timeout, the same per-call-override pattern `TranscribeV1TranscribeStart` already uses for its own 5s | §5.1 |
| MEDIUM-2 | Rev 15's public, arbitrarily-callable endpoint removed the implicit "just created/reused, therefore live" guarantee `Start`'s old hook relied on — a terminated/deleted AIcall could still pass steps 3-6, spawn step 7's 45s goroutine, and start a billed STT session, caught only much later (and less directly) by §5.4.1's own unrelated status check | Step 2 (renamed "AIcall gate") now also requires `c.Status == progressing && c.TMDelete == nil`, rejecting before any of steps 3-6 run | §5.1.1 step 2, §6, §7 item 1 |
| MEDIUM-3 | The rollback-on-any-error snippet HIGH-3's first fix used treated every `TranscribeV1TranscribeStart` error identically, silently dropping §6's already-documented `TRANSCRIBE_ALREADY_PROGRESSING` reuse-on-conflict behaviour (the read-then-create race `startLive`'s own comments acknowledge) | Restored as an explicit discriminated `switch` branch: on `TRANSCRIBE_ALREADY_PROGRESSING`, re-run the list once and rewrite this AIcall's state to the winner instead of rolling back and giving up | §5.2.2, §6, §7 item 2 |
| MEDIUM-4 | The new route's handler orchestrated three separate business-handler calls plus a conditional goroutine spawn directly in `pkg/listenhandler`, unlike every sibling route (`terminate`, `Get`, `Delete`), each of which makes exactly one business-handler call | Resolved by the same `ProcessListen` collapse as HIGH-1/HIGH-2 — the handler now makes one call, matching `processV1AIcallsIDTerminatePost`'s shape exactly | §5.1 |
| LOW-1 | Several current-state (non-historical) mentions of the now-superseded unified `ensureListen`/`ensureListenAsync` names survived rev 15's own sweep, in §5.1.1's step-7 prose, §5.2.2/§5.2.4, §6, §7, §9, and §5.12/§5.13 | Swept to the rev-16 names (`ProcessListen`/`checkListenEligible`/`runListenStart`) everywhere current-state prose referenced them; historical §0 rows and §10.x matrices intentionally left as an accurate record of what each revision said at the time, per this document's established convention | §5.1.1, §5.2, §6, §7, §9, §5.12, §5.13 |
| LOW-2 | Three citations drifted: `ai_aicalls.go:113-131` (actually `:110-127`, `:131` falls inside a different function), `servicehandler/aicall.go:250-270` (actually `:249-277`, the cited range stopped mid-function), `transcribehandler/start.go:158-161` (actually `:160-163`) | All three corrected | §5.1, §5.1.1 |
| LOW-3 | §6's new error rows describe rev 15/16's new failure branches as folding into existing `skipped_*`/`failed` metric buckets, but no explicit label was actually added to §5.13's enumerated set the way every other outcome is listed | Recorded as an explicit open item rather than left implicit — a label name (or an explicit decision to fold silently) is needed before implementation | §11 item 16 (new) |

### 10.12 Review-response matrix (round 14 → rev 17)

**Round 14 independently re-verified every one of round 13's ten
findings against current source, not §10.11's own description** — and
confirmed all ten genuinely and honestly fixed, BLOCKING-1's permission
surface in particular checked line-for-line against
`bin-api-manager/docs/auth.md`, the actual `terminate`/`ServiceAgentTranscribeList`
gates, and the actual `gen.go` route registrations. It also traced the
full `TRANSCRIBE_ALREADY_PROGRESSING` round trip end to end and confirmed
the discrimination MEDIUM-3 (round 13) required is genuinely achievable
— just not via the helper rev 16's snippet invented. What round 14 found
instead were two new defects, both arising from the same unexamined
premise: that concurrent `runListenStart` goroutines for one AIcall
(already documented, §5.1.1's own LOW-6 note) still resolve to the same
transcribe id, which stopped being true the moment rev 16 moved the write
before creation.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| HIGH-1 | `UpdateListenState`'s rev-14 `owns`-merge rule ("OR a `true` in, never overwrite with `false`") was unconditional on transcribe id. On the create-then-fall-back-to-reuse branch (§5.2.2's `switch`), the row is written first against a speculative id (`owns=true`), then — on `TRANSCRIBE_ALREADY_PROGRESSING` — against a *different*, winning id (`owns=false`); the unconditional merge would carry the stale `true` forward onto the winning id's row, making this AIcall believe it owns a session it does not, which §5.7.2's stop path would then incorrectly stop | The OR-merge is scoped to same-transcribe-id writes only; a write against a differing id sets `owns` directly from the caller's value, no carry-over | §5.2.4, §7 item 2 |
| HIGH-2 | Two concurrent `runListenStart` goroutines for the *same* AIcall (an expected consequence of §5.1.1 step 7's own retry, per the LOW-6 note) can now each pass §5.2.2's `List` check and mint their own speculative transcribe id before either finishes writing — one goroutine's pre-write can `SREM` the other's already-live session out of the resolver set (rev-4's stale-id logic, misapplied to a session that was never stale), or a later rollback from one goroutine can delete DB/Redis state belonging to the other's live, billed session | Wrapped the whole reuse-check-through-write sequence in a new per-AIcall `SET NX EX` lock (`ai:listen:startlock:<aicall_id>`, new config `aicall_listen_start_lock_ttl_seconds`) — reversing the earlier "considered and rejected" decision, whose stated reasoning covered only cross-AIcall duplication (already prevented by §5.2.2's `List`-based dedup guard), not this same-AIcall race rev 16's write-ordering change newly exposed | §5.2.2, §5.12, §7 item 2 |
| MEDIUM-1 | `cerrors.IsReason(err, "...")`, used in rev 16's `switch` to discriminate `TRANSCRIBE_ALREADY_PROGRESSING`, does not exist anywhere in this codebase | Replaced with the actual established pattern (`errors.As(err, &ve) && ve.Reason == "..."` against `*cerrors.VoipbinError`), verified end-to-end: transcribe-manager's actual error construction, the wire round trip, and ai-manager's actual error-recovery call sites all confirmed to use this shape | §5.2.2 |
| MEDIUM-2 | BLOCKING-1's (round 13) endpoint-path fix swept most of the document but missed four current-state mentions of the old top-level `/v1/aicalls/{id}/listen` public path: §7 item 20 (frontend test spec), §8 step 4 (rollout), §5.10.2 ×2 (mandated RST deliverable, including instructing it to follow `terminate`'s pattern rather than `claim`'s) | All four corrected to `/service_agents/aicalls/{id}/listen` | §7 item 20, §8, §5.10.2 |
| MEDIUM-3 | The 10s RPC timeout's stated justification — "none of the three RPCs are cache-first" — was itself factually wrong for `CallV1CallGet`, which the actual `bin-call-manager/pkg/dbhandler/call.go:115-130` shows *is* cache-first; the cited `callhandler/db.go:171-185` only shows delegation to the (cache-first) `dbhandler`, not a bypass of it | Restated on the reasoning that actually holds regardless of caching: three sequential cross-service RPC hops, each independently subject to its own default timeout, can sum to roughly 3× a single hop's timeout worst-case | §5.1 |
| LOW-1 | §5.1's `checkListenEligible` snippet (`ctx, c`) and §5.1.1's prose (claiming `ProcessListen` resolves `a` first, then calls `checkListenEligible(ctx, c, a)`) disagreed on the function's own signature | `checkListenEligible` resolves `a` itself internally (matching the snippet); §5.1.1's prose corrected to match | §5.1.1 |
| LOW-2 | Two citation/convention slips introduced by round 13's own BLOCKING-1 fix: `serviceagent_transcribe.go:41-44` cited for the auth-check shape is actually the unrelated `if token == ""` block (the real lines are `:27-29`/`:45-48`); the new ownership check called `AIV1AIcallGet` directly where the sibling `ServiceAgentAIcallGet` uses the private two-level `aicallGet` helper `bin-api-manager/CLAUDE.md`'s two-level handler pattern expects reused | Both corrected | §5.1 |
| LOW-3 | A handful of current-state (non-historical) mentions of the superseded `ensureListen` name survived rev 16's own sweep: §8's flag-gating table (a different row from the one round 13 already fixed) and §11 item 13's prose | Corrected to `checkListenEligible`/`ProcessListen` | §8, §11 item 13 |
| LOW-4 | The conflict-recovery branch's redundant `ListenTranscribeAIcallRemove` call (immediately after an `UpdateListenState` that had already `SREM`'d the same membership) was harmless but unexplained — direct evidence the two mechanisms hadn't been reconciled, which HIGH-1/HIGH-2 then found the substance of | Left in place (still correct — an idempotent duplicate) but now explained inline: it removes only the AIcall's own never-created speculative id, never the winner's membership, which `UpdateListenState` already registered correctly | §5.2.2 |
| LOW-5 | Three small cross-reference errors: §5.2.4 claimed §5.4.1's precondition "would also refuse to act on a mismatched `listen_transcribe_id`" (false — §5.4.1 step 1 only checks the field is set, never compares it); §9's cachehandler scope note pointed at "§11 item 7" (actually item 9, renumbered as the document grew); §5.8 and §9 disagreed on which file holds `rollbackListenState` | All three corrected | §5.2.4, §9, §5.8 |

### 10.13 Review-response matrix (round 15 → rev 18)

**Round 15 confirmed rev 17's diagnosis of round 14's HIGH-1/HIGH-2 and
its own `owns`-merge rule were correct** — the reviewer stated explicitly
they "could not construct a case where the scoped rule produces a wrong
`owns` value" — and independently re-verified round 14's other four
findings against current source rather than trusting §10.12's own
description, confirming all genuinely fixed (including re-tracing the
`TRANSCRIBE_ALREADY_PROGRESSING` error round-trip end to end a second
time). What it found instead was that rev 17's own new lock — the fix
for round 14's HIGH-2 — was itself under-specified, in a way that could
reopen the very race it was built to close.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| HIGH-1 | Two compounding defects in the new lock: (a) its TTL (`15`s, §5.12) was sized against only `TranscribeV1TranscribeStart`'s explicit 5000ms timeout, but the lock also wraps up to two `TranscribeV1TranscribeList` calls, whose RPC client hardcodes an uncontrollable **30000ms** timeout — nominally summing to as much as 65s in-lock, comfortably exceeding a 15s TTL; (b) the lock's release (`h.cache.Del`, unconditional) and acquisition (a constant value, not a per-goroutine token) meant one goroutine's release could delete a *different*, still-legitimately-working goroutine's lock once the first's TTL had lapsed — silently reopening round 14's HIGH-2 rather than closing it | Re-derived the TTL from the goroutine's own outer `AIcallListenEnsureGoroutineTimeoutSeconds` (default unchanged at `45`; lock TTL default raised from `15` to `60`, strictly above it) rather than from the RPC clients' own internal timeouts — no call inside the lock can outlive the `ctx` it runs under regardless of its own timeout constant, so a TTL exceeding the outer goroutine timeout can never expire under genuinely ongoing work; gave the lock a per-goroutine ownership token, released only via an atomic Redis `EVAL` compare-and-delete (`ListenStartLockRelease`, new cache primitive) rather than a bare, ownerless `Del` | §5.2.2, §5.12 |
| MEDIUM-1 | §5.2.4's own description of the bug round 14's HIGH-1 fixed stated the consequence backwards: a stale carried-forward `owns=true` makes §5.7.2's stop path **incorrectly stop** a session this AIcall doesn't own (`!owns` evaluates `false`, so the guard clause is skipped) — not, as an earlier draft said, "correctly skip stopping it." §10.11's own matrix already had this right, so the two sections contradicted each other | §5.2.4's prose corrected to match §10.11's (and the actual mechanics) | §5.2.4 |
| MEDIUM-2 | The two source citations offered as evidence for round 14 MEDIUM-1's fix (`disabled.go:24-28`, `bin-direct-manager/.../main.go:104-112`) don't contain the `errors.As`+`Reason` pattern they were cited for — neither file has an `errors.As` call performing a `Reason` comparison | Repointed to citations that actually show the pattern: `transcribehandler/stop.go:196-205`, and `bin-storage-manager/pkg/filehandler/signing.go:79` for the exact one-line-wrapper shape used here | §5.2.2 |
| MEDIUM-3 | §5.2.4 overclaimed the lock's scope — "within one AIcall only one write sequence is ever in flight" is true only for the create-or-reuse sequence the lock actually wraps, not for teardown paths (`clearListenState`/§5.7.3, `stopListenByCallID`/§5.7.1), which don't take this lock and can still interleave with it | Restated narrower, with the existing bounded-harm reasoning (§6, `isValidReference`) named as why this stays an accepted precision gap rather than a new correctness issue | §5.2.4 |
| LOW-1 | The lock's own two outcomes (a losing goroutine on contention; a `SetNX`/release Redis error) had no §6 error-table row and no §5.13 metric label | Two new §6 rows added; folded into §11 item 16's existing open-item tracking, with `skipped_start_locked` flagged as worth its own label specifically (a useful concurrency-pressure signal, unlike the other fold-candidates) | §6, §11 item 16 |
| LOW-2 | §5.1.1's own LOW-6 note still described the same-AIcall concurrent-goroutine race as "already covered by §5.2.2's reuse rule and §6's `TRANSCRIBE_ALREADY_PROGRESSING` row alone" — the exact unexamined premise round 14's HIGH-2 found insufficient | Repointed at §5.2.2's lock, with the correction noted explicitly | §5.1.1 step 7 |
| LOW-3 | §9 said "seven flags" against §5.12's actual twelve, and its `cachehandler`/Redis-primitive enumeration omitted the lock's two new primitives (`SetNX` reuse, `ListenStartLockRelease`) | Both corrected | §9 |
| LOW-4 | A genuine Go-level bug in rev 17's own snippet: the reuse-path `TranscribeV1TranscribeList` call's error return was silently dropped (`existing := ...` instead of `existing, err := ...`), so an RPC failure would have read as "no existing session found" and started a duplicate — inconsistent with the same snippet's other two `List`/write calls, which already handled their errors correctly | Fixed to check and propagate the error, matching the pattern used everywhere else in the same snippet | §5.2.2 |
| LOW-5 | §5.1 said the two-level `aicallGet` helper performs the ownership `CustomerID` compare itself — imprecise; the helper only fetches, and the compare is the caller's own, matching the sibling `ServiceAgentAIcallGet`'s actual division of labour | Restated precisely | §5.1 |
| LOW-6 | Four spots this document's own MEDIUM-2 (round 14) sweep touched were attributed to "path corrected in rev 16," when the text at each of those specific spots actually changed in rev 17 (rev 16 only moved the route itself, in §5.1). **This row's own "Where" column was itself wrong — corrected in rev 19, review round 16 finding LOW-1**: it named §5.10.1a, which was never touched by this sweep and correctly still reads "corrected in rev 16" (§5.10.1a describes the route itself, not one of the four re-mentions); the actual fifth site the rev-17 sweep touched, §9, was omitted | Corrected to name the revision that actually touched each spot | §5.10.2 (×2), §7 item 20, §8 step 4, §9 |
| LOW-7 | Neither the rev-4 SREM-from-old-id rule nor the new scoped `owns`-merge rule stated which read of "the row's current value" they meant — the calling goroutine's own (potentially stale) in-hand `c`, or a fresh DB read inside `UpdateListenState` itself | Clarified as a fresh read; confirmed the ambiguity was harmless for `owns` specifically (only the SREM half was ever exposed to it), given the lock already serializes this AIcall's own writers | §5.2.4 |

---

### 10.14 Review-response matrix (round 16 → rev 19)

**Round 16 independently re-verified round 15's own findings against actual current source rather than trusting §10.13's description** — confirming the TTL-derivation reasoning, the `TranscribeV1TranscribeList` 30000ms hardcoded-timeout citation, the corrected `errors.As`+`Reason` citations, and the corrected owns-merge consequence description were all genuinely right. What it found was that rev 18's own fixes were incompletely or, in one case, incorrectly applied — the TTL default was raised in the prose that derives it but not in the config table that governs it, and the release fix (a genuine improvement over rev 17) introduced a context-lifetime defect of its own that only became reachable once the TTL was sized correctly.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| HIGH-1 | §5.12's `aicall_listen_start_lock_ttl_seconds` row and §11 item 13 still said `15` (proposed) with the withdrawn "sum the RPC timeouts, stay well inside the goroutine timeout" rationale — directly contradicting §5.2.2's rev-18 rule that the TTL must instead *exceed* the goroutine timeout, default `60` | Both updated to `60` with §5.2.2's actual derivation | §5.12, §11 item 13 |
| MEDIUM-1 | §5.2.2's prose said the lock's ownership token was `newTranscribeID`, but the code block four lines later mints a separate `lockToken := h.utilHandler.UUIDCreate()` — and `newTranscribeID` isn't minted until after lock acquisition, nor at all on the reuse path, making the prose's claim unimplementable, not just imprecise | Prose corrected to name `lockToken` and explain why it must be independent of `newTranscribeID` | §5.2.2 |
| MEDIUM-2 | The deferred release ran under the acquiring goroutine's own `ctx`; a goroutine reaching its own outer timeout cancels `ctx`, so the release call would fail immediately — stranding the lock in exactly the legitimate-work scenario the TTL-vs-timeout margin exists to keep working, and invalidating the withdrawn "margin for the deferred release to run" claim from rev 18 | Release now runs under a context detached from `ctx`'s own cancellation (`context.WithoutCancel`, precedented at `bin-schedule-manager/pkg/dispatchhandler/manual.go:102`), bounded by its own new, short timeout config | §5.2.2, §5.12 |
| MEDIUM-3 | Raising the TTL strictly above the goroutine timeout means a genuinely crashed goroutine (pod loss, the release `defer` never runs) now strands the lock for the full TTL rather than for less than the outer timeout as rev 17's original default implied; §7 item 2's own "a later goroutine can acquire it and proceed normally" test claim went stale against this without being updated | §7 item 2 corrected to the TTL-elapsed condition it now actually requires; the accepted trade-off (slower crash recovery in exchange for the TTL never lapsing under legitimate work) stated explicitly rather than left implicit | §7 item 2, §5.12, §5.2.2 |
| MEDIUM-4 | The lock's compare-and-delete release semantics (a stale-token release is a deliberate no-op, not a delete) and its Redis-error path had no regression test of their own | Added directly to §7 item 2, alongside a new test that the release call itself runs on a detached, non-cancelled context | §7 item 2 |
| LOW-1 | §10.13's own LOW-6 "Where" column named §5.10.1a as one of the four spots corrected in rev 17, but §5.10.1a was never touched by that sweep (it correctly still reads "corrected in rev 16") and the actual fifth site touched, §9, was omitted | Corrected | §10.13 |
| LOW-2 | §5.2.4's LOW-7 fix cited its own supporting claim with a bare, brittle doc-internal line reference (`` `:1013-1014` above ``), which every subsequent revision's edits shift out from under | Replaced with a named-subsection reference | §5.2.4 |
| LOW-3 | §6's `skipped_start_locked` mention read as though the label were already added to §5.13's enumerated set, when §11 item 16 states plainly the label is still an open, undecided item | Reworded to state it as this design's proposed label, not a settled one | §6 |
| LOW-4 | §0's rev-18 row's "default raised to `60`" aside reads, on a literal parse, as attached to `AIcallListenEnsureGoroutineTimeoutSeconds` rather than the lock TTL it actually describes | Left as the historical record, per this document's own convention of not rewriting past revision rows — the row is ambiguously worded, not factually wrong, and the ambiguity is noted here instead | §10.14 (this row) |
| LOW-5 | §5.8's file-list bullet for `listen_trigger.go` omitted the lock's two cache primitives, which §9's parallel enumeration already listed — the two sections disagreed again, the same class of drift LOW-5 (round 14) had already corrected once | `ListenStartLockAcquire`/`ListenStartLockRelease` added to §5.8's bullet | §5.8 |
| LOW-6 | The lock's acquire/release API was asymmetric: acquisition was a raw, inline `SetNX` call with the key built at the call site, while release was a named function that rebuilt the same key format internally — the format lived in two places and could drift | Given a matched `ListenStartLockAcquire`/`ListenStartLockRelease` pair; the key format now lives in exactly one place | §5.2.2, §9, §5.8 |
| LOW-7 | The illustrative `dupFilters` pseudo-block earlier in §5.2.2 showed `existing := TranscribeV1TranscribeList(...)` with no error check — the exact dropped-error shape round 15's LOW-4 had just fixed a few paragraphs later in the real snippet, left as a stale, confusing echo | Rewritten as the actual `dupFilters` binding, with its own error check, referenced by name from the lock sequence below | §5.2.2 |
| LOW-8 | `dupFilters` was referenced by name in the lock's Go snippet but never bound anywhere in it | Bound once in the illustrative block above (LOW-7's fix) and referenced by name | §5.2.2 |
| LOW-9 | The deferred release call discarded its error return without the explicit `_ =` this snippet uses everywhere else it intentionally ignores an error — inconsistent, and the kind of thing an `errcheck` gate would flag | Naturally fixed by MEDIUM-2's rewrite, which wraps the release in a closure using `_ =` | §5.2.2 |
| LOW-10 | The conflict-recovery branch's `existing, errList := ...` shadowed the create path's own `existing`/`errList` names a few lines above, in a section whose most recent bug (round 15 LOW-4) was about that exact pair's error handling | Renamed to `existingRetry`/`errListRetry` | §5.2.2 |
| LOW-11 | Pre-existing, not introduced in rev 18: §5.1's response-timing prose described step 7's confbridge wait as "up-to-45s," conflating the confbridge-wait budget (`aicall_listen_confbridge_ready_max_wait_seconds`, `30`s) with the goroutine's own separate outer timeout (`45`s) | Corrected to name the right config and its actual value | §5.1 |

---

### 10.15 Review-response matrix (round 17 → rev 20)

**Round 17 independently re-verified all six of round 16's findings against current text and, where load-bearing, against actual source** — including confirming `context.WithoutCancel` really is used at the cited precedent (`bin-schedule-manager/pkg/dispatchhandler/manual.go:102`) and that this monorepo's Go version (`go 1.27.1`) supports it — and confirmed every one genuinely fixed, with no reservations. What it found instead was that rev 19's own edits, while correctly fixing round 16's specific findings, left several of the sections they touched inconsistent with each other or with immediately adjacent text the edits didn't reach.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| B-1 | The Status line and §0 row 19 both undercounted round 16 as "3 MEDIUM" against §10.14's own 4-row (MEDIUM-1 through MEDIUM-4) enumeration | Fixed on the Status line, which is continuously-updated current-state summary; row 19's own text left alone per this document's convention (a factual miscount on a still-current summary line is corrected; a frozen historical row is not rewritten — the two are treated differently on purpose, not inconsistently) | Status line |
| B-2 | §10.14's own LOW-7 row claimed the illustrative `dupFilters` block had been rewritten "with its own error check" — it had not; it still declared `existing, err := TranscribeV1TranscribeList(...)` with no `if err != nil`, the same dropped-error shape under a different mask | The call removed from the illustrative block entirely; it now only binds the filter map, referenced by name from both real call sites, each of which already has its own correct error handling | §5.2.2 |
| B-3 | Rev 19's own LOW-10 fix (renaming shadowed variables) introduced a brand-new brittle doc-internal line citation ("line-990") in its explanatory comment, already off by one line | Replaced with a description naming the create-path call directly, no line number | §5.2.2 |
| B-4 | §5.8's file-list bullet, rewritten in rev 19, ended up claiming `rollbackListenState` — an `aicallHandler`-level helper, not a cache primitive — lived in `pkg/cachehandler` alongside the lock's two primitives, contradicting §9's placement of it in `listen_trigger.go`; separately, §9's own `cachehandler` enumeration still omitted `ListenTranscribeAIcallRemove` | §5.8 split into two bullets matching §9's placement exactly: `rollbackListenState` stays in `listen_trigger.go`; `ListenTranscribeAIcallRemove`/`ListenStartLockAcquire`/`ListenStartLockRelease` are the cache primitives, now listed in both §5.8 and §9 | §5.8, §9 |
| B-5 | §9 said "twelve flags" against §5.12's actual thirteen, the moment rev 19 added `aicall_listen_start_lock_release_timeout_seconds` without updating this count | Corrected to "thirteen" | §9 |
| B-6 | §6's lock-error row and §7 item 2's matching test both described `ListenStartLockAcquire`/`ListenStartLockRelease` errors identically — but a deferred `Release` error can't yield "no transcribe is started" (the `Release` runs after `TranscribeV1TranscribeStart` may have already succeeded) or "metered `failed`" (its error is deliberately discarded via `_ =`); the test text also mislabeled the lock as reachable from `checkListenEligible`, which never touches it | Split into an acquire-error row/test (fail-closed, metered `failed`, zero `TranscribeV1TranscribeStart` calls, attributed to `runListenStart` not `checkListenEligible`) and a release-error row/test (best-effort, never metered, falls back to the lock's own TTL) | §6, §7 item 2 |
| B-7 | An acquire call whose `SET NX` lands server-side but errors client-side (timeout, connection reset) registers no `defer`, so nothing would ever release that lock — a stranding path the "reserved for an actual crash" residual-risk claim didn't name | A best-effort release with the same token is now attempted on the acquire-error path itself before returning the error, collapsing this case into the already-accepted crash residual rather than leaving it as a separate, silent one | §5.2.2 |
| LOW-1 | `tr`, captured from `TranscribeV1TranscribeStart`'s return, was never used anywhere except inside a comment — real Go would not compile this | Replaced with `_` | §5.2.2 |
| LOW-2 | §7 item 2 asserted `skipped_start_locked` as a settled label in two places, after §6 had already been reworded (rev 19, round 16 LOW-3) to call it "proposed, not yet added to §5.13" | Both §7 mentions reworded to match §6 | §7 item 2 |
| LOW-3 | Rev 19 reworded §9's debounce-lock cross-reference ("already established in §5.3.4") but left §5.2.2's parallel mention ("already uses") in the older phrasing | Made consistent | §5.2.2 |

**Nitpicks recorded, not separately fixed** (round 17's own assessment: none block approval; **the two doc-internal line citations below were themselves found stale by review round 19 finding LOW-4 and are given as descriptive references instead, rather than re-pinned to new line numbers that will just as surely drift again**): a residual "already uses"-style phrasing elsewhere, scoped unambiguously by an explicit §5.3.4 reference; §5.1's response-timing prose naming "step 7's 45s goroutine" attributes the goroutine's own outer timeout to step 7 by name, a narrower imprecision than LOW-11's already-fixed budget/timeout conflation; §5.1.1's mention of the internal `POST /v1/aicalls/{id}/listen` route is accurate per §5.1's own statement that the internal path never moved, just reads adjacent to BLOCKING-1's public-path history; and round 17's own explicit confirmation that this document's row-freezing convention (§0's historical rows left alone; only continuously-updated summary text like the Status line corrected) was applied consistently, not selectively, in resolving B-1.

---

### 10.16 Review-response matrix (round 18 → rev 21)

**Round 18 independently re-derived all ten of round 17's findings from current text rather than trusting §10.15, and re-verified every load-bearing source citation a second time** (`context.WithoutCancel`'s precedent and this monorepo's Go version, `TranscribeV1TranscribeList`'s hardcoded timeout, `TranscribeV1TranscribeStart`'s signature) — all still accurate — and further confirmed B-7's new best-effort-release code is valid, non-leaking Go. Nine of round 17's ten findings were confirmed genuinely and completely fixed, no reservations. What it found was one real compile-breaking defect inside the very block rev 20 rewrote to fix a *different* compile-breaking defect (B-2), plus a cluster of small attribution/wording residue from the same revision's other fixes.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| MEDIUM-1 | §5.2.2's own code comment, added by rev 20 to explain a discarded return value, cited "round 17 finding B-9" — an id that does not exist in this document's numbering (round 17's range is B-1 through B-7); the finding that actually produced this edit was LOW-1 | Corrected to cite LOW-1 | §5.2.2 |
| MEDIUM-2 | The `dupFilters` illustrative block, rewritten in rev 20 to fix B-2's dropped-error shape, introduced a different compile error: it declared `map[string]any` with bare string keys, but `TranscribeV1TranscribeList`'s actual parameter type is `map[tmtranscribe.Field]any` (`bin-common-handler/pkg/requesthandler/transcribe_transcribes.go:40`) — a distinct named type Go does not implicitly convert between | Rekeyed with the actual `tmtranscribe.Field` constants (`FieldCustomerID`, `FieldReferenceID`, `FieldStatus`, `FieldDeleted`), verified present at `bin-transcribe-manager/models/transcribe/field.go` | §5.2.2 |
| LOW-1 | §7 item 2's new deferred-release test case cited "round 17 finding B-6/MEDIUM-4," reading as though MEDIUM-4 were also round 17's when it is round 16's | Reworded to "B-6, extending round 16's MEDIUM-4 coverage" | §7 item 2 |
| LOW-2 | The §5.2.2 "accepted residual" paragraph and §5.12's lock-TTL row both named crash-only (pod loss) stranding as the sole cause of a full-TTL strand, without naming B-7's own new residual (an ambiguous acquire error whose best-effort release also fails) | Both now name the second residual explicitly | §5.2.2, §5.12 |
| LOW-3 | §7 item 2's happy-path assertion still referenced `tr.ID`, a variable LOW-1's own rev-20 rename removed from §5.2.2's snippet | Reworded to describe the mocked return value without naming a variable | §7 item 2 |
| LOW-4 | §5.2.4's `UpdateListenState` description also referenced `tr.ID`, but `tr` was never in scope in that section at all — its own parameter is `transcribeID` | Corrected to `transcribeID` in both mentions | §5.2.4 |
| LOW-5 | The Redis resolver set's `12h` membership TTL was the only listen timing constant in this design with no §5.12 config row and no stated reason why not | A sentence added stating this is deliberate — a safety-margin bound, not a value expected to need tuning — so it is not promoted to a fourteenth flag | §5.2.4 |
| LOW-6 | **Mis-cited by rev 21 itself, review round 19 finding LOW-3: the quoted framing ("factually-wrong text (corrected) … merely historical text (not rewritten)") is not in §10.15's B-1 row at all — §10.15's B-1 row uses only the operative-rule phrasing. The quote is verbatim in §0's own row 20.** So it is row 20, not §10.15's B-1 row, whose framing argues (as literally phrased) for correcting row 19's miscount rather than leaving it, against the operative rule that row 20's own parenthetical states correctly a few words earlier in the same sentence ("continuously-updated current-state summary, not a frozen historical row") (**descriptor corrected in rev 23, review round 20 finding LOW-2, from "its own trailing clause" — accurate for §10.15's B-1 row, whose operative-rule statement really is trailing, but not for row 20, whose statement comes earlier in the sentence**) | Row 20 left as the historical record of round 17's own review, per this document's convention, rather than rewritten — the mis-citation corrected here instead, since this matrix (unlike §0's own rows) is not itself frozen | §10.16 (this row) |

---

### 10.17 Review-response matrix (round 19 → rev 22)

**Round 19 independently re-derived all eight of round 18's findings from current text and re-verified every load-bearing citation a second time, plus performed a full compile-read of §5.2.2's entire lock snippet** (not just the lines MEDIUM-2 named) — every variable, every RPC call's argument count and types, checked against the actual signatures in `bin-common-handler`/`bin-transcribe-manager`, and against the `Field` constants used in the corrected `dupFilters` map. All eight were confirmed genuinely and completely fixed. **This is the sub-loop's first APPROVE.** One more consecutive APPROVE (round 20) closes it.

| # | Review finding | Resolution | Where |
|---|---|---|---|
| LOW-1 | The "accepted residual" paragraph, extended in rev 21 to name a second stranding cause, pointed to it as "(above)" — the actual text is ~40 lines below, not above | Corrected to "(below)" | §5.2.2 |
| LOW-2 | A third `tr.ID` reference survived in §5.2.4's historical description of rev 2's original key format, beyond the two rev-21 LOW-4 fixed | Corrected to `transcribeID` | §5.2.4 |
| LOW-3 | §10.16's own LOW-6 row attributed a quoted framing phrase to "§10.15's own B-1 row," but that exact phrase is not there — it is verbatim in §0's row 20 | Attribution corrected in §10.16's row; §0 row 20 itself left unrewritten (frozen, per convention) | §10.16 |
| LOW-4 | §10.15's own nitpick paragraph carried two doc-internal line citations already stale by the time round 19 checked them | Replaced with descriptive references — the same fix this recurring anti-pattern already received in round 16 (LOW-2) and round 17 (B-3) | §10.15 |
| LOW-5 | Two mentions of "the `List` bullet/call … above" in §5.2.2 went stale the moment rev 20's B-2 fix removed the illustrative block's own `List` call | Reworded to reference the reuse-check bullet and the `dupFilters` binding directly | §5.2.2 |

---

## 11. Open items before implementation sign-off

1. **`in`/`out` speaker mapping empirical verification (§5.9) — blocking,
   narrowed in rev 11.** The general channel-relative mechanism is now
   confirmed against real production transcript data, and which leg gets
   transcribed is now backed by a stronger, code-checked invariant
   (`in == Case.Peer`, guaranteed CRM-eligible by `case_create` itself —
   §5.1.1 step 7) rather than assumed. What remains open: capture one real
   or staged **agent-bridged** call and confirm the mapping against known
   speaker identity end-to-end before merge. A reversed mapping is a
   silent correctness failure, not a cosmetic one. See item 11 for the
   residual-risk vector review round 9 surfaced while narrowing this item
   (correcting rev 11's own first-draft framing of it), and item 12 for a
   separate residual risk rev 11 itself first surfaced, whose "refused
   vs. unguarded" framing review round 9 later corrected.
2. **Confirm the deployed Redis version supports `LPOP key count` (Redis
   ≥ 6.2) — blocking, elevated in rev 4 (review round 3: this is not a
   deferrable implementation nicety).** §5.4.1's atomic pop-all of the
   `pending` buffer is load-bearing for the "no line lost to a
   concurrent appender" property; if the deployed Redis predates 6.2,
   §5.4.1 needs a different primitive (e.g. `MULTI`/`EXEC` wrapping
   `LRANGE`+`LTRIM`) chosen *before* implementation starts, not
   discovered during it. A five-minute check (`redis-cli INFO server`
   against the production instance) resolves this; do it before rev 4 is
   considered final.
3. **Pre-existing orphaned `tool`-role message defect (§5.6.3) —
   narrowed in rev 4.** §5.4.5's fix (tagging listen-turn tool rows
   `Origin=listen_internal` and excluding them from every future replay)
   means this feature can no longer *create* an instance of the defect.
   The defect itself — an agent-initiated tool call's own tool-call row
   already gets filtered by `run.py:450`'s empty-content check today,
   independent of listening — is real, predates this design, and is
   worth confirming against production traffic/logs. It is a genuine
   platform bug, but no longer gated on or made worse by this feature
   shipping, so it returns to a follow-up ticket (recommend filing
   promptly regardless, given its severity if confirmed) rather than a
   rollout blocker.
4. **No Jira ticket filed.** Recommend filing a `VOIP-*` ticket for this
   feature before implementation starts, per project convention — and a
   **separate** ticket for item 3 above.
5. **Confirm whether a new system `customer_id` sentinel (§5.2.1,
   `IDAIManagerListen`) needs a real `bin-customer-manager` row or is
   usable as a bare constant.** Small, but needs an answer before
   `pkg/aicallhandler` can reference it.
6. **Follow-up ticket (separate):** `transcripthandler.dbDelete` publishes
   `EventTypeTranscriptCreated` on delete
   (`bin-transcribe-manager/pkg/transcripthandler/db.go:33`). This design
   defends against it rather than fixing it, because changing the emitted
   event type is a routing-key-visible change affecting every current
   subscriber. It should be fixed on its own ticket.
7. **Follow-up ticket (separate):** decouple `Send`'s cooldown from
   `tm_update` onto a dedicated `tm_last_send`. Pre-existing fragility
   (`send.go:27-32` + `dbhandler/aicall.go:240`) that this design bounds
   but does not remove.
8. **Follow-up ticket (separate):** webhook noise from tool-call rows.
   §5.6.4/§5.10.1 hide the two per-tool-call noise rows from the *panel*
   (a client-side render filter), but the tenant's own webhook consumer
   still receives every `aimessage_created` delivery for them, exactly as
   today. Whether to suppress those webhooks too (and whether any tenant
   automation actually depends on receiving them) is a genuinely separate
   decision from the frontend fix shipped with this design.
9. **Implementation-time decision, not blocking:** where listen's new
   Redis operations live — extend `pkg/cachehandler` or give them their
   own small package (§9's scope note).
10. **Product decision (not blocking implementation):** whether Insight
    listening becomes a billed line item, and if so under which meter
    (§3, §5.2.1). The architecture keeps the STT cost off the customer's
    transcription bill, which makes this a clean, deliberate pricing
    choice rather than an accident.
11. **Staff-calling-in-as-a-CRM-eligible-peer inversion (§5.9) — new in
    rev 11, corrected by review round 9, not blocking this design's
    scope, but should get an explicit product decision before rollout.**
    Round 9 found rev 11's first framing of this risk (an agent-outbound
    click-to-call leg) is very likely impossible: `case_create`'s own
    `isCRMEligiblePeer` check already excludes agent/extension/SIP peers,
    so that exact scenario cannot produce a Case at all
    (`bin-flow-manager/pkg/activeflowhandler/actionhandle.go:1259-1275`).
    The real, narrower vector is an **inbound** call whose peer address is
    CRM-eligible (`tel`/`email`) but is actually staff calling in via a
    plain DID rather than the agent-dial path — `case_create` cannot
    distinguish that from a genuine customer, and `in`/`out` would be
    silently inverted for such a call. §5.1.1 step 7's participant-count
    guard does not catch it (a 2-party bridge still reads as 2 parties).
    Out of scope to fix here; needs its own follow-up ticket if this
    scenario is a realistic concern for any customer's call routing.
12. **Call-transfer interaction with listening (§5.1.1 step 7, §5.9) —
    new in rev 11, not investigated.** Whether a `bin-transfer-manager`
    transfer changes which leg is transcribed mid-listen, or produces a
    transient 3-party bridge state, is unconfirmed. §5.1.1 step 7's guard
    is enforced only at listen-start, with **no ongoing re-check** once a
    session is live — so a transfer occurring mid-session is unguarded,
    not merely refused, contrary to what an earlier draft of this item
    implied. Recommend a follow-up ticket rather than blocking this
    design on it, since transfer-during-listen is a narrower scenario
    than the base feature.
13. **New config, needs a value before implementation (§5.1.1 step 7,
    §5.12):** `aicall_listen_confbridge_ready_poll_interval_seconds`
    (proposed default `2`), `aicall_listen_confbridge_ready_max_wait_seconds`
    (proposed default `30`), and — **added after review round 10 finding
    MEDIUM-B**, since the design otherwise left this unbounded/implicit —
    `aicall_listen_ensure_goroutine_timeout_seconds` (proposed default
    `45`, strictly greater than the max-wait value above). Together these
    are the bounded retry that review round 9 required after finding the
    original one-shot participant-count check fails closed on a perfectly
    normal call (agent still ringing when `ProcessListen` first runs). All
    three defaults are this design's proposal, not yet validated against
    real hold/ring-time distributions — in particular, since review round
    10's HIGH-A fix means a stably-wrong topology and a merely-slow ring
    now share the same `skipped_confbridge_not_ready` outcome and the same
    30s budget, an unusually long real ring time would show up
    indistinguishably from a genuine topology problem, which is a
    reason to err toward a longer default here rather than a shorter one.
    Confirm before rev 11 is considered final for implementation. **Rev
    17, review round 14 finding HIGH-2 adds a fourth (default corrected
    from `15` to `60` in rev 18-19, §5.12)**:
    `aicall_listen_start_lock_ttl_seconds` (proposed default `60`,
    §5.2.2, §5.12) — the per-AIcall create-or-reuse lock's TTL, needing
    the same validate-before-final treatment as the three above. **Rev
    19 adds a fifth**: `aicall_listen_start_lock_release_timeout_seconds`
    (proposed default `3`, §5.2.2, §5.12) — the detached-context bound on
    the same lock's `Release` call.
14. **Product decision, not blocking (§5.1, new in rev 15, path
    corrected in rev 16):** the new
    `POST /service_agents/aicalls/{id}/listen` endpoint deliberately
    returns no listening-status field — the caller cannot tell "started"
    from "reused" from "not eligible" from "still waiting on confbridge"
    from the response alone. This matches the CEO/CTO's own stated scope for
    this API (separation of concerns, not status visibility — 2026-09-04
    design discussion). If a future UI need arises (e.g. a "🎧
    listening" indicator in the panel), closing this gap does not require
    blocking the endpoint's own HTTP response on step 7's confbridge wait
    — the AIcall's own `ListenCallID`/`Metadata[listen_transcribe_id]`
    fields (§5.8) already carry the eventual outcome and could be
    surfaced through the existing WebSocket/poll transport the panel
    already uses for messages, on a follow-up ticket.
15. **Naming, not blocking (§5.1, new in rev 15, updated in rev 16):**
    `ProcessListen`/`checkListenEligible`/`runListenStart`/`rollbackListenState`/
    `ListenTranscribeAIcallRemove` are this design's proposed names,
    chosen to be self-describing and consistent with this file's existing
    naming (`UpdateListenState`, `clearListenState`,
    `ListenTurnPipecatcallIDAdd`) — not verified against any
    linter/style convention beyond that. Fine to bikeshed at
    implementation time; not a design-level decision.
16. **New metric label, needs a name before implementation (§6, §5.13 —
    new in rev 16, review round 13 finding LOW-3; extended in rev 18,
    review round 15 finding LOW-1; §6's own rows split further in rev
    20, review round 17 finding B-6, once acquire- and release-side lock
    errors turned out to need different handling — the label question
    below is unaffected by that split, since a release error is
    deliberately never metered at all, §6):** §6's new rows for
    `checkListenEligible`'s AIcall-liveness rejection (MEDIUM-2), the
    pre-write/rollback failures (HIGH-3), and — newly, rev 18 — the
    per-AIcall start lock's own two outcomes (`skipped_start_locked`, a
    losing goroutine; `failed` on a `ListenStartLockAcquire` Redis error
    specifically, **not** a `ListenStartLockRelease` one, §6) are
    described only as folding into the existing `skipped_*`/`failed`
    buckets of `aicall_listen_start_total` (§5.13)
    — no new label was actually added to that metric's enumerated set
    the way §5.13's table lists every other outcome explicitly. Either
    add explicit labels (e.g. `skipped_aicall_not_live`,
    `failed_prewrite`, `skipped_start_locked`) or state plainly that
    these fold into `skipped_not_listenable`/`failed` and accept the
    reduced observability; small either way, but should be a deliberate
    choice recorded in §5.13's table before implementation, not left
    implicit. `skipped_start_locked` in particular is worth its own
    label regardless of the others' fate: unlike the other new outcomes,
    a sustained non-zero rate is a genuinely useful signal (heavy
    concurrent re-open pressure on one AIcall), not just a fail-closed
    edge case.
