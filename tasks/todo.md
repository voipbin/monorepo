# VOIP-1455 - emit_info_card Insight tool (backend)

Design doc: `docs/plans/2026-09-04-insight-assistant-emit-info-card-design.md`
Branch: `VOIP-1455-Add-emit-info-card-insight-tool`
Scope: backend only (this worktree/PR). Frontend (`CardBlock` rendering +
VOIP-1458 allowlist fix) is a separate PR in monorepo-javascript, tracked
separately once this backend PR is ready.

## Phase 0 - Review gates (per CLAUDE.md review-loop policy)

- [x] Issue analysis review loop: 5 rounds (1-3 REQUEST_CHANGES, 4+5
      consecutive APPROVE). See design doc §1 for the condensed conclusions.
- [x] Design review round 1 (architect, REQUEST_CHANGES): D1 primary/
      secondary enforcement backwards, D4 comparator wrong (blocks isn't a
      wire field), D3 empty-state gate bug, AVAILABLE_TOOLS sync missing,
      4 hedges to confirm, 1 citation fix -- all fixed
- [x] Design review round 2 (code-reviewer, REQUEST_CHANGES): all 7 round-1
      fixes confirmed accurate; new finding -- square-main shares VOIP-1458's
      defect class, filed as VOIP-1459 -- documented and referenced
- [x] Design review round 3 (architect, REQUEST_CHANGES): round-2 fixes
      confirmed accurate; **blocking new finding** -- getPipecatcallMessages
      rebuilds LLM history from DB on every turn and re-injects card JSON
      (D2's original fix only covered the first turn) -- added D2's second
      fix (strip blocks in start.go:637-657), plus 1 test-name correction
      and 2 wording consistency fixes
- [x] Design review round 4 (code-reviewer, APPROVE) -- confirmed round-3
      fix accurate; 1 non-blocking suggestion (unmarshal-failure fallback)
      added
- [x] Design review round 5 (architect, REQUEST_CHANGES): **3rd injection
      point found** -- the tool-call REQUEST message (`tool.go:47`) stores
      the LLM's raw `Function.Arguments` (same content as the card, just
      pre-execution), and `start.go:648-650` replays it verbatim every
      turn via `tool_calls`. Round-3's fix (part 1) only stripped the
      tool-RESULT message's content; this is a separate copy in a
      different message. Added D2 fix part 2 (neuter `Arguments` in the
      tool_calls copy while keeping the entry itself, required for
      chat-completion API tool_calls/tool-message pairing). Also fixed a
      "20 other tools" -> "21" count typo. Counter reset to 0 (this round
      was REQUEST_CHANGES).
- [x] Design review round 6 (code-reviewer, REQUEST_CHANGES): traced D2
      part 2's rationale one layer further (Python's `run.py:449,634`
      `valid_messages` filter) and found it **already drops the tool-call
      request message today** (content always ""), independent of what
      round-5's fix does -- meaning part 2's "required for API pairing"
      justification doesn't match current behavior. Deeper investigation
      (agent af22bc96038a6cb4f) confirmed via live production log
      inspection (Komodo API) that this leaves the paired tool-RESULT
      message orphaned in real traffic today, provider-dependent outcome
      (Gemini degrades silently, OpenAI/Grok has no defense). This is an
      **independent, pre-existing defect affecting all 6 existing Insight
      tools**, filed as VOIP-1460 (Highest). D2 part 2's rationale rewritten
      as defense-in-depth (correct regardless of whether/when VOIP-1460 is
      fixed) rather than a claim about current API-call correctness.
      Counter reset to 0.
