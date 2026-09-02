# get_call_transcript implementation plan (rev1)

Design doc (finalized, 2 consecutive design-review approvals, rounds 10+11
of an 11-round loop -- rev1 correction, plan round-1 review, LOW finding:
this header previously miscited "rounds 13+14 of a 14-round loop", which
does not match the design doc's own authoritative count at its Status line
and §8 -- "eleven rounds", closed at rounds 10+11):
`docs/plans/2026-09-03-insight-assistant-get-call-transcript-design.md`

Jira: VOIP-1453

Goal + acceptance criteria: add a 6th read-only Insight tool,
`get_call_transcript(call_id)`, exactly per the design doc §3.0-§3.7:
entry guard (surface-reduction only, not security) → tenant-only access
check on the unscoped `CallV1CallGet` (LLM-suppliable `call_id`, no
narrowing to Case peer/contact) → pagination-until-exact fetch of
`Transcribe` sessions (§3.3) → pagination-until-exact fetch of `Transcript`
rows per session (§3.4) → strict-total-order cross-session merge (§3.5) →
gap markers for genuine fetch-layer truncation (§3.6) → header/lines
rendering with `pagedOut=false` (§3.7). Ships in v1 with `transcribe_start`
(live in-call transcription) only; `transcribe_recording` is a different
resource, explicitly deferred to a future tool (design §2). Auto-activated
for all `tool_names=["all"]` Insight AIs per 대표님's explicit tenant-only
access decision (design §0.2/§7).

**Single commit**: steps 1 through the staging+commit step land as ONE
commit, following the `get_contact_profile` precedent (see that ticket's
own plan) — the tree is deliberately incomplete between step 1 (new
`AllInsightToolNames` entry) and step 6 (its `ToolDefinition`), and
between step 1 and step 3 (`allowed_tools_test.go` allowlist). If the code
review loop requires changes, add fixup commits on the same branch and
re-run the verification/staging steps before pushing again.

## Steps (in order; each is a checkpoint)

- [ ] 1. `bin-ai-manager/models/tool/main.go`: add `ToolNameGetCallTranscript`
      const, appended to `AllInsightToolNames` (design §6).
- [ ] 2. `bin-ai-manager/models/message/tool.go`: add
      `FunctionCallNameGetCallTranscript` const (design §6).
- [ ] 3. `bin-ai-manager/models/ai/allowed_tools_test.go`: add
      `tool.ToolNameGetCallTranscript` to the `knownReadOnly` allowlist
      (design §6 — a deliberate read-only consent-gate test; update in the
      same change or `TestAllInsightToolNamesAreReadOnly` fails red).
      Attest explicitly in this step and in the PR body that this tool
      performs no writes anywhere in its RPC chain (`CallV1CallGet`,
      `TranscribeV1TranscribeList`, `TranscribeV1TranscriptList` are all reads).
