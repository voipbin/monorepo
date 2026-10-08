# VOIP-1577 Flow Builder returns no draft and runs away

This document holds the issue analysis, the design and the implementation plan of one small change. Reviewers must verify every claim against the code (`git show origin/main:<path>`), not against this text. The measured numbers come from throwaway probes whose raw output is quoted below; the probe file itself is not part of the change.

## 1. Issue analysis

### Observed (production, after VOIP-1576 was deployed)

- Nine real `POST /v1.0/flow_builder/chat` calls by a temporary admin agent, with prompts that ask for a draft. The new diagnostic line (`The flow builder model call finished.`) and the client-side response were compared per call.
- In every `ok` call (7 of 7) the response had only the key `message`. `has_draft`, `draft_discarded` and `empty_draft` were all false, so the model sent no draft at all. Two bodies were read: both say "I have created the draft" but carry no draft.
- Some `message` values are huge: 14,629 chars (1,905 completion tokens, 14.9 s) and 30,593 chars (35 s) whose tail repeats filler words ("freely smoothly effortlessly easily ..."). One call took 40.0 s with 0 tokens (`outcome=timeout`), one 26.3 s with `finish_reason=other`, 0 tokens, `invalid_kind=unparsable`.

### Local reproduction (same model `gemini-3.8-flash`, `reasoning_effort=none`, the production `RunFlowTurn`, `FlowConfig(DefaultConfig())`, 90 s timeout; a developer key, not the production key; 4 prompts x 3 repetitions = 12 calls per cell)

| allowed types | response_format | drafts returned (graph present) | `finish_reason=length` (8192 tokens, 48 to 60 s) |
|---|---|---|---|
| 10 | `json_schema` (what production sends) | 0 of 12 | 3 of 12 |
| 10 | `json_object` | 12 of 12 | 0 |
| 10 | none | 11 of 12 | 0 |
| 30 (every type the server exposes) | `json_schema` | 0 of 12 | 7 of 12 |
| 30 | `json_object` | 12 of 12 | 0 |

In `json_object` mode every call ended with `finish_reason=stop` in 2.4 to 6.6 s.

