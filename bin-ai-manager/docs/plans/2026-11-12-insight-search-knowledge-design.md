# Allow search_knowledge for Insight AI

## 1. Ticket review

- `models/tool/main.go`: `ToolNameSearchKnowledge` is only in `AllToolNames`; `AllInsightToolNames` lacks it.
- `models/ai/tool_validation.go`: `AllowedToolNames` and `ExpandAllToolNames` use `insightToolSet()` for `TypeInsight`, so an Insight AI cannot save `search_knowledge`, `["all"]` does not expand to it, and `toolhandler.GetByNames` does not expose it.
- bin-pipecat-manager `run.go` and `runner.go` build tools with `GetByNames(ai.Type, ai.ToolNames)` and drop `search_knowledge` when `RagID` is nil. Every other pipecat path uses `TypeNormal`.
- `aicallhandler.toolHandleSearchKnowledge` does not look at the AI type; it fails when `RagID` is nil and otherwise calls `RagV1RagQuery`.

## 2. Change

Add `ToolNameSearchKnowledge` to `tool.AllInsightToolNames`. The tool only reads the knowledge base (query embedding, vector search), so it belongs to the guarded list and not to a side-effect list. No new list, flag or migration.

Effect: save-time validation accepts it, `["all"]` includes it, `GetByNames(TypeInsight)` returns it, and pipecat hides it when `RagID` is nil. An existing Insight AI with `["all"]` and a `rag_id` gets the tool after deploy.

Out of scope: system prompt, tool description, listen prompt, square-admin mirror. The OpenAPI enum value already exists. Only descriptions go stale: the sentence "(15 tools in total, with the 8 other Insight tools)" (the whole parenthesis is deleted, because "8 other" also becomes wrong) appears in `bin-openapi-manager/openapi/openapi.yaml` (~2305), `openapi/paths/ais/main.yaml` (~106) and `openapi/paths/ais/id.yaml` (~147). Regenerate in order: `go generate ./...` in bin-openapi-manager, then in bin-api-manager (this also refreshes `gens/openapi_server/gen.go`, and `gens/openapi_redoc/openapi.json` and `api.html`, which are git-tracked and only regenerated when `npx` exists; check `npx` first and report if missing). Then `go build ./...` in bin-api-manager.

## 3. Files

Code: `models/tool/main.go` (list and comments).
Comments: `models/ai/tool_validation.go` and `models/tool/main.go` are updated; the comments in `models/ai/main.go`, `pkg/aicallhandler/main.go` and `pkg/teamhandler/handler.go` only say "AllInsightToolNames plus ..." and stay correct. `ai_struct_ai.rst` (tool_names and type descriptions) and the `tool_names` description in the ais path yaml files list the Insight tool names and gain search_knowledge.
Test points (the exact set is confirmed by `go test`):
- `allowed_tools_test.go`: also `Test_ExpandAllToolNames` insight case (`wantNot` built from AllToolNames minus extras), `Test_ExpandAllToolNames_InsightIsAllowedSet` (`want != 15` and its comment). Normal `wantDisallowed` must not contain search_knowledge (Normal allows it); Insight `wantDisallowed` is `withoutNames(AllToolNames, extras)` and must also drop it; count 15 -> 16; `knownReadOnly` allowlist gains it; `TestValidateToolNames_WriteToolNeverAllowedForInsight` derives its allowed set from AllInsightToolNames and needs no change.
- `bin-ai-manager/pkg/toolhandler/main_test.go`: the three Insight subtests sum four lists (AllInsight, SMS, email, call) with no overlap; `wantCount` 15 -> 16 and the "15 in total" comment.
- `definitions_resource_test.go`: search_knowledge gets its own group (RAG lookup, not covered by the run_llm wording contract); the run_llm description test is untouched.
- `bin-pipecat-manager/pkg/toolhandler/main_test.go` `Test_ToolHandler_GetByNames_InsightAllReturnsFifteenTools` (seed, `want`, `seen`): the fixture seeds the cache with AllToolNames plus AllInsightToolNames, so search_knowledge would be seeded twice; seed the de-duplicated union. 15 -> 16 and rename the function.
Files: `models/ai/allowed_tools_test.go`, `pkg/toolhandler/main_test.go`, `pkg/toolhandler/definitions_resource_test.go`, `bin-pipecat-manager/pkg/toolhandler/main_test.go`. The existing `Test_toolHandleSearchKnowledge` covers RagID nil and set and does not use the AI type (no new handler test; Insight type is not read by the handler).
Docs: `bin-ai-manager/docs/domain.md` (search_knowledge row, line ~182); `ai_struct_tool.rst` (a separate paragraph in the Insight section, not the "exclusive to type=insight" list at ~924, because the tool is shared with Normal; also the "Seven more tools" paragraph at ~941 stays correct as it counts only those seven); `ai_overview.rst` (the Insight tool note at lines ~330-339, which lists the separate smaller set, and the `["all"]` note ~799). docsdev is rebuilt clean and force-added.

## 4. Verify

1. Insight save with `search_knowledge` is accepted; Normal behaviour unchanged.
2. `ExpandAllToolNames(TypeInsight)` contains it; count is 16 (15 today: 8 + SMS 2 + email 2 + call 3; AllInsightToolNames becomes 9).
3. pipecat `GetByNames(Insight, all)` returns 16 tools (currently 15).
4. Handler with `RagID` set calls `RagV1RagQuery`; with nil fails (mock tests, no real calls).
5. Full verification workflow in bin-ai-manager and bin-pipecat-manager.

## 5. Impact

Behaviour change only for Insight AIs that already hold a `rag_id`. Cost: one internal query embedding per call, no customer billing. Insight listen turns reach tools through pipecat `GetByNames(ai.Type, ...)`, same as above. Trigger to revisit: abnormal embedding volume, in particular from Insight listen turns.
