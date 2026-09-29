# MCP tool schema rejected by Gemini: issue analysis

Status: issue analysis (pre-design), revision 6. Review loop closed: rounds 1-4 REQUEST
CHANGES, rounds 5-6 APPROVE (2 consecutive); round-6 Medium/Low/Nit applied.
Scope: PR #1349 (NOJIRA-Add-mcp-tool-exposure-b2), deployed to prod ahead of merge
by the CEO for a live check.

## 1. Symptom (prod, 2026-09-29, 08:27:47 to 08:28:40 UTC)

- AI `6e391666` (type normal, `engine_model gemini.gemini-2.5-flash`, `tool_names ["all"]`,
  one whitelisted MCP server `6f539b8b...`, a GitHub-style MCP server).
- 6 user turns across 3 messaging aicalls (`72bb3ed4`, `4c96a062`, `9cd84f48`, the last
  being the one reported). None produced an assistant reply.
- Control: the same AI with 16 tools (before the MCP server was attached) worked at
  08:02 to 08:03 (prompt ~7.7k tokens, completions 3/13/71). Its earlier failures at 07:57
  to 07:58 were a separate, since-fixed "API key not valid". So key, model, and prompt size
  are ruled out.

## 2. Evidence (Loki, read-only)

- ai-manager: every turn started a fresh pipecatcall (e.g. `f16a48ca`, `fdbf95f0`). Normal.
- pipecat-manager Go: `Retrieved tools for pipecat call tool_count=61` (built-ins plus MCP).
  PR B2's transport worked as designed.
- pipecat-script-runner (Python), prod banner `Pipecat 1.4.0`: at `_stream_content`, `ERROR
  push_error_frame:697 - GoogleLLMService#N exception (pydantic/main.py:263): Unknown error
  occurred: 76 validation errors for GenerateContentConfig`. There are 6 distinct failures,
  one per turn (each emits several log records: ERROR, WARNING, continuation lines), and every
  one reports exactly 76. Reassembled from the ~40 KB runner record (the Go-side copy is
  cut at 16 KiB), the full set is:
  - 74 x `properties.<owner|repo>.x-mcp-header: Extra inputs are not permitted`, across 37
    tool declarations;
  - 1 x list `type` (`['string','number','boolean']`) at declaration 35,
    `issue_fields.items.properties.value.type`;
  - 1 x `tools.0.callable` (pydantic union-branch noise, not a real cause).
  No `enum` errors occurred in prod; the enum case below is synthetic.
- Validation fails client-side, before any network request. `pipecat/services/google/llm.py`
  (1.4.0) turns it into a non-fatal `push_error`, so the turn ends with zero tokens.
- Visibility: the Python runner logged it at ERROR. The Go side receives the RTVI `error`
  frame and logs it at DEBUG as `Unrecognized RTVI message type: error` (`runner.go`
  `receiveMessageFrameTypeMessage`, `default:` branch), so it never reaches ai-manager or
  the customer. To the customer the AI is simply silent.

## 3. Root cause (reproduced offline, no API key)

Path: ai-manager `resolveMcpOnly` puts each MCP tool's raw `inputSchema` into
`tool.Tool.Parameters` (only size and JSON-object checks, `decodeToolSchema`). Pipecat Go
appends it. Python `run.py _openai_tools_to_standard` builds
`FunctionSchema(properties=params["properties"], required=params["required"])` (top-level
`$defs`, `additionalProperties`, `$schema` are dropped here). pipecat `GeminiLLMAdapter`
converts. `GoogleLLMService` builds `google.genai.types.GenerateContentConfig(tools=...)`,
whose pydantic models are `extra="forbid"`.

Reproductions (`~/.hermes/cache/scratch/pc/`):

| probe | pipecat 1.4.0 / genai 1.75.0 (prod) | pipecat 1.12.0 / genai 2.25.0 |
|---|---|---|
| `repro.py` (x-mcp-header, list type, non-string enum) | FAIL | OK (adapter adapts, logs) |
| `probe_session_survival.py`: real `GoogleLLMService._stream_content`, stub client, built-ins only | reaches network | reaches network |
| same, built-ins plus one `x-mcp-header` tool | FAIL before network (`ValidationError`) | reaches network |

pipecat 1.4.0's `gemini_adapter.py` strips only `additionalProperties`. Because the whole
`GenerateContentConfig` is validated at once, one bad MCP tool kills the request, built-ins
included. The design's D13 property ("MCP half is best-effort, built-ins always survive")
holds at the Go RPC layer only, not at the provider layer.

