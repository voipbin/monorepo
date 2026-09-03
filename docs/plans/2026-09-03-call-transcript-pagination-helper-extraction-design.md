# Extract `toolHandleGetCallTranscript`'s pagination loops into a shared generic helper

Date: 2026-09-03
Status: Finalized rev4 (rounds 1-6 design review complete; rounds 5+6 unanimous consecutive APPROVE, both reviewers). Ready for implementation.
Follow-up to: PR #1258 (VOIP-1453), code review round 2/3 non-blocking finding

## Revision history

- rev1: incorporated round-1 design review (2 reviewers -- security APPROVE
  with 2 hardening recommendations, correctness/idiom REQUEST_CHANGES on
  comment-relocation gaps). Security review confirmed the generic helper is
  behaviorally byte-identical to both existing loops (filter-then-decide
  ordering, `capped`/`possiblyIncomplete` biconditional, no predicate
  mixup risk) and recommended two doc-comment additions (both folded in):
  stating `keep` is the sole tenant/deletion enforcement boundary, and
  flagging the `limit`-vs-page-size coupling as load-bearing but
  unenforced. Correctness review independently confirmed the same
  mechanical equivalence but found the proposed snippets, while logically
  correct, dropped several load-bearing rationale comments from the
  original inline code without relocating them anywhere in the refactored
  source (only in this doc) -- specifically the "filter FIRST, caller-
  supplied filters are not server-enforced" invariant, the nil-TMCreate
  SQL three-valued-logic defensive rationale, the ISO8601Layout-vs-
  RFC3339Nano token-format rationale, and the "honest failure, not masked"
  policy note at §3.3's call site -- plus flagged that the CURRENT inline
  comment at `tool_insight.go:910-913` becomes a dangling cross-reference
  post-refactor and must be deleted, not left pointing at removed code.
  Fixed: all 5 round-1-named rationale comments now inlined directly into
  the helper's doc comment and loop body (§3.2) and into both call sites
  (§3.3/§3.4) rather than only existing in this design doc; §5's
  verification plan now explicitly calls out deleting the dangling
  cross-reference.
- rev2: incorporated round-2 design review (security APPROVE, confirming
  both round-1 hardening recommendations landed accurately with no new
  inaccuracy from the other comment relocations; correctness REQUEST_CHANGES
  on a fresh full read that found rev1's "all rationale comments" claim was
  an overclaim -- it was true for the 5 items round 1 named, but round 2
  found two MORE load-bearing comments the rev1 pass had missed: the
  `TranscribeID`-mismatch check's "why" (prevents a same-tenant row from a
  different session mislabeling which language was spoken), and §3.3's
  "exclusion rule stated generically, not IDAIManager-specifically" forward-
  compatibility rationale with its cross-reference to summaryhandler/
  start.go). Fixed: both comments now inlined at their call sites (§3.3's
  intro comment above the `paginateUntilExact` call, §3.4's `keep` closure's
  `TranscribeID` branch). Also closed round 2's one minor, non-blocking
  observation (uint64 parameter vs. the two limit constants' declared
  types) by confirming directly against source that both are untyped
  constants requiring no explicit conversion (§5 step 6).
