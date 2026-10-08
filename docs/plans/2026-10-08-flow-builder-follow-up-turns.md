# VOIP-1578 Flow Builder fails on follow-up question turns

This document holds the issue analysis, the design and the implementation plan of one small change. Reviewers must verify every claim against the code (`git show origin/main:<path>`). The measurements come from a throwaway probe whose raw output is quoted below; the probe file itself is not part of the change.

## 1. Issue analysis

### Observed

- The CEO used the Flow AI Builder in square-admin (production, after VOIP-1577). Turn 1 ("a typical parcel delivery company") got a question. Turn 2 ("I want delivery tracking") ended with "No usable response. Trying again may use another daily attempt." and a Try again button, which is the `BUILDER_RESPONSE_INVALID` message.
- Production diagnostic lines (VOIP-1576) of the last 90 minutes for that conversation: turn 1 `outcome=ok`; every turn with `user_turns` 2, 3 and 4 (5 calls in total) `outcome=invalid_response`, `invalid_kind=unparsable`, `finish_reason=stop`, 2.3 to 3.6 s, 82 to 110 completion tokens, 144 to 193 response chars, `json_mode=object`. The model answered quickly and completely, and the answer was not accepted by the parser.

### Reproduction (local, developer key, `gemini-3.8-flash`, reasoning none, 27 to 30 allowed types, the production `flowMessages` and system prompt; the conversation of the screenshot)

| response_format | history sent as today (assistant turn as plain text) | assistant turn wrapped as `{"message": "..."}` |
|---|---|---|
| `json_object` | 4 of 4 plain Korean sentences, `FlowParse` fails ("no usable response object") | 8 of 8 `{"message": ...}`, `FlowParse` accepts |
| none | 4 of 4 plain sentences, fails | not run |
| `json_schema` (before VOIP-1577) | 4 of 4 JSON, accepted | not run |

The table counts only whether `FlowParse` accepts the answer, not whether the answer carries a draft; that is measured below.

So with `json_object` the provider does not force the answer shape and, on a later turn, the model copies the form of the previous assistant message in the history, which is plain text.

### Second probe: does wrapping the history change when a draft is produced?

The probe file is `/tmp/probe/wrap_probe_test.go` (build tag `flow_probe`, run in a throwaway worktree; not part of the change). It builds the real `flowMessages` output for five conversations with the production system prompt (30 allowed types), `json_object`, `gemini-3.8-flash`, reasoning none, 4 calls per cell, and runs each both as sent today and with the assistant turns wrapped. Wrapped and unwrapped runs are the only difference. The `CurrentDraft` of the change-request cases has two nodes. The history of the "draft then change" cases holds only the visible message of the draft turn, as the client sends it.

| conversation | as sent today: parse ok / draft | wrapped: parse ok / draft |
|---|---|---|
| Korean, question, question, fixed phrase (draft expected) | 4/4, 4/4 | 4/4, 4/4 |
| English, same | 4/4, 4/4 | 4/4, 4/4 |
| Korean, draft turn then a change request with a current draft (draft expected) | 4/4, 4/4 | 4/4, 4/4 |
| English, same | 4/4, 4/4 | 4/4, 4/4 |
| Korean, question, question, a plain answer (no draft expected) | **2/4**, 0/4 | **4/4**, 0/4 |

A third run of the same probe (`/tmp/probe/wrap_probe2_test.go`, same settings, 4 calls per cell) covers the trigger that the prompt names first, the user approving a summary in natural language ("yes, go ahead"):

| conversation | as sent today: parse ok / draft | wrapped: parse ok / draft |
|---|---|---|
| Korean, one question, a summary from the assistant, "응, 그렇게 해줘." (draft expected) | 4/4, 4/4 | 4/4, 4/4 |
| English, same, "Yes, go ahead." | 4/4, 4/4 | 4/4, 4/4 |
| Korean, three questions, a summary, "좋아 그렇게 만들어줘" (5 user turns, draft expected) | 4/4, 4/4 | 4/4, 4/4 |

So in all three triggers (fixed phrase, change request, approval of a summary) wrapping does not make the model leave out the draft after earlier assistant turns (the draft cases are 4/4 in both forms), and it fixes the one case that failed (the plain answer to a question, which is the screenshot case). The rows of the first table other than the wrapped 8 of 8 come from an earlier version of the probe whose file was not kept; with 4 calls per cell they show a direction, and the production log (5 of 5 follow-up turns invalid) is the supporting evidence. How often the unwrapped form fails depends on the wording of the conversation: the screenshot conversation failed 4 of 4 in the first table and a similar one 2 of 4 here. The 8 of 8 in the first table comes from `/tmp/probe/inv_probe_test.go`; the other tables come from the two `wrap_probe` files. The fixed phrase and the change request succeed even without wrapping because the user turn is explicit; the failure sits in the turns where the model has to continue a question dialogue.

### Code facts