Why design and review missed it: the design bounded schema size and JSON-object shape but
never addressed provider schema dialects. No test exercises a real provider SDK validator
(the Python pytest mocks all of pipecat and is not run in CI; Go tests stop at the RPC).

## 4. Does upgrading pipecat help? (checked, not assumed)

- Upstream CHANGELOG 1.8.0 (2026-08-26), PR #4939, makes the Gemini adapter drop `x-*`
  keys, convert a list `type` to `anyOf`, and drop a non-string `enum`, and names GitHub
  MCP's `x-mcp-header` explicitly. 1.4.0 to 1.7.0 lack it (grep of each wheel); 1.8.0 onward
  have it. So the exact prod failure is fixed by pipecat >= 1.8.0.
- It does not fix the class. `probe_constructs.py` on 1.12.0 still FAILs on `$ref` (dangling,
  since run.py drops `$defs`), `const`, `oneOf`, `examples`, all common in Pydantic/FastMCP-
  generated schemas. Upstream is a denylist: any construct it does not know still kills the
  whole session.
- Version provenance (corrected): prod is NOT controlled by `uv.lock`. The Dockerfile
  pip-installs `requirements.txt` (`pipecat-ai[...]>=1.4.0,<2.0`, `pipecat-ai-flows>=1.2.0,<2.0`).
  The latest `pipecat-ai-flows` (1.4.0) requires `pipecat-ai>=1.4.0,<1.5.0`; flows 1.3.0
  allows `<2` (PyPI metadata, checked). With the unchanged `requirements.txt`, pip 26 on
  Python 3.12 picks flows 1.4.0 + pipecat-ai 1.4.0 + google-genai 1.75.0 (what prod runs),
  while `uv pip compile` picks flows 1.3.0 + pipecat-ai 1.12.0 + google-genai 2.25.0. The
  effective prod version therefore depends on the resolver and is held back only by a
  third-party flows constraint. That is a latent reproducibility risk on its own.
- Upgrade cost: 8 minor releases. The probe (upgrade skill script) on 1.8.0 and 1.12.0 imports
  every runner module (exit 0). Required migration: `pipecat-ai-flows` is bundled as
  `pipecat.flows` since 1.5.0, so `from pipecat_flows import ...` (`run.py:51`,
  `team_flow.py:8`) must move and the standalone package be dropped from
  `requirements.txt`/`pyproject.toml`/`uv.lock` (it logs an ERROR at import otherwise). The
  unit tests mock pipecat, so a live smoke matrix is mandatory. This is a separate, larger
  risk surface than this bug.
- OpenAI/Grok: `OpenAILLMAdapter` passes the MCP schema through unchanged on both versions
  (`probe_openai.py`). Whether OpenAI/Grok accept these constructs is unverified (needs a live
  call). No failure has been observed on those providers.

## 5. Options

