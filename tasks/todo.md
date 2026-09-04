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
- [ ] Code review loop after backend implementation (min 3, until 2 consecutive approvals)

## Phase 1 - Backend implementation (bin-ai-manager)

- [ ] `models/tool/main.go`: `ToolNameEmitInfoCard`, add to `AllInsightToolNames`,
      reword read-only invariant comment
- [ ] `pkg/toolhandler/definitions.go`: tool definition (RunLLM:true + guard
      description, JSON Schema params per D1)
- [ ] `models/message/tool.go` (or wherever FunctionCallName lives):
      `FunctionCallNameEmitInfoCard`
- [ ] `pkg/aicallhandler/tool.go`: `CardField`/`CardBlock` types,
      `Blocks omitempty` on `messageContent`, dispatch map entry
- [ ] New `pkg/aicallhandler/tool_emit_info_card.go`: handler, validation/
      truncation per D1 caps
- [ ] Bypass `unmarshalToolResponse` for this tool only (D2, first-turn fix)
      -- verify isolated branch, no other tool affected
- [ ] `start.go:637-657` `getPipecatcallMessages`, part 1: strip `blocks`
      from `role: tool` message content when rebuilding LLM history (D2
      fix part 1, design review round 3; fall back to original string on
      unmarshal error; no-op for every other tool)
- [ ] `start.go:637-657` `getPipecatcallMessages`, part 2: replace
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
- [ ] `models/ai/allowed_tools_test.go`: `knownReadOnly` +
      `ToolNameEmitInfoCard`, reword doc comment
- [ ] Confirm `AllowedToolNames` derives from `AllInsightToolNames` (no dup
      list to update)

## Phase 2 - Docs

- [ ] RST: `ai_overview.rst`, `ai_struct_tool.rst`, `ai_struct_ai.rst`
- [ ] Clean Sphinx rebuild (`rm -rf build && python3 -m sphinx -M html source build`)
      + `git add -f docsdev/build/`
- [ ] OpenAPI: `AIManagerToolName` enum + `x-enum-varnames` in
      `bin-openapi-manager/openapi/openapi.yaml`

## Phase 3 - Tests

- [ ] Unit tests for `toolHandleEmitInfoCard` (valid input, truncation,
      empty fields, missing description)
- [ ] Unit test: LLM-facing return excludes Blocks, includes the title-only
      `Message` trace (not empty, not the field values), uses
      `resource_type: "card"` / `resource_id: ""` (first-turn fix)
- [ ] Unit test: `getPipecatcallMessages` output for a card-bearing turn,
      on a *second* call simulating the next turn -- (a) tool-result
      content has no `blocks`, (b) tool-call `Arguments` is the
      placeholder not the original card JSON, (c) the tool_calls entry is
      still present with original id/type/name (defense-in-depth guard,
      not current API-pairing necessity -- see Phase 1 note above /
      VOIP-1460). This 3-part test is the design's most important
      regression guard -- parts 1+2 of the fix must BOTH be exercised.
- [ ] `allowed_tools_test.go` consent-gate test passes
- [ ] Full verification workflow: `go mod tidy && go mod vendor && go
      generate ./... && go test ./... && golangci-lint run -v --timeout 5m`

## Phase 4 - Ship

- [ ] `git fetch origin main` + conflict check
- [ ] Commit (VOIP-1455 title format, project-prefixed bullets, no AI
      attribution per this repo's CLAUDE.md)
- [ ] Push, open PR
- [ ] Note in PR: frontend work is a separate PR in monorepo-javascript,
      depends on this PR's tool name/wire format (D1) landing first (or at
      least being stable) but not on deployment order (§1.1)

## Working Notes

- No Alembic migration -- Blocks embedded in existing `ai_messages.content`
  TEXT column as JSON.
- `bin-pipecat-manager` needs a rebuild+redeploy after this merges (no code
  change there) -- deployment order vs this PR is unconstrained (§1.1).
- Frontend PR (separate): allowlist fix for VOIP-1458 lands in the same PR
  as CardBlock rendering, same render site in both panels.

## Results (fill in at the end)

- What changed:
- How verified:
- Lessons (append to tasks/lessons.md if any):