- rev3: incorporated round-3 design review (security APPROVE, confirming
  the two round-2-added comments are documentation-only with no logic
  change and no security-relevant inaccuracy; correctness REQUEST_CHANGES
  on a fourth-attempt fresh full comparison against source, finding ONE
  more previously-missed comment -- the "Otherwise: keep paging, because
  the shortfall might be entirely accounted for by excluded rows"
  rationale explaining the loop's implicit fall-through-to-next-page
  branch, at source `tool_insight.go:800-804`). Fixed: restored into
  §3.2's helper loop body, in the same position relative to the overflow
  break and nil-TMCreate guard it occupies in the original code. Given
  this is now the third consecutive round where a fresh comparison found
  one previously-missed comment, a self-review before dispatching round 4
  built an exhaustive line-by-line inventory of EVERY comment in the
  source function (not another "fresh read") and cross-checked each one's
  disposition (relocated verbatim / paraphrased into the generic helper's
  general SECURITY note / correctly left untouched as downstream,
  unchanged code / correctly flagged for deletion as a dangling
  cross-reference). This caught one MORE previously-missed comment before
  round 4 even ran: the inline rationale immediately preceding the
  `possiblyIncomplete := ...` computation ("possiblyIncomplete captures
  every path that did not reach a proof... falling out of the loop
  without either would otherwise silently report [sessionCapped/capped]
  as false, exactly the unmarked hole this design exists to prevent") --
  only the helper's TOP doc-comment paraphrased this substance; the
  inline comment at the actual computation site was missing. Now restored
  at that exact site. Round 4 is scoped explicitly as a
  comment-completeness-only re-check, given the exhaustive-inventory
  method (not a "fresh read") is what actually closed the gap this time.
- rev4: incorporated round-4 design review (security APPROVE, confirming
  the two round-3-added comments were pure documentation with no logic
  change and no security-relevant inaccuracy; correctness REQUEST_CHANGES
  -- using the exhaustive line-by-line inventory method as instructed
  (33-row classification table, every comment in the source range
  categorized and cross-checked), which found what rounds 1-3's "fresh
  read" approach had structurally missed: a SECOND dangling comment,
  parallel to the one already caught (`tool_insight.go:910-913`), at
  `tool_insight.go:924-926` ("Mirrors §3.3's possiblyIncomplete term
  exactly") -- attached to §3.4's own local `possiblyIncomplete`/
  `sessionFetchTruncated` computation, which this refactor ALSO eliminates
  by moving it inside the shared helper, making this comment's premise
  (mirroring a separate, duplicate computation) stop being true for the
  same structural reason as the first dangling comment. Also flagged one
  minor completeness gap (the `pagesExhausted := false` declaration's
  substance-overlapping-but-technically-undocumented disposition).
  Fixed: added a second "Implementation note" naming the 924-926 comment
  explicitly and explaining why it must also be deleted; updated §5 step 2
  to require deleting BOTH dangling comments, not just the first; added an
  explicit one-line disposition comment at the `pagesExhausted := false`
  declaration site. This is the fourth consecutive round to find a gap in
  the comment-preservation exercise, but the first to be found by an
  exhaustive inventory rather than a holistic read -- round 5 should
  re-apply the SAME exhaustive-inventory method (not revert to "fresh
  read") as final confirmation.

## 1. Problem

`bin-ai-manager/pkg/aicallhandler/tool_insight.go`'s `toolHandleGetCallTranscript`
(lines 675-1056) is ~382 lines, well over the repo's `<50-line` function-size
guideline. Two independent code reviewers flagged this as a maintainability
issue and recommended extracting the near-identical §3.3 (Transcribe session
list) and §3.4 (Transcript segment list, per session) pagination-until-exact
loops into a shared helper, explicitly deferred to a follow-up rather than
done under review pressure.

## 2. Goal

Reduce duplication and function length WITHOUT changing any observable
behavior. The 41 existing `Test_toolHandleGetCallTranscript_*` tests must
continue passing **unchanged** — this is a structural refactor, not a logic
change. If any existing test's assertion needs to change, that is a signal
the refactor introduced a behavior change and must be treated as a defect,
not accepted as "test needed updating."

**Non-goals:** changing any comparator, threshold, error-handling policy, or
message wording. The two loops have genuinely different error-handling
policies at their call sites (§3.3's RPC failure fails the whole tool; §3.4's
RPC failure degrades one session visibly) — this MUST stay a caller-side
decision, not be absorbed into the shared helper.

## 3. Design

### 3.1 Why the two loops are extractable

Diffing §3.3 (`tool_insight.go:753-834`) against §3.4's inner loop
(`:859-931`), the loop bodies are structurally identical modulo:
- The RPC call and its filter map (different type, different fields)
- The per-row recheck predicate (different fields per type)
- The page-size/cap constant (`insightCallTranscribeSessionLimit` vs
  `resourceListPageSize`)
- What happens to the flags/verified slice AFTER the loop (session-list
  gates the whole tool's not-found handling; transcript-list gates a
  per-session gap marker)

The loop's own internals -- fetch, filter-then-decide, `pagesExhausted`,
`possiblyIncomplete`, the nil-`TMCreate` guard, the `ISO8601Layout` token
construction -- are byte-for-byte identical in logic between the two sites.

### 3.2 Shared helper (generic, Go 1.18+ -- an established idiom in this
repo: `bin-api-manager/pkg/servicehandler/aggregated_events.go`,
`bin-call-manager/pkg/dbhandler/normalization.go`,
`bin-common-handler/pkg/databasehandler/main.go`,
`bin-flow-manager/pkg/activeflowhandler/common.go` all already use generics)

```go
// paginateUntilExact fetches pages of T from fetch, keeping only items for
// which keep returns true, until it has unambiguous proof of either
// overflow (more than `limit` genuine items exist) or exhaustion (the
// source is genuinely out of rows) -- or gives up honestly at `maxPages`.
// See docs/plans/2026-09-03-insight-assistant-get-call-transcript-design.md
// §3.3 (lines ~103-220) for the full history of how this invariant was
// derived across 11 design-review rounds; do not change the comparators
// below without re-reading that section first.
//
// capped is true iff genuine overflow occurred OR the loop could not
// obtain proof within maxPages (the caller must treat this as "possibly
// incomplete", not "definitely not capped" -- see the design doc's
// possiblyIncomplete discussion).
//
// SECURITY: `keep` is the ONLY tenant/deletion enforcement this helper
// performs -- paginateUntilExact itself does no filtering by identity. Every
// call site MUST independently re-verify ownership fields (CustomerID,
// ReferenceID/TranscribeID, TMDelete) inside `keep`, mirroring §3.3/§3.4 of
// the design doc, because the underlying RPC's filter map is caller-supplied
// and NOT server-enforced. Do not add a new call site whose `keep` trusts
// the RPC's own filtering.
//
// `limit` MUST match the `+1` sentinel size actually requested inside
// `fetch` (i.e. fetch should request `limit+1` items per page) -- the
// pagesExhausted/overflow proof below depends on that exact coupling, and
// nothing in this function's signature enforces it. If a future edit
// changes one without the other, `capped` silently stops meaning what it
// claims to mean.
func paginateUntilExact[T any](
	ctx context.Context,
	maxPages int,
	limit uint64,
	fetch func(ctx context.Context, pageToken string) ([]T, error),
	tmCreateOf func(T) *time.Time,
	keep func(T) bool,
) (verified []T, capped bool, err error) {
	pageToken := ""
	pagesExhausted := false // true only once a short page PROVES no more source rows remain -- see the exhaustion break below, not restated here to avoid duplicating that comment (round-4 finding, minor: intentionally not duplicated at the declaration site)
	for page := 0; page < maxPages; page++ {
		items, ferr := fetch(ctx, pageToken)
		if ferr != nil {
			// Whole-call-failure vs. degrade-one-unit-visibly is a POLICY
			// decision that varies by caller (fail the whole tool vs. skip
			// just this unit) -- deliberately left to the caller, not
			// decided here. Any partial `verified` accumulated so far in
			// THIS call is discarded on error, matching both of this
			// function's original call sites' behavior exactly (neither
			// ever returned a partial page's worth of rows alongside an
			// error).
			return nil, false, ferr
		}

		// Filter FIRST, before deciding whether to fetch another page or
		// stop -- every field `keep` checks is caller-supplied to the RPC
		// and NOT server-enforced (RPC list endpoints parse filters from
		// the request body with no injected/enforced identity constraint),
		// so an excluded row must never be allowed to count toward "we
		// have enough genuine rows" or "the source is exhausted." This
		// ordering is the single most safety-critical property of this
		// function -- do not move the exhaustion/overflow checks above it.
		for _, it := range items {
			if keep(it) {
				verified = append(verified, it)
			}
		}
		if uint64(len(items)) < limit+1 {
			pagesExhausted = true
			break // raw page returned fewer than requested -- source is genuinely exhausted, regardless of how many were excluded
		}
		if uint64(len(verified)) > limit {
			break // already have unambiguous proof of overflow -- no need to keep paging just to count higher
		}

		// Otherwise: the page was full AND we still don't have more than
		// `limit` worth of GENUINE rows -- keep paging, because the
		// shortfall might be entirely accounted for by excluded rows in
		// this page.
		//
		// This nil-TMCreate guard is DEFENSIVE, at an untrusted RPC/
		// deserialization boundary -- for the two current call sites it is
		// NOT reachable via their DB queries in normal operation (both
		// filter with WHERE tm_create < token, and under standard SQL
		// three-valued logic NULL < <any value> evaluates to NULL, not
		// TRUE, so a tm_create IS NULL row would be excluded by the WHERE
		// clause on every page). Kept as defense-in-depth against a
		// malformed or future-changed RPC response, not because current
		// production data can trigger it -- but a NEW caller of this
		// generic helper must not assume that same DB-level guarantee
		// without re-verifying it for its own data source.
		last := items[len(items)-1]
		tm := tmCreateOf(last)
		if tm == nil {
			break // cannot safely construct a continuation token; `capped` below will honestly reflect "possibly incomplete"
		}
		// utilhandler.ISO8601Layout, NOT time.RFC3339Nano: this function's
		// two current callers both talk to bin-transcribe-manager, whose
		// own default token (TimeGetCurTime()) uses this fixed-precision
		// layout, not RFC3339Nano's variable-precision format -- a token
		// built with the wrong layout will not error, it will silently
		// mis-paginate. A future caller against a DIFFERENT RPC must
		// confirm which layout THAT service's own list endpoint expects
		// rather than assuming this one.
		pageToken = tm.UTC().Format(utilhandler.ISO8601Layout)
	}
	// possiblyIncomplete captures every path that did not reach a proof
	// (overflow or exhaustion) within the page budget -- falling out of
	// the loop without either would otherwise silently report `capped` as
	// false, exactly the unmarked hole this function exists to prevent.
	possiblyIncomplete := !pagesExhausted && uint64(len(verified)) <= limit
	capped = uint64(len(verified)) > limit || possiblyIncomplete
	if uint64(len(verified)) > limit {
		verified = verified[:limit]
	}
	return verified, capped, nil
}
```

**Deliberate design choices:**
- `err` is returned to the caller, NOT handled inside the helper -- §3.3 and
  §3.4 have different failure policies (fail-whole-tool vs degrade-one-
  session), which must stay caller-side.
- `keep` is a predicate closure, not a struct of filter fields -- lets each
  call site keep its own `log.Warnf` per exclusion reason exactly as today,
  with zero behavior change to logging.
- `tmCreateOf` is a plain accessor closure (not a `TMCreater` interface) --
  simplest thing that works for two call sites; an interface would be
  over-engineering for N=2.
- No behavior is added (no new logging, no new flags) -- this is pure
  extraction.

### 3.3 Call sites after extraction

§3.3 (session list, replaces `tool_insight.go:753-834`):
```go
// The exclusion rule below is stated GENERICALLY ("exclude any row whose
// CustomerID != c.CustomerID"), not as "exclude IDAIManager specifically",
// so it stays correct if a future system-initiated transcriber appears
// (design §3.3's summaryhandler/start.go IDAIManager note).
verified, sessionCapped, err := paginateUntilExact(ctx, insightCallTranscribeFetchMaxPages, insightCallTranscribeSessionLimit,
	func(ctx context.Context, pageToken string) ([]tmtranscribe.Transcribe, error) {
		return h.reqHandler.TranscribeV1TranscribeList(ctx, pageToken, insightCallTranscribeSessionLimit+1, map[tmtranscribe.Field]any{
			tmtranscribe.FieldCustomerID:    c.CustomerID,
			tmtranscribe.FieldReferenceType: tmtranscribe.ReferenceTypeCall,
			tmtranscribe.FieldReferenceID:   callID,
			tmtranscribe.FieldDeleted:       false,
		})
	},
	func(t tmtranscribe.Transcribe) *time.Time { return t.TMCreate },
	func(t tmtranscribe.Transcribe) bool {
		if t.CustomerID != c.CustomerID {
			log.Warnf("Skipping cross-customer transcribe session. call_id: %s, transcribe_id: %s, transcribe_customer_id: %s", callID, t.ID, t.CustomerID)
			return false
		}
		if t.ReferenceType != tmtranscribe.ReferenceTypeCall || t.ReferenceID != callID {
			log.Warnf("Skipping transcribe session with mismatched reference. call_id: %s, transcribe_id: %s, transcribe_reference_type: %s, transcribe_reference_id: %s", callID, t.ID, t.ReferenceType, t.ReferenceID)
			return false
		}
		return t.TMDelete == nil
	},
)
if err != nil {
	// Honest failure, not masked -- this is a list call, not a
	// not-found-shaped Get, and there is no partial result to salvage
	// yet (unlike §3.4's per-session degrade-visibly policy below, a
	// session-list failure fails the whole tool).
	log.Errorf("Could not list transcribe sessions. call_id: %s, err: %v", callID, err)
	fillFailed(res, fmt.Errorf("resource lookup failed"))
	return res
}
```

§3.4 (per-session transcript list, replaces `tool_insight.go:859-931`,
inside the `for _, t := range verified` loop):
```go
verifiedTranscripts, sessionFetchTruncated, err := paginateUntilExact(ctx, insightCallTranscribeFetchMaxPages, resourceListPageSize,
	func(ctx context.Context, pageToken string) ([]tmtranscript.Transcript, error) {
		return h.reqHandler.TranscribeV1TranscriptList(ctx, pageToken, resourceListPageSize+1, map[tmtranscript.Field]any{
			tmtranscript.FieldCustomerID:   c.CustomerID,
			tmtranscript.FieldTranscribeID: t.ID,
			tmtranscript.FieldDeleted:      false,
		})
	},
	func(tr tmtranscript.Transcript) *time.Time { return tr.TMCreate },
	func(tr tmtranscript.Transcript) bool {
		if tr.CustomerID != c.CustomerID {
			log.Warnf("Skipping cross-customer transcript row. call_id: %s, transcribe_id: %s, transcript_id: %s, transcript_customer_id: %s", callID, t.ID, tr.ID, tr.CustomerID)
			return false
		}
		if tr.TranscribeID != t.ID {
			// TranscribeID is exactly as unenforced as CustomerID/Deleted --
			// recheck it too. Without this, a same-tenant row from a
			// DIFFERENT session could render under THIS session's
			// t.Language tag, mislabeling which language was actually
			// spoken on that line.
			log.Warnf("Skipping transcript row with mismatched transcribe_id. call_id: %s, transcribe_id: %s, transcript_id: %s, transcript_transcribe_id: %s", callID, t.ID, tr.ID, tr.TranscribeID)
			return false
		}
		return tr.TMDelete == nil
	},
)
if err != nil {
	// Partial-failure policy: one session's TranscriptList call failing
	// does NOT fail the whole tool -- skip this session, but make the
	// skip VISIBLE (sessionsUnavailable, surfaced in the header) rather
	// than a silent drop, matching renderTranscribe's own "(transcripts
	// unavailable)" degrade-VISIBLY precedent. Contrast with §3.3's
	// call site above, which fails the whole tool on error -- this is
	// the caller-side policy difference paginateUntilExact deliberately
	// stays out of.
	log.Errorf("Could not list transcripts for session. call_id: %s, transcribe_id: %s, err: %v", callID, t.ID, err)
	sessionsUnavailable++
	continue
}
```

**Implementation note (round-1 design review finding, dangling comment #1):**
the CURRENT inline code has a comment at `tool_insight.go:910-913` ("Identical
nil-TMCreate guard and ISO8601Layout token format as §3.3's loop above -- see
that block's comment for the full rationale") pointing at the §3.3 loop that
this refactor removes. That comment must be DELETED as part of this change
(not left dangling) -- the rationale it pointed to now lives directly inside
`paginateUntilExact`'s own doc comment and inline comments (§3.2 above), which
both call sites already get "for free" by calling the shared function, so no
replacement cross-reference is needed at the call site at all.

**Implementation note (round-4 design review finding, dangling comment #2,
structurally identical to #1 above but missed by rounds 1-3's "fresh read"
passes -- only caught by round 4's exhaustive line-by-line inventory):** the
CURRENT inline code ALSO has a comment at `tool_insight.go:924-926` ("Mirrors
§3.3's possiblyIncomplete term exactly -- falling out of the loop at
insightCallTranscribeFetchMaxPages without proof of overflow or exhaustion
must not silently read as 'not truncated'") immediately before the local
`possiblyIncomplete`/`sessionFetchTruncated` computation this refactor also
removes -- that local computation no longer exists post-refactor (it moves
entirely inside `paginateUntilExact`, shared with §3.3), so this comment's own
premise ("mirrors §3.3's [separate, duplicate] term") stops being true once
there is no longer a duplicate computation to mirror. This comment must ALSO
be DELETED as part of this change, for the same reason as dangling comment #1
-- the rationale it pointed to is now the single shared inline comment at the
`possiblyIncomplete := ...` line inside `paginateUntilExact` itself (§3.2
above), not a second, parallel computation at this call site.

Everything downstream of these two call sites (not-found handling, gap
marker construction, merge/sort/render) is UNCHANGED -- it already consumes
`verified`/`sessionCapped` and `verifiedTranscripts`/`sessionFetchTruncated`
as opaque values, so it does not need to know they now come from a shared
helper.

## 4. Testing

No NEW tests required -- this is a pure refactor and the existing 41
`Test_toolHandleGetCallTranscript_*` tests already exercise every code path
`paginateUntilExact` replaces (multi-page continuation, H>=2 hidden rows,
nominal path, page-cap exhaustion, nil-TMCreate guard, RPC failure at both
call sites, filter-then-cap ordering at both layers). If all 41 pass
unchanged after the refactor, that IS the regression proof.

Add ONE small new unit test directly for `paginateUntilExact` in isolation
(a generic-friendly table test with a trivial `int`-like type), since it's
now a standalone, independently-testable unit -- this is additive, not a
replacement for the existing 41.

## 5. Verification plan

1. Extract `paginateUntilExact` as a new unexported function in
   `tool_insight.go` (or a new file `tool_pagination.go` in the same
   package if that reads more cleanly at implementation time).
2. Rewrite §3.3/§3.4's call sites to use it, preserving every comment
   currently attached to the replaced logic (move comments to the helper's
   doc comment or the call site as appropriate -- don't just delete them,
   per §3.2/§3.3's rev1-corrected snippets above). Specifically: DELETE
   BOTH now-dangling cross-reference/duplicate-computation comments --
   `tool_insight.go:910-913` ("see that block's comment above") AND
   `tool_insight.go:924-926` ("Mirrors §3.3's possiblyIncomplete term
   exactly") -- rather than leaving either pointing at code that no
   longer exists (round-4 finding: rounds 1-3 only caught the first of
   these two).
3. Run `go test ./pkg/aicallhandler/... -run Test_toolHandleGetCallTranscript -v` --
   all 41 must pass with NO assertion changes.
4. Full verification workflow: `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`.
5. Confirm the function length reduction: `toolHandleGetCallTranscript` should drop to roughly 200-250 lines (from ~382).
6. (Round-2 finding, minor, confirmed non-issue) `insightCallTranscribeSessionLimit`
   (`tool_insight.go:56`) and `resourceListPageSize` (`tool_resource.go:42`)
   are both declared as untyped constants (`const x = 10`, no explicit
   type), so they convert implicitly to `paginateUntilExact`'s `uint64`
   parameter with no cast needed at either call site -- verified directly
   against the current source, not assumed.
