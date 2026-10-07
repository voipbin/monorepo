# VOIP-1576 Flow Builder LLM call diagnostics

This document holds the issue analysis, the design and the implementation plan of one small change. Reviewers must verify every claim against the code (`git show origin/main:<path>`), not against this text.

## 1. Issue analysis

### Observed (post-deploy check of VOIP-1573, production)

- The Flow Builder draft step fails almost every time: 5 of 6 attempts that should produce a draft failed at about 40 seconds (`BUILDER_TIMEOUT` 4 times, `BUILDER_RESPONSE_INVALID` once). One attempt returned a message without a draft.
- The question step of the same builder answers in 2 to 4 seconds. The Assistant Builder, with the same model settings, returns a draft in about 8 seconds (2 of 2).
- Model, reasoning effort and timeout are the defaults (`gemini-3.8-flash`, `none`, 40 seconds) because no `AI_BUILDER_*` variable is set in production. The Flow output cap is the constant 8192 (`flowDefaultMaxOutputTokens`).
- 40 seconds is `AI_BUILDER_LLM_TIMEOUT_SECONDS`, and `ErrTimeout` only comes from the handler's own `context.WithTimeout(cfg.LLMTimeout)` around `SendOnce` (`flow_turn.go`), so those four failures are the model call not answering within its own deadline. Redis counter wait is outside that deadline.
- The one `BUILDER_RESPONSE_INVALID` came back at about 40.2 seconds, and the production log of that call carries `result=invalid_response` without an `llm_error` field. So it was `ErrTruncated` or `ErrInvalidResponse`, not a provider error class. A truncation at 40 seconds would mean the model reached `max_tokens` (8192) about when the deadline fell, which fits a runaway completion; this is a hint, not proof, because the sentinel is not logged.
- The production log lines of the four timeouts show `result=llm_error` and no `llm_error` field. That shape alone is not unique to `ErrTimeout`: the caller-deadline branch (`LLMError{timeout}`) and the non-sentinel default branch of `mapTurnError` look the same. They are told apart here by the API reason `BUILDER_TIMEOUT` (api-manager can also return it after its own 55 second RPC wait, but the observed time was about 40 seconds, not 55) and by the elapsed time. The new `outcome` field separates `timeout`, `llm_timeout` and `error` explicitly.

### What the current logs and metrics tell, and what they cannot

- Available today: `promFlowBuilderChatTotal{result}` and `promFlowBuilderChatDuration` (whole `Chat`, including semaphore, Redis counter and prompt build), and one `fail()` Info line with the result label. `llm_error=<code>` is logged only in the `ErrLLM` branch of `mapTurnError`. The API reasons `BUILDER_TIMEOUT` and `BUILDER_RESPONSE_INVALID` separate a timeout from an unusable answer.
- Missing: which sentinel produced `invalid_response` (`ErrTruncated` or `ErrInvalidResponse`), `finish_reason`, token usage when the call failed (`RunFlowTurn` returns `nil` on the timeout and provider-error paths), the duration of the model call alone, and any size of the request.
- `SendOnce` (`engine_openai_handler/send_once.go`) is non-streaming and calls the provider exactly once without retry, so a hidden retry is not a cause.
- Provider error text is intentionally never logged (it may hold the prompt). `ClassifyLLMError` reduces it to a fixed code (`timeout`, `canceled`, `auth`, `rate_limit`, `provider_5xx`, `provider_4xx`, `other`).

### Hypotheses and what each diagnostic can show (honest limits)

| Hypothesis | Visible in the new log | Limit |
|---|---|---|
| H1 The provider hangs on the structured-output (`json_schema`) path that produces a draft | `outcome=timeout`, `elapsed_ms` near the deadline, zero tokens | Looks identical to H3 on a timeout |
| H2 The provider returns 429 or 5xx | `outcome=llm_rate_limit` or `llm_provider_5xx` | Cannot show a status when our own deadline fires first |
| H3 The completion is very long (runaway output) | `outcome=truncated`, `finish_reason=length`, `completion_tokens` near 8192, large `response_chars`; or a late `ok` with large tokens | On a pure timeout there is no response, so H3 and H1 cannot be told apart from the log alone |
| H4 The draft output structure (not the input size) is what the provider cannot finish | Supported by H1 or H3 evidence together with `has_draft` on the successful turns | The input is the same for the question step and the draft step apart from the conversation, so input size alone cannot explain it (system prompt and schema are constant for a given type set) |

