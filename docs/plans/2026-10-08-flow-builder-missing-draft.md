# VOIP-1577 Flow Builder returns no draft and runs away

This document holds the issue analysis, the design and the implementation plan of one small change. Reviewers must verify every claim against the code (`git show origin/main:<path>`), not against this text. The measured numbers come from throwaway probes whose raw output is quoted below; the probe file itself is not part of the change.

## 1. Issue analysis

### Observed (production, after VOIP-1576 was deployed)

- Nine real `POST /v1.0/flow_builder/chat` calls by a temporary admin agent, with prompts that ask for a draft. The new diagnostic line (`The flow builder model call finished.`) and the client-side response were compared per call.
- In every `ok` call (7 of 7) the response had only the key `message`. `has_draft`, `draft_discarded` and `empty_draft` were all false, so the model sent no draft at all. Two bodies were read: both say "I have created the draft" but carry no draft.
- Some `message` values are huge: 14,629 chars (1,905 completion tokens, 14.9 s) and 30,593 chars (35 s) whose tail repeats filler words ("freely smoothly effortlessly easily ..."). One call took 40.0 s with 0 tokens (`outcome=timeout`), one 26.3 s with `finish_reason=other`, 0 tokens, `invalid_kind=unparsable`.

### Local reproduction (same model `gemini-3.8-flash`, `reasoning_effort=none`, the production `RunFlowTurn`, `FlowConfig(DefaultConfig())`, 90 s timeout; a developer key, not the production key; 4 prompts x 3 repetitions per mode)

| allowed types | response_format | drafts returned (graph present) | `finish_reason=length` (8192 tokens, 48 to 60 s) |
|---|---|---|---|
| 10 | `json_schema` (what production sends) | 0 of 12 | 3 of 12 |
| 10 | `json_object` | 12 of 12 | 0 |
| 10 | none | 11 of 12 | 0 |
| 30 (every type the server exposes) | `json_schema` | 0 of 12 | 7 of 12 |
| 30 | `json_object` | 12 of 12 | 0 |

In `json_object` mode every call ended with `finish_reason=stop` in 2.4 to 6.6 s.

### Code facts

- `DefaultConfig()` sets `JSONMode: JSONModeSchema` (`config.go:64`). `FlowConfig` (`flow_main.go`) only raises `MaxOutputTokens` and clears `SystemPrompt`, so the Flow Builder sends `json_schema` with `Strict:false` built by `FlowResponseSchema(allowed)` (`flow_schema.go`, `flowResponseFormat` in `flow_turn.go`). `JSONMode` is not an environment setting.
- The system prompt already states the answer shape in words (`flow_prompt.go:159-163`: `message`, optional `draft` with `nodes`, `assumptions`, plus two worked examples), and `FlowParse` plus `AssembleFlowDraft` validate everything deterministically (types against the allowed set, options, labels), so the code does not rely on the schema for correctness. The comment at `config.go:10-12` says Parse "still validates everything".
- The Assistant Builder uses its own `Config` and is not affected by a change to `FlowConfig`.

### Conclusion and what is not known

- The cause in production is the `json_schema` response format of the Flow Builder: with it the provider returns no draft and, in a large share of calls, generates until the 8192 token cap, which takes 48 s or more and therefore hits the 40 s timeout (a plain timeout shows 0 tokens because no answer arrives). Without the schema (`json_object` or none) the same model, prompts, prompt text and parser give a draft every time.
- Why the provider degrades under this schema (for example the large `enum` of node types, nesting, or its handling of non-strict schemas) is not established and does not need to be for the fix. The two local runs used a developer key; production showed the same symptoms with its own key, so it is the response format, not the key.
- The probe prompts say "draft it now". The product prompt asks the model to draft only after approval or a fixed phrase (`flow_prompt.go:161`), so a message-only answer can be legitimate on a first turn. This does not explain the difference between modes (same prompts, 0 of 12 against 12 of 12), the runaway generations, or the timeouts.
- Not covered by the measurement: multi-turn conversations with a current draft, the fixed phrase, Korean input, and the other prompt rules. The fix is validated by repeating the production scenario after deploy (section 3).

