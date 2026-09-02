# get_contact_profile implementation plan (rev3, round-1/2/3 plan review incorporated)

Design doc (finalized, 2 consecutive design-review approvals, rounds 6+7):
docs/plans/2026-09-02-insight-assistant-get-contact-profile-design.md

Goal + acceptance criteria: add a 5th read-only Insight tool,
`get_contact_profile`, exactly per the design doc §3 (implicit Case
scoping, no LLM-suppliable identifier, mandatory response-side tenant
check on the unscoped `ContactV1ContactGet`, header/lines rendering split,
own truncation note instead of `renderBodyLines`' misleading marker).
Ships in v1 with addresses (Type+Target only, capped 5), auto-activated for
all `tool_names=["all"]` Insight AIs per 대표님's explicit decision
(design doc §3.5/§5).

**Single commit**: steps 1 through **12b** (the staging+commit step) land
as ONE commit. Do not commit at any intermediate step -- the tree is
deliberately red/incomplete between step 1 (new `AllInsightToolNames`
entry) and step 6 (its `ToolDefinition`), and between step 1 and step 3
(`allowed_tools_test.go` allowlist). If the code review loop (step 13)
requires changes, add fixup commits on the same branch and re-run steps
11-12 before pushing again -- the single-commit rule governs the initial
landing, not every commit ever made on the branch (squash merge collapses
review-fix commits later, per root CLAUDE.md).

**Design doc status**: the design doc's own Status line (line 4) is stale
-- it says "Draft rev5 ... pending final confirmation round" but the
design was in fact finalized with 2 consecutive approvals at rounds 6+7.
Update that line to "Finalized (rounds 6+7 approved)" as part of this
change, so the two documents don't disagree.

## Steps (in order; each is a checkpoint)

- [ ] 1. `bin-ai-manager/models/tool/main.go`: add
      `ToolNameGetContactProfile` + append to `AllInsightToolNames` (design §3.5)
- [ ] 2. `bin-ai-manager/models/message/tool.go`: add
      `FunctionCallNameGetContactProfile` (design §3.5)
- [ ] 3. `bin-ai-manager/models/ai/allowed_tools_test.go`: add
      `tool.ToolNameGetContactProfile` to `knownReadOnly` (design §3.5 —
      this is a deliberate consent-gate test; must be updated in the same
      change, not discovered as a red test)
- [ ] 4. `bin-ai-manager/pkg/aicallhandler/tool_insight.go`: implement
      `toolHandleGetContactProfile` per design §3.0-§3.4 exactly (entry
      guard → Case get → nil-ContactID short-circuit → Contact get →
      mandatory tenant check w/ nil-safe forensic audit log → header/lines
      render w/ own truncation note, `pagedOut=false` to `renderBodyLines`).
      Includes two NEW helpers that don't exist anywhere in the repo today
      -- write them in this step: the `insightContactAddressLimit = 5`
      const, and the `contactDisplayName(contact)` fallback-name helper.
- [ ] 5. `bin-ai-manager/pkg/aicallhandler/tool.go`: wire
      `FunctionCallNameGetContactProfile` into the `mapFunctions` dispatch
      map (design §3.5)
- [ ] 6. `bin-ai-manager/pkg/toolhandler/definitions.go`: add the
      `ToolDefinition` (WHEN TO USE / WHEN NOT TO USE, `RunLLM: true`,
      no-argument `Parameters`) (design §3.5)
- [ ] 7. `bin-ai-manager/pkg/aicallhandler/tool_insight_test.go`: add all
      **10** table-driven test cases from design §4 (not 9 -- corrected in
      round-1 plan review):
      1. `ReferenceType` guard → `fillFailed`, zero RPC calls
      2. Happy path: full profile (name/company/job_title + 2 addresses)
      3. Happy path: sparse profile (fallback name composition, omitted lines)
      4. Happy path: >5 addresses → cap + own `(showing 5 of N)` note in
         header + built-in `renderBodyLines` marker text absent (round-5 regression)
      5. Case not found → masked, `ContactV1ContactGet` NOT called
      6. Case cross-tenant → masked, `ContactV1ContactGet` NOT called
      7. **`ContactID == nil`** → distinct `"no contact profile found"`
         (NOT masked), `ContactV1ContactGet` asserted `Times(0)` -- this is
         the single most security-load-bearing assertion in the set (pins
         "never call the unscoped RPC with a nil id"); do not drop it
      8. Contact not found → masked
      9. Contact cross-tenant (the mandatory §3.2 step 5 check) → masked,
         no panic
      10. `contact == nil` defensive branch → masked, no panic (distinct
          from #9's non-nil-but-wrong-tenant case)
- [ ] 7a. `bin-ai-manager/docs/domain.md`: steps 1-2 touch `models/.../*.go`,
      which the repo's `check-service-docs.sh` PostToolUse hook maps to this
      file (root CLAUDE.md service-docs-sync table). The existing "LLM
      Tools" table is already stale (omits all 4 current Insight tools) --
      do NOT scope-creep into backfilling that; either add one row for
      `get_contact_profile` or record in the PR body that no doc change was
      warranted. Do not silently ignore the hook warning either way.
- [ ] 8. `bin-openapi-manager/openapi/openapi.yaml`: add
      `get_contact_profile` to BOTH the `AIManagerToolName` enum values AND
      the parallel `x-enum-varnames` list (design §3.5 rev4 correction —
      both lists, in lockstep, or codegen produces a mismatched const name)
- [ ] 8a. Decide-and-record (flagged in round-1 plan review, not in the
      design doc): `bin-openapi-manager/openapi/paths/ais/main.yaml:86-87`
      and `id.yaml:126-127` carry a customer-visible prose sentence
      enumerating Insight tool names by name, which is ALREADY stale
      (missing `get_related_cases`/`get_case_notes`) and renders into the
      public redoc HTML this change regenerates (step 9). Either fix both
      to enumerate all 5 tools, or explicitly note in the PR body that this
      is pre-existing staleness left out of scope. Do not leave it
      unmentioned.
- [ ] 9. Regenerate derived artifacts, **bin-openapi-manager FIRST, then
      bin-api-manager** (per `bin-openapi-manager/CLAUDE.md` consumer-order
      rule): `go generate ./...` in each. Do not hand-edit generated files.
- [ ] 9a. **Inspect the generated diff before proceeding -- do not trust a
      green exit code.** `bin-api-manager/openapi/config_redoc/generate.go`
      shells out to `npx @redocly/cli`, and on any npx/network failure it
      prints "skipping generation" and **still exits 0** -- a silent no-op
      that a plain `go generate && echo ok` would miss entirely. Run:
      `git status --short bin-openapi-manager/gens bin-api-manager/gens`
      and confirm **all four** artifacts changed:
      `bin-openapi-manager/gens/models/gen.go`,
      `bin-api-manager/gens/openapi_server/gen.go`,
      `bin-api-manager/gens/openapi_redoc/openapi.json`,
      `bin-api-manager/gens/openapi_redoc/api.html`.
      If the redoc pair is unchanged, the toolchain silently no-opped --
      fix it and rerun; do not proceed with stale redoc artifacts.
      Then `git diff` the redoc pair: if the diff is large beyond the new
      enum entry, that's `@redocly/cli` version drift (it resolves latest
      at run time) -- inspect and decide whether to accept before committing.
      Note (round-1 plan review): this is an additive enum constant only,
      so no other consumer needs a rebuild beyond `bin-api-manager` (already
      covered by step 12). If `golangci-lint` in step 12 surfaces
      pre-existing findings inside `gens/` (generated code), record them,
      do not fix them -- out of scope, avoid scope creep.
- [ ] 10. RST docs (round-3 plan review: corrected file list -- one file
      was missing). Update ALL THREE files in `bin-api-manager/docsdev/source/`
      that enumerate the Insight tool set by name:
      - `ai_overview.rst` (~line 329-330).
      - `ai_struct_tool.rst` -- three separate spots, not just "list the
        new tool": the summary table (~841-844), a new per-tool section
        with its own `.. _ai-struct-tool-get_contact_profile:` anchor
        (matching the existing per-tool section pattern), and the
        `run_llm` table (~1033-1036).
      - `ai_struct_ai.rst` -- **missing from rev2, added in rev3**: two
        spots, the `tool_names` field doc (~line 67, "only Insight tool
        names are permitted (currently ...)") and the `insight` AI type
        description (~line 124, "Restricted to the Insight tool set
        (...)"). Both currently enumerate tools by name and are already
        stale (list only 2 of the current 4) -- fix to enumerate all 5,
        or explicitly record the pre-existing staleness as out-of-scope in
        the PR body (same fix-or-record framing as step 8a). Do not leave
        unmentioned -- this file's rebuilt HTML ships on docs.voipbin.net
        in the same commit that adds the 5th tool, so silence here would
        commit a customer-facing page that undercounts the tool set by one.

      Note (round-4 plan review): `ai_tutorial.rst:347` also contains an
      Insight tool name list (`"tool_names": ["get_contact_interactions",
      "get_conversation_content"]`), but it's a customer-chosen SUBSET
      inside a `curl` example payload, not an enumeration of the full tool
      set -- leave it unchanged, it doesn't go stale when a 5th tool is
      added. A grep for tool names while doing this step will surface it;
      don't edit it.

      Then clean rebuild (`rm -rf build && python3 -m sphinx -M html source build`).
      **Do NOT stage `build/` yet** -- staging happens once, at step 12b,
      after verification (staging before step 11-12 risks staging
      pre-verification/pre-format content).
- [ ] 11. Verify — run in `bin-ai-manager`:
      `go mod tidy && go mod vendor && go generate ./... && go test ./... && golangci-lint run -v --timeout 5m`
- [ ] 12. Verify — run the same 5-step workflow in `bin-openapi-manager`
      and `bin-api-manager` (schema/codegen changes touch both).
- [ ] 12a. Pull main + conflict check (root CLAUDE.md, mandatory before
      PR): `git fetch origin main` →
      `git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main | grep -E "^(CONFLICT|changed in both)"`
      → `git log --oneline HEAD..origin/main`. If conflicts exist: resolve,
      then re-run steps 11-12 in full before continuing.
- [ ] 12b. Stage everything, including
      `git add -f bin-api-manager/docsdev/build/` (root `.gitignore`
      excludes `build/`). Do NOT stage `vendor/`. Verify final diff with
      `git diff --cached` (not `git status` two-letter codes -- see
      project git-workflow lessons). Commit as a single commit titled
      `NOJIRA-Add-get-contact-profile-insight-tool`, body with
      `bin-<service>:` prefixed bullets per root CLAUDE.md commit format.
      No AI attribution.
- [ ] 13. Code review loop (min 3 rounds, 2 consecutive approvals, per
      project CLAUDE.md Review Loop Policy). **Loop ends at approval --
      report the verdict and STOP. Do not merge without explicit
      instruction.**
- [ ] 14. Push branch (`-u` if new) + create PR (title = branch name, body
      = narrative summary + `bin-<service>:` bullets, NO markdown headers,
      NO test-plan section, NO AI attribution, per root CLAUDE.md PR format).
      Report to 대표님 with the verification story; wait for explicit
      "merge" before ever running `gh pr merge`.

## Working notes / constraints carried from the design doc

- `ContactV1ContactGet` has NO tenant scoping server-side — the response
  check in step 4 above is the ONLY enforcement. Do not comment it as
  "belt-and-suspenders" in code.
- Never call `ContactV1ContactGet` with a nil/zero `ContactID`.
- `insightContactAddressLimit = 5`, NOT `insightMaxListLimit` (50) — a new,
  separate constant.
- `renderBodyLines` must be called with `pagedOut=false` always for this
  tool — the truncation note is handled in `header`, not via that flag.
- Every response path (masked or success) uses
  `fillSuccess(res, "contact_profile", c.ReferenceID.String(), body)`.
- No `tags:` line in v1 (dropped per design §6).
- Only `Type`/`Target` rendered per address — never `Detail`/`Name`.