Because H1 and H3 are not separable on a timeout, the plan includes a controlled experiment after the first read of the logs (section 3, step 6).

### Is it still valid, and should we proceed

Valid: the failure is reproducible in production, and the existing logs cannot discriminate between most of the hypotheses. The change is small, reversible, adds no behavior and is a prerequisite of the fix. The CEO decided to add diagnostics and redeploy, and decided not to revert the front entry card.

Alternatives considered: reproducing locally with the production key or the existing `builderhandler/eval` harness (needs the secret and CEO approval, kept as a later step); a new histogram with an outcome label (useful later for the OQ10 timeout ratio, deferred to avoid scope growth); raising the timeout blindly (a guess, and the limit is bounded by the 55 second RPC wait, so at most 50 seconds, see `operations.md`).

## 2. Design

### Rule

The diagnostics carry counts, sizes, durations, booleans, configured constants and fixed classes only. They never carry the customer's messages, the model's answer, the system prompt text, the schema text, the draft content or any provider error text.

### Changes

1. `FlowTurnResult` (flow_turn.go) gets a `Diag` field (`FlowTurnDiag`):
   - `Elapsed` (duration of the `SendOnce` call only, measured around that call),
   - `SystemChars` (runes of the system prompt), `RequestChars` (runes of all message contents sent, system prompt included, which also covers the current draft block and the history),
   - `SchemaBytes` (bytes of the JSON schema text, 0 unless the schema mode is used; the schema string is built once and reused for the request),
   - `ResponseChars` (runes of the answer content, 0 when none),
   - `UserTurns`, `HistoryMessages`, `AllowedTypes` (count), `CurrentDraftPresent` (`req.CurrentDraft != nil`, not whether it rebuilds to a graph; its size is inside `RequestChars`),
   - `BuildElapsed` (from the start of `RunFlowTurn` to just before `SendOnce`: system prompt, schema and message building),
   - `InvalidKind`, a fixed value set by `RunFlowTurn` at the three places that return `ErrInvalidResponse`: `nil_response` (`resp == nil`), `no_choices` (`len(resp.Choices) == 0`) and `unparsable` (`FlowParse` failed); empty otherwise. `FinishReason`, `ResponseChars` and the usage are filled before the `ErrTruncated` and parse returns.
