# MCP tools on realtime voice calls: documentation wording (design)

## Problem

The public docs and OpenAPI descriptions say realtime voice call sessions do not receive MCP tools. The code does not exclude voice calls:

- `bin-api-manager` starts a single-AI voice call through `startAIcallByRealtime` -> `startPipecatcall`.
- `bin-pipecat-manager/pkg/pipecatcallhandler/runner.go` appends the result of `AIV1AIcallToolList` for every non-team AIcall, with no voice/text branch.
- `bin-ai-manager/pkg/aicallhandler/mcp_tool.go` `ResolveMcpTools` returns nil only when the exposure flag is off or the AIcall is team-typed.

Voice calls receive the whitelisted MCP tools of a single AI. Team-typed AI calls do not. The voice exclusion is stale wording from an earlier phase.

## Change (wording only, no code or behavior change)

Remove the voice exclusion, keep the team exclusion, and add "voice calls" to the list of sessions that discover and advertise tools. The list uses the words "voice calls" so that the stale-phrase grep below stays at 0. The scope of usable servers is limited per AI by `mcp_server_ids`.

| File | Location |
|---|---|
| `bin-api-manager/docsdev/source/ai_struct_mcpserver.rst` | tool use scope note, status table `active` row |
| `bin-api-manager/docsdev/source/ai_struct_ai.rst` | `mcp_server_ids` field |
| `bin-api-manager/docsdev/source/ai_overview.rst` | MCP note |
| `bin-openapi-manager/openapi/openapi.yaml` | `mcp_server_ids` description, `McpServer` description |

Generated artifacts are regenerated, not hand edited: `go generate` in `bin-openapi-manager` (`gens/models/gen.go`) and then `bin-api-manager` (`gens/openapi_server/gen.go`, `gens/openapi_redoc/openapi.json`, `api.html`; the redoc step needs `npx` and silently skips without it, so check the result with grep) and `rm -rf build && python3 -m sphinx -M html source build` with `git add -f docsdev/build/`.

No latency figures are added to the docs. Left unchanged: `bin-ai-manager/docs/domain.md`, `openapi/paths/mcpservers/main.yaml`, `ai_struct_mcpserver.rst` line 111 area, `aicall_struct_aicall.rst`, `bin-mcp-manager`, `tool_names` text.

## Verification

1. `grep -rniE "realtime voice call|voice call sessions|not realtime voice" bin-api-manager bin-openapi-manager` returns 0 and the same check on newline-joined text (`tr '\n' ' '`) to catch wrapped lines such as `openapi.yaml` `McpServer` in sources, generated files and build output (including `searchindex.js` and `api.html`).
2. `grep -rniE "team-typed|team AI"` still matches at each edited location, and each edited sentence reads correctly with only the team subject left (for example "do not either", "never receive").
3. `git diff --stat` shows only the files above and their generated counterparts; generated diffs contain only lines tied to this wording, apart from tracked binary `build/doctrees/*.doctree` files that Sphinx rewrites for the three edited pages.
4. `go generate`, `go build ./...` and tests in both services pass.