### Proceed?

Yes. The feature does not deliver a draft in production, the fix is one line of configuration in the Flow path, it is reversible, and the alternatives are worse: raising the timeout (50 s at most) does not help a runaway generation; lowering the output cap hides the missing draft; rewriting the schema is a guess at a provider behaviour that is not understood.

## 2. Design

- `FlowConfig` sets `JSONMode = JSONModeObject`. Nothing else changes: prompt, parser, assembler, error mapping, metrics, timeout, output cap, API and frontend stay as they are. No new setting or flag is added.
- The schema is therefore no longer sent for the Flow Builder. `FlowResponseSchema`, `flowResponseFormat` and the other modes stay, because the evaluation tooling selects a mode by name and other callers of `Config` still use `JSONModeSchema`.
- The diagnostic line stays correct: `json_mode` reads `object` and `schema_bytes` reads 0 (the schema is built only for the schema mode, `flowUsesSchema`). `operations.md` already derives these from the mode; the section gets one sentence about why `json_mode=object` is expected.
- The comment at `config.go:10-12` and the comment on `FlowConfig` say which mode the Flow Builder uses and why (measured, with the reference to this ticket).
- Risk: `json_object` guarantees valid JSON but not the field shape, so a malformed answer ends as `BUILDER_RESPONSE_INVALID` instead of being steered by a schema. In the local runs there were 0 parse failures in 24 `json_object` calls. `FlowParse` tolerates a code fence or prose around the object, and the parser and assembler are the existing safety net.
- Rollback: revert the commit and redeploy ai-manager (CI build and Komodo deploy). No data or API change.

## 3. Implementation plan

1. `bin-ai-manager/pkg/builderhandler/flow_main.go`: in `FlowConfig`, set `c.JSONMode = JSONModeObject` with a comment that states the measured reason (VOIP-1577). `config.go`: adjust the `JSONModeSchema` comment so it no longer implies that it is the production mode for every builder.
2. Tests in `flow_diag_test.go` and `flow_chat_test.go` that depend on the old default:
   - `Test_RunFlowTurn_diag` uses `FlowConfig(testCfg())` and expects `SchemaBytes > 0`; it must set `JSONModeSchema` explicitly for that case.
   - A new test pins the contract: `FlowConfig(DefaultConfig()).JSONMode == JSONModeObject`, and a `Chat` call through the real handler sends `response_format` of type `json_object` (recorded by the sender fake) and writes a diagnostic line with `json_mode=object` and `schema_bytes=0`.
   - The Assistant Builder keeps `JSONModeSchema`: a test asserts `DefaultConfig().JSONMode == JSONModeSchema`.
3. `bin-ai-manager/docs/operations.md`: in the Flow Builder section, state that the Flow Builder sends `json_object` and why, and that `json_mode=object` and `schema_bytes=0` are the expected values.
4. Gates: `cd bin-ai-manager && go test ./... && golangci-lint run`; from the repository root, after committing and fetching `origin/main`, `bash scripts/check-test-conventions.sh`.
5. Verification after deploy (read-only plus the minimum LLM calls of the earlier check, a temporary admin that is deleted afterwards, no calls, SMS, email or number purchases): repeat the nine-call scenario and compare the diagnostic lines. Expected: `json_mode=object`, `finish_reason=stop`, `outcome=ok` with `has_draft=true` for explicit draft requests, no `timeout`, no `truncated`. If a draft is still missing for explicit requests, the next suspect is the prompt rule at `flow_prompt.go:161`, not the response format.
6. The change touches only `bin-ai-manager/`, so CI builds and deploys only ai-manager; the release approval after merge belongs to the CEO.