A. Go-side allowlist normalization in ai-manager, where the raw schema with `$defs` is still
   available. Rewrite each MCP input schema into a provider-neutral subset and enforce BOTH
   keys and value shapes:
   - keys kept: `type`, `description`, `properties`, `required`, `items`, `enum`, `anyOf`,
     `format`, `minimum`, `maximum`, `minItems`, `maxItems`. Everything else (`x-*`, `$schema`,
     `examples`, `title`, `default`, `additionalProperties`, `exclusiveMinimum`, `pattern`, ...)
     dropped;
   - value shapes: `type` must be one of the JSON Schema type strings (a list becomes `anyOf` of
     single types); every subschema (each `properties` value, `items`, each `anyOf` member)
     must be an object (boolean subschemas and tuple `items: [...]` are invalid);
     `required` must be a list of strings naming existing properties; `description` must be a
     string; numeric bounds must be numbers;
   - server-side-conservative rules (the genai client accepts these, but the Gemini server is
     publicly reported to reject them, so the client validator is not proof):
     `format` kept only for known-safe pairs (string: `date-time`, `enum`; integer: `int32`,
     `int64`; number: `float`, `double`), otherwise dropped (e.g. `uri`, `email`); `enum` kept
     only when `type` is `string` and every member is a string, otherwise dropped (`format:
     enum` itself is kept only when an `enum` is present);
   - free-form objects (nested `object` with absent or empty `properties`, including one that
     becomes empty after its own properties are dropped) are KEPT, normalized to exactly
     `{type: object, description}` with no `properties` key. This is the shape prod already
     proves Gemini accepts: built-in `set_variables.variables` (required) and
     `create_call.variables` / `actions[].option` are `{type: object, additionalProperties: ...}`;
     pipecat 1.4.0 strips `additionalProperties` and genai emits `{"type":"OBJECT"}`
     (`rv4_probe_builtins.py`), and those tools were in the 08:02 control request on
     gemini-2.5-flash, which returned normal completions (Loki: `Received 16 tools`, prompt
     tokens 7715/7725/7755). Built-in `create_call.actions[].option` is already plain
     `{type: object, description}`, exactly the normalized target shape. The in-house proof
     covers gemini-2.5-flash only; the other offered Gemini models (`ai/main.go`: 2.5-pro,
     2.0-flash, pro-latest) are unverified and unused in the last 7 days. Public reports of
     `properties: should be non-empty for OBJECT type` (Google AI forum 34581, 64086, the
     latter with a follow-up on gemini-2.5-flash-lite-preview) exist. The keep rule adds no
     new risk class, however: built-ins send this exact `{type: OBJECT}` shape to any Gemini
     model whose AI carries `set_variables` or `create_call`, through the same adapter path,
     so a model that rejected it would already fail for those AIs today. It is new exposure
     only for an AI on another model whose tool list excludes both; that counterfactual is
     untested (no prod traffic on those models). An
     explicit empty `properties: {}` on a nested
     object is not proven in-house, so it is stripped to match the proven shape. The
     top-level parameters object with empty `properties` is kept as is (built-ins
     `stop_flow`/`stop_service` send it; also in the control);
   - an `array` without `items` is kept conservative: nothing in-house shows it accepted
     (all 7 array built-ins carry `items`), and the `items: missing field` rejection is quoted
     in forum 34581. Such a
     property is dropped when optional and the tool is dropped when it is required. It is not
     re-typed as a JSON string, because call arguments go to the MCP server raw.
   - unusable-subschema cascade (single rule, local semantics). A subschema is unusable when
     it is a typeless schema with nothing to infer, an `array` without `items`, an
     unresolvable or cyclic `$ref`, a multi-member `allOf`, or an `anyOf` left with no usable
     member. Handling, applied bottom-up: an unusable `anyOf` member is removed from its
     `anyOf`; an unusable `items` makes its array unusable; an unusable property is removed
     from its object when that object does not list it in `required`, and makes the object
     itself unusable when it does. Required-ness therefore propagates only up an unbroken
     `required` chain, and the tool is dropped only when the top-level parameters object
     becomes unusable. `required` is pruned to the surviving properties;
   - sources for the server-side rules: google-gemini/gemini-cli#2237 (400 `only 'enum' and
     'date-time' are supported for STRING type`, gemini-2.5-pro, 2025-06) and a Google AI
     developer-forum report (`enum: only allowed for STRING type`). Whether today's API still
     enforces these is not verified (no paid calls); the rules are chosen to be safe either
     way. The live call in section 7 checks the kept shapes;
   - missing `type`: inferred when unambiguous (`properties` present: `object`; `items`
     present: `array`; an all-string `enum`: `string`, which repairs the common Zod shape
     `{enum: [...]}` instead of losing the constraint; `anyOf` present: left as the `anyOf`).
     A subschema with no `type` and nothing to infer from ("any JSON value", e.g. GitHub MCP's
     projects `value`) is unusable (cascade rule above);
   - conversions: `oneOf` to `anyOf`; `const` string to single-value `enum` with `type: string`
     (non-string: drop); a single-member `allOf` is merged into its parent (the common Pydantic
     wrapper around `$ref`), a multi-member `allOf` is unusable (cascade rule above); nullability expressed as
     `anyOf` with `{"type":"null"}` (probe-verified on genai 1.75 and 2.25), not the OpenAPI-only
     `nullable`;
   - `$ref` inlined from the raw schema's `$defs`/`definitions`, depth- and cycle-bounded
     (failures are unusable, cascade rule above);
   - a tool whose schema cannot be normalized is dropped and logged; the rest survive.
   Impact measured against the GitHub MCP server source (`rv4_ghmcp_impact.py`, commit
   85598ba snapshots, with the earlier drop-free-form rule): the incident server's 44
   snapshotted tools lose nothing; across all 125 GitHub MCP tools 2 were dropped and 2 lost
   a property. With the free-form keep rule and the local cascade above
   (`rv6_local_cascade.py`): 0 tools dropped, 1 property lost (`projects_write.updated_field`,
   optional at top level; its `value` is typeless inside every `oneOf` member), and all 125
   tools pass genai 1.75. (An any-depth reading, `rv5_ghmcp_keep_freeform.py`, would instead
   drop `projects_write` entirely; the local reading is chosen because it keeps more tools.)
   The local snapshots
   carry no `x-mcp-header`; Loki is the only evidence for that construct.
   Only the advertised schema is rewritten; call arguments still go raw to the MCP server,
   whose own validation plus B24 (`isError`) handles any constraint loss. Pros: allowlist, so
   unknown constructs cannot reach any provider; one place, provider-agnostic, Go-tested,
   independent of pipecat version. Cons: loses some constraints; the subset must be proven
   acceptable to Gemini (offline via the genai validator), OpenAI and Grok (live).
B. Upgrade pipecat to >= 1.8.0 (latest 1.12.0). Fixes the reported case, not the class. Needs
   its own design, pin work, flows migration, and live smoke matrix.
C. Python sanitizer in `run.py _openai_tools_to_standard`. `$defs` are already lost there;
   Python pytest is not run in CI and mocks pipecat. Duplicates upstream's adapter.
D. Raw schema via Gemini `parameters_json_schema` through pipecat's
   `custom_tools[AdapterType.GEMINI]` bypass. Client-side validation passes on 1.4.0 and
   1.12.0 (`probe_jsonschema.py`); server acceptance and pipecat dispatch for custom tools
   are unverified; Gemini-only.
E. Per-tool isolation on the Gemini path (Python). Hook: `run.py create_llm_service`, `gemini`
   branch, immediately after `standard_tools = _openai_tools_to_standard(tools)` and before
   `ToolsSchema(...)`/`LLMContext(...)`. For each `FunctionSchema`, run the installed
   `GeminiLLMAdapter().to_provider_tools_format(ToolsSchema([fs]))`, then
   `GenerateContentConfig(tools=...)`, keep the ones that pass, drop and log the rest. No
   pipecat fork. Tools are fixed once per session via `LLMContext`, so a single check at
   build time suffices, and using the installed adapter makes E track upstream adapter
   changes automatically. Verified on pipecat 1.4.0 (`rv2_probe_e_hook.py`): good plus bad
   tool fails as a set; the filter keeps only the good one and the filtered set validates.
   Limit: E catches only client-side pydantic rejection. A server-side 400 (e.g. the `format`
   and `enum` cases in A) still fails the whole request, built-ins included, and E cannot see
   it. That residual risk is what A's conservative rules and the pre-merge live Gemini call
   target.

## 6. Recommendation (revised)

- A plus E in PR #1349. A is the fix (removes the class for every provider, preserves tools).
  E is defense in depth on the Gemini path against client-side rejection of anything A
  misses. Server-side rejection remains a residual risk covered only by A's conservative
  rules and the live call before merge.
- Also in PR #1349: make the Go side log the RTVI `error` frame at WARN (today DEBUG) so the
  failure is visible where the rest of the session is logged. A metric is optional.
- B as a separate track (CEO approved the split). Reframe it as: the runner version floats
  and is held back only by `pipecat-ai-flows`' `<1.5.0` cap; pin it deliberately, migrate
  flows to `pipecat.flows`, and bring the runner current. It also closes the reported case
  a second way.
- D rejected for now (unproven server-side, Gemini-only). C rejected (wrong layer; `$defs`
  lost).
- Mitigation until fixed: `MCP_TOOL_EXPOSURE_ENABLED=false` on ai-manager, or clear
  `mcp_server_ids` on the test AI. Only AIs with MCP servers are affected.

## 7. Scope notes

- Team AIcalls do not receive MCP tools (`runner.go` only calls the RPC on the single-AI
  aicall branch; `team_flow.py` builds its own tools), so team flows are out of scope.
- Offline validation against the genai client is necessary but not sufficient: the client
  accepts constructs the Gemini server is reported to reject (section 5A). Before merge, one
  live Gemini call with the normalized real fixture (the GitHub-style server's tools,
  including the list `type` case, plus a synthetic tool carrying a nested free-form object
  in the normalized `{type: object}` shape, an explicit nested `properties: {}` (to decide
  whether stripping it is needed), a nullable `anyOf` with `{type: null}` (the Pydantic
  `Optional` shape, client-verified only), and a typeless string enum, to confirm the rules above
  produce a request the server accepts) is required, plus one live call each for OpenAI and Grok,
  whose acceptance of the normalized schema is not verified offline at all.