- [ ] 4. `bin-ai-manager/pkg/aicallhandler/tool_insight.go`: implement
      `toolHandleGetCallTranscript` per design §3.0-§3.7 exactly.
      **Copy the design's §3.0-§3.7 code blocks directly (including
      §3.2's guard condition); adapt only what's needed to compile.** Do NOT re-derive the comparators,
      guard conditions, or the `possiblyIncomplete` logic from the
      surrounding prose or from memory of the design discussion — this
      document's own revision history (revisions rev5, rev9, rev10 -- each
      fixed a stale-prose-next-to-correct-mechanism instance) shows that exact
      failure mode (a correct mechanism sitting next to stale/summarized
      prose) repeatedly reintroduced retired bugs during review. The code
      blocks in the design doc are the authoritative source, not a
      paraphrase of them. This is the largest step; break it into the
      design doc's own sub-sections while writing, in this order:
      - §3.0 entry guard (`c.ReferenceType != aicall.ReferenceTypeContactCase`
        → `fillFailed`), explicitly commented as surface-reduction, not security.
      - §3.1 `call_id` argument: 3-branch validation (missing/malformed/
        `uuid.Nil`) mirroring `get_conversation_content`'s precedent
        (`tool_insight.go:190-206`).
      - §3.2 tenant-only access check: `CallV1CallGet(ctx, callID)`
        (unscoped RPC) + mandatory recheck with ALL THREE disjuncts from
        the design's code block, not just the middle one:
        `call == nil || call.CustomerID != c.CustomerID || call.CustomerID == uuid.Nil`
        (the `uuid.Nil` branch is not redundant — it matters if
        `c.CustomerID` is itself zero-valued). Copy this guard verbatim,
        same as §3.3/§3.4/§3.5/§3.7's code blocks. Not-found and
        cross-tenant both collapse to the byte-identical
        `msgResourceNotFound`. No narrowing to Case peer/contact (confirmed
        product decision, design §0.2). Soft-deleted calls ARE returned
        (no filter added — accepted per design §3.2/§5). **Per design §3.2:
        do NOT comment this check as "defensive" or "redundant" — it is the
        SOLE tenant enforcement on the call itself, load-bearing exactly
        like `get_contact_profile`'s mandatory contact check.**
      - §3.3 Transcribe session list: the bounded pagination-until-exact
        loop exactly as specified in the design doc's current code block —
        `insightCallTranscribeSessionLimit = 10`,
        `insightCallTranscribeFetchMaxPages = 5` package-level constants
        (grep the file first to confirm neither name collides with an
        existing constant — as of this plan's writing they do not, but
        re-verify at implementation time). Per-row rechecks are MANDATORY:
        four filter keys (`CustomerID`, `ReferenceType`, `ReferenceID`,
        `TMDelete`) across the design's own THREE branches — `CustomerID`
        gets its own `log.Warnf` branch; `ReferenceType`/`ReferenceID`
        SHARE one branch and one `log.Warnf` (not two separate branches —
        matching the design's code block exactly, `if t.ReferenceType !=
        ... || t.ReferenceID != ...`); `TMDelete` gets its own branch with
        NO log call (`if t.TMDelete != nil { continue }`, silent skip).
        Do not add a fourth `Warnf` or split the reference-type/reference-id
        check into two branches — that would deviate from the design's
        authoritative code block. The nil-`TMCreate` defensive guard uses
        `utilhandler.ISO8601Layout`, NOT `time.RFC3339Nano`, for the
        pagination token; `pagesExhausted`/`possiblyIncomplete`/
        `sessionCapped` exactly as derived in the design's rev10 code
        block. Not-found handling guarded by `sessionCapped` (distinct
        "no transcript found" vs "could not confirm" messages). State the
        exclusion rule GENERICALLY in code comments (design §3.3:217:
        "exclude any row whose `CustomerID != c.CustomerID`"), not as
        "exclude IDAIManager specifically" -- stays correct if a future
        system-initiated transcriber appears.
      - §3.4 Transcript segment fetch per session: same pagination-loop
        structure against `TranscribeV1TranscriptList`, its own
        `possiblyIncomplete`/`sessionFetchTruncated`. Per-row rechecks are
        MANDATORY here too, individually: `CustomerID`, `TranscribeID`,
        and `TMDelete` on `Transcript` rows (design §3.4's three
        filter-key rechecks). Carry `t.Language` (from the already-fetched,
        already-tenant-verified parent `Transcribe`) onto every row from
        that session -- `Transcript` has no language field of its own; this
        is a prerequisite for both the `[ts direction lang]` render format
        (§3.7) and for correctly labeling concurrent differently-languaged
        sessions. The per-session partial-failure policy
        (`sessionsUnavailable`, one session's RPC failure doesn't fail the
        tool), and the gap-marker construction guarded by
        `sessionFetchTruncated && len(verifiedTranscripts) > 0`.
      - §3.5 `transcriptLine` struct + `sort.SliceStable` merge on the
        5-component strict total-order key `(tmCreate, rank, transcribeID,
        transcriptID, seq)`. **The primary sort key is `TMCreate`, NOT
        `TMTranscript`** (design §3.5:389) -- `TMTranscript` exists on the
        `Transcript` model (`bin-transcribe-manager/models/transcript/transcript.go:20`)
        and its name reads like the obvious transcript timestamp, but it
        is a per-session OFFSET from that session's own start; sorting on
        it would incorrectly interleave concurrent sessions. This is the
        single most attractive wrong choice in the whole implementation --
        do not reach for it. Nil `TMCreate` sorts LAST (a new decision
        this design makes, not an inherited file convention -- see design
        §3.5's rev3 correction). New imports needed beyond what
        `tool_insight.go` currently has (`context`, `encoding/json`,
        `fmt`, `strings`, `time`, `uuid`, `errors`, `logrus` are already
        present; `sort` is NOT) -- confirm the full new-import list at
        implementation time rather than assuming `sort` is the only one:
        it also needs `bin-common-handler/pkg/utilhandler` (for
        `ISO8601Layout`) and the `tmtranscribe`/`tmtranscript` model
        packages (already used under identical aliases in this same
        package's `tool_resource.go`, so no new aliasing convention is
        introduced).
      - §3.6 gap marker semantics (the exception clause: zero verified
        rows → zero marker, not "for every flagged session"). Add TWO
        code comments design §3.6 mandates at the `renderBodyLines` call
        site (§3.7), per the design's rev2 precision note (design:409):
        (a) name `renderBodyLines`' TWO internal truncation mechanisms
        explicitly rather than treating it as one opaque step -- the
        backward-walk truncation (`tool_resource.go:331-338`) AND the
        separate, coarser final `capSummaryRunes(sb.String())` string-tail
        cut (`tool_resource.go:363`); (b) reference the gap-marker
        mechanism's own contiguous-newest-suffix composition invariant
        (design §3.6's cross-file dependency note) so a future change to
        `renderBodyLines` doesn't silently break this tool's gap-marker
        honesty guarantee. Both comments are required, not alternatives.
      - §3.7 rendering: sort → render `[]transcriptLine` into
        `[]string` (display timestamp format is `time.RFC3339` here --
        note this is DIFFERENT from §3.3/§3.4's pagination-token format,
        `utilhandler.ISO8601Layout`; do not conflate the two), header
        construction (`transcript_lines: N`, conditional
        `transcribe_sessions: N (capped, more may exist)` — no numeric
        total, conditional `sessions_unavailable: N`),
        `renderBodyLines(header, renderedLines, false, "transcript lines")`,
        `fillSuccess(res, "call_transcript", callID.String(), body)`.
      - **Implementation-time verification (design §6):** confirm
        `utilhandler.ISO8601Layout` is actually importable from
        `tool_insight.go` (it lives in `bin-common-handler/pkg/utilhandler`,
        already an indirect dependency), and that the token's wire
        round-trip through the RPC transport preserves it unchanged --
        do not just assume the design doc's citation is current.
- [ ] 5. `bin-ai-manager/pkg/aicallhandler/tool.go`: wire
      `FunctionCallNameGetCallTranscript` into the `mapFunctions` dispatch
      map (design §6).
- [ ] 6. `bin-ai-manager/pkg/toolhandler/definitions.go`: add the
      `ToolDefinition` — `RunLLM: true`, WHEN TO USE / WHEN NOT TO USE
      style, `call_id` as a required string parameter, and explicit
      `get_contact_interactions`-first chaining language in the
      description (design §3.1/§6), matching `get_conversation_content`'s
      own chaining-hint precedent.
- [ ] 7. `bin-ai-manager/pkg/aicallhandler/tool_insight_test.go`: add all
      table-driven test cases from design §4 (the full, corrected-through-rev10
      list — re-read design §4 directly while writing tests, it is the
      authoritative source; do not work from memory of the design
      discussion). At minimum, ensure these are present and not skipped:
      1. `ReferenceType` guard → `fillFailed`, zero RPC calls.
      2. Empty / malformed / `uuid.Nil` `call_id` → `fillFailed`, three
         distinct cases.
      3. Call not found → masked; call cross-tenant → masked; both assert
         byte-identical `msgResourceNotFound` AND zero
         `TranscribeV1TranscribeList` calls on both paths (proves the
         tenant check short-circuits before any transcript fan-out —
         design §4 lines 493-494).
      4. Happy path: single session, several transcripts, correct
         `[ts direction lang] message` rendering.
      5. Happy path: no Transcribe sessions found (post-filter) → distinct
         "no transcript found for this call", non-masked.
      6. Cross-tenant Transcribe row filtered (IDAIManager scenario):
         excluded, no leak.
      7. Cross-tenant Transcript row filtered (`CustomerID` mismatch):
         excluded, others kept.
      7b. **Transcript row with mismatched `TranscribeID`** (design:514,
          rev4): excluded, does NOT render under the wrong session's
          `t.Language` — distinct from item 7, this is the load-bearing
          test for the language-mislabeling risk §3.4's `t.Language`
          carry-over creates if unchecked.
      7c. **Transcript-layer `TMDelete` recheck** (design:516, rev5,
          explicitly added because an earlier draft covered only the
          Transcribe-layer soft-delete recheck): a `Transcript` row with
          non-nil `TMDelete` → excluded.
      8. Multi-session merge, chronological order across sessions/languages.
      9. Per-session fetch truncation → gap marker at correct position, a
         concurrent untruncated session's rows correctly surround it.
      9b. **§3.4 silent-hole false negative, CORRECTED fixture** (design:522,
          rev8 H2 -- do NOT use the fixture below, it was the retired,
          self-contradictory version): a session with
          `resourceListPageSize + 1` (NOT exactly `resourceListPageSize`)
          genuine rows plus one or more excluded rows landing inside the
          raw fetch → `sessionFetchTruncated == true`, gap marker fires,
          all `resourceListPageSize` kept rows are genuine (none displaced).
      10. Nil `TMCreate` on a transcript row (sort/render path, design:502):
          defensive fixture (MOCK-CONSTRUCTED, not a claim about reachable
          production data per design §3.3/§3.5's rev9 correction) → sorts
          last, is PREFERENTIALLY RETAINED under render-budget truncation
          (explicit assertion — this is the non-obvious downstream
          consequence of the nil-sorts-last rule, don't drop it), renders
          "unknown", no panic.
      10b. **Nil-`TMCreate` boundary row during pagination halts the loop
          without panicking** (design:513, rev8 H1 -- distinct from item
          10 above, which is about sort/render, not the pagination loop
          itself): a raw page whose LAST row has `TMCreate == nil` and
          where the page was otherwise full → assert no panic, the loop
          stops at that page, and `sessionCapped`/`sessionFetchTruncated`
          read `true` via `possiblyIncomplete`.
      11. Session-count cap: >10 verified sessions → capped to 10, header
          `transcribe_sessions: 10 (capped, more may exist)`, no numeric
          total, AND `renderBodyLines`' own marker does NOT fire purely
          from the session cap (only from actual line overflow) —
          regression test for the `pagedOut=sessionCapped` bug an early
          design draft had.
      12. Filter-then-cap ordering at BOTH the session layer and the
          transcript layer (two separate tests, mirroring the design's
          rev3/rev4 regression tests).
      13. Wrong-call / soft-deleted Transcribe row rechecked → excluded.
      14. Per-session RPC failure degrades VISIBLY (`sessions_unavailable`
          header field), doesn't fail the whole tool.
      14b. **`TranscribeV1TranscribeList` RPC error** (design:515, rev5 --
          a DIFFERENT policy from item 14): the session-list call itself
          returns a non-not-found error → `fillFailed`, an honest failure
          (not masked, not degraded), since this is the first list call
          in the chain and there is no partial result to salvage.
      15. Header `transcript_lines: N` reflects only real rows.
      16. **Pagination loop actually pages when needed** (multi-page
          continuation, `pageToken` uses `utilhandler.ISO8601Layout` — NOT
          `RFC3339Nano`; assert against the LITERAL expected format string,
          not just internal self-consistency with whatever the code produced).
          ALSO assert the final `verified`/`verifiedTranscripts` slice
          contains every genuine row across all pages, capped correctly
          at the configured limit. Use a gomock call-count assertion
          (`Times()`) to assert the RPC is called more than once, not
          just "eventually contains all rows" as an indirect signal.
      17. **`sessionCapped`/`sessionFetchTruncated` do NOT leak
          filtered-row existence for H>=2 hidden rows**, spread across
          page boundaries. Assert BYTE-IDENTICAL rendered output AND
          header between the with-hidden-rows and without-hidden-rows
          fixtures, not just equal `sessionCapped` booleans.
      18. **Nominal path, exactly `limit` genuine rows, WITH and WITHOUT
          exclusions** — both must read `false`, no false-positive cap/gap
          (regression test for the rev6 `>=`-comparator bug).
      19. **Pagination loop respects `insightCallTranscribeFetchMaxPages`**
          and `possiblyIncomplete` makes the page-cap exit honest (>=45
          excluded rows at session layer, >=405 at transcript layer —
          use the CORRECT layer-specific threshold, not 45 for both;
          assert against the literal threshold value, not a hand-waved
          "large number"). ALSO assert no numeric hidden-row count is
          exposed anywhere in the header at this boundary.
      20. **`sessionCapped`-guarded not-found message**: `verified` empty +
          `sessionCapped=true` → distinct "could not confirm" message, not
          a false "no transcript found".
      21. OUTER RPC fan-out count (session-iteration loop) invariant to
          hidden-row presence; INNER pagination RPC count is NOT invariant
          — pin the actual variance (1 page vs 2+ pages), do not assert a
          false invariance.
      22. `seq` tiebreak determinism (identical `TMCreate`/`rank`/
          `TranscribeID`/`TranscriptID` -> order matches fetch order,
          deterministic across repeated runs); gap-boundary nil-`TMCreate`
          handling (marker copies the nil verbatim, placed per the
          nil-sorts-last rule, not asserted at a specific position);
          `transcript_lines` pre-render-vs-rendered divergence (header
          reflects the full pre-truncation merged count even when fewer
          lines are actually rendered -- assert both numbers independently);
          all-sessions-unavailable terminal state (`Result: "success"`,
          `transcript_lines: 0`, `sessions_unavailable: N`, no panic on an
          empty merge).

      **This numbered list (including the `N`/`Nb`/`Nc` items) is meant to
      be exhaustive against design §4 as of rev10, but re-read design §4
      directly while writing tests regardless — it is the authoritative
      source, this list is a checklist derived from it, not a replacement
      for it. If §4 and this list ever disagree, §4 wins and this plan
      should be corrected.
- [ ] 8. `bin-ai-manager/docs/domain.md`: add one row for `get_call_transcript`
      to the existing "LLM Tools" table (service-docs-sync hook target;
      design §6). Do not backfill other already-missing rows (scope discipline).
- [ ] 9. OpenAPI + RST wiring (design §6, mirrors `get_contact_profile`'s
      own required wiring exactly — do not skip, `AIManagerToolName` is a
      closed enum):
      - `bin-openapi-manager/openapi/openapi.yaml`: add `get_call_transcript`
        to BOTH the `AIManagerToolName` enum value list AND the parallel
        `x-enum-varnames` list, in lockstep.
      - `bin-openapi-manager/openapi/paths/ais/main.yaml` and `id.yaml`:
        update the Insight tool-name prose from 5 to 6 tools.
      - Regenerate: `go generate ./...` in `bin-openapi-manager` first,
        then `bin-api-manager` (`gens/openapi_server/gen.go`,
        `gens/openapi_redoc/openapi.json` + `api.html`).
      - RST docs (`bin-api-manager/docsdev/source/`): `ai_overview.rst`
        (tool-set mention), `ai_struct_tool.rst` (summary table + a new
        per-tool section WITH ITS OWN ANCHOR matching the existing
        per-tool pattern + `run_llm` table), `ai_struct_ai.rst` (two spots
        enumerating the Insight tool set by name). Clean rebuild:
        `cd bin-api-manager/docsdev && rm -rf build && python3 -m sphinx -M html source build`,
        then `git add -f bin-api-manager/docsdev/build/` per root CLAUDE.md's
        RST-sync rule.
- [ ] 10. Update the design doc's own Status line if any further drift is
      noticed during implementation (should not be needed — it already
      accurately reads "Draft rev10 ... design phase complete" after the
      review loop's own closing rounds).
- [ ] 11. **Verification workflow** (root CLAUDE.md, mandatory, run from
      `bin-ai-manager/` — and `bin-openapi-manager/`, `bin-api-manager/` for
      step 9's changes):
      ```
      go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m
      ```
      Do not skip any step. Confirm `go test ./...` output for
      `tool_insight_test.go` and `allowed_tools_test.go` specifically shows
      the new tests passing, not just "no failures" (i.e., actually verify
      the new test names appear in verbose output).
- [ ] 12. Stage and commit as ONE commit (per the single-commit rule
      above). Commit message: title matches branch name
      (`VOIP-1453-Add-call-transcript-insight-tool`), body lists affected
      projects with `bin-<service>:` prefixes per root CLAUDE.md's format.
      No AI attribution.
- [ ] 13. Pull latest `main`, check for conflicts
      (`git fetch origin main && git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"`),
      resolve if any, re-run step 11 if resolution touched code.
- [ ] 14. Push branch, create PR (title = branch name, body in the
      established narrative + `bin-<service>:` bullet format, no headers,
      no test-plan section, no AI attribution).
- [ ] 15. **Code review loop** (root CLAUDE.md mandatory policy): minimum
      3 rounds, 2 CONSECUTIVE approvals required, max 30 rounds. Dispatch
      independent reviewer agents (code-quality + security, mirroring the
      design review's two-reviewer pattern) against the actual diff. At
      least one round must explicitly diff the test suite's assertions
      against design §4 line-by-line (not just review the implementation
      for style) — this is the only gate that catches a test sharing the
      same bug as its implementation (e.g. a pagination test asserting
      against whatever token format the code happens to produce, rather
      than the literal `utilhandler.ISO8601Layout` string design §3.3
      requires). At least one round must ALSO explicitly diff the
      implementation's §3.2-§3.7 logic against the design's own code
      blocks section-by-section (not just style-review it) — this is what
      gives the design-reopen branch below a concrete procedure: if the
      code matches the design's stated logic exactly and a reviewer still
      finds a defect, that's unambiguously a design-level issue (reopen);
      if the code deviates from the design's code block, that's an
      implementation slip (fix in code, no reopen needed).
      **Design-reopen branch:** if a code-review round finds a substantive
      correctness or security defect in the DESIGN's own logic (i.e. the
      design doc's invariant is itself wrong for some case the design
      review loop didn't consider — not a transcription/implementation
      slip), STOP the code-review loop, do not patch around it at the code
      level, and re-open the design doc for a new revision round instead.
      Only resume this step once the design is re-approved. This has
      concrete precedent within this same design's own history (rounds 6
      and 7 each found the document's own stated invariant was
      mathematically/logically wrong, not just a citation slip, each
      caught by a later independent reviewer) — treat a code reviewer
      finding the same class of issue as equally credible, not as
      out-of-scope for the coding stage.