2. `RunFlowTurn` returns a non-nil result for every path after the request is built (timeout, provider error, nil response, no choices, truncated, unparsable), so `Diag` and the usage (zero when no answer arrived) exist on failure. Its error return values and sentinels are unchanged. The two guard returns before the request is built (nil sender, nil request) still return a nil result; the rule is single: when the result is nil the logging helper is not called (a nil guard in `Chat`).
3. `Chat` emits exactly one Info line `The flow builder model call finished.` for every call that reached the model, through one helper that is called right before each of the two returns that follow `RunFlowTurn` (the error return and the final success return), so it can also carry what happened after the call (draft assembly). It is not a deferred function, so a panic never produces a false `ok` line. "Reached the model" means `RunFlowTurn` returned a non-nil result, which now happens on every path where `SendOnce` was called (the guard returns for a nil sender or request still return nil, and the six refusals before `RunFlowTurn` never log this line). The sentinel error is captured in a local variable (`errTurn`) before `mapTurnError` converts it, and classified from that local. If `RunFlowTurn` or the draft assembly after it panics, no line is written here (a panic in `AssembleFlowDraft` happens after the model answered; the existing panic handling upstream logs it), and the panic is not recovered here. The fields:
   - `outcome`: `ok`, `timeout` (the handler's own deadline, `ErrTimeout`), `truncated`, `invalid_response`, `llm_<code>` using the `ClassifyLLMError` code (`llm_timeout` is the caller's context deadline and is a different thing from `timeout`; `llm_canceled`, `llm_auth`, `llm_rate_limit`, `llm_provider_5xx`, `llm_provider_4xx`, `llm_other`), `error` for a non-sentinel error (defensive: with the nil-result rule above no current path reaches it). The two timeout kinds are not merged.
   - `invalid_kind`: for `invalid_response` only, `nil_response`, `no_choices` or `unparsable`, so the three `ErrInvalidResponse` sources are separate.
   - timing: `elapsed_ms` (the `SendOnce` call), `build_ms` (prompt, schema and message building inside `RunFlowTurn`), `pre_call_ms` (from the start of `Chat` to the call of `RunFlowTurn`: request validation and the Redis counter; the semaphore is a non-blocking select and has no wait), `chat_ms` (start of `Chat` to the helper call; it differs from `promFlowBuilderChatDuration` by microseconds).
   - `finish_reason` normalized by an allowlist (`stop`, `length`, `content_filter`, `tool_calls`, `function_call`, anything else `other`, no answer `none`).
   - `prompt_tokens`, `completion_tokens`, `response_chars`, `system_chars`, `request_chars`, `schema_bytes`, `allowed_types`, `user_turns`, `history_messages`, `current_draft_present`.
   - for the `ok` outcome, three separate booleans so "message only", "draft not decodable" and "nothing survived the filters" are told apart: `has_draft` (`resp.Draft != nil`), `draft_discarded` (the exact warning `WarnDraftDiscarded` in `Parsed.Warnings`, which `FlowParse` adds when the draft could not be decoded) and `empty_draft` (the exact `flowbuilder.WarningEmptyDraft` in the response warnings, which means a graph was decoded but `AssembleFlowDraft` kept no node or removed the start node, for example all types unsupported). Matching is by whole-string equality, never by substring, because other warnings contain model-supplied labels.
   - configuration: `model`, `reasoning_effort`, `max_tokens`, `llm_timeout_ms`, `json_mode` (a fixed string derived with the same condition as the request format: `none`, `object`, or `schema` for every other value including the empty string, so it matches what is sent).
4. The existing result labels, metrics, error mapping, response shape and API are unchanged. The existing `fail()` line stays, so a failed call logs two lines with the same `customer_id`.
5. The line is Info so it shows in production. The builder is a low-traffic admin feature; one line per model call is acceptable.

### Failure behavior

If a diagnostic value is not available it is logged as 0 or `none`. A nil result is allowed. Logging never changes the returned error or the response.

### Tests

- Exactly-one-line and boundary: for every model-reaching case below, the diagnostic line is counted and must be exactly one; for each of the six refusals before the model (no key, invalid request, no usable type, busy, counter failure, daily limit) the count must be zero.
- A table test drives `Chat` through the existing fake sender for each outcome: ok with a draft, ok without a draft, ok with a discarded draft, `context.DeadlineExceeded` from the sender (`timeout`), finish reason length (`truncated`), unparsable content, nil response, no choices (a new fake sender is needed because the existing one always returns one choice), provider 429, provider 500, and the caller's context deadline (`llm_timeout`, built with an already-expired parent context and a blocking sender; a sender that merely returns `context.DeadlineExceeded` with a live parent context is the `timeout` case, and both are tested). It asserts the single diagnostic line, `outcome`, `invalid_kind` (all three values), the `has_draft` / `draft_discarded` / `empty_draft` combination (an all-unsupported-types answer must give `empty_draft` true and `draft_discarded` false), that numeric fields are present, and that `pre_call_ms`, `build_ms` and `elapsed_ms` are plausible (non-negative, `elapsed_ms` taken from a sender that sleeps a known short time).
- Privacy: the diagnostic line's field names are checked against an allowlist (the new fields plus the existing `func`, `customer_id` and `message_count`); every value is checked by `fmt.Sprint` against sentinel strings placed in the customer message, in the model answer (including inside the draft) and in the provider error text (429 and 500). It uses the existing log-capture style of `Test_FlowChat_logsNeverCarryCustomerInput`.
- Panic boundary: a sender that panics (recovered by the test) yields zero diagnostic lines.
- `finish_reason` normalization has a table test, including a hostile provider string.
- Existing tests keep passing unchanged.