- `flowMessages` (`flow_turn.go`) sends each history message as `{Role, Content: m.Content}`; the assistant entries are the visible `message` text that the client stores and sends back (`flowbuilder.ChatRequest.Messages`).
- `FlowParse` accepts a JSON object that has a non-empty `message`; text without such an object returns `ErrInvalidResponse` (`flow_parse.go`).
- Before VOIP-1577 the schema forced the JSON form on every turn, which hid the problem. The change of VOIP-1577 (`FlowConfig` fixes `json_object`) exposed it. Its checks covered a first turn, the fixed phrase and a change request; they did not cover a second question turn. This is a regression of that change.
- The Assistant Builder builds its own messages (`turn.go`) and is not affected.

### Conclusion and not known

- Cause: plain-text assistant turns in the history make the model answer in plain text when no schema is enforced. Wrapping them in the JSON form that the prompt demands fixes the reproduction (8 of 8).
- Not measured: conversations with more than 5 user turns (the checkpoint wording that the prompt adds on later turns is part of that), other models, and the production key. Each cell has 4 calls, so the numbers show a direction, not a rate.
- The earlier claim that VOIP-1577's checks left out the second question turn is an inference from its review record, which lists only the first turn, the fixed phrase and the change request; no multi-turn question case is named there. The post-deploy check repeats the screenshot scenario against production.
- Proceed: yes. Every conversation with a follow-up question fails today, and the fix is small and reversible. A revert of VOIP-1577 is not better: it brings back the missing draft and the runaway generations.

## 2. Design

- `flowMessages` sends each assistant history entry as the JSON object that the prompt asks the model to produce: `{"message": "<the text>"}`, encoded with HTML escaping off (so `&`, `<` stay as written) and non-ASCII text left as is. User entries and the session facts block are unchanged.
- The encoder is `json.Encoder` with HTML escaping off, and its trailing newline is removed (the same `TrimRight` as the session facts block). `U+2028` and `U+2029` are written as `\u2028` and `\u2029`, and invalid UTF-8 is replaced by the character U+FFFD; the output is valid JSON, and a test on invalid UTF-8 must assert the replacement, not equality with the input. An empty message never reaches this code (`ValidateRequest` rejects it).
- This is a server-side change. The client keeps sending the visible text, so the API, the OpenAPI spec and the frontend do not change.
- An assistant entry that is already a JSON object with a `message` is not specially handled: the client only stores the visible message text, never the raw model answer.
- No change to the parser, the prompt, the response format, error mapping, metrics or the Assistant Builder. The parser is not made lenient on purpose: accepting free text as a message would hide a real format failure; if invalid answers continue after this change, that is the next step to design.
- Risk: the model now sees its own earlier turns in the JSON form, which is what the prompt describes, so no new behaviour is asked of it. Messages that contain quotes or newlines are escaped by the encoder.
- Alternative not chosen: add "answer with a JSON object only" to the session facts block of the last user message. It leaves the plain-text history in place, so the model still copies the earlier form, and it adds prompt tokens to every call.
- Rollback: revert the commit and redeploy ai-manager.

## 3. Implementation plan

1. `bin-ai-manager/pkg/builderhandler/flow_turn.go`: in `flowMessages`, encode assistant history entries as `{"message": content}` (a small helper, HTML escaping off) and keep user entries as they are. Update the doc comment of `flowMessages`.
2. Tests in `flow_history_test.go`, reached through the real `Chat` path with the recording sender so the request that is sent is what is checked:
   - an assistant turn with Korean text, a double quote, a backslash, a newline, `&` and `<` is sent as a valid JSON object with exactly one key, `message`, whose value equals the original text (decoded and compared as values); user turns are sent as typed, also when the typed text itself looks like JSON; the last user turn keeps the session facts block;
   - an assistant turn that is already `{"message":"x"}` text is wrapped once more, so the model receives exactly the text the user saw, as a message (documented in the design: the client never stores a raw model answer);
   - a request with assistant turns and a `CurrentDraft` (the next turn of a draft conversation) is sent with wrapped assistant turns and the draft in the facts block;
   - a history without assistant turns is unchanged;
   - the first message of the request is still the system prompt.
   Before the change is applied, the new tests are run once and must fail (the existing tests all pass without the change, so only the new tests pin it); after the change they pass.
3. `bin-ai-manager/docs/operations.md`: one sentence in the Flow Builder section that history assistant turns are sent as JSON and why (VOIP-1578), and that `request_chars` therefore counts the wrapping (about 14 characters plus escapes per assistant turn).
4. Gates: `cd bin-ai-manager && go test ./... && golangci-lint run`; from the repository root, after committing and fetching `origin/main`, `bash scripts/check-test-conventions.sh`.
5. Verification after deploy (read-only plus the minimum LLM calls, a temporary admin that is deleted afterwards, no calls, SMS, email or number purchases): repeat the screenshot conversation (turn 1, a question, turn 2, a question, then the fixed phrase) and the three product scenarios of VOIP-1577. Expected: no `invalid_response`, every turn `outcome=ok`, a draft on the fixed phrase and on a change request after earlier question turns and after a draft turn. Include a conversation of at least 6 user turns and a summary approved in natural language ("yes, go ahead") that must produce a draft.
6. Only `bin-ai-manager/` changes, so CI builds and deploys only ai-manager; the release approval after merge belongs to the CEO.