- [ ] 16. Report to 대표님, wait for explicit "머지 진행해줘"-style
      instruction before merging (NEVER merge without it, even after
      approval — per root CLAUDE.md and this session's own prior precedent
      with `get_contact_profile`). Include in the report design §7's
      caching note: `bin-pipecat-manager` caches tool definitions and
      refreshes on its own schedule, so the tool may be invisible to
      already-running Insight sessions until pipecat-manager's cache
      refreshes after `bin-ai-manager` deploys -- fail-safe, not
      instantaneous, so 대표님 isn't surprised if it's not immediately
      visible post-deploy.

## Working notes / constraints carried from the design doc

- **Access control is tenant-only, not case-scoped** — this is the single
  most important behavioral difference from every other Insight tool
  shipped so far. Do not "fix" this during implementation by adding a
  peer/contact check; it was an explicit, reviewed product decision.
- **Do not simplify the pagination loop back to a single fixed-size
  fetch.** The bounded pagination-until-exact structure (§3.3/§3.4) is the
  load-bearing fix for a leak that took 7 design-review rounds to close
  correctly. If it looks over-engineered during implementation, re-read
  design §3.3's "Why this is genuinely leak-free" note before touching it.
- **`utilhandler.ISO8601Layout`, not `time.RFC3339Nano`**, for the
  pagination token format — this specific mistake was made and caught
  twice during design review; it will compile fine either way, so the
  test suite (step 7, item 16) is what actually catches it.
- **The nil-`TMCreate` guards in §3.3/§3.4 are defense-in-depth, not
  dead code to be removed** — they guard an RPC/deserialization boundary
  that is unreachable via the current DB query in production (verified in
  design review), but keep them; do not "clean up" what looks like
  unreachable code here.
- **`sessionCapped`/`sessionFetchTruncated` semantics**: `true` iff
  genuine overflow of the configured limit, OR the pagination loop could
  not obtain proof within its page budget (`possiblyIncomplete`). This is
  a biconditional, not a raw count — do not add a numeric "of N" total to
  any header field (would leak hidden-row existence).