## 3. Implementation plan

1. `bin-ai-manager/pkg/builderhandler/flow_turn.go`: add `FlowTurnDiag`, fill it in `RunFlowTurn`, return a non-nil result on the timeout and provider-error paths (the nil-response and no-choices paths already return one). `flowResponseFormat(mode, allowed)` currently calls `FlowResponseSchema(allowed)` itself; change it to take the already built schema string so it is built once and `SchemaBytes` can be measured; it is built under the same condition as the `default` branch of the `flowResponseFormat` switch (every mode other than `JSONModeNone` and `JSONModeObject`), so `SchemaBytes` matches what is sent (its only caller is `RunFlowTurn`, and no test calls it directly).
2. `bin-ai-manager/pkg/builderhandler/flow_chat.go`: add the outcome classifier, the `finish_reason` normalizer and the logging helper called before the two returns after `RunFlowTurn`; capture `pre_call_ms` and the config values. `RunFlowTurn` has one caller (`flow_chat.go`), no mock or eval harness uses it, and the interface in `flow_main.go` does not change, so no mock regeneration is needed.
3. Tests in `bin-ai-manager/pkg/builderhandler/flow_chat_test.go` and a new `flow_diag_test.go`, named `Test_...` per `scripts/check-test-conventions.sh` (no testify, gomock controller named `mc`), reusing `newFlowTestHandler` and the existing sender fakes, plus new fakes for the cases the existing `chatSender` cannot produce: one returning no choices, one returning a nil response, and one that sleeps a known short time (for the `elapsed_ms` check); the panic case reuses the existing `panicSender` in `chat_test.go`.
4. `bin-ai-manager/docs/operations.md`: in the `Flow Builder (VOIP-1573)` section, describe the log line, its fields and how to read it (the timeout case, the truncated case); state that metrics are unchanged.
5. Verification: `cd bin-ai-manager && go test ./... && golangci-lint run -v --timeout 5m`; commit first, then from the repository root `git fetch origin main` and `bash scripts/check-test-conventions.sh` (the script only checks committed added lines, so on uncommitted work it checks nothing).
6. After merge and deploy (ai-manager only, through CircleCI and Komodo, approval by the CEO): repeat the production scenario with an admin test agent (an extra admin must exist so the test agent can be deleted), read the line per replica, and decide with this table:
   - `truncated` with `completion_tokens` near 8192, or a late `ok` with large tokens: H3 (runaway output).
   - `llm_rate_limit` or `llm_provider_5xx`: H2.
   - `timeout` with zero tokens: H1 or H3 cannot be separated yet. Then, with the CEO's approval, run the controlled experiment. The value 50 is valid (`validateBuilderConfig` rejects only values above 50), but it cannot be set from the Komodo UI: `bin-ai-manager/komodo/docker-compose.yml` lists its `environment:` entries explicitly and has no `AI_BUILDER_*`, and the CircleCI deploy rewrites the stack file from git on every deploy. So the experiment is a one-line compose change (`- AI_BUILDER_LLM_TIMEOUT_SECONDS=50`), a CI deploy, the test, and a revert with another deploy. Because it is a separate change in the same repository, it needs the CEO's decision on a second PR. Side effects to state in that request: the timeout and the concurrency semaphore are shared with the Assistant Builder, a 50 second call leaves only 5 seconds before the 55 second RPC wait, each deploy recreates the containers, and non-Swarm Compose does not do rolling updates, so both replicas are recreated together (`docs/workflows/manager-replica-scaling.md`), which briefly interrupts in-flight builder calls. The experiment is then run; a draft that completes between 40 and 50 seconds is read together with its `completion_tokens` (large tokens mean H3; a slow answer with small tokens points to the provider being slow, H1), and still timing out at 50 seconds means H1 or a longer run.
   - Compare with the Assistant Builder turn on the same agent for the same period.

Logs are read with `docker logs` per replica soon after the test (json-file logging keeps three files of 10 MB, so older lines rotate away).

Rollback of the diagnostics: revert the commit and redeploy (CircleCI build and Komodo deploy); it has no schema, config or API change. Rollback of the experiment is separate: revert the compose line and redeploy.