- [x] Design review round 7 (architect, REQUEST_CHANGES): round-6's
      corrected rationale was applied to §1.2/§8 only, not propagated to
      §4/§6/todo.md (F1 -- those still asserted the retracted "API pairing
      requires it" claim); D2's resource_type/resource_id justification
      cited a nonexistent codebase precedent (F2 -- every real tool uses a
      non-empty resource_type, fixed to `"card"`); `messageContent.Message`
      was undecided across 3 sections with both readings broken (F3 --
      resolved: title-only trace, identical across storage/LLM-feedback/
      frontend, D5 amended to never render `content.Message` for cards).
      Also: citation fix (run.py:450 and :637, two sites) and confirmed a
      4th message-read path (`renderAIcall`) is already safe, no work
      needed. Counter reset to 0.
- [x] Design review round 8 (code-reviewer, REQUEST_CHANGES): F2/F3/N1-N3
      all confirmed correctly reflected; F1 was only partially propagated
      -- Phase 3's test bullet (this file) still had the retracted
      "protocol-pairing guard" phrasing while Phase 0/1 and design doc
      §4/§6 already had the corrected "defense-in-depth" framing -- fixed.
      Counter reset to 0.
- [x] Design review round 9 (architect, APPROVE) -- confirmed todo.md
      Phase 3 fix applied correctly, full grep sweep found no remaining
      "protocol-pairing" phrasing outside Phase 0's historical review log
      (correctly left as-is, audit trail). All prior 8 rounds' decisions
      cross-checked against code, zero citation errors found. Streak 1/2 --
      round 8 was REQUEST_CHANGES, so one more consecutive APPROVE needed.
- [x] Design review round 10 (code-reviewer, APPROVE) -- **2 consecutive
      approvals (rounds 9+10), design review loop satisfied.** Re-verified
      ~15 files including frontend cross-repo citations, zero errors found.
      Design review loop closed at round 10 (min 2, 8 REQUEST_CHANGES + 2
      consecutive APPROVE). Ready for backend implementation.
- [x] Code review round 1 (code-reviewer, APPROVE) -- D1 truncation
      (rune-based, no mid-char cuts), D2 (bypass, Message trace,
      resource_type/id, 3-injection-point fix incl. non-mutating copy),
      read-only invariant rewording, RunLLM guard text, test quality all
      verified against code. Zero findings.
- [x] Code review round 2 (security-reviewer, APPROVE) -- input validation/
      DoS (rune-truncation safe, JSON depth bounded by stdlib), tenant
      isolation, all 3 re-injection points independently re-verified closed,
      webhook exposure, OpenAPI additive-only, log hygiene. 2 LOW/informational
      notes (both pre-existing, all-21-tools patterns, not this PR's
      regression, not blocking): (a) field-count truncation happens after
      full JSON unmarshal, not before parsing -- LLM-output-bounded in
      practice; (b) the tool-call REQUEST message's raw Arguments is
      webhook-published before the handler's truncation runs -- follow-up
      candidate, out of this ticket's scope (all 21 tools share this).
- [x] Code review round 3 (go-reviewer, APPROVE) -- confirmed `message.ToolCall`/
      `FunctionCall` are pure value types field-by-field (no pointer/slice/map
      members), so Part 2's slice copy is genuinely safe (not a shallow-copy
      trap); concurrency (no mutable package-level state, `-race` clean, 860
      tests); JSON/naming idiom consistency; test idiom consistency (table-
      driven, reflect.DeepEqual, matches package convention); checked for
      over-defensive code too, found none.
      **3 rounds run (min met), all 3 APPROVE (exceeds 2-consecutive
      requirement) -- code review loop satisfied.**

## Phase 1 - Backend implementation (bin-ai-manager)

- [x] `models/tool/main.go`: `ToolNameEmitInfoCard`, add to `AllInsightToolNames`,
      reword read-only invariant comment
- [x] `pkg/toolhandler/definitions.go`: tool definition (RunLLM:true + guard
      description, JSON Schema params per D1)
- [x] `models/message/tool.go` (or wherever FunctionCallName lives):
      `FunctionCallNameEmitInfoCard`
- [x] `pkg/aicallhandler/tool.go`: `CardField`/`CardBlock` types,
      `Blocks omitempty` on `messageContent`, dispatch map entry
- [x] New `pkg/aicallhandler/tool_emit_info_card.go`: handler, validation/
      truncation per D1 caps
- [x] Bypass `unmarshalToolResponse` for this tool only (D2, first-turn fix)
      -- verify isolated branch, no other tool affected. Implemented as a
      small extracted helper `emitInfoCardLLMResult(tmpContent
      *messageContent) map[string]any` (not in the original design plan,
      added during implementation purely so the bypass's exact key set is
      independently unit-testable without mocking the full `ToolHandle`
      dispatch path, which no existing test in this package does).
- [x] `start.go:637-657` `getPipecatcallMessages`, part 1: strip `blocks`
      from `role: tool` message content when rebuilding LLM history (D2
      fix part 1, design review round 3; fall back to original string on
      unmarshal error; no-op for every other tool)
- [x] `start.go:637-657` `getPipecatcallMessages`, part 2: replace
      `emit_info_card` tool-call entries' `Function.Arguments` with a
      placeholder (e.g. `{}`) in a COPIED `[]message.ToolCall` (do not
      mutate `m.ToolCalls` in place) when attaching `tool_calls` to
      rebuilt history -- KEEP the tool_calls entry itself (id/type/name),
      only neuter Arguments. Defense-in-depth, not current API-pairing
      necessity: Python's `run.py` filter already drops this entry
      downstream regardless (VOIP-1460), but keeping it neutered here is
      what stays safe once VOIP-1460 is fixed and the entry starts
      reaching the LLM again (D2 fix part 2, design review rounds 5-7 --
      3rd injection point, not covered by part 1)