Second probe, for the product rules (same model and settings, 12 allowed types, `FlowConfig(DefaultConfig())`, 3 repetitions per cell, 30 calls per mode in total with the first probe's cells excluded):

| scenario | `json_schema` | `json_object` |
|---|---|---|
| ambiguous first turn, English ("I want a phone flow for my pizza shop.") | no draft, a question (3 of 3) | no draft, a question (3 of 3) |
| ambiguous first turn, Korean | no draft (3 of 3) | no draft (3 of 3) |
| 3-turn conversation ending with the fixed phrase "Please create the draft with the information so far." | draft 0 of 3, 2 runaways at the cap (46 s, 53 s) | draft 3 of 3, 4.0 to 5.8 s |
| same with the Korean fixed phrase | draft 0 of 3 | draft 3 of 3, 6.2 to 6.6 s |
| change request with a current draft | draft 0 of 3, 2 runaways at the cap (47 s, 60 s) | draft 3 of 3, 2.5 to 4.2 s |

So `json_object` keeps the rule that no draft is written on an ambiguous first turn (the early-draft rule at `flow_prompt.go:161` holds in both modes), and it returns drafts for the fixed phrases and for change requests, where `json_schema` returned none.

### Code facts

- `DefaultConfig()` sets `JSONMode: JSONModeSchema` (`config.go:64`). `FlowConfig` (defined at `flow_turn.go:29`, called at `flow_main.go:91`) only raises `MaxOutputTokens` and clears `SystemPrompt`, so the Flow Builder sends `json_schema` with `Strict:false` built by `FlowResponseSchema(allowed)` (`flow_schema.go`, `flowResponseFormat` in `flow_turn.go`). `JSONMode` is not an environment setting.
- The system prompt already states the answer shape in words (`flow_prompt.go:159-163`: `message`, optional `draft` with `nodes`, `assumptions`, plus two worked examples), and `FlowParse` plus `AssembleFlowDraft` validate everything deterministically (types against the allowed set, options, labels), so the code does not rely on the schema for correctness. The comment at `config.go:10-12` says Parse "still validates everything".
- The Assistant Builder uses its own `Config` and is not affected by a change to `FlowConfig`.

### Conclusion and what is not known

- The cause in production is the `json_schema` response format of the Flow Builder: with it the provider returns no draft and, in a large share of calls, generates until the 8192 token cap, which takes 48 s or more and therefore hits the 40 s timeout (a plain timeout shows 0 tokens because no answer arrives). Without the schema (`json_object` or none) the same model, prompts, prompt text and parser give a draft every time.
- Why the provider degrades under this schema (for example the large `enum` of node types, nesting, or its handling of non-strict schemas) is not established and does not need to be for the fix. The two local runs used a developer key; production showed the same symptoms with its own key, so it is the response format, not the key.
- The probe prompts say "draft it now". The product prompt asks the model to draft only after approval or a fixed phrase (`flow_prompt.go:161`), so a message-only answer can be legitimate on a first turn. This does not explain the difference between modes (same prompts, 0 of 12 against 12 of 12), the runaway generations, or the timeouts.
- Not covered by the measurement: longer conversations (more than 3 turns), the quality of the drafts (only presence, finish reason and time were measured), the other prompt rules, and the production key. The fix is validated by repeating the production scenario after deploy (section 3).
- Production settings are taken from the VOIP-1576 diagnostic lines of the production check, not assumed: `model=gemini-3.8-flash`, `reasoning_effort=none`, `max_tokens=8192`, `llm_timeout_ms=40000`.
- The Assistant Builder uses the same `json_schema` path (`turn.go` `responseFormat`) with its own, smaller schema and returned drafts in about 8 s in production. Whether it degrades under some inputs is not measured and is out of scope here; it is a possible follow-up if it is ever seen.

### Proceed?

Yes. The feature does not deliver a draft in production, the fix is one line of configuration in the Flow path, it is reversible, and the alternatives are worse: raising the timeout (50 s at most) does not help a runaway generation; lowering the output cap hides the missing draft; rewriting the schema is a guess at a provider behaviour that is not understood.

## 2. Design

- `FlowConfig` (`flow_turn.go:29`) sets `JSONMode = JSONModeObject`. Nothing else changes: prompt, parser, assembler, error mapping, metrics, timeout, output cap, API and frontend stay as they are. No new setting or flag is added.
- The schema is therefore no longer sent for the Flow Builder. `FlowResponseSchema`, `flowResponseFormat` and the other modes stay, because the evaluation tooling selects a mode by name and other callers of `Config` still use `JSONModeSchema`.
- The diagnostic line stays correct: `json_mode` reads `object` and `schema_bytes` reads 0 (the schema is built only for the schema mode, `flowUsesSchema`). `operations.md` already derives these from the mode; the section gets one sentence about why `json_mode=object` is expected.
- Three comments that become false are corrected: `config.go:10-12` (the `JSONModeSchema` constant), `config.go:35-36` ("production uses JSONModeSchema") and the `FlowConfig` doc comment at `flow_turn.go:25-28` ("same ... JSON mode"). The `FlowConfig` comment states the mode and why (measured, ticket VOIP-1577).
- Risk: `json_object` guarantees valid JSON but not the field shape, so a malformed answer ends as `BUILDER_RESPONSE_INVALID` instead of being steered by a schema. In the local runs there were 0 parse failures in 24 `json_object` calls. `FlowParse` tolerates a code fence or prose around the object, and the parser and assembler are the existing safety net.
- Rollback: revert the commit and redeploy ai-manager (CI build and Komodo deploy). No data or API change.

## 3. Implementation plan

1. `bin-ai-manager/pkg/builderhandler/flow_turn.go`: in `FlowConfig`, set `c.JSONMode = JSONModeObject`, and fix the `FlowConfig` doc comment (`:25-28`). `config.go`: fix the comments at `:10-12` and `:35-36`.
2. Tests that depend on the old default. Every test that reaches `FlowConfig` is checked, not only the ones listed:
   - `flow_diag_test.go:216-217` (`Test_FlowChat_modelCallDiagnostic`, through `newFlowTestHandler` and `NewBuilderHandlers`) expects `json_mode == "schema"`; it expects `object` after the change, and the real path decides it (the test does not set the mode).
   - `Test_RunFlowTurn_diag` (`:430` and the other uses of `FlowConfig(testCfg())` at `:439`, `:448`, `:460`, `:507`, `:558`, `:712`, `:740`) expects `SchemaBytes > 0`; where a schema is needed the test sets `JSONModeSchema` explicitly. The per-mode request format test already sets the mode itself.
   - New test: a `Chat` call through the real handler with a recording sender asserts that the request carried `response_format` of type `json_object` with no `JSONSchema`, and that the diagnostic line says `json_mode=object` and `schema_bytes=0`. That is the contract; a comparison of two constants is not added, and the existing `DefaultConfig().JSONMode == JSONModeSchema` check in `turn_test.go:340` already keeps the Assistant Builder on the schema.
3. `bin-ai-manager/docs/operations.md`: in the Flow Builder section, state that the Flow Builder sends `json_object` and why, and that `json_mode=object` and `schema_bytes=0` are the expected values.
4. Gates: `cd bin-ai-manager && go test ./... && golangci-lint run`; from the repository root, after committing and fetching `origin/main`, `bash scripts/check-test-conventions.sh`.
5. Verification after deploy (read-only plus the minimum LLM calls of the earlier check, a temporary admin that is deleted afterwards, no calls, SMS, email or number purchases): repeat the nine-call scenario and compare the diagnostic lines. Add the three product scenarios of the second probe (an ambiguous first turn, a conversation ending with the fixed phrase, a change request with a current draft). Expected: `json_mode=object`, `schema_bytes=0`, `finish_reason=stop`, `outcome=ok`, no `timeout`, no `truncated`; `has_draft=true` for the fixed phrase and the change request, `has_draft=false` with a question for the ambiguous first turn. If a draft is still missing for explicit requests, the next suspect is the prompt rule at `flow_prompt.go:161`, not the response format.
6. The change touches only `bin-ai-manager/`, so CI builds and deploys only ai-manager; the release approval after merge belongs to the CEO.
