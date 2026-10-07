# VOIP-1576 Flow Builder LLM call diagnostics

This document holds the issue analysis, the design and the implementation plan of one small change. Reviewers must verify every claim against the code (`git show origin/main:<path>`), not against this text.

## 1. Issue analysis

### Observed (post-deploy check of VOIP-1573, production)

- The Flow Builder draft step fails almost every time: 5 of 6 attempts that should produce a draft failed at about 40 seconds (`BUILDER_TIMEOUT` 4 times, `BUILDER_RESPONSE_INVALID` once). One attempt returned a message without a draft.
- The question step of the same builder answers in 2 to 4 seconds. The Assistant Builder, with the same model settings, returns a draft in about 8 seconds (2 of 2).
- Model, reasoning effort and timeout are the defaults (`gemini-3.8-flash`, `none`, 40 seconds) because no `AI_BUILDER_*` variable is set in production. The Flow output cap is the constant 8192.
- 40 seconds is `AI_BUILDER_LLM_TIMEOUT_SECONDS`, so the model call is not answering in time.

### Why the current logs cannot tell the cause

- `Chat` in `bin-ai-manager/pkg/builderhandler/flow_chat.go` logs only a result label through `fail()`. A timeout (`ErrTimeout`) and an unusable answer both reach `h.fail` with the label `llm_error` or `invalid_response`, and `mapTurnError` adds no detail for a timeout.
- `RunFlowTurn` in `flow_turn.go` returns `nil` for the result when the call times out or the provider errors, so no elapsed time, finish reason or token usage exists on those paths.
- `SendOnce` (`engine_openai_handler/send_once.go`) calls the provider exactly once without retry, so a hidden internal retry is not a cause.
- Provider error text is intentionally never logged (it may hold the prompt). `ClassifyLLMError` already reduces it to a fixed code (`timeout`, `auth`, `rate_limit`, `provider_5xx`, `provider_4xx`, `other`), but `Chat` logs that code only for the `ErrLLM` branch.

### Hypotheses the logs must separate

1. The provider hangs on the structured-output path that produces a draft (`json_schema` in `flowResponseFormat`).
2. The provider answers with an error class (429, 5xx) that is surfaced as a timeout or an invalid answer.
3. The output is large and does not finish in 40 seconds (long completion, visible in completion tokens and response length).
4. The prompt or schema is large (visible in system prompt length and schema size).

### Is it still valid, and should we proceed

Valid: the failure is reproducible in production, and the existing logs cannot discriminate between the hypotheses. The change is small, reversible, adds no new behavior and is a prerequisite of the fix. The CEO decided to add diagnostics and redeploy, and decided not to revert the front entry card.

Alternatives considered: reproducing locally with the production key (needs the secret, needs CEO approval); a new metric with an outcome label (useful later for the OQ10 timeout ratio, deferred to avoid scope growth); raising the timeout blindly (guesses, and the limit is bounded by the 55 second RPC wait).

## 2. Design

### Rule

The diagnostics carry counts, sizes, durations and fixed classes only. They never carry the customer's messages, the model's answer, the system prompt text, the schema text or any provider error text.

### Changes

1. `FlowTurnResult` (flow_turn.go) gets a `Diag` field (`FlowTurnDiag`): `Elapsed` (time.Duration of the model call), `SystemChars`, `SchemaBytes` (0 unless the JSON schema mode is used), `ResponseChars`, `HistoryMessages`, `UserTurns`, `HasCurrentDraft`, `AllowedTypes`.
2. `RunFlowTurn` returns a non-nil result for every path after the request is built, including timeout and provider error, so `Diag` and the usage (zero when the provider did not answer) exist on failure. Its error return values and sentinels are unchanged.
3. `Chat` writes exactly one Info line `The flow builder model call finished.` after `RunFlowTurn` with the fields: `outcome`, `elapsed_ms`, `finish_reason`, `prompt_tokens`, `completion_tokens`, `response_chars`, `system_chars`, `schema_bytes`, `allowed_types`, `user_turns`, `history_messages`, `has_current_draft`, `model`, `reasoning_effort`, `max_tokens`, `llm_timeout_ms`, `json_mode`. `outcome` is one of `ok`, `timeout`, `truncated`, `invalid_response`, `llm_<code>` (the `ClassifyLLMError` code), `error`.
4. The existing result labels, metrics, error mapping, response shape and API are unchanged. The existing `fail()` line stays.
5. The line is Info so it shows in production (the builder is a low-traffic admin feature, one line per model call).

### Failure behavior

If a diagnostic value cannot be computed, it is logged as 0. Logging must never change the returned error or the response.

### Tests

- A table test drives `Chat` with a fake sender for each outcome (ok with a draft, ok without a draft, deadline exceeded, truncated by length, empty choices, unparsable content, provider 429, provider 500) and asserts the one diagnostic line, its `outcome`, and that the numeric fields exist.
- A privacy test uses a sentinel string in the customer message, in the model answer and in the provider error text and asserts that no log entry of the call contains it.
- Existing tests keep passing unchanged.

## 3. Implementation plan

1. `bin-ai-manager/pkg/builderhandler/flow_turn.go`: add `FlowTurnDiag`, fill it in `RunFlowTurn`, return a non-nil result on the timeout and provider-error paths.
2. `bin-ai-manager/pkg/builderhandler/flow_chat.go`: add a helper that classifies the outcome and logs the line; call it right after `RunFlowTurn`.
3. Tests in `bin-ai-manager/pkg/builderhandler/flow_chat_test.go` (or a new `flow_diag_test.go`), named `Test_...` per `scripts/check-test-conventions.sh`.
4. `bin-ai-manager/docs/operations.md`: describe the log line and how to read it for the timeout case.
5. Verification: `go test ./bin-ai-manager/...`, `golangci-lint`, `bash scripts/check-test-conventions.sh`.
6. Deploy order: ai-manager only. After the deploy, repeat the production scenario (admin test agent, draft step) and read the log line per replica.

Rollback: revert the commit; no schema, config or API change.