- [x] `models/ai/allowed_tools_test.go`: `knownReadOnly` +
      `ToolNameEmitInfoCard`, reword doc comment
- [x] Confirm `AllowedToolNames` derives from `AllInsightToolNames` (no dup
      list to update) -- confirmed by reading `models/ai/tool_validation.go`;
      no code change needed, matches design doc §4.

## Phase 2 - Docs

- [x] RST: `ai_overview.rst`, `ai_struct_tool.rst`, `ai_struct_ai.rst`
- [x] Clean Sphinx rebuild (`rm -rf build && python3 -m sphinx -M html source build`)
      + `git add -f docsdev/build/` -- build succeeded with zero warnings/errors
      for the three edited RST files; verified `emit_info_card` appears in the
      built HTML.
- [x] OpenAPI: `AIManagerToolName` enum + `x-enum-varnames` in
      `bin-openapi-manager/openapi/openapi.yaml` -- regenerated
      `gens/models/gen.go` via `go generate ./...`; verified `bin-ai-manager`
      and `bin-api-manager` both still build against the updated types.

## Phase 3 - Tests

- [x] Unit tests for `toolHandleEmitInfoCard` (valid input, truncation,
      empty fields, missing description) -- `tool_emit_info_card_test.go`:
      `Test_toolHandleEmitInfoCard` (3 cases: full input, empty fields,
      missing description), `Test_toolHandleEmitInfoCard_Truncation` (all 5
      D1 caps: title/description/label/value length + field count),
      `Test_toolHandleEmitInfoCard_MissingTitle` (required-field rejection).
- [x] Unit test: LLM-facing return excludes Blocks, includes the title-only
      `Message` trace (not empty, not the field values), uses
      `resource_type: "card"` / `resource_id: ""` (first-turn fix) --
      `Test_emitInfoCardLLMResult` (exact 5-key map, no `blocks` key) +
      `Test_toolHandleEmitInfoCard_Message_NeverContainsFieldValues`.
- [x] Unit test: `getPipecatcallMessages` output for a card-bearing turn,
      on a *second* call simulating the next turn -- (a) tool-result
      content has no `blocks`, (b) tool-call `Arguments` is the
      placeholder not the original card JSON, (c) the tool_calls entry is
      still present with original id/type/name (defense-in-depth guard,
      not current API-pairing necessity -- see Phase 1 note above /
      VOIP-1460). This 3-part test is the design's most important
      regression guard -- parts 1+2 of the fix must BOTH be exercised.
      Added to `start_test.go`'s `Test_getPipecatcallMessages` table: a
      standalone blocks-strip case, a standalone Arguments-neuter case, a
      non-card no-op guard case, and the combined full-turn case exercising
      (a)+(b)+(c) together in one table entry.
- [x] `allowed_tools_test.go` consent-gate test passes -- confirmed via full
      `go test ./...` run (see below).
