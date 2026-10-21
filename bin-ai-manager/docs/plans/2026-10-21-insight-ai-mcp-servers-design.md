# Insight AIs use their own whitelisted MCP servers

## 1. Behavior

An Insight AI is offered the tools of the MCP servers in its own `mcp_server_ids`, in every session it runs (Q&A turns and live-call listen turns), exactly like a Normal AI. With an empty `mcp_server_ids` it has only the Insight built-in tools (`AllInsightToolNames`). Team AIcalls get no MCP tools. Servers that are deleted, inactive or owned by another customer are skipped at discovery and refused at call time, and a call is refused when the server is no longer in the AI's list. There is no read-only filter.

## 2. Code

- The saving path (`listenhandler/v1_ais.go`, `ValidateMcpServerIDs`) and pipecat-manager (`runnerStartScript` appends the `AIV1AIcallToolList` result; `ToolHandle` routes `mcp_` names to `toolHandleMcpCall`) do not look at the AI type, so no change is needed there.
- `ResolveMcpTools` and `discoverMcpTools` no longer return early for `ai.TypeInsight`. Every other consumer (`startAIcallByMessaging`, `writeInsightSessionMetadata`, `refreshMcpToolMap`, `persistToolMap`) goes through `discoverMcpTools`.
- `writeInsightSessionMetadata` runs `resolveMcpToolMap` before it re-reads the AIcall, so a metadata key written while the network discovery runs is not reverted.
- Unchanged: `mcpServerIsUsable`, the call-time whitelist check, the 256-tool cap, `mcp_tool_exposure_enabled`. No schema, migration or API shape change.

## 3. Docs

`docs/domain.md`, `docs/architecture.md`, the comment above `InsightSystemPrompt` and `models/ai/main.go`, docsdev RST (`ai_overview.rst`, `ai_struct_ai.rst`, `ai_struct_mcpserver.rst`, `aicall_struct_aicall.rst`) with the rebuilt HTML, and the OpenAPI descriptions (`openapi.yaml`, `paths/mcpservers/main.yaml`, `paths/ais/main.yaml`, `paths/ais/id.yaml`) with regenerated types.

## 4. Tests

- `Test_ResolveMcpTools_Insight`: whitelisted server advertised and stored in the tool map, empty list advertises nothing, other-customer server looked up but not listed (aicall is a contact case with a listen call).
- `Test_discoverMcpTools_Insight`: own server listed, other-customer server skipped.
- `Test_writeInsightSessionMetadata_McpToolMap`: tool map stored, key written during discovery kept.
- `Test_ToolHandle_mcpToolOnListenTurn`: a listen-turn `mcp_` call reaches the MCP server.
- `Test_ResolveMcpTools_Team` still covers the team case.