- [x] Full verification workflow: `go mod tidy && go mod vendor && go
      generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
      -- all green in `bin-ai-manager`: `go mod tidy`/`go mod vendor`
      produced no diff, `go generate ./...` no output, `go test ./...` 1410
      passed / 38 packages, `golangci-lint run` no issues.

## Phase 4 - Ship

- [x] `git fetch origin main` + conflict check -- no conflicts, branch base
      was current
- [x] Committed (VOIP-1455 title format, project-prefixed bullets, no AI
      attribution per this repo's CLAUDE.md) -- 2 commits (design doc WIP +
      implementation)
- [x] Pushed, opened PR: https://github.com/voipbin/monorepo/pull/1264
- [x] Noted in PR: frontend work is a separate PR in monorepo-javascript,
      depends on this PR's tool name/wire format (D1) landing first (or at
      least being stable) but not on deployment order (§1.1)

**PR #1264 open, awaiting human review/merge (not merged by this session
per policy -- merge requires explicit instruction).**

## Working Notes

- No Alembic migration -- Blocks embedded in existing `ai_messages.content`
  TEXT column as JSON.
- `bin-pipecat-manager` needs a rebuild+redeploy after this merges (no code
  change there) -- deployment order vs this PR is unconstrained (§1.1).
- Frontend PR (separate): allowlist fix for VOIP-1458 lands in the same PR
  as CardBlock rendering, same render site in both panels.

## Results (fill in at the end)

- What changed:
  - `bin-ai-manager/models/tool/main.go`: added `ToolNameEmitInfoCard`,
    registered it in `AllInsightToolNames`, reworded the read-only invariant
    comment.
  - `bin-ai-manager/models/message/tool.go`: added
    `FunctionCallNameEmitInfoCard`.
  - `bin-ai-manager/pkg/toolhandler/definitions.go`: added the
    `emit_info_card` tool definition (`RunLLM: true`, anti-duplicate-
    narration guard text, JSON Schema with `title`/`description`/`fields`
    and `maxLength`/`maxItems` hints).
  - `bin-ai-manager/pkg/aicallhandler/tool.go`: added `CardField`/`CardBlock`
    types, `Blocks []CardBlock \`json:"blocks,omitempty"\`` on
    `messageContent`, the `mapFunctions` dispatch entry, and the
    `emit_info_card`-only `unmarshalToolResponse` bypass (extracted into
    `emitInfoCardLLMResult`).
  - `bin-ai-manager/pkg/aicallhandler/tool_emit_info_card.go` (new): the
    `toolHandleEmitInfoCard` handler plus the size-guard constants/
    `truncateCardText` helper (handler-side truncation is the real
    enforcement per D1, not the JSON Schema hints).
  - `bin-ai-manager/pkg/aicallhandler/start.go`: `getPipecatcallMessages`
    now (part 1) strips a `role:tool` message's `blocks` key when rebuilding
    LLM history, and (part 2) neuters `emit_info_card` tool-call entries'
    `Function.Arguments` to `"{}"` in a copied slice while preserving the
    entry itself.
  - `bin-ai-manager/models/ai/allowed_tools_test.go`: added
    `tool.ToolNameEmitInfoCard` to `knownReadOnly`, reworded the doc
    comment.
  - `bin-ai-manager/pkg/aicallhandler/tool_emit_info_card_test.go` (new) and
    `bin-ai-manager/pkg/aicallhandler/start_test.go` (extended): new unit
    tests, see Phase 3.
  - `bin-api-manager/docsdev/source/ai_overview.rst`,
    `ai_struct_tool.rst`, `ai_struct_ai.rst`: documented `emit_info_card`
    (tool tables, its own reference section, `run_llm` defaults table,
    `type=insight` tool-set mentions). `bin-api-manager/docsdev/build/`:
    clean-rebuilt and force-added.
  - `bin-openapi-manager/openapi/openapi.yaml`: added `emit_info_card` /
    `AIManagerToolNameEmitInfoCard` to the `AIManagerToolName` enum +
    `x-enum-varnames`; `bin-openapi-manager/gens/models/gen.go` regenerated.
- How verified:
  - `bin-ai-manager`: full verification workflow green -- `go mod tidy`
    (no diff), `go mod vendor` (no diff), `go generate ./...` (no output),
    `go test ./...` (1410 passed, 38 packages), `golangci-lint run -v
    --timeout 5m` (no issues).
  - `bin-openapi-manager`: `go generate ./...` regenerated `gen.go`
    correctly (verified `AIManagerToolNameEmitInfoCard` present); `go build
    ./...` succeeds (no test files in this service, matches its CLAUDE.md).
  - Cross-service sanity: `bin-ai-manager` and `bin-api-manager` both still
    `go build ./...` cleanly against the regenerated OpenAPI types.
  - `bin-api-manager/docsdev`: clean Sphinx rebuild
    (`rm -rf build && python3 -m sphinx -M html source build`) succeeded
    with zero warnings/errors for the three edited RST files; confirmed
    `emit_info_card` appears in the built HTML for all three pages.
- Deviations from the design doc (for the next code-review round):
  - The design doc's D2 "Confirmed locally implementable" note describes an
    inline `if function.Name == ... { <build map> } else { unmarshalToolResponse(msg) }`
    branch directly in `ToolHandle`. Implemented instead as that same
    `if/else` branch calling a small extracted helper,
    `emitInfoCardLLMResult(tmpContent *messageContent) map[string]any`, so
    the bypass's exact 5-key output shape (no `blocks` key) could be unit-
    tested directly -- no existing test in this package invokes the full
    `ToolHandle` dispatch path (it requires mocking `Get`/two
    `messageHandler.Create` calls/etc.), so testing the bypass logic in
    isolation was the more consistent, lower-risk option. Behavior is
    unchanged; this is a pure extract-function refactor.
  - Everything else matches the design doc's file/line plan; no other
    deviations found during implementation.
- Not done (out of scope for this delegation, left unchecked deliberately):
  - Phase 0's "Code review loop after backend implementation" -- this pass
    was implementation + tests + docs + verification only. The review loop
    is a separate step against this diff.
  - Phase 4 (git fetch/conflict check, commit, push, PR) -- not started;
    this delegation stopped at "implementation + tests + docs + green
    verification," matching the instructions given.
- Lessons (append to tasks/lessons.md if any): none from this pass -- the
  design doc's 10-round review loop had already surfaced and resolved the
  hard problems (the 3 card-JSON injection points, the `Message` field's
  three-consumer conflict, the `resource_type`/`resource_id` precedent) before
  implementation started, so implementation proceeded without new findings
  that would change the design.
